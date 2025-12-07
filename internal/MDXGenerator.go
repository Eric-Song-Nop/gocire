package internal

import (
	"fmt"
	"strings"
	"unicode" // Add this import

	"github.com/sourcegraph/scip/bindings/go/scip"
)

// MDXGenerator generates MDX (Markdown with JSX) code from source code
// by combining SCIP analysis tokens with syntax highlighting information.
// It produces MDX with React components and proper JSX escaping.
type MDXGenerator struct {
	sourceLines      []string      // Split source code lines for processing
	comments         []CommentInfo // Comments to interleave
	CodeWrapperStart string        // Custom opening HTML/JSX for code blocks
	CodeWrapperEnd   string        // Custom closing HTML/JSX for code blocks
}

// NewMDXGenerator creates a new MDXGenerator instance from the given source lines.
func NewMDXGenerator(sourceLines []string) *MDXGenerator {
	return &MDXGenerator{
		sourceLines:      sourceLines,
		comments:         []CommentInfo{},                  // Initialize empty, will be set by GenerateMDX
		CodeWrapperStart: "<pre><code className=\"cire\">", // Default MDX wrapper
		CodeWrapperEnd:   "</code></pre>",                  // Default MDX wrapper
	}
}

// GenerateMDX generates MDX JSX code with proper escaping for JSX
func (m *MDXGenerator) GenerateMDX(tokens []TokenInfo, comments []CommentInfo) string {
	m.comments = comments
	var sb strings.Builder

	// Calculate file end position
	lastLineIdx := len(m.sourceLines) - 1
	fileEndPos := scip.Position{Line: 0, Character: 0} // Default for empty file
	if len(m.sourceLines) > 0 {
		fileEndPos = scip.Position{Line: int32(lastLineIdx), Character: int32(len([]rune(m.sourceLines[lastLineIdx])))}
	}

	currentPos := scip.Position{Line: 0, Character: 0}
	tokenIdx := 0
	commentIdx := 0
	inCodeBlock := false

	// Helper to get the start position of the next token or "infinity"
	getNextTokenStart := func() scip.Position {
		if tokenIdx < len(tokens) {
			return tokens[tokenIdx].Span.Start
		}
		return scip.Position{Line: 999999, Character: 999999} // "Infinity"
	}

	// Helper to get the start position of the next comment or "infinity"
	getNextCommentStart := func() scip.Position {
		if commentIdx < len(m.comments) {
			return m.comments[commentIdx].Span.Start
		}
		return scip.Position{Line: 999999, Character: 999999} // "Infinity"
	}

	for {
		// Break condition: if currentPos reached fileEndPos AND no more tokens/comments
		if scip.Position.Compare(currentPos, fileEndPos) >= 0 && tokenIdx >= len(tokens) && commentIdx >= len(m.comments) {
			break
		}

		nextTokenStart := getNextTokenStart()
		nextCommentStart := getNextCommentStart()

		// Determine the end of the current gap (code or text)
		gapEnd := fileEndPos
		if scip.Position.Compare(nextTokenStart, gapEnd) < 0 {
			gapEnd = nextTokenStart
		}
		if scip.Position.Compare(nextCommentStart, gapEnd) < 0 {
			gapEnd = nextCommentStart
		}

		// Process gap text (code/plain text)
		if scip.Position.Compare(currentPos, gapEnd) < 0 {
			gapContent := getSourceFromSpan(m.sourceLines, scip.Range{Start: currentPos, End: gapEnd})

			// Trim leading whitespace if starting a new code block (after a comment or at file start)
			if !inCodeBlock {
				gapContent = strings.TrimLeftFunc(gapContent, unicode.IsSpace)
			}

			// If this gap is immediately before a comment, trim trailing whitespace
			if scip.Position.Compare(gapEnd, nextCommentStart) == 0 {
				gapContent = strings.TrimRightFunc(gapContent, unicode.IsSpace)
			}

			if gapContent != "" {
				if !inCodeBlock {
					sb.WriteString(m.CodeWrapperStart)
					sb.WriteString("\n")
					inCodeBlock = true
				}
				sb.WriteString("<span className=\"cire_text\">{`")
				sb.WriteString(escapeMDXForTemplateLiteral(gapContent))
				sb.WriteString("`}</span>")
			}
			currentPos = gapEnd
		}

		// Process next event
		if scip.Position.Compare(currentPos, nextCommentStart) == 0 && scip.Position.Compare(nextCommentStart, nextTokenStart) <= 0 {
			// Current event is a comment (or comment and token start at same pos, prefer comment)
			comment := m.comments[commentIdx]

			// Close code block if open
			if inCodeBlock {
				sb.WriteString(m.CodeWrapperEnd)
				sb.WriteString("\n")
				inCodeBlock = false
			}

			// Output comment content (prose)
			sb.WriteString(comment.Content)
			sb.WriteString("\n") // Add a newline after the comment content

			currentPos = comment.Span.End
			commentIdx++

			// Skip any tokens entirely within this comment's span
			for tokenIdx < len(tokens) && scip.Position.Compare(tokens[tokenIdx].Span.End, currentPos) <= 0 {
				tokenIdx++
			}
		} else if scip.Position.Compare(currentPos, nextTokenStart) == 0 {
			// Current event is a token
			token := tokens[tokenIdx]

			// Open code block if not already in one
			if !inCodeBlock {
				sb.WriteString(m.CodeWrapperStart)
				sb.WriteString("\n")
				inCodeBlock = true
			}

			m.outputTokenJSX(token, &sb)
			currentPos = token.Span.End
			tokenIdx++
		} else if scip.Position.Compare(currentPos, fileEndPos) >= 0 {
			// Reached end of file, and all tokens/comments processed within the loop break conditions.
			break
		} else {
			// This case should ideally not be hit if all tokens/comments are covered and currentPos advances.
			// As a safeguard, advance currentPos to prevent infinite loops if something unexpected occurs.
			currentPos = scip.Position{Line: currentPos.Line, Character: currentPos.Character + 1}
			if currentPos.Character > int32(len(m.sourceLines[currentPos.Line])) {
				currentPos = scip.Position{Line: currentPos.Line + 1, Character: 0}
			}
			if currentPos.Line >= int32(len(m.sourceLines)) {
				currentPos = fileEndPos
			}
		}
	}

	// Final closing for any open code block
	if inCodeBlock {
		sb.WriteString(m.CodeWrapperEnd)
		sb.WriteString("\n")
	}

	return sb.String()
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

	var innerContentBuilder strings.Builder
	switch {
	case token.IsDefinition:
		fmt.Fprintf(&innerContentBuilder, `<span id="%s" className="%s">%s</span>`,
			escapeMDXAttribute(token.Symbol), cssClass, templateContent)
	case token.IsReference:
		fmt.Fprintf(&innerContentBuilder, `<a href="#%s" className="%s">%s</a>`,
			escapeMDXAttribute(token.Symbol), cssClass, templateContent)
	case cssClass != "":
		fmt.Fprintf(&innerContentBuilder, `<span className="%s">%s</span>`,
			cssClass, templateContent)
	default:
		innerContentBuilder.WriteString("<span className=\"cire_text\">")
		innerContentBuilder.WriteString(templateContent)
		innerContentBuilder.WriteString("</span>")
	}

	finalOutput := innerContentBuilder.String()

	if len(token.Document) > 0 {
		doc := strings.Join(token.Document, "\n")
		htmlDoc := RenderMarkdown(doc)
		escapedHTML := escapeForJSTemplateLiteral(htmlDoc)
		// Use rc-tooltip API with dangerouslySetInnerHTML to render HTML from Markdown
		fmt.Fprintf(sb, `<Tooltip overlay={<div className="cire-markdown" dangerouslySetInnerHTML={{ __html: `+"`"+`%s`+"`"+` }} />} placement="top" trigger={['hover']}>%s</Tooltip>`,
			escapedHTML, finalOutput)
	} else {
		sb.WriteString(finalOutput)
	}

	// Inlay hints are currently disabled to reduce output noise
	// To enable: change 'false' to 'true'
	if len(token.Document) > 0 && false {
		sb.WriteString(" ")
		for _, hint := range token.Document {
			sb.WriteString(escapeMDXForTemplateLiteral(hint))
		}
	}
}

// escapeForJSTemplateLiteral escapes characters for a JavaScript template literal (backticked string).
// It does NOT escape HTML entities, allowing safe embedding of HTML strings.
func escapeForJSTemplateLiteral(text string) string {
	if text == "" {
		return ""
	}
	result := text
	// Escape backslashes first!
	result = strings.ReplaceAll(result, "\\", "\\\\")
	// Escape backticks
	result = strings.ReplaceAll(result, "`", "\\`")
	// Escape variable interpolation start
	result = strings.ReplaceAll(result, "${", "\\${")
	return result
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
