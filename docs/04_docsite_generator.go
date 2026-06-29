package docs

import "github.com/Eric-Song-Nop/gocire/internal/project"

// # Docsite generator
//
// The docsite generator is the static-site backend for `gocire`. It is inspired
// by Docusaurus product structure, but it is not a Docusaurus plugin. The first
// bundled site target is Astro.
//
// The generator follows these conventions:
//
// - source files under `docs` become documentation pages,
// - source files under `blogs` become blog posts,
// - every other supported source file becomes a source context page,
// - source context pages are not top-level navigation entries.
//
// A documentation page is still a normal source file. Standalone comments become
// prose, and nearby code becomes semantic code blocks. This keeps examples,
// links, hover text, and definition navigation tied to real source.
func IsNarrativePage(kind project.PageKind) bool {
	return kind == project.PageKindDocs || kind == project.PageKindBlog
}

// The Astro backend owns the website shell: navigation, sidebars, theme,
// tooltip runtime, generated route index, and static assets. It consumes the
// same source analysis as Markdown and MDX rather than inventing a separate
// analysis path.
func IsFirstClassNavigation(kind project.PageKind) bool {
	return kind == project.PageKindDocs || kind == project.PageKindBlog
}

// Configuration starts with defaults. A root `.gocire.yml` can override the
// site title, project root, docs/blog directories, source route prefix, include
// rules, exclude rules, and output directory. The current backend also includes
// sidebar navigation, theme-aware code highlighting, Markdown-rendered hover
// tooltips, and LSP inlay hints.
func DefaultSourceRoutePrefix() string {
	return "/_source"
}
