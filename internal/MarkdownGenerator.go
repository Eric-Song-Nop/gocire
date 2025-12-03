package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

// MarkdownGenerator generates markdown code from source code
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

// GenerateMarkdown do the Markdown generation process
//
// Make sure that all tokens are sorted and not intersect with each other before generation.
func (m *MarkdownGenerator) GenerateMarkdown(tokens []TokenInfo) string {
	return "<pre><code class='cire'>" + m.generateMarkdownCode(tokens) + "\n</code></pre>"
}

func (m *MarkdownGenerator) generateMarkdownCode(tokens []TokenInfo) string {
	var sb strings.Builder
	currentPos := scip.Position{Line: 0, Character: 0}

	for _, token := range tokens {
		m.outputGapText(currentPos, token.Span.Start, &sb)

		m.outputTokenHTML(token, &sb)
		currentPos = token.Span.End
	}

	m.outputRemainingText(currentPos, &sb)
	return sb.String()
}

func (m *MarkdownGenerator) outputGapText(start, end scip.Position, sb *strings.Builder) {
	if scip.Position.Compare(start, end) == 0 {
		return
	}

	gapRange := scip.Range{Start: start, End: end}
	content := getSourceFromSpan(m.sourceLines, gapRange)

	sb.WriteString(escapeHTML(content))
}

func (m *MarkdownGenerator) outputRemainingText(startPos scip.Position, sb *strings.Builder) {
	if len(m.sourceLines) == 0 {
		return
	}

	lastLineIdx := len(m.sourceLines) - 1
	lastLine := m.sourceLines[lastLineIdx]
	fileEndPos := scip.Position{
		Line:      int32(lastLineIdx),
		Character: int32(len([]rune(lastLine))),
	}

	if scip.Position.Compare(startPos, fileEndPos) >= 0 {
		return
	}

	endRange := scip.Range{Start: startPos, End: fileEndPos}
	content := getSourceFromSpan(m.sourceLines, endRange)
	sb.WriteString(escapeHTML(content))
}

func (m *MarkdownGenerator) outputTokenHTML(token TokenInfo, sb *strings.Builder) {
	content := getSourceFromSpan(m.sourceLines, token.Span)
	escapedContent := escapeHTML(content)

	var cssClass string
	if token.HighlightClass != "" {
		cssClass = token.HighlightClass
	}

	switch {
	case token.IsDefinition:
		fmt.Fprintf(sb, `<span id="%s" class="%s definition">%s</span>`,
			escapeHTML(token.Symbol), cssClass, escapedContent)
	case token.IsReference:
		fmt.Fprintf(sb, `<a href="#%s" class="%s reference">%s</a>`,
			escapeHTML(token.Symbol), cssClass, escapedContent)
	case cssClass != "":
		fmt.Fprintf(sb, `<span class="%s">%s</span>`,
			cssClass, escapedContent)
	default:
		sb.WriteString(escapedContent)
	}

	// TODO: don't show inlay hints for now
	if len(token.InlayText) > 0 && false {
		sb.WriteString(" ")
		for _, hint := range token.InlayText {
			sb.WriteString(escapeHTML(hint))
		}
	}
}

func getSourceFromSpan(sourceLines []string, s scip.Range) string {
	startLine := s.Start.Line
	endLine := s.End.Line

	if startLine < 0 || endLine < 0 || startLine >= int32(len(sourceLines)) {
		return ""
	}
	if endLine >= int32(len(sourceLines)) {
		endLine = int32(len(sourceLines)) - 1
	}

	if startLine == endLine {
		if startLine >= int32(len(sourceLines)) {
			return ""
		}
		line := sourceLines[startLine]
		runes := []rune(line)
		startChar := s.Start.Character
		endChar := s.End.Character

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

	var result strings.Builder

	if startLine < int32(len(sourceLines)) {
		firstLine := sourceLines[startLine]
		firstRunes := []rune(firstLine)
		startChar := max(s.Start.Character, 0)
		if startChar <= int32(len(firstRunes)) {
			result.WriteString(string(firstRunes[startChar:]))
		}
		result.WriteString("\n")
	}

	for i := startLine + 1; i < endLine; i++ {
		if i < int32(len(sourceLines)) {
			result.WriteString(sourceLines[i])
			result.WriteString("\n")
		}
	}

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
