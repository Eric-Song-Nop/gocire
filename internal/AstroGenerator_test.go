package internal

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

func TestGenerateAstroSourceModeKeepsCommentsInCode(t *testing.T) {
	sourceLines := []string{
		"// Intro comment",
		"func main() {",
		"\tprintln(\"hi\")",
		"}",
	}
	gen := NewAstroGenerator(sourceLines)

	output := gen.GenerateAstro(nil, []CommentInfo{
		{
			Content: "Intro comment",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 16},
			},
		},
	}, AstroPageOptions{
		Title:          "Example",
		Kind:           "source",
		Language:       "go",
		SourcePath:     "main.go",
		RenderMode:     AstroRenderModeSource,
		CodePageImport: "../layouts/CodePage.astro",
	})

	expectedParts := []string{
		`import CodePage from "../layouts/CodePage.astro";`,
		`renderMode="source"`,
		`<pre class="cire-code"><code class="cire language-go" data-language="go">`,
		`// Intro comment`,
		`func main() &#123;`,
		`println(&quot;hi&quot;)`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("output missing %q\nGot:\n%s", part, output)
		}
	}

	if strings.Contains(output, "<p>Intro comment</p>") || strings.Contains(output, `class="cire-prose"`) {
		t.Fatalf("source mode rendered standalone comment as prose\nGot:\n%s", output)
	}
	if strings.Count(output, `<pre class="cire-code">`) != 1 {
		t.Fatalf("source mode should render one complete source code block\nGot:\n%s", output)
	}
}

func TestGenerateAstroNarrativeModeInterleavesProseAndCode(t *testing.T) {
	sourceLines := []string{
		"// Intro comment",
		"func main() {",
		"\tprintln(\"hi\")",
		"}",
	}
	gen := NewAstroGenerator(sourceLines)

	output := gen.GenerateAstro([]TokenInfo{
		{
			HighlightClass: "function",
			Span: scip.Range{
				Start: scip.Position{Line: 2, Character: 1},
				End:   scip.Position{Line: 2, Character: 8},
			},
		},
	}, []CommentInfo{
		{
			Content: "Intro comment",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 16},
			},
		},
	}, AstroPageOptions{
		Title:      "Example",
		Kind:       "blog",
		Language:   "go",
		SourcePath: "docs/main.go",
		RenderMode: AstroRenderModeNarrative,
	})

	expectedParts := []string{
		`renderMode="narrative"`,
		`<div class="cire-prose"><p>Intro comment</p>`,
		`<pre class="cire-code"><code class="cire language-go" data-language="go">`,
		`func main() &#123;`,
		`<span class="function">println</span>`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("output missing %q\nGot:\n%s", part, output)
		}
	}

	if strings.Contains(output, "// Intro comment") {
		t.Fatalf("narrative mode should not keep standalone source comment in code\nGot:\n%s", output)
	}
}

func TestGenerateAstroPassesMetadataPropsToCodePage(t *testing.T) {
	gen := NewAstroGenerator([]string{"package main"})

	output := gen.GenerateAstro(nil, nil, AstroPageOptions{
		Title:      "Metadata",
		Kind:       "blog",
		Language:   "go",
		SourcePath: "blogs/post.go",
		Date:       " 2026-06-30 ",
		Tags:       []string{"go", " ", "astro"},
		Author:     " Ada Lovelace ",
		RenderMode: AstroRenderModeSource,
	})

	expectedParts := []string{
		`date="2026-06-30"`,
		`author="Ada Lovelace"`,
		`tags={["go", "astro"]}`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("output missing %q\nGot:\n%s", part, output)
		}
	}
	if strings.Contains(output, `" "`) {
		t.Fatalf("metadata output should omit blank tags\nGot:\n%s", output)
	}
}

