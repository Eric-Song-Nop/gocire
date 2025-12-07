package internal

import (
	"testing"
)

func TestCleanNodeContent(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		language string
		want     string
	}{
		// ---------------------------------------------------------------------
		// Line Comments (Go, Java, etc.)
		// ---------------------------------------------------------------------
		{
			name:     "Line comment - simple",
			content:  "// simple",
			language: "go",
			want:     "simple",
		},
		{
			name:     "Line comment - one space",
			content:  "//  one space",
			language: "go",
			want:     " one space",
		},
		{
			name:     "Line comment - no space",
			content:  "//nospace",
			language: "go",
			want:     "nospace",
		},
		{
			name:     "Line comment - trailing space",
			content:  "// trailing   ",
			language: "go",
			want:     "trailing",
		},

		// ---------------------------------------------------------------------
		// Block Comments - Starred (Javadoc style)
		// ---------------------------------------------------------------------
		{
			name:     "Block Starred - Javadoc standard",
			content:  "/**\n * Header\n *   Indented\n */",
			language: "java",
			want:     "Header\n  Indented",
		},
		{
			name:     "Block Starred - Go style",
			content:  "/*\n * Header\n * Body\n */",
			language: "go",
			want:     "Header\nBody",
		},
		{
			name:     "Block Starred - with empty lines",
			content:  "/**\n *\n * Text\n *\n */",
			language: "java",
			want:     "\nText\n",
		},
		{
			name:     "Block Starred - minimal",
			content:  "/** doc */",
			language: "java",
			// inner: "* doc "
			// split lines: ["* doc "]
			// line starts with *? Yes.
			// cleaned: "doc"
			want: "doc",
		},

		// ---------------------------------------------------------------------
		// Block Comments - Raw (Indented style)
		// ---------------------------------------------------------------------
		{
			name:     "Block Raw - Simple",
			content:  "/*\n  Line 1\n  Line 2\n*/",
			language: "go",
			want:     "Line 1\nLine 2",
		},
		{
			name:     "Block Raw - Nested Indent",
			content:  "/*\n  Line 1\n    Line 2\n*/",
			language: "go",
			want:     "Line 1\n  Line 2",
		},
		{
			name:     "Block Raw - Mixed indent",
			content:  "/*\n\tLine 1\n\tLine 2\n*/",
			language: "go",
			want:     "Line 1\nLine 2",
		},
		{
			name:     "Block Raw - Single Line",
			content:  "/* single line */",
			language: "go",
			want:     "single line",
		},

		// ---------------------------------------------------------------------
		// Python / Ruby (#)
		// ---------------------------------------------------------------------
		{
			name:     "Python Line",
			content:  "# comment",
			language: "python",
			want:     "comment",
		},
		{
			name:     "Python Indent",
			content:  "#  comment",
			language: "python",
			want:     " comment",
		},

		// ---------------------------------------------------------------------
		// Haskell
		// ---------------------------------------------------------------------
		{
			name:     "Haskell Line",
			content:  "-- comment",
			language: "haskell",
			want:     "comment",
		},
		{
			name: "Haskell Block Inline",
			// The current implementation strictly checks for "{- " and " -}"
			content:  "{- comment -}",
			language: "haskell",
			want:     "comment",
		},
		{
			name: "Haskell Block Multiline",
			// The current implementation strictly checks for "{- " and " -}"
			content:  "{- \n  Line 1\n  Line 2\n -}",
			language: "haskell",
			want:     "Line 1\nLine 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanNodeContent(tt.content, tt.language)
			if got != tt.want {
				t.Errorf("cleanNodeContent(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}
}

func TestIsCommentStandalone(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		commentChar int // character index of the start of the comment
		want        bool
	}{
		{
			name:        "Standalone - beginning of file",
			source:      "// Comment\nfunc main() {}",
			commentChar: 0,
			want:        true,
		},
		{
			name:        "Standalone - on new line",
			source:      "func init() {\n    // Comment\n}",
			commentChar: 16, // Index of "// Comment"
			want:        true,
		},
		{
			name:        "Standalone - with leading spaces",
			source:      "func init() {\n    // Comment\n}",
			commentChar: 16, // Index of "// Comment"
			want:        true,
		},
		{
			name:        "Inline - after code",
			source:      "func main() { var x int // Comment\n}",
			commentChar: 22, // Index of "// Comment"
			want:        false,
		},
		{
			name:        "Inline - after non-whitespace code",
			source:      "a := 1// Comment",
			commentChar: 7, // Index of "// Comment"
			want:        false,
		},
		{
			name:        "Standalone - only comment in file",
			source:      "// Only comment",
			commentChar: 0,
			want:        true,
		},
		{
			name:        "Standalone - with tabs",
			source:      "\t\t// Comment",
			commentChar: 2, // Index of "// Comment"
			want:        true,
		},
		{
			name:        "Inline - tabs before code",
			source:      "\t\tvar x int // Comment",
			commentChar: 13, // Index of "// Comment"
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isCommentStandalone([]byte(tt.source), tt.commentChar)
			if got != tt.want {
				t.Errorf("isCommentStandalone(%q, %d) = %v, want %v", tt.source, tt.commentChar, got, tt.want)
			}
		})
	}
}

// Ensure the function is exported for testing (if it wasn't already in the same package)
// Since this test is in package 'internal', it can access unexported 'cleanNodeContent'.
