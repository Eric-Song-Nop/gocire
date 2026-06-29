package docs

import (
	"github.com/Eric-Song-Nop/gocire/internal"
	"github.com/Eric-Song-Nop/gocire/internal/project"
	"github.com/sourcegraph/scip/bindings/go/scip"
)

// # Architecture
//
// The implementation is shaped like a compiler pipeline.
//
// Source files are scanned into a project model, each file is analyzed into
// tokens, comments, hover data, inlay hints, and definition locations. A
// backend then renders the shared analysis result into Markdown, MDX, or Astro.
// Adding a new output format should not require a new analyzer.
//
// The important boundary is this: analyzers know about source code and language
// servers, while generators know about output markup. Project-level data such
// as routes, pages, and navigation sits between them.
func SourceRouteManifestForFiles(root string, files []string) (internal.SourceRouteManifest, error) {
	return internal.NewSourceRouteManifestWithPrefix(root, "/_source", files)
}

// Page kind controls presentation, not analysis. A docs page and a source page
// can both use the same token stream, but they render comments differently.
func RenderModeDescription(kind project.PageKind) string {
	switch kind {
	case project.PageKindDocs:
		return "narrative documentation"
	case project.PageKindBlog:
		return "dated narrative post"
	default:
		return "source exploration"
	}
}

// Jump-to-definition uses a route manifest rather than a global semantic index.
// This keeps files independently generatable: a token only needs the definition
// location returned by LSP and a stable source path to route mapping.
func DefinitionAnchorForRoute(route string, line, column int32) string {
	return route + internal.LineColumnAnchor(scip.Position{
		Line:      line,
		Character: column,
	})
}
