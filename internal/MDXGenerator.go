package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

// MDXGenerator generates MDX JSX code from source code
type MDXGenerator struct {
	sourceLines []string
}

func NewMDXGenerator(sourcePath string) (*MDXGenerator, error) {
	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, err
	}
	sourceLines := strings.Split(string(sourceContent), "\n")
	return &MDXGenerator{
		sourceLines: sourceLines,
	}, nil
}

// GenerateMDX generates MDX JSX code with proper escaping for JSX
func (m *MDXGenerator) GenerateMDX(tokens []TokenInfo) string {
	var sb strings.Builder

	// Start the JSX component
	sb.WriteString("<pre><code className=\"cire\">\n")

	currentPos := scip.Position{Line: 0, Character: 0}

	for _, token := range tokens {
		m.outputGapText(currentPos, token.Span.Start, &sb)
		m.outputTokenJSX(token, &sb)
		currentPos = token.Span.End
	}

	m.outputRemainingText(currentPos, &sb)

	// Close the JSX component
	sb.WriteString("</code></pre>\n")

	return sb.String()
}

func (m *MDXGenerator) outputGapText(start, end scip.Position, sb *strings.Builder) {
	if scip.Position.Compare(start, end) == 0 {
		return
	}

	gapRange := scip.Range{Start: start, End: end}
	content := getSourceFromSpan(m.sourceLines, gapRange)

	// Escape the content for JSX
	sb.WriteString(escapeMDX(content))
}

func (m *MDXGenerator) outputRemainingText(startPos scip.Position, sb *strings.Builder) {
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
	sb.WriteString(escapeMDX(content))
}

func (m *MDXGenerator) outputTokenJSX(token TokenInfo, sb *strings.Builder) {
	content := getSourceFromSpan(m.sourceLines, token.Span)
	escapedContent := escapeMDX(content)

	var cssClass string
	if token.HighlightClass != "" {
		cssClass = token.HighlightClass
	}

	switch {
	case token.IsDefinition:
		fmt.Fprintf(sb, `<span id="%s" className="%s">%s</span>`,
			escapeMDX(token.Symbol), cssClass, escapedContent)
	case token.IsReference:
		fmt.Fprintf(sb, `<a href="#%s" className="%s">%s</a>`,
			escapeMDX(token.Symbol), cssClass, escapedContent)
	case cssClass != "":
		fmt.Fprintf(sb, `<span className="%s">%s</span>`,
			cssClass, escapedContent)
	default:
		sb.WriteString(escapedContent)
	}

	// TODO: don't show inlay hints for now
	if len(token.InlayText) > 0 && false {
		sb.WriteString(" ")
		for _, hint := range token.InlayText {
			sb.WriteString(escapeMDX(hint))
		}
	}
}

// escapeMDX escapes characters for MDX JSX content
// This handles HTML entities, JSX-specific characters, and Markdown conflicts
func escapeMDX(text string) string {
	if text == "" {
		return ""
	}

	result := text

	// HTML entities (must be escaped for valid HTML/JSX) - do this first
	result = strings.ReplaceAll(result, "&", "&amp;")
	result = strings.ReplaceAll(result, "<", "&lt;")
	result = strings.ReplaceAll(result, ">", "&gt;")
	result = strings.ReplaceAll(result, "\"", "&quot;")
	result = strings.ReplaceAll(result, "'", "\\'")

	// JSX-specific characters that could break JSX parsing
	result = strings.ReplaceAll(result, "{", "\\{")
	result = strings.ReplaceAll(result, "}", "\\}")

	// Markdown conflicts that could interfere with MDX parsing
	result = strings.ReplaceAll(result, "*", "\\*") // Prevent italic/bold parsing
	result = strings.ReplaceAll(result, "#", "\\#") // Prevent heading parsing
	result = strings.ReplaceAll(result, "`", "\\`") // Prevent inline code parsing
	result = strings.ReplaceAll(result, "|", "\\|") // Prevent table parsing

	return result
}
