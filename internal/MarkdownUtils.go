package internal

import (
	"bytes"
	"fmt"
	"strings"
	"unicode"

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

type MarkdownPageRenderer struct {
	gm       goldmark.Markdown
	slugs    *headingSlugger
	headings []MarkdownHeading
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

func NewMarkdownPageRenderer() *MarkdownPageRenderer {
	return &MarkdownPageRenderer{
		gm:    newMarkdownPageRenderer(),
		slugs: newHeadingSlugger(),
	}
}

func (r *MarkdownPageRenderer) RenderFragment(input string) string {
	if r == nil {
		return RenderMarkdown(input)
	}

	source := []byte(input)
	document := r.gm.Parser().Parse(text.NewReader(source), parser.WithContext(parser.NewContext()))
	r.assignHeadingIDs(document, source)

	var buf bytes.Buffer
	if err := r.gm.Renderer().Render(&buf, source, document); err != nil {
		return input
	}
	return buf.String()
}

func (r *MarkdownPageRenderer) Headings() []MarkdownHeading {
	if r == nil || len(r.headings) == 0 {
		return nil
	}
	headings := make([]MarkdownHeading, len(r.headings))
	copy(headings, r.headings)
	return headings
}

func (r *MarkdownPageRenderer) assignHeadingIDs(document ast.Node, source []byte) {
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		heading, ok := node.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		title := compactMarkdownHeadingTitle(string(heading.Text(source)))
		id := markdownHeadingID(heading)
		if id == "" {
			id = r.slugs.Unique(markdownHeadingSlug(title))
		} else {
			id = r.slugs.Unique(id)
		}
		heading.SetAttribute([]byte("id"), []byte(id))

		if title != "" {
			r.headings = append(r.headings, MarkdownHeading{
				Level: heading.Level,
				ID:    id,
				Title: title,
			})
		}
		return ast.WalkContinue, nil
	})
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
	return newMarkdownRendererWithParserOptions(parser.WithAutoHeadingID())
}

func newMarkdownPageRenderer() goldmark.Markdown {
	return newMarkdownRendererWithParserOptions(parser.WithHeadingAttribute())
}

func newMarkdownRendererWithParserOptions(opts ...parser.Option) goldmark.Markdown {
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
		goldmark.WithParserOptions(opts...),
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

type headingSlugger struct {
	used   map[string]bool
	counts map[string]int
}

func newHeadingSlugger() *headingSlugger {
	return &headingSlugger{
		used:   make(map[string]bool),
		counts: make(map[string]int),
	}
}

func (s *headingSlugger) Unique(base string) string {
	if s == nil {
		return base
	}
	base = strings.Trim(base, "-")
	if base == "" {
		base = "heading"
	}

	next := s.counts[base]
	for {
		candidate := base
		if next > 0 {
			candidate = fmt.Sprintf("%s-%d", base, next)
		}
		if !s.used[candidate] {
			s.used[candidate] = true
			s.counts[base] = next + 1
			return candidate
		}
		next++
	}
}

func markdownHeadingSlug(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}

	var sb strings.Builder
	lastDash := false
	for _, r := range title {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			sb.WriteRune(unicode.ToLower(r))
			lastDash = false
			continue
		}
		if sb.Len() > 0 && !lastDash {
			sb.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(sb.String(), "-")
}
