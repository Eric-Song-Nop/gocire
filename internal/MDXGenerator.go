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

	// Use unified template literal format, avoid nesting
	sb.WriteString("<span className=\"cire_text\">{`")
	sb.WriteString(escapeMDXForTemplateLiteral(content))
	sb.WriteString("`}</span>")
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

	// Use unified template literal format
	sb.WriteString("<span className=\"cire_text\">{`")
	sb.WriteString(escapeMDXForTemplateLiteral(content))
	sb.WriteString("`}</span>")
}

func (m *MDXGenerator) outputTokenJSX(token TokenInfo, sb *strings.Builder) {
	content := getSourceFromSpan(m.sourceLines, token.Span)
	escapedContent := escapeMDXForTemplateLiteral(content) // Use template literal escaping

	var cssClass string
	if token.HighlightClass != "" {
		cssClass = token.HighlightClass
	}

	// Build template literal content
	templateContent := "{`" + escapedContent + "`}"

	switch {
	case token.IsDefinition:
		fmt.Fprintf(sb, `<span id="%s" className="%s">%s</span>`,
			escapeMDXAttribute(token.Symbol), cssClass, templateContent)
	case token.IsReference:
		fmt.Fprintf(sb, `<a href="#%s" className="%s">%s</a>`,
			escapeMDXAttribute(token.Symbol), cssClass, templateContent)
	case cssClass != "":
		fmt.Fprintf(sb, `<span className="%s">%s</span>`,
			cssClass, templateContent)
	default:
		sb.WriteString("<span className=\"cire_text\">")
		sb.WriteString(templateContent)
		sb.WriteString("</span>")
	}

	// TODO: don't show inlay hints for now
	if len(token.InlayText) > 0 && false {
		sb.WriteString(" ")
		for _, hint := range token.InlayText {
			sb.WriteString(escapeMDXForTemplateLiteral(hint))
		}
	}
}

// escapeMDXForTemplateLiteral escapes characters for MDX template literal content
// This handles HTML entities and template literal-specific characters
func escapeMDXForTemplateLiteral(text string) string {
	if text == "" {
		return ""
	}

	result := text

	// HTML entity escaping (required)
	result = strings.ReplaceAll(result, "&", "&amp;")
	result = strings.ReplaceAll(result, "<", "&lt;")
	result = strings.ReplaceAll(result, ">", "&gt;")

	// Template literal specific escaping
	result = strings.ReplaceAll(result, "\\", "\\\\") // Escape backslashes
	result = strings.ReplaceAll(result, "`", "\\`")   // Escape backticks
	result = strings.ReplaceAll(result, "${", "\\${") // Prevent variable interpolation

	// Tab normalization (convert to escape sequence)
	result = strings.ReplaceAll(result, "\t", "\\t")

	// Other characters that need explicit handling in template literals
	result = strings.ReplaceAll(result, "\r", "\\r") // Carriage return

	return result
}

// escapeMDXAttribute escapes characters for MDX JSX attribute values
// This handles HTML entities and JSX-specific characters for attributes
func escapeMDXAttribute(text string) string {
	if text == "" {
		return ""
	}

	result := text

	// Complete HTML entity escaping
	result = strings.ReplaceAll(result, "&", "&amp;")
	result = strings.ReplaceAll(result, "<", "&lt;")
	result = strings.ReplaceAll(result, ">", "&gt;")
	// result = strings.ReplaceAll(result, "\"", "&quot;")
	result = strings.ReplaceAll(result, "'", "&#39;")

	// Tab normalization
	result = strings.ReplaceAll(result, "\t", "\\t")

	// JSX special character escaping for attributes
	result = strings.ReplaceAll(result, "{", "\\{")
	result = strings.ReplaceAll(result, "}", "\\}")

	return result
}
