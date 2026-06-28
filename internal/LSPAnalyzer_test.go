package internal

import (
	"strings"
	"testing"

	"github.com/Eric-Song-Nop/gocire/internal/lsp"
	"github.com/sourcegraph/scip/bindings/go/scip"
)

func TestTokenInfosFromInlayHintsConvertsLabels(t *testing.T) {
	hints := []lsp.InlayHint{
		{
			Position:     lsp.Position{Line: 1, Character: 7},
			Label:        "i32",
			PaddingLeft:  true,
			PaddingRight: true,
		},
		{
			Position: lsp.Position{Line: 2, Character: 3},
			Label: []interface{}{
				map[string]interface{}{"value": "name"},
				map[string]interface{}{"value": ": "},
			},
		},
		{
			Position: lsp.Position{Line: 3, Character: 5},
			Label: []lsp.InlayHintLabelPart{
				{Value: "Result"},
				{Value: "<T>"},
			},
		},
		{
			Position: lsp.Position{Line: 4, Character: 1},
			Label:    "",
		},
	}

	tokens := tokenInfosFromInlayHints(hints)
	if len(tokens) != 3 {
		t.Fatalf("expected 3 inlay tokens, got %d: %#v", len(tokens), tokens)
	}

	expectedLabels := []string{" i32 ", "name: ", "Result<T>"}
	for i, label := range expectedLabels {
		if tokens[i].InlayHintLabel != label {
			t.Fatalf("token %d label = %q, want %q", i, tokens[i].InlayHintLabel, label)
		}
		if scip.Position.Compare(tokens[i].Span.Start, tokens[i].Span.End) != 0 {
			t.Fatalf("token %d span should be zero-length: %+v", i, tokens[i].Span)
		}
		if tokens[i].Document != nil {
			t.Fatalf("token %d should not use Document for inlay hints: %#v", i, tokens[i].Document)
		}
	}
}

func TestFullDocumentInlayHintRangeTrimsTrailingSplitLine(t *testing.T) {
	r := fullDocumentInlayHintRange([]byte("let x = 1\n"))
	if r.Start.Line != 0 || r.Start.Character != 0 {
		t.Fatalf("range start = %+v, want 0:0", r.Start)
	}
	if r.End.Line != 0 || r.End.Character != len([]rune("let x = 1")) {
		t.Fatalf("range end = %+v, want 0:%d", r.End, len([]rune("let x = 1")))
	}

	r = fullDocumentInlayHintRange([]byte(strings.Join([]string{"first", "second"}, "\n")))
	if r.End.Line != 1 || r.End.Character != len([]rune("second")) {
		t.Fatalf("multi-line range end = %+v, want 1:%d", r.End, len([]rune("second")))
	}
}
