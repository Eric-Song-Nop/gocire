package internal

import (
	"bytes"

	katex "github.com/FurqanSoftware/goldmark-katex"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// RenderMarkdown converts a CommonMark string to HTML using Goldmark,
// with CommonMark and GFM extensions enabled.
func RenderMarkdown(input string) string {
	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown for tables, task lists, etc.
			&katex.Extender{},
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Automatically generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(), // Render newlines as <br>
			html.WithUnsafe(),    // Allow embedding of raw HTML
		),
	)

	var buf bytes.Buffer
	if err := gm.Convert([]byte(input), &buf); err != nil {
		return input
	}
	return buf.String()
}
