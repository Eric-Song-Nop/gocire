package internal

import (
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
		`<a id="def&lt;foo&gt;&amp;&quot;" href="/docs?a=1&amp;b=&quot;two&quot;" class="identifier reference" data-hover="SG92ZXIgPGRvYz4gJiAicXVvdGVzIgpsaW5lIHt0d299">foo</a>`,
		`&lt;bar&gt;`,
	}
	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Fatalf("output missing %q\nGot:\n%s", part, output)
		}
	}

	disallowed := []string{"className=", "<Tooltip", "dangerouslySetInnerHTML"}
	for _, part := range disallowed {
		if strings.Contains(output, part) {
			t.Fatalf("Astro output should not contain React syntax %q\nGot:\n%s", part, output)
		}
	}
}
