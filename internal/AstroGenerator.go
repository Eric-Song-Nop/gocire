package internal

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/sourcegraph/scip/bindings/go/scip"
)

type AstroRenderMode string

const (
	AstroRenderModeNarrative AstroRenderMode = "narrative"
	AstroRenderModeSource    AstroRenderMode = "source"
)

const defaultAstroCodePageImport = "../components/CodePage.astro"

type AstroPageOptions struct {
	Title          string
	Kind           string
	Language       string
	SourcePath     string
	Date           string
	Tags           []string
	Author         string
	RenderMode     AstroRenderMode
	CodePageImport string
}

type AstroTableOfContentsItem struct {
	Level int
	ID    string
	Title string
}

// AstroGenerator generates complete Astro pages from source analysis data.
type AstroGenerator struct {
	sourceLines []string
}

func NewAstroGenerator(sourceLines []string) *AstroGenerator {
	return &AstroGenerator{
		sourceLines: sourceLines,
	}
}

func (g *AstroGenerator) GenerateAstro(tokens []TokenInfo, comments []CommentInfo, opts AstroPageOptions) string {
	mode := normalizeAstroRenderMode(opts.RenderMode)
	importPath := opts.CodePageImport
	if importPath == "" {
		importPath = defaultAstroCodePageImport
	}
	date := strings.TrimSpace(opts.Date)
	tags := normalizeAstroStringList(opts.Tags)
	author := strings.TrimSpace(opts.Author)

	var body string
	if mode == AstroRenderModeSource {
		body = g.generateSourceAstro(tokens, opts)
	} else {
		body = g.generateNarrativeAstro(tokens, comments, opts)
	}
	toc := astroTableOfContentsForComments(comments, mode)

	var sb strings.Builder
	fmt.Fprintf(&sb, "---\nimport CodePage from %s;\n---\n\n", strconv.Quote(importPath))
	fmt.Fprintf(
		&sb,
		`<CodePage title="%s" kind="%s" language="%s" sourcePath="%s" renderMode="%s"`,
		escapeAstroAttribute(opts.Title),
		escapeAstroAttribute(opts.Kind),
		escapeAstroAttribute(opts.Language),
		escapeAstroAttribute(opts.SourcePath),
		escapeAstroAttribute(string(mode)),
	)
	if date != "" {
		writeAstroAttribute(&sb, "date", date)
	}
	if author != "" {
		writeAstroAttribute(&sb, "author", author)
	}
	if len(tags) > 0 {
		fmt.Fprintf(&sb, " tags={%s}", astroStringArrayLiteral(tags))
	}
	if len(toc) > 0 {
		fmt.Fprintf(&sb, " toc={%s}", astroTableOfContentsLiteral(toc))
	}
	sb.WriteString(">")
	sb.WriteString("\n")
	fmt.Fprintf(
		&sb,
		`<article class="cire-page cire-page--%s" data-kind="%s" data-language="%s" data-source-path="%s">`,
		escapeAstroAttribute(string(mode)),
		escapeAstroAttribute(opts.Kind),
		escapeAstroAttribute(opts.Language),
		escapeAstroAttribute(opts.SourcePath),
	)
	sb.WriteString("\n")
	sb.WriteString(body)
	if body != "" && !strings.HasSuffix(body, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString("</article>\n</CodePage>\n")

	return sb.String()
}

func astroTableOfContentsForComments(comments []CommentInfo, mode AstroRenderMode) []AstroTableOfContentsItem {
	if mode != AstroRenderModeNarrative {
		return nil
	}

	items := make([]AstroTableOfContentsItem, 0)
	titleItems := make([]AstroTableOfContentsItem, 0)
	for _, comment := range comments {
		for _, heading := range ExtractMarkdownHeadings(comment.Content) {
			if heading.Level < 1 || heading.Level > 4 {
				continue
			}
			item := AstroTableOfContentsItem{
				Level: heading.Level,
				ID:    heading.ID,
				Title: heading.Title,
			}
			if heading.Level == 1 {
				titleItems = append(titleItems, item)
				continue
			}
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return titleItems
	}
	return items
}

func astroTableOfContentsLiteral(items []AstroTableOfContentsItem) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, item := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(
			&sb,
			"{ level: %d, id: %s, title: %s }",
			item.Level,
			strconv.Quote(item.ID),
			strconv.Quote(item.Title),
		)
	}
	sb.WriteString("]")
	return sb.String()
}

func normalizeAstroStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func astroStringArrayLiteral(values []string) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, value := range values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strconv.Quote(value))
	}
	sb.WriteString("]")
	return sb.String()
}

func normalizeAstroRenderMode(mode AstroRenderMode) AstroRenderMode {
	switch mode {
	case AstroRenderModeSource:
		return AstroRenderModeSource
	default:
		return AstroRenderModeNarrative
	}
}

func (g *AstroGenerator) generateSourceAstro(tokens []TokenInfo, opts AstroPageOptions) string {
	var sb strings.Builder
	g.openAstroCodeBlock(&sb, opts)
	sb.WriteString(g.generateAstroCode(tokens))
	g.closeAstroCodeBlock(&sb)
	return sb.String()
}

