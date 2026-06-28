package internal

import (
	"path/filepath"
	"testing"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

func TestSourceRouteManifestRouteAndAnchorRules(t *testing.T) {
	root := t.TempDir()
	manifest := newTestSourceRouteManifest(t, root, []string{"rel/path.go", `rel\windows.go`})

	route, ok := manifest.RouteFor("rel/path.go")
	if !ok {
		t.Fatal("RouteFor returned ok=false")
	}
	if route != "/_source/rel/path.go.html" {
		t.Fatalf("route = %q, want /_source/rel/path.go.html", route)
	}

	route, ok = manifest.RouteFor("rel/windows.go")
	if !ok {
		t.Fatal("RouteFor returned ok=false for backslash source path")
	}
	if route != "/_source/rel/windows.go.html" {
		t.Fatalf("backslash route = %q, want /_source/rel/windows.go.html", route)
	}

	if route := SourceRoute(`rel\path.go`); route != "/_source/rel/path.go.html" {
		t.Fatalf("SourceRoute with backslashes = %q, want /_source/rel/path.go.html", route)
	}

	if anchor := LineAnchor(scip.Position{Line: 12, Character: 4}); anchor != "#L13" {
		t.Fatalf("LineAnchor = %q, want #L13", anchor)
	}
	if anchor := LineColumnAnchor(scip.Position{Line: 12, Character: 4}); anchor != "#L13C5" {
		t.Fatalf("LineColumnAnchor = %q, want #L13C5", anchor)
	}
}

func TestSourceRouteManifestUsesConfiguredRoutePrefix(t *testing.T) {
	root := t.TempDir()
	manifest, err := NewSourceRouteManifestWithPrefix(root, "/code/", []string{"rel/path.go"})
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	route, ok := manifest.RouteFor("rel/path.go")
	if !ok {
		t.Fatal("RouteFor returned ok=false")
	}
	if route != "/code/rel/path.go.html" {
		t.Fatalf("route = %q, want /code/rel/path.go.html", route)
	}
}

func TestResolveDefinitionHrefSameFileAndCrossFile(t *testing.T) {
	root := t.TempDir()
	manifest := newTestSourceRouteManifest(t, root, []string{
		"rel/current.go",
		"rel/target.go",
	})

	currentSourcePath := filepath.Join(root, "rel", "current.go")
	sameFile := SourceLocation{
		Path: currentSourcePath,
		Range: scip.Range{
			Start: scip.Position{Line: 1, Character: 3},
			End:   scip.Position{Line: 1, Character: 9},
		},
	}
	href, ok, warning := ResolveDefinitionHref(currentSourcePath, sameFile, manifest)
	if !ok {
		t.Fatalf("same-file ResolveDefinitionHref returned ok=false, warning=%v", warning)
	}
	if href != "#L2C4" {
		t.Fatalf("same-file href = %q, want #L2C4", href)
	}

	crossFile := SourceLocation{
		URI: fileURI(filepath.Join(root, "rel", "target.go")),
		Range: scip.Range{
			Start: scip.Position{Line: 9, Character: 0},
			End:   scip.Position{Line: 9, Character: 6},
		},
	}
	href, ok, warning = ResolveDefinitionHref(currentSourcePath, crossFile, manifest)
	if !ok {
		t.Fatalf("cross-file ResolveDefinitionHref returned ok=false, warning=%v", warning)
	}
	if href != "/_source/rel/target.go.html#L10C1" {
		t.Fatalf("cross-file href = %q, want /_source/rel/target.go.html#L10C1", href)
	}

	if href, ok := manifest.ResolveDefinition(currentSourcePath, crossFile); !ok || href != "/_source/rel/target.go.html#L10C1" {
		t.Fatalf("ResolveDefinition = %q, %v; want /_source/rel/target.go.html#L10C1, true", href, ok)
	}
}

func TestResolveDefinitionHrefMissingTargets(t *testing.T) {
	root := t.TempDir()
	currentSourcePath := filepath.Join(root, "rel", "current.go")
	manifest := newTestSourceRouteManifest(t, root, []string{currentSourcePath})

	tests := []struct {
		name       string
		definition SourceLocation
		warning    string
	}{
		{
			name: "inside root but not in manifest",
			definition: SourceLocation{
				Path: filepath.Join(root, "rel", "missing.go"),
				Range: scip.Range{
					Start: scip.Position{Line: 0, Character: 0},
					End:   scip.Position{Line: 0, Character: 1},
				},
			},
			warning: LinkResolutionWarningMissingRoute,
		},
		{
			name: "outside root",
			definition: SourceLocation{
				URI: fileURI(filepath.Join(t.TempDir(), "external.go")),
				Range: scip.Range{
					Start: scip.Position{Line: 0, Character: 0},
					End:   scip.Position{Line: 0, Character: 1},
				},
			},
			warning: LinkResolutionWarningOutsideRoot,
		},
		{
			name: "unsupported uri",
			definition: SourceLocation{
				URI: "https://example.com/external.go",
				Range: scip.Range{
					Start: scip.Position{Line: 0, Character: 0},
					End:   scip.Position{Line: 0, Character: 1},
				},
			},
			warning: LinkResolutionWarningInvalidLocation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			href, ok, warning := ResolveDefinitionHref(currentSourcePath, tt.definition, manifest)
			if ok {
				t.Fatalf("ResolveDefinitionHref returned href=%q, ok=true; want ok=false", href)
			}
			if warning == nil || warning.Code != tt.warning {
				t.Fatalf("warning = %#v, want code %q", warning, tt.warning)
			}
		})
	}
}

func TestResolveTokenLinksOnlySetsDefinitionAnchorWhenLocationExists(t *testing.T) {
	root := t.TempDir()
	currentSourcePath := filepath.Join(root, "rel", "current.go")
	manifest := newTestSourceRouteManifest(t, root, []string{currentSourcePath})

	tokens := []TokenInfo{
		{
			Symbol:       "legacy_symbol",
			IsDefinition: true,
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 6},
			},
		},
		{
			IsDefinition: true,
			Definition: &SourceLocation{
				Path: currentSourcePath,
				Range: scip.Range{
					Start: scip.Position{Line: 1, Character: 2},
					End:   scip.Position{Line: 1, Character: 8},
				},
			},
			Span: scip.Range{
				Start: scip.Position{Line: 1, Character: 2},
				End:   scip.Position{Line: 1, Character: 8},
			},
		},
	}

	warnings := ResolveTokenLinks(currentSourcePath, tokens, manifest)
	if len(warnings) != 0 {
		t.Fatalf("ResolveTokenLinks returned warnings: %#v", warnings)
	}
	if tokens[0].Anchor != "" {
		t.Fatalf("legacy symbol-backed definition Anchor = %q, want empty", tokens[0].Anchor)
	}
	if tokens[1].Anchor != "L2C3" {
		t.Fatalf("location-backed definition Anchor = %q, want L2C3", tokens[1].Anchor)
	}
}

func newTestSourceRouteManifest(t *testing.T, root string, sourcePaths []string) SourceRouteManifest {
	t.Helper()

	manifest, err := NewSourceRouteManifest(root, sourcePaths)
	if err != nil {
		t.Fatalf("NewSourceRouteManifest returned error: %v", err)
	}
	return manifest
}

func fileURI(path string) string {
	return "file://" + filepath.ToSlash(path)
}
