package internal

import (
	"testing"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

// Helper function to create a test token
func createTestToken(symbol string, isRef, isDef bool, highlightClass string, startLine, startChar, endLine, endChar int32) TokenInfo {
	return TokenInfo{
		Symbol:         symbol,
		IsReference:    isRef,
		IsDefinition:   isDef,
		HighlightClass: highlightClass,
		Span: scip.Range{
			Start: scip.Position{Line: startLine, Character: startChar},
			End:   scip.Position{Line: endLine, Character: endChar},
		},
	}
}

func TestSortTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []TokenInfo
		expected []TokenInfo
	}{
		{
			name: "already sorted",
			tokens: []TokenInfo{
				createTestToken("a", false, true, "function", 1, 0, 1, 5),
				createTestToken("b", true, false, "variable", 2, 0, 2, 3),
			},
			expected: []TokenInfo{
				createTestToken("a", false, true, "function", 1, 0, 1, 5),
				createTestToken("b", true, false, "variable", 2, 0, 2, 3),
			},
		},
		{
			name: "unsorted tokens",
			tokens: []TokenInfo{
				createTestToken("c", true, false, "variable", 3, 0, 3, 2),
				createTestToken("a", false, true, "function", 1, 0, 1, 5),
				createTestToken("b", true, false, "variable", 2, 0, 2, 3),
			},
			expected: []TokenInfo{
				createTestToken("a", false, true, "function", 1, 0, 1, 5),
				createTestToken("b", true, false, "variable", 2, 0, 2, 3),
				createTestToken("c", true, false, "variable", 3, 0, 3, 2),
			},
		},
		{
			name: "same start line, different end positions (reverse order)",
			tokens: []TokenInfo{
				createTestToken("short", false, true, "function", 1, 0, 1, 3),
				createTestToken("long", false, true, "function", 1, 0, 1, 10),
			},
			expected: []TokenInfo{
				createTestToken("short", false, true, "function", 1, 0, 1, 3),
				createTestToken("long", false, true, "function", 1, 0, 1, 10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SortBySpan(tt.tokens)
			for i, expectedToken := range tt.expected {
				if i >= len(tt.tokens) {
					t.Errorf("Token count mismatch, expected at least %d tokens, got %d", i+1, len(tt.tokens))
					break
				}
				actual := tt.tokens[i]
				if actual.Symbol != expectedToken.Symbol {
					t.Errorf("Token %d symbol mismatch: expected %s, got %s", i, expectedToken.Symbol, actual.Symbol)
				}
				if scip.Position.Compare(actual.Span.Start, expectedToken.Span.Start) != 0 {
					t.Errorf("Token %d start position mismatch: expected %+v, got %+v", i, expectedToken.Span.Start, actual.Span.Start)
				}
			}
		})
	}
}

func TestMergeSplitTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []TokenInfo
		expected []TokenInfo
	}{
		{
			name:     "empty input",
			tokens:   []TokenInfo{},
			expected: []TokenInfo{},
		},
		{
			name: "single token - no overlap",
			tokens: []TokenInfo{
				createTestToken("func", false, true, "function", 1, 0, 1, 10),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 0, 1, 10),
			},
		},
		{
			name: "overlapping tokens on same line",
			tokens: []TokenInfo{
				createTestToken("outer", false, true, "function", 1, 0, 1, 15),
				createTestToken("inner", true, false, "variable", 1, 5, 1, 10),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 0, 1, 5),              // Before inner starts
				createTestToken("inner", true, false, "variable", 1, 5, 1, 10), // Where both are active
				createTestToken("", false, false, "", 1, 10, 1, 15),            // After inner ends
			},
		},
		{
			name: "nested tokens with different properties",
			tokens: []TokenInfo{
				createTestToken("func", false, true, "function", 1, 0, 1, 20),
				createTestToken("var", true, false, "variable", 1, 5, 1, 8),
				createTestToken("keyword", true, false, "keyword", 1, 6, 1, 7),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 0, 1, 5),              // Before var starts
				createTestToken("var", true, false, "variable", 1, 5, 1, 6),    // var only
				createTestToken("keyword", true, false, "keyword", 1, 6, 1, 7), // all three
				createTestToken("var", true, false, "variable", 1, 7, 1, 8),    // func + var
				createTestToken("", false, false, "", 1, 8, 1, 20),             // func only
			},
		},
		{
			name: "tokens on multiple lines",
			tokens: []TokenInfo{
				createTestToken("multiline", false, true, "function", 1, 5, 3, 10),
				createTestToken("inline", true, false, "variable", 2, 0, 2, 15),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 5, 2, 0),               // Before inline starts
				createTestToken("inline", true, false, "variable", 2, 0, 2, 15), // Both active
				createTestToken("", false, false, "", 2, 15, 3, 10),             // After inline ends
			},
		},
		{
			name: "multiple separate tokens",
			tokens: []TokenInfo{
				createTestToken("first", false, true, "function", 1, 0, 1, 5),
				createTestToken("second", true, false, "variable", 1, 10, 1, 15),
				createTestToken("third", false, true, "class", 2, 0, 2, 8),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 0, 1, 5),
				createTestToken("", false, false, "", 1, 10, 1, 15),
				createTestToken("", false, false, "", 2, 0, 2, 8),
			},
		},
		{
			name: "completely overlapping tokens",
			tokens: []TokenInfo{
				createTestToken("large", false, true, "function", 1, 0, 1, 20),
				createTestToken("medium", true, false, "variable", 1, 2, 1, 18),
				createTestToken("small", false, true, "class", 1, 5, 1, 15),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 0, 1, 2),                // large only
				createTestToken("medium", true, false, "variable", 1, 2, 1, 5),   // large + medium
				createTestToken("small", false, true, "class", 1, 5, 1, 15),      // all three
				createTestToken("medium", true, false, "variable", 1, 15, 1, 18), // large + medium
				createTestToken("", false, false, "", 1, 18, 1, 20),              // large only
			},
		},
		{
			name: "adjacent tokens (no overlap)",
			tokens: []TokenInfo{
				createTestToken("first", false, true, "function", 1, 0, 1, 5),
				createTestToken("second", true, false, "variable", 1, 5, 1, 10),
			},
			expected: []TokenInfo{
				createTestToken("", false, false, "", 1, 0, 1, 5),
				createTestToken("", false, false, "", 1, 5, 1, 10),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MergeSplitTokens(tt.tokens)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d segments, got %d", len(tt.expected), len(result))
				t.Logf("Result segments:")
				for i, seg := range result {
					t.Logf("  %d: span=%+v, symbol='%s', ref=%v, def=%v, class='%s'",
						i, seg.Span, seg.Symbol, seg.IsReference, seg.IsDefinition, seg.HighlightClass)
				}
				t.Logf("Expected segments:")
				for i, seg := range tt.expected {
					t.Logf("  %d: span=%+v, symbol='%s', ref=%v, def=%v, class='%s'",
						i, seg.Span, seg.Symbol, seg.IsReference, seg.IsDefinition, seg.HighlightClass)
				}
				return
			}

			for i, expectedSegment := range tt.expected {
				actual := result[i]

				// Check span positions
				if scip.Position.Compare(actual.Span.Start, expectedSegment.Span.Start) != 0 {
					t.Errorf("Segment %d start position mismatch: expected %+v, got %+v",
						i, expectedSegment.Span.Start, actual.Span.Start)
				}
				if scip.Position.Compare(actual.Span.End, expectedSegment.Span.End) != 0 {
					t.Errorf("Segment %d end position mismatch: expected %+v, got %+v",
						i, expectedSegment.Span.End, actual.Span.End)
				}

				// Check properties (only if expected has non-default values)
				if expectedSegment.Symbol != "" && actual.Symbol != expectedSegment.Symbol {
					t.Errorf("Segment %d symbol mismatch: expected '%s', got '%s'",
						i, expectedSegment.Symbol, actual.Symbol)
				}
				if expectedSegment.IsReference && actual.IsReference != expectedSegment.IsReference {
					t.Errorf("Segment %d IsReference mismatch: expected %v, got %v",
						i, expectedSegment.IsReference, actual.IsReference)
				}
				if expectedSegment.IsDefinition && actual.IsDefinition != expectedSegment.IsDefinition {
					t.Errorf("Segment %d IsDefinition mismatch: expected %v, got %v",
						i, expectedSegment.IsDefinition, actual.IsDefinition)
				}
				if expectedSegment.HighlightClass != "" && actual.HighlightClass != expectedSegment.HighlightClass {
					t.Errorf("Segment %d HighlightClass mismatch: expected '%s', got '%s'",
						i, expectedSegment.HighlightClass, actual.HighlightClass)
				}
			}
		})
	}
}

