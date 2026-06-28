package internal

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

const (
	LinkResolutionWarningInvalidLocation = "invalid_location"
	LinkResolutionWarningOutsideRoot     = "outside_root"
	LinkResolutionWarningMissingRoute    = "missing_route"
)

type LinkResolutionWarning struct {
	Code    string
	Message string
	Path    string
	RelPath string
}

func (w LinkResolutionWarning) String() string {
	if w.Message != "" {
		return w.Message
	}
	if w.RelPath != "" {
		return fmt.Sprintf("%s: %s", w.Code, w.RelPath)
	}
	if w.Path != "" {
		return fmt.Sprintf("%s: %s", w.Code, w.Path)
	}
	return w.Code
}

type SourceRouteManifest struct {
	Root        string
	RoutePrefix string
	Routes      map[string]string
}

func NewSourceRouteManifest(root string, sourcePaths []string) (SourceRouteManifest, error) {
	return NewSourceRouteManifestWithPrefix(root, "/_source", sourcePaths)
}

func NewSourceRouteManifestWithPrefix(root string, routePrefix string, sourcePaths []string) (SourceRouteManifest, error) {
	rootPath, err := cleanAbsPath(root)
	if err != nil {
		return SourceRouteManifest{}, err
	}

	manifest := SourceRouteManifest{
		Root:        rootPath,
		RoutePrefix: normalizeRoutePrefix(routePrefix),
		Routes:      make(map[string]string, len(sourcePaths)),
	}

	for _, sourcePath := range sourcePaths {
		relPath, ok := manifest.RelPathForSourcePath(sourcePath)
		if !ok {
			continue
		}
		manifest.Routes[relPath] = manifest.SourceRoute(relPath)
	}

	return manifest, nil
}

func SourceRoute(relPath string) string {
	return sourceRoute("/_source", relPath)
}

func (m SourceRouteManifest) SourceRoute(relPath string) string {
	return sourceRoute(m.RoutePrefix, relPath)
}

func sourceRoute(routePrefix, relPath string) string {
	routePrefix = normalizeRoutePrefix(routePrefix)
	relPath = normalizeRelPath(relPath)
	if routePrefix == "/" {
		return "/" + relPath + ".html"
	}
	return routePrefix + "/" + relPath + ".html"
}

func LineAnchor(pos scip.Position) string {
	return "#" + LineAnchorID(pos)
}

func LineColumnAnchor(pos scip.Position) string {
	return "#" + LineColumnAnchorID(pos)
}

func LineAnchorID(pos scip.Position) string {
	return fmt.Sprintf("L%d", oneBased(pos.Line))
}

func LineColumnAnchorID(pos scip.Position) string {
	return fmt.Sprintf("L%dC%d", oneBased(pos.Line), oneBased(pos.Character))
}

func SourcePositionAnchor(pos scip.Position) string {
	return LineColumnAnchor(pos)
}

func SourcePositionAnchorID(pos scip.Position) string {
	return LineColumnAnchorID(pos)
}

func ResolveTokenLinks(currentSourcePath string, tokens []TokenInfo, manifest SourceRouteManifest) []LinkResolutionWarning {
	var warnings []LinkResolutionWarning
	for i := range tokens {
		if tokens[i].IsDefinition && tokens[i].Definition != nil {
			tokens[i].Anchor = SourcePositionAnchorID(tokens[i].Span.Start)
			continue
		}
		if tokens[i].Definition == nil {
			continue
		}

		href, ok, warning := ResolveDefinitionHref(currentSourcePath, *tokens[i].Definition, manifest)
		if warning != nil {
			warnings = append(warnings, *warning)
		}
		if ok {
			tokens[i].Href = href
		}
	}
	return warnings
}