func (g *AstroGenerator) generateNarrativeAstro(tokens []TokenInfo, comments []CommentInfo, opts AstroPageOptions) string {
	var sb strings.Builder
	fileEndPos := g.fileEndPosition()
	currentPos := scip.Position{Line: 0, Character: 0}
	tokenIdx := 0
	commentIdx := 0
	inCodeBlock := false

	nextTokenStart := func() scip.Position {
		if tokenIdx < len(tokens) {
			return tokens[tokenIdx].Span.Start
		}
		return scip.Position{Line: 999999, Character: 999999}
	}
	nextCommentStart := func() scip.Position {
		if commentIdx < len(comments) {
			return comments[commentIdx].Span.Start
		}
		return scip.Position{Line: 999999, Character: 999999}
	}

	for {
		if scip.Position.Compare(currentPos, fileEndPos) >= 0 && tokenIdx >= len(tokens) && commentIdx >= len(comments) {
			break
		}

		tokenStart := nextTokenStart()
		commentStart := nextCommentStart()
		gapEnd := fileEndPos
		if scip.Position.Compare(tokenStart, gapEnd) < 0 {
			gapEnd = tokenStart
		}
		if scip.Position.Compare(commentStart, gapEnd) < 0 {
			gapEnd = commentStart
		}

		if scip.Position.Compare(currentPos, gapEnd) < 0 {
			gapContent := getSourceFromSpan(g.sourceLines, scip.Range{Start: currentPos, End: gapEnd})
			if !inCodeBlock {
				gapContent = strings.TrimLeftFunc(gapContent, unicode.IsSpace)
			}
			if scip.Position.Compare(gapEnd, commentStart) == 0 {
				gapContent = strings.TrimRightFunc(gapContent, unicode.IsSpace)
			}

			if gapContent != "" {
				if !inCodeBlock {
					g.openAstroCodeBlock(&sb, opts)
					inCodeBlock = true
				}
				sb.WriteString(escapeAstroText(gapContent))
			}
			currentPos = gapEnd
		}

		if scip.Position.Compare(currentPos, commentStart) == 0 && scip.Position.Compare(commentStart, tokenStart) <= 0 {
			comment := comments[commentIdx]
			if inCodeBlock {
				g.closeAstroCodeBlock(&sb)
				inCodeBlock = false
			}
			sb.WriteString(`<div class="cire-prose">`)
			sb.WriteString(escapeAstroTemplate(RenderMarkdown(comment.Content)))
			sb.WriteString("</div>\n")

			currentPos = comment.Span.End
			commentIdx++

			for tokenIdx < len(tokens) && scip.Position.Compare(tokens[tokenIdx].Span.End, currentPos) <= 0 {
				tokenIdx++
			}
		} else if scip.Position.Compare(currentPos, tokenStart) == 0 {
			token := tokens[tokenIdx]
			if !inCodeBlock {
				g.openAstroCodeBlock(&sb, opts)
				inCodeBlock = true
			}
			g.outputAstroToken(token, &sb)
			currentPos = token.Span.End
			tokenIdx++
		} else if scip.Position.Compare(currentPos, fileEndPos) >= 0 {
			break
		} else {
			currentPos = g.advancePosition(currentPos, fileEndPos)
		}
	}

	if inCodeBlock {
		g.closeAstroCodeBlock(&sb)
	}

	return sb.String()
}

func (g *AstroGenerator) generateAstroCode(tokens []TokenInfo) string {
	var sb strings.Builder
	currentPos := scip.Position{Line: 0, Character: 0}

	for _, token := range tokens {
		g.outputAstroGap(currentPos, token.Span.Start, &sb)
		g.outputAstroToken(token, &sb)
		currentPos = token.Span.End
	}

	g.outputAstroRemaining(currentPos, &sb)
	return sb.String()
}

func (g *AstroGenerator) openAstroCodeBlock(sb *strings.Builder, opts AstroPageOptions) {
	codeClass := "cire"
	if opts.Language != "" {
		codeClass += " language-" + opts.Language
	}

	sb.WriteString(`<div class="cire-code-block" data-code-block>`)
	fmt.Fprintf(sb, `<pre class="cire-code"><code class="%s"`, escapeAstroAttribute(codeClass))
	if opts.Language != "" {
		fmt.Fprintf(sb, ` data-language="%s"`, escapeAstroAttribute(opts.Language))
	}
	sb.WriteString(">")
}

func (g *AstroGenerator) closeAstroCodeBlock(sb *strings.Builder) {
	sb.WriteString("</code></pre></div>\n")
}

func (g *AstroGenerator) outputAstroGap(start, end scip.Position, sb *strings.Builder) {
	if scip.Position.Compare(start, end) == 0 {
		return
	}

	content := getSourceFromSpan(g.sourceLines, scip.Range{Start: start, End: end})
	sb.WriteString(escapeAstroText(content))
}

