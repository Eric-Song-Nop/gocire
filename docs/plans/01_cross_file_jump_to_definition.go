package plans

import (
	cire "github.com/Eric-Song-Nop/gocire/internal"
	"github.com/sourcegraph/scip/bindings/go/scip"
)

// # Cross-file jump to definition
//
// This plan focuses on one language feature: linking a token in one generated
// page to its definition in another generated page.
//
// If LSP reports that a token in `main.go` is defined in
// `internal/store.go`, the generated output for `main.go` should link to the
// generated source page for `internal/store.go`.
//
// ```text
// main.go token
// -> LSP definition: internal/store.go line 12 character 4
// -> generated link: /_source/internal/store.go.html#L13C5
// ```
//
// LSP positions are zero-based. Generated anchors are one-based.
func GeneratedDefinitionHref(route string, line, column int32) string {
	return route + cire.LineColumnAnchor(scip.Position{
		Line:      line,
		Character: column,
	})
}

// Files should remain independently generatable. File A does not need file B to
// have been analyzed or rendered first. It only needs stable mapping rules:
//
// - source path to generated source page route,
// - source position to generated anchor.
//
// That avoids making a global semantic index mandatory for the first project
// export loop.
func BuildSourceRouteMapping(root string, files []string) (cire.SourceRouteManifest, error) {
	return cire.NewSourceRouteManifestWithPrefix(root, "/_source", files)
}

// Missing targets should produce warnings, not failed builds. Common missing
// targets are standard library files, dependencies outside the project root,
// skipped files, or unsupported languages.
func CanLinkDefinition(manifest cire.SourceRouteManifest, relPath string) bool {
	_, ok := manifest.RouteForRelPath(relPath)
	return ok
}