func ResolveDefinitionHref(currentSourcePath string, definition SourceLocation, manifest SourceRouteManifest) (href string, ok bool, warning *LinkResolutionWarning) {
	targetPath, ok := definitionSourcePath(definition)
	if !ok {
		return "", false, &LinkResolutionWarning{
			Code:    LinkResolutionWarningInvalidLocation,
			Message: "definition location does not include a file path or file URI",
		}
	}

	targetRelPath, ok := manifest.RelPathForSourcePath(targetPath)
	if !ok {
		return "", false, &LinkResolutionWarning{
			Code:    LinkResolutionWarningOutsideRoot,
			Message: "definition target is outside the source route manifest root",
			Path:    targetPath,
		}
	}

	if currentRelPath, ok := manifest.RelPathForSourcePath(currentSourcePath); ok && currentRelPath == targetRelPath {
		return LineColumnAnchor(definition.Range.Start), true, nil
	}

	route, ok := manifest.RouteForRelPath(targetRelPath)
	if !ok {
		return "", false, &LinkResolutionWarning{
			Code:    LinkResolutionWarningMissingRoute,
			Message: "definition target is not present in the source route manifest",
			Path:    targetPath,
			RelPath: targetRelPath,
		}
	}

	return route + LineColumnAnchor(definition.Range.Start), true, nil
}

func (m SourceRouteManifest) ResolveDefinition(currentSourcePath string, definition SourceLocation) (string, bool) {
	href, ok, _ := ResolveDefinitionHref(currentSourcePath, definition, m)
	return href, ok
}

func (m SourceRouteManifest) RouteFor(relPath string) (string, bool) {
	return m.RouteForRelPath(relPath)
}

func (m SourceRouteManifest) RouteForRelPath(relPath string) (string, bool) {
	route, ok := m.Routes[normalizeRelPath(relPath)]
	return route, ok
}

func (m SourceRouteManifest) RouteForSourcePath(sourcePath string) (route string, relPath string, ok bool) {
	relPath, ok = m.RelPathForSourcePath(sourcePath)
	if !ok {
		return "", "", false
	}
	route, ok = m.RouteForRelPath(relPath)
	return route, relPath, ok
}

func (m SourceRouteManifest) RelPathForSourcePath(sourcePath string) (string, bool) {
	if sourcePath == "" {
		return "", false
	}

	sourcePath = normalizeOSPath(sourcePath)

	var absSourcePath string
	if filepath.IsAbs(sourcePath) {
		absSourcePath = filepath.Clean(sourcePath)
	} else {
		absSourcePath = filepath.Join(m.Root, sourcePath)
	}

	relPath, err := filepath.Rel(m.Root, absSourcePath)
	if err != nil || isOutsideRelPath(relPath) {
		return "", false
	}

	return normalizeRelPath(relPath), true
}

func definitionSourcePath(definition SourceLocation) (string, bool) {
	if definition.Path != "" {
		return definition.Path, true
	}
	if definition.URI == "" {
		return "", false
	}

	parsed, err := url.Parse(definition.URI)
	if err != nil {
		return "", false
	}
	if parsed.Scheme == "" {
		return definition.URI, true
	}
	if parsed.Scheme != "file" {
		return "", false
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		return "//" + parsed.Host + parsed.Path, true
	}
	return parsed.Path, true
}

func cleanAbsPath(pathValue string) (string, error) {
	pathValue = normalizeOSPath(pathValue)
	if pathValue == "" {
		pathValue = "."
	}
	return filepath.Abs(pathValue)
}

func normalizeRelPath(relPath string) string {
	relPath = strings.ReplaceAll(relPath, "\\", "/")
	relPath = path.Clean(relPath)
	if relPath == "." {
		return ""
	}
	relPath = strings.TrimPrefix(relPath, "./")
	relPath = strings.TrimLeft(relPath, "/")
	return relPath
}

func normalizeRoutePrefix(routePrefix string) string {
	routePrefix = strings.TrimSpace(routePrefix)
	if routePrefix == "" {
		return "/"
	}
	routePrefix = path.Clean("/" + strings.TrimLeft(routePrefix, "/"))
	if routePrefix != "/" {
		routePrefix = strings.TrimRight(routePrefix, "/")
	}
	return routePrefix
}

func normalizeOSPath(pathValue string) string {
	return filepath.FromSlash(strings.ReplaceAll(pathValue, "\\", "/"))
}

func isOutsideRelPath(relPath string) bool {
	return relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) || filepath.IsAbs(relPath)
}

func oneBased(value int32) int {
	if value < 0 {
		return 1
	}
	return int(value) + 1
}
