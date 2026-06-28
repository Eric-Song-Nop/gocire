package blogs

import (
	"strings"

	cire "github.com/Eric-Song-Nop/gocire/internal"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

// # Shipping the first Astro backend
//
// The Astro backend is a parallel generator backend. It does not replace the
// Markdown or MDX generators, and it does not need a new analysis format. The
// same `sourceLines`, tokens, comments, and resolved links are enough.
//
// This post is intentionally small, but it references real renderer APIs so the
// generated blog page can test hover cards, syntax highlighting, and
// jump-to-definition from narrative code blocks.
func RenderSourcePreview(source string) string {
	lines := strings.Split(source, "\n")
	generator := cire.NewAstroGenerator(lines)

	return generator.GenerateAstro(nil, nil, cire.AstroPageOptions{
		Title:      "Generated source preview",
		Kind:       string(project.PageKindSource),
		Language:   "go",
		SourcePath: "internal/AstroGenerator.go",
		RenderMode: cire.AstroRenderModeSource,
	})
}

// Documentation and blog files are rendered in narrative mode. Source
// exploration files are rendered as complete code, keeping comments inside the
// code block. The classification comes from the project scanner, not from a
// special documentation file format.
func RenderModeForPageKind(kind project.PageKind) cire.AstroRenderMode {
	switch kind {
	case project.PageKindDocs, project.PageKindBlog:
		return cire.AstroRenderModeNarrative
	default:
		return cire.AstroRenderModeSource
	}
}
