package internal

import (
	"strings"
	"testing"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

func TestNewMarkdownGenerator(t *testing.T) {
	content := "line1\nline2\nline3"
	sourceLines := strings.Split(content, "\n")

	t.Run("Success", func(t *testing.T) {
		gen := NewMarkdownGenerator(sourceLines)
		if len(gen.sourceLines) != 3 {
			t.Errorf("Expected 3 lines, got %d", len(gen.sourceLines))
		}
		if gen.sourceLines[0] != "line1" {
			t.Errorf("Expected line1, got %s", gen.sourceLines[0])
		}
	})
}

func TestGenerateMarkdown(t *testing.T) {
	// Prepare source content
	// Line 0: package main
	// Line 1:
	// Line 2:	func main() {
	// Line 3:	    print("Hello")
	// Line 4:	}
	content := "package main\n\nfunc main() {\n\tprint(\"Hello\")\n}"
	sourceLines := strings.Split(content, "\n")

	gen := NewMarkdownGenerator(sourceLines)

	// Define tokens
	// Token 1: "package" keyword
	// Line 0, chars 0-7

	token1 := TokenInfo{
		HighlightClass: "keyword",
		Span: scip.Range{
			Start: scip.Position{Line: 0, Character: 0},
			End:   scip.Position{Line: 0, Character: 7},
		},
	}

	// Token 2: "main" definition
	// Line 2, chars 5-9

	token2 := TokenInfo{
		Symbol:         "main_func",
		IsDefinition:   true,
		HighlightClass: "function",
		Span: scip.Range{
			Start: scip.Position{Line: 2, Character: 5},
			End:   scip.Position{Line: 2, Character: 9},
		},
	}

	// Token 3: "print" reference
	// Line 3, chars 1-6 (tab is char 0)

	token3 := TokenInfo{
		Symbol:         "print_ref",
		IsReference:    true,
		HighlightClass: "builtin",
		Span: scip.Range{
			Start: scip.Position{Line: 3, Character: 1},
			End:   scip.Position{Line: 3, Character: 6},
		},
	}

	tokens := []TokenInfo{token1, token2, token3}

	// Generate markdown
	output := gen.GenerateMarkdown(tokens)

	// Verify output
	// We expect:
	// <pre><code class='cire'>
	// <span class="keyword">package</span> main
	//
	// func <span id="main_func" class="function definition">main</span>() {
	// 	<a href="#print_ref" class="builtin reference">print</a>("Hello")
	// }
	// </code></pre>

	// Note: escaping might happen.
	// "package" -> class="keyword"
	// "main" (line 0) -> plain text
	// "func" -> plain text (since no token for it in this test)
	// "main" (line 2) -> definition span
	// "print" -> reference link

	expectedParts := []string{
		"<pre><code class='cire'>",
		`<span class="keyword">package</span>`,
		" main\n\nfunc ",
		`<span id="main_func" class="function definition">main</span>`,
		"() {\n\t",
		`<a href="#print_ref" class="builtin reference">print</a>`,
		"(&quot;Hello&quot;)\n}",
		"</code></pre>",
	}

	for _, part := range expectedParts {
		if !strings.Contains(output, part) {
			t.Errorf("Output missing expected part: %q\nGot:\n%q", part, output)
		}
	}
}

func TestGenerateMarkdownUsesResolvedHrefAndAnchor(t *testing.T) {
	sourceLines := []string{"foo"}
	gen := NewMarkdownGenerator(sourceLines)

	output := gen.GenerateMarkdown([]TokenInfo{
		{
			HighlightClass: "function",
			Href:           "/_source/internal/foo.go.html#L1C1",
			Anchor:         "L1C1",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 3},
			},
		},
	})

	expected := `<a id="L1C1" href="/_source/internal/foo.go.html#L1C1" class="function reference">foo</a>`
	if !strings.Contains(output, expected) {
		t.Fatalf("output missing resolved link %q\nGot:\n%s", expected, output)
	}
}

