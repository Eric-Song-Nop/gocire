package docs

import (
	cire "github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
	"github.com/sourcegraph/scip/bindings/go/scip"
)

// # gocire overview
//
// `gocire` generates code-centered documentation directly from source files.
// It treats comments, syntax tokens, hover text, inlay hints, and definition
// locations as documentation data instead of asking authors to copy code into a
// separate Markdown file.
//
// The project now has two complementary modes:
//
// - single-file export to Markdown or MDX,
// - project export to an Astro static site.
//
// Project export is the main product direction. Files in `docs` and `blogs`
// become narrative pages. Other supported source files become source context
// pages that readers reach through language navigation, especially cross-file
// jump to definition.
func DefaultDocsiteConfig() *projectconfig.ProjectConfig {
	return projectconfig.DefaultConfig()
}

// The default content model is intentionally conventional. `docs` is the
// documentation tree, `blogs` is the blog tree, and all other included source
// files become source exploration pages under the source route prefix.
func ClassifyDocumentationFile(file project.SourceFile) project.PageKind {
	return file.Kind
}

// The generated site is static HTML. Hover cards use a small JavaScript runtime
// for placement, but source navigation and inlay hints are ordinary markup,
// links, and anchors.
func StaticSourceAnchor(line, column int32) string {
	return cire.LineColumnAnchor(scip.Position{
		Line:      line,
		Character: column,
	})
}