func TestMergeSplitTokensPreservesDefinitionLinkFields(t *testing.T) {
	definition := &SourceLocation{
		URI:  "file:///repo/internal/target.go",
		Path: "/repo/internal/target.go",
		Range: scip.Range{
			Start: scip.Position{Line: 6, Character: 2},
			End:   scip.Position{Line: 6, Character: 8},
		},
	}

	tokens := []TokenInfo{
		{
			Symbol:      "target_symbol",
			IsReference: true,
			Definition:  definition,
			Href:        "/_source/internal/target.go.html#L7C3",
			Anchor:      "L7C3",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 0},
				End:   scip.Position{Line: 0, Character: 6},
			},
		},
		{
			HighlightClass: "function",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 2},
				End:   scip.Position{Line: 0, Character: 4},
			},
		},
	}

	result, err := MergeSplitTokens(tokens)
	if err != nil {
		t.Fatalf("MergeSplitTokens returned error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 split segments, got %d: %#v", len(result), result)
	}

	for i, segment := range result {
		if segment.Definition == nil {
			t.Fatalf("segment %d lost Definition", i)
		}
		if segment.Definition.URI != definition.URI {
			t.Errorf("segment %d Definition.URI = %q, want %q", i, segment.Definition.URI, definition.URI)
		}
		if scip.Position.Compare(segment.Definition.Range.Start, definition.Range.Start) != 0 {
			t.Errorf("segment %d Definition.Range.Start = %+v, want %+v", i, segment.Definition.Range.Start, definition.Range.Start)
		}
		if segment.Href != "/_source/internal/target.go.html#L7C3" {
			t.Errorf("segment %d Href = %q", i, segment.Href)
		}
		if segment.Anchor != "L7C3" {
			t.Errorf("segment %d Anchor = %q", i, segment.Anchor)
		}
	}

	if result[1].HighlightClass != "function" {
		t.Fatalf("overlap segment HighlightClass = %q, want function", result[1].HighlightClass)
	}
}

func TestMergeSplitTokensPreservesPointTokens(t *testing.T) {
	tokens := []TokenInfo{
		createTestToken("value", false, true, "variable", 0, 0, 0, 10),
		{
			InlayHintLabel: ": i32",
			Span: scip.Range{
				Start: scip.Position{Line: 0, Character: 4},
				End:   scip.Position{Line: 0, Character: 4},
			},
		},
		createTestToken("inner", true, false, "function", 0, 6, 0, 8),
	}

	result, err := MergeSplitTokens(tokens)
	if err != nil {
		t.Fatalf("MergeSplitTokens returned error: %v", err)
	}
	if len(result) != 5 {
		t.Fatalf("expected 5 tokens including point token, got %d: %#v", len(result), result)
	}

	point := result[1]
	if point.InlayHintLabel != ": i32" {
		t.Fatalf("point token InlayHintLabel = %q, want : i32", point.InlayHintLabel)
	}
	if scip.Position.Compare(point.Span.Start, point.Span.End) != 0 {
		t.Fatalf("point token span should be zero-length: %+v", point.Span)
	}
	if scip.Position.Compare(point.Span.Start, (scip.Position{Line: 0, Character: 4})) != 0 {
		t.Fatalf("point token position = %+v, want line 0 char 4", point.Span.Start)
	}

	for i, token := range []TokenInfo{result[0], result[2], result[3], result[4]} {
		if token.InlayHintLabel != "" {
			t.Fatalf("ordinary split segment %d unexpectedly inherited inlay label %q", i, token.InlayHintLabel)
		}
	}
	if scip.Position.Compare(result[0].Span.End, point.Span.Start) != 0 ||
		scip.Position.Compare(result[2].Span.Start, point.Span.End) != 0 {
		t.Fatalf("ordinary spans were not split around point token: %#v", result)
	}
}

// Benchmark tests
func BenchmarkMergeSplitTokens(b *testing.B) {
	tokens := []TokenInfo{
		createTestToken("outer", false, true, "function", 1, 0, 1, 100),
		createTestToken("inner1", true, false, "variable", 1, 10, 1, 90),
		createTestToken("inner2", false, true, "class", 1, 20, 1, 80),
		createTestToken("inner3", true, false, "keyword", 1, 30, 1, 70),
	}

	for b.Loop() {
		_, err := MergeSplitTokens(tokens)
		if err != nil {
			b.Fatalf("Unexpected error: %v", err)
		}
	}
}