func TestGenerateMDXUsesResolvedHrefAndAnchor(t *testing.T) {
	sourceLines := []string{"foo"}
	gen := NewMDXGenerator(sourceLines)

	output := gen.GenerateMDX([]TokenInfo{
		{
			HighlightClass: "function",
			Href:           "/_source/internal/foo.go.html#L1C1",
			Anchor:         "L1C1",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 3},
			},
		},
	}, nil)

	expectedStart := `<a id="L1C1" href="/_source/internal/foo.go.html#L1C1" className="function reference">`
	if !strings.Contains(output, expectedStart) || !strings.Contains(output, "{`foo`}</a>") {
		t.Fatalf("output missing resolved MDX link\nGot:\n%s", output)
	}
}

func TestGenerateMDXOutputsInlayHintOnly(t *testing.T) {
	sourceLines := []string{"let value = call()"}
	gen := NewMDXGenerator(sourceLines)

	output := gen.GenerateMDX([]TokenInfo{
		{
			InlayHintLabel: "`${type}`",
			Document:       []string{"hover must not render"},
			Href:           "#bad",
			Anchor:         "bad",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 9},
				End:   scip.Position{Line: 0, Character: 9},
			},
		},
	}, nil)

	if !strings.Contains(output, `className="inlay-hint"`) {
		t.Fatalf("output missing inlay hint span\nGot:\n%s", output)
	}
	if !strings.Contains(output, `\${type}`) {
		t.Fatalf("output did not escape template interpolation\nGot:\n%s", output)
	}
	if strings.Count(output, "\\`") < 2 {
		t.Fatalf("output did not escape inlay hint backticks\nGot:\n%s", output)
	}
	for _, unwanted := range []string{"className=\"cire_text\">{``}</span>", "<Tooltip", "href=", "id=\"bad\"", "hover must not render"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("inlay hint output should not contain %q\nGot:\n%s", unwanted, output)
		}
	}
}

func TestGenerateMarkdownOutputsInlayHint(t *testing.T) {
	sourceLines := []string{"let value = call()"}
	gen := NewMarkdownGenerator(sourceLines)

	output := gen.GenerateMarkdown([]TokenInfo{
		{
			InlayHintLabel: ": <T> &",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 9},
				End:   scip.Position{Line: 0, Character: 9},
			},
		},
	})

	expected := `<span class="inlay-hint">: &lt;T&gt; &amp;</span>`
	if !strings.Contains(output, expected) {
		t.Fatalf("output missing escaped inlay hint %q\nGot:\n%s", expected, output)
	}
}

func TestGetSourceFromSpan(t *testing.T) {
	lines := []string{
		"line0",
		"line1",
		"line2",
	}

	tests := []struct {
		name     string
		span     scip.Range
		expected string
	}{
		{
			name: "SingleLine",
			span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 4},
			},
			expected: "line",
		},
		{
			name: "SingleLineFull",
			span: scip.Range{
				Start: scip.Position{Line: 1, Character: 0},
				End:   scip.Position{Line: 1, Character: 5},
			},
			expected: "line1",
		},
		{
			name: "MultiLine",
			span: scip.Range{
				Start: scip.Position{Line: 0, Character: 4}, // "0"
				End:   scip.Position{Line: 1, Character: 4}, // "line"
			},
			expected: "0\nline",
		},
		{
			name: "MultiLineFull",
			span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 2, Character: 5},
			},
			expected: "line0\nline1\nline2",
		},
		{
			name: "OutOfBoundsStartLine",
			span: scip.Range{
				Start: scip.Position{Line: -1, Character: 0},
				End:   scip.Position{Line: 0, Character: 1},
			},
			expected: "",
		},
	}

	// Since getSourceFromSpan is private (lowercase), we cannot test it directly from a test package usually.
	// However, since we are in package internal (same package), we can access it.
	// Note: the file package declaration must match.

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We need to expose or copy the logic, OR just verify it via public methods?
			// But wait, I am writing this in package internal, so I can access private functions.
			result := getSourceFromSpan(lines, tt.span)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}
