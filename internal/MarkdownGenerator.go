package internal

import (
	"os"
	"strings"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

type MarkdownGenerator struct {
	sourceLines []string
}

func NewMarkdownGenerator(sourcePath string) (*MarkdownGenerator, error) {
	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	sourceLines := strings.Split(string(sourceContent), "\n")
	return &MarkdownGenerator{
		sourceLines: sourceLines,
	}, nil
}

func (m *MarkdownGenerator) GenerateMarkdown(tokens []TokenInfo) string {
	return "<pre><code class='cire'>" + "\n</code></pre>"
}

func getSourceFromSpan(sourceLines []string, s scip.Range) string {
	startLine := s.Start.Line
	endLine := s.End.Line

	// Check bounds
	if startLine < 0 || endLine < 0 || startLine >= int32(len(sourceLines)) {
		return ""
	}
	if endLine >= int32(len(sourceLines)) {
		endLine = int32(len(sourceLines)) - 1
	}

	// Single line case
	if startLine == endLine {
		if startLine >= int32(len(sourceLines)) {
			return ""
		}
		line := sourceLines[startLine]
		runes := []rune(line)
		startChar := s.Start.Character
		endChar := s.End.Character

		// Check character bounds
		if startChar < 0 || startChar > int32(len(runes)) {
			startChar = 0
		}
		if endChar < 0 || endChar > int32(len(runes)) {
			endChar = int32(len(runes))
		}
		if endChar <= startChar {
			return ""
		}

		return string(runes[startChar:endChar])
	}

	// Multi-line case
	var result strings.Builder

	// First line
	if startLine < int32(len(sourceLines)) {
		firstLine := sourceLines[startLine]
		firstRunes := []rune(firstLine)
		startChar := max(s.Start.Character, 0)
		if startChar <= int32(len(firstRunes)) {
			result.WriteString(string(firstRunes[startChar:]))
		}
		result.WriteString("\n")
	}

	// Middle lines
	for i := startLine + 1; i < endLine; i++ {
		if i < int32(len(sourceLines)) {
			result.WriteString(sourceLines[i])
			result.WriteString("\n")
		}
	}

	// Last line
	if endLine < int32(len(sourceLines)) {
		lastLine := sourceLines[endLine]
		lastRunes := []rune(lastLine)
		endChar := min(max(s.End.Character, 0), int32(len(lastRunes)))
		if endChar > 0 {
			result.WriteString(string(lastRunes[:endChar]))
		}
	}

	return result.String()
}

func escapeHTML(text string) string {
	if text == "" {
		return ""
	}

	result := text
	result = strings.ReplaceAll(result, "&", "&amp;")
	result = strings.ReplaceAll(result, "<", "&lt;")
	result = strings.ReplaceAll(result, ">", "&gt;")
	result = strings.ReplaceAll(result, "\"", "&quot;")
	result = strings.ReplaceAll(result, "'", "&#39;")

	return result
}

func escapeMarkdown(text string) string {
	if text == "" {
		return ""
	}

	result := text
	result = strings.ReplaceAll(result, "\\", "\\\\")
	result = strings.ReplaceAll(result, "`", "\\`")
	result = strings.ReplaceAll(result, "*", "\\*")
	result = strings.ReplaceAll(result, "#", "\\#")
	result = strings.ReplaceAll(result, "+", "\\+")
	result = strings.ReplaceAll(result, "-", "\\-")
	result = strings.ReplaceAll(result, "_", "\\_")
	result = strings.ReplaceAll(result, ".", "\\.")
	result = strings.ReplaceAll(result, "!", "\\!")
	result = strings.ReplaceAll(result, "[", "\\[")
	result = strings.ReplaceAll(result, "]", "\\]")
	result = strings.ReplaceAll(result, "(", "\\(")
	result = strings.ReplaceAll(result, ")", "\\)")
	result = strings.ReplaceAll(result, "{", "\\{")
	result = strings.ReplaceAll(result, "}", "\\}")
	result = strings.ReplaceAll(result, "|", "\\|")
	result = strings.ReplaceAll(result, "^", "\\^")
	result = strings.ReplaceAll(result, "~", "\\~")

	return result
}