func (g *AstroGenerator) outputAstroRemaining(startPos scip.Position, sb *strings.Builder) {
	fileEndPos := g.fileEndPosition()
	if scip.Position.Compare(startPos, fileEndPos) >= 0 {
		return
	}

	content := getSourceFromSpan(g.sourceLines, scip.Range{Start: startPos, End: fileEndPos})
	sb.WriteString(escapeAstroText(content))
}

func (g *AstroGenerator) outputAstroToken(token TokenInfo, sb *strings.Builder) {
	if token.InlayHintLabel != "" {
		sb.WriteString(`<span class="inlay-hint" data-inlay-hint aria-hidden="true">`)
		sb.WriteString(escapeAstroText(token.InlayHintLabel))
		sb.WriteString("</span>")
		return
	}

	content := getSourceFromSpan(g.sourceLines, token.Span)
	escapedContent := escapeAstroText(content)

	cssClass := token.HighlightClass
	id := token.Anchor
	href := token.Href
	encodedHover, encodedHoverHTML, hasHover := encodeAstroHover(token.Document)

	switch {
	case href != "":
		referenceClass := strings.TrimSpace(cssClass + " reference")
		sb.WriteString("<a")
		if id != "" {
			writeAstroAttribute(sb, "id", id)
		}
		writeAstroAttribute(sb, "href", href)
		if referenceClass != "" {
			writeAstroAttribute(sb, "class", referenceClass)
		}
		if hasHover {
			writeAstroHoverAttributes(sb, encodedHover, encodedHoverHTML)
		}
		sb.WriteString(">")
		sb.WriteString(escapedContent)
		sb.WriteString("</a>")
	case id != "":
		definitionClass := strings.TrimSpace(cssClass + " definition")
		sb.WriteString("<span")
		writeAstroAttribute(sb, "id", id)
		if definitionClass != "" {
			writeAstroAttribute(sb, "class", definitionClass)
		}
		if hasHover {
			writeAstroHoverAttributes(sb, encodedHover, encodedHoverHTML)
		}
		sb.WriteString(">")
		sb.WriteString(escapedContent)
		sb.WriteString("</span>")
	case cssClass != "" || hasHover:
		sb.WriteString("<span")
		if cssClass != "" {
			writeAstroAttribute(sb, "class", cssClass)
		}
		if hasHover {
			writeAstroHoverAttributes(sb, encodedHover, encodedHoverHTML)
		}
		sb.WriteString(">")
		sb.WriteString(escapedContent)
		sb.WriteString("</span>")
	default:
		sb.WriteString(escapedContent)
	}
}

func encodeAstroHover(document []string) (encodedRaw string, encodedHTML string, ok bool) {
	hover := strings.Join(document, "\n")
	if hover == "" {
		return "", "", false
	}

	renderedHover := RenderMarkdown(hover)
	return base64.StdEncoding.EncodeToString([]byte(hover)),
		base64.StdEncoding.EncodeToString([]byte(renderedHover)),
		true
}

func writeAstroHoverAttributes(sb *strings.Builder, encodedHover string, encodedHoverHTML string) {
	writeAstroAttribute(sb, "data-hover", encodedHover)
	writeAstroAttribute(sb, "data-hover-html", encodedHoverHTML)
}

func (g *AstroGenerator) fileEndPosition() scip.Position {
	if len(g.sourceLines) == 0 {
		return scip.Position{Line: 0, Character: 0}
	}

	lastLineIdx := len(g.sourceLines) - 1
	return scip.Position{
		Line:      int32(lastLineIdx),
		Character: int32(len([]rune(g.sourceLines[lastLineIdx]))),
	}
}

func (g *AstroGenerator) advancePosition(currentPos, fileEndPos scip.Position) scip.Position {
	if len(g.sourceLines) == 0 {
		return fileEndPos
	}

	next := scip.Position{Line: currentPos.Line, Character: currentPos.Character + 1}
	if currentPos.Line >= 0 && currentPos.Line < int32(len(g.sourceLines)) && next.Character > int32(len([]rune(g.sourceLines[currentPos.Line]))) {
		next = scip.Position{Line: currentPos.Line + 1, Character: 0}
	}
	if scip.Position.Compare(next, fileEndPos) > 0 || next.Line >= int32(len(g.sourceLines)) {
		return fileEndPos
	}
	return next
}

func writeAstroAttribute(sb *strings.Builder, name string, value string) {
	fmt.Fprintf(sb, ` %s="%s"`, name, escapeAstroAttribute(value))
}

func escapeAstroText(text string) string {
	return escapeAstroTemplate(escapeHTML(text))
}

func escapeAstroTemplate(text string) string {
	if text == "" {
		return ""
	}

	result := text
	result = strings.ReplaceAll(result, "{", "&#123;")
	result = strings.ReplaceAll(result, "}", "&#125;")
	return result
}

func escapeAstroAttribute(text string) string {
	if text == "" {
		return ""
	}

	result := escapeAstroTemplate(escapeHTML(text))
	result = strings.ReplaceAll(result, "\n", "&#10;")
	result = strings.ReplaceAll(result, "\r", "&#13;")
	result = strings.ReplaceAll(result, "\t", "&#9;")
	return result
}
