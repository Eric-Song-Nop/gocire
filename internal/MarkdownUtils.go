package internal

import (
	"bytes"
	"strings"

	katex "github.com/FurqanSoftware/goldmark-katex"
	formatter "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlight "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

type MarkdownHeading struct {
	Level int
	ID    string
	Title string
}

// RenderMarkdown converts a CommonMark string to HTML using Goldmark,
// with CommonMark and GFM extensions enabled.
func RenderMarkdown(input string) string {
	gm := newMarkdownRenderer()

	var buf bytes.Buffer
	if err := gm.Convert([]byte(input), &buf); err != nil {
		return input
	}
	return buf.String()
}

func ExtractMarkdownHeadings(input string) []MarkdownHeading {
	gm := newMarkdownRenderer()
	source := []byte(input)
	document := gm.Parser().Parse(text.NewReader(source), parser.WithContext(parser.NewContext()))

	headings := make([]MarkdownHeading, 0)
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		id := markdownHeadingID(heading)
		title := compactMarkdownHeadingTitle(string(heading.Text(source)))
		if id != "" && title != "" {
			headings = append(headings, MarkdownHeading{
				Level: heading.Level,
				ID:    id,
				Title: title,
			})
		}
		return ast.WalkContinue, nil
	})
	return headings
}

func newMarkdownRenderer() goldmark.Markdown {
	gm := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM, // GitHub Flavored Markdown for tables, task lists, etc.
			&katex.Extender{},
		),
		goldmark.WithExtensions(
			highlight.NewHighlighting(highlight.WithFormatOptions(
				formatter.WithClasses(true),
			), highlight.WithGuessLanguage(true)),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(), // Automatically generate heading IDs
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(), // Allow embedding of raw HTML
		),
	)
	return gm
}

func markdownHeadingID(heading *ast.Heading) string {
	attr, ok := heading.AttributeString("id")
	if !ok {
		return ""
	}
	switch value := attr.(type) {
	case []byte:
		return string(value)
	case string:
		return value
	default:
		return ""
	}
}

func compactMarkdownHeadingTitle(title string) string {
	return strings.Join(strings.Fields(title), " ")
}