func TestGenerateAstroEscapesLinkAnchorAndHoverAttributes(t *testing.T) {
	sourceLines := []string{`foo <bar>`}
	gen := NewAstroGenerator(sourceLines)

	output := gen.GenerateAstro([]TokenInfo{
		{
			HighlightClass: "identifier",
			Href:           `/docs?a=1&b="two"`,
			Anchor:         `def<foo>&"`,
			Document:       []string{`Hover <doc> & "quotes"`, `line {two}`},
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 3},
			},
		},
	}, nil, AstroPageOptions{
		RenderMode: AstroRenderModeSource,
	})

	expectedParts := []string{
		`<a id="def&lt;foo&gt;&amp;&quot;" href="/docs?a=1&amp;b=&quot;two&quot;" class="identifier reference"`,
		`>foo</a>`,
		`&lt;bar&gt;`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("output missing %q\nGot:\n%s", part, output)
		}
	}

	decodedHover := decodeAstroAttributeBase64(t, extractAstroAttribute(t, output, "data-hover"))
	if decodedHover != "Hover <doc> & \"quotes\"\nline {two}" {
		t.Fatalf("data-hover should preserve raw hover markdown\nGot:\n%s", decodedHover)
	}

	decodedHoverHTML := decodeAstroAttributeBase64(t, extractAstroAttribute(t, output, "data-hover-html"))
	if strings.Contains(decodedHoverHTML, "&#123;") || strings.Contains(decodedHoverHTML, "&lt;bar&gt;") {
		t.Fatalf("data-hover-html should contain rendered markdown before Astro text escaping\nGot:\n%s", decodedHoverHTML)
	}

	disallowed := []string{"className=", "<Tooltip", "dangerouslySetInnerHTML"}
	for _, part := range disallowed {
		if strings.Contains(output, part) {
			t.Fatalf("Astro output should not contain React syntax %q\nGot:\n%s", part, output)
		}
	}
}

func TestGenerateAstroOutputsRenderedHoverHTMLAttribute(t *testing.T) {
	sourceLines := []string{`main`}
	gen := NewAstroGenerator(sourceLines)

	hoverMarkdown := strings.Join([]string{
		"**Function** docs",
		"",
		"```go",
		"func main() {}",
		"```",
	}, "\n")

	output := gen.GenerateAstro([]TokenInfo{
		{
			HighlightClass: "function",
			Document:       []string{hoverMarkdown},
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 4},
			},
		},
	}, nil, AstroPageOptions{
		RenderMode: AstroRenderModeSource,
		Language:   "go",
	})

	rawHover := decodeAstroAttributeBase64(t, extractAstroAttribute(t, output, "data-hover"))
	if rawHover != hoverMarkdown {
		t.Fatalf("data-hover should preserve raw hover markdown\nGot:\n%s", rawHover)
	}

	renderedHover := decodeAstroAttributeBase64(t, extractAstroAttribute(t, output, "data-hover-html"))
	expectedParts := []string{
		"<strong>Function</strong>",
		`class="chroma"`,
		"func",
		"main",
	}
	for _, part := range expectedParts {
		if !strings.Contains(renderedHover, part) {
			t.Fatalf("rendered hover HTML missing %q\nGot:\n%s", part, renderedHover)
		}
	}
	if strings.Contains(renderedHover, "```") {
		t.Fatalf("rendered hover HTML should not contain raw code fence\nGot:\n%s", renderedHover)
	}
}

func TestGenerateAstroOutputsInlayHintOnly(t *testing.T) {
	sourceLines := []string{"let value = call()"}
	gen := NewAstroGenerator(sourceLines)

	output := gen.GenerateAstro([]TokenInfo{
		{
			InlayHintLabel: ": <T> {x} &",
			Document:       []string{"hover must not render"},
			Href:           "#bad",
			Anchor:         "bad",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 9},
				End:   scip.Position{Line: 0, Character: 9},
			},
		},
	}, nil, AstroPageOptions{
		RenderMode: AstroRenderModeSource,
	})

	expected := `<span class="inlay-hint" data-inlay-hint aria-hidden="true">: &lt;T&gt; &#123;x&#125; &amp;</span>`
	if !strings.Contains(output, expected) {
		t.Fatalf("output missing escaped inlay hint %q\nGot:\n%s", expected, output)
	}
	for _, unwanted := range []string{"data-hover", "data-hover-html", "href=", "id=\"bad\"", "hover must not render"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("inlay hint output should not contain %q\nGot:\n%s", unwanted, output)
		}
	}
}

func extractAstroAttribute(t *testing.T, output string, attr string) string {
	t.Helper()

	marker := attr + `="`
	start := strings.Index(output, marker)
	if start < 0 {
		t.Fatalf("output missing attribute %s\nGot:\n%s", attr, output)
	}
	start += len(marker)
	end := strings.Index(output[start:], `"`)
	if end < 0 {
		t.Fatalf("unterminated attribute %s\nGot:\n%s", attr, output)
	}
	return output[start : start+end]
}

func decodeAstroAttributeBase64(t *testing.T, value string) string {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		t.Fatalf("attribute is not valid base64: %v\nValue:\n%s", err, value)
	}
	return string(decoded)
}
