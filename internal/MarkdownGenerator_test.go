package internal

import (
	"testing"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

// Helper function to create a test span
func createTestSpan(startLine, startChar, endLine, endChar int32) scip.Range {
	return scip.Range{
		Start: scip.Position{Line: startLine, Character: startChar},
		End:   scip.Position{Line: endLine, Character: endChar},
	}
}

func TestGetSourceFromSpan(t *testing.T) {
	sourceLines := []string{
		"package main",
		"",
		"import \"fmt\"",
		"",
		"func main() {",
		"	fmt.Println(\"Hello, World!\")",
		"}",
		"",
		"// End of file",
	}

	tests := []struct {
		name     string
		span     scip.Range
		expected string
	}{
		{
			name:     "out of bounds - start line beyond file",
			span:     createTestSpan(10, 0, 11, 5),
			expected: "",
		},
		{
			name:     "out of bounds - negative line",
			span:     createTestSpan(-1, 0, 0, 5),
			expected: "",
		},
		{
			name:     "single line - normal range",
			span:     createTestSpan(0, 0, 0, 7), // "package"
			expected: "package",
		},
		{
			name:     "single line - entire line",
			span:     createTestSpan(0, 0, 0, 12), // "package main"
			expected: "package main",
		},
		{
			name:     "single line - partial line",
			span:     createTestSpan(0, 8, 0, 12), // "main"
			expected: "main",
		},
		{
			name:     "single line - with out of bounds chars",
			span:     createTestSpan(0, -5, 0, 20), // Should be clamped to full line
			expected: "package main",
		},
		{
			name:     "single line - empty line",
			span:     createTestSpan(1, 0, 1, 5), // Empty line
			expected: "",
		},
		{
			name:     "single line - import statement",
			span:     createTestSpan(2, 8, 2, 11), // "fmt"
			expected: "fmt",
		},
		{
			name:     "multi-line - function definition",
			span:     createTestSpan(4, 0, 6, 1), // "func main() {\n\tfmt.Println(\"Hello, World!\")\n}"
			expected: "func main() {\n\tfmt.Println(\"Hello, World!\")\n}",
		},
		{
			name:     "multi-line - with partial last line",
			span:     createTestSpan(5, 1, 5, 18), // "fmt.Println(\"Hello"
			expected: "fmt.Println(\"Hell",
		},
		{
			name:     "multi-line - from middle to end",
			span:     createTestSpan(5, 1, 6, 1), // "Println(\"Hello, World!\")\n}"
			expected: "fmt.Println(\"Hello, World!\")\n}",
		},
		{
			name:     "multi-line - partial start and end",
			span:     createTestSpan(4, 5, 6, 0), // "main() {\n\tfmt.Println(\"Hello, World!\")"
			expected: "main() {\n\tfmt.Println(\"Hello, World!\")\n",
		},
		{
			name:     "multi-line - with empty lines",
			span:     createTestSpan(1, 0, 3, 6), // "\nimport \"fmt\"\n"
			expected: "\nimport \"fmt\"\n",
		},
		{
			name:     "single character",
			span:     createTestSpan(0, 0, 0, 1), // "p"
			expected: "p",
		},
		{
			name:     "zero length span",
			span:     createTestSpan(0, 5, 0, 5),
			expected: "",
		},
		{
			name:     "span at end of file",
			span:     createTestSpan(8, 0, 8, 16), // "// End of file"
			expected: "// End of file",
		},
		{
			name:     "span with special characters",
			span:     createTestSpan(5, 14, 5, 28), // "Hello, World!\""
			expected: "Hello, World!\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSourceFromSpan(sourceLines, tt.span)
			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestGetSourceFromSpanEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		sourceLines []string
		span        scip.Range
		expected    string
	}{
		{
			name:        "empty source file",
			sourceLines: []string{},
			span:        createTestSpan(0, 0, 0, 5),
			expected:    "",
		},
		{
			name:        "single line source file",
			sourceLines: []string{"hello world"},
			span:        createTestSpan(0, 0, 0, 5),
			expected:    "hello",
		},
		{
			name:        "single character lines",
			sourceLines: []string{"a", "b", "c"},
			span:        createTestSpan(0, 0, 2, 1),
			expected:    "a\nb\nc",
		},
		{
			name:        "lines with unicode characters",
			sourceLines: []string{"你好世界", "テスト"},
			span:        createTestSpan(0, 2, 1, 2),
			expected:    "世界\nテス",
		},
		{
			name:        "very long line",
			sourceLines: []string{"this is a very long line that might cause issues with character indexing and bounds checking"},
			span:        createTestSpan(0, 10, 0, 20),
			expected:    "very long ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getSourceFromSpan(tt.sourceLines, tt.span)
			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

// Benchmark tests
func BenchmarkGetSourceFromSpan_SingleLine(b *testing.B) {
	sourceLines := []string{
		"package main",
		"import \"fmt\"",
		"func main() { fmt.Println(\"Hello\") }",
	}
	span := createTestSpan(1, 7, 1, 11) // "fmt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getSourceFromSpan(sourceLines, span)
	}
}

func BenchmarkGetSourceFromSpan_MultiLine(b *testing.B) {
	sourceLines := []string{
		"package main",
		"",
		"func main() {",
		"	fmt.Println(\"Hello, World!\")",
		"}",
	}
	span := createTestSpan(2, 0, 4, 1) // "func main() {\n\tfmt.Println(\"Hello, World!\")\n}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getSourceFromSpan(sourceLines, span)
	}
}
