package internal

import (
	"bytes"
	"os"
	"strings"
	"unicode"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CommentAnalyzer struct {
	language string
}

func NewCommentAnalyzer(language string) *CommentAnalyzer {
	return &CommentAnalyzer{
		language: language,
	}
}

func getCommentQuery(language string) (string, error) {
	switch strings.ToLower(language) {
	case "go", "golang", "java", "js", "javascript", "ts", "typescript", "rust", "c", "cpp", "c++", "csharp", "c#", "cs", "php", "dart", "ruby", "python", "py", "haskell":
		return "(comment)+ @comment", nil
	default:
		return "", errors.Newf("unsupported language: %s", language)
	}
}

func cleanNodeContent(content string, language string) string {
	// Helper to clean line comments
	cleanLine := func(text, prefix string) string {
		text = strings.TrimPrefix(text, prefix)
		if strings.HasPrefix(text, " ") {
			text = text[1:]
		}
		return strings.TrimRightFunc(text, unicode.IsSpace)
	}

	// Helper to dedent block comments
	dedent := func(lines []string) []string {
		if len(lines) == 0 {
			return lines
		}
		// Find common indent
		commonIndent := -1
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			indent := 0
			for _, r := range line {
				if unicode.IsSpace(r) {
					indent++
				} else {
					break
				}
			}
			if commonIndent == -1 || indent < commonIndent {
				commonIndent = indent
			}
		}

		if commonIndent <= 0 {
			return lines
		}

		var result []string
		for _, line := range lines {
			if len(line) >= commonIndent {
				result = append(result, line[commonIndent:])
			} else {
				result = append(result, line)
			}
		}
		return result
	}

	switch strings.ToLower(language) {
	case "go", "golang", "java", "js", "javascript", "ts", "typescript", "rust", "c", "cpp", "c++", "csharp", "c#", "cs", "php", "dart":
		if strings.HasPrefix(content, "//") {
			return cleanLine(content, "//")
		}
		if strings.HasPrefix(content, "/*") && strings.HasSuffix(content, "*/") {
			inner := content[2 : len(content)-2]
			lines := strings.Split(inner, "\n")

			// Check if this is a "starred" block (e.g. javadoc)
			isStarred := true
			// verify if all non-empty lines start with *, ignoring initial whitespace
			hasContent := false
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if trimmed == "" {
					continue
				}
				hasContent = true
				if !strings.HasPrefix(trimmed, "*") {
					isStarred = false
					break
				}
			}

			var cleaned []string
			if isStarred && hasContent {
				for _, line := range lines {
					trimmed := strings.TrimLeftFunc(line, unicode.IsSpace)
					if strings.HasPrefix(trimmed, "*") {
						c := strings.TrimPrefix(trimmed, "*")
						if strings.HasPrefix(c, " ") {
							c = c[1:]
						}
						cleaned = append(cleaned, strings.TrimRightFunc(c, unicode.IsSpace))
					} else if strings.TrimSpace(line) == "" {
						// Keep empty lines as empty
						cleaned = append(cleaned, "")
					}
				}

				// Remove leading/trailing empty lines (only the wrapper lines)
				if len(cleaned) > 0 && cleaned[0] == "" {
					cleaned = cleaned[1:]
				}
				if len(cleaned) > 0 && cleaned[len(cleaned)-1] == "" {
					cleaned = cleaned[:len(cleaned)-1]
				}
			} else {
				// Not a starred block, use dedent logic
				// First, remove empty leading/trailing lines which are common in /*\n ... \n*/
				start := 0
				end := len(lines)
				if len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
					start = 1
				}
				if len(lines) > start && strings.TrimSpace(lines[len(lines)-1]) == "" {
					end = len(lines) - 1
				}

				rawLines := lines[start:end]
				// Remove trailing whitespace from each line before dedent
				for i, l := range rawLines {
					rawLines[i] = strings.TrimRightFunc(l, unicode.IsSpace)
				}
				cleaned = dedent(rawLines)
			}
			return strings.Join(cleaned, "\n")
		}
	case "python", "py", "ruby":
		if strings.HasPrefix(content, "#") {
			return cleanLine(content, "#")
		}
	case "haskell":
		if strings.HasPrefix(content, "--") {
			return cleanLine(content, "--")
		}
		if strings.HasPrefix(content, "{- ") && strings.HasSuffix(content, " -}") {
			inner := content[2 : len(content)-2]
			// Haskell blocks can be inline or multi-line.
			// Simple trim for inline, or dedent for multi-line.
			if !strings.Contains(inner, "\n") {
				return strings.TrimSpace(inner)
			}
			lines := strings.Split(inner, "\n")
			// Remove empty first/last lines if present
			start := 0
			end := len(lines)
			if len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
				start = 1
			}
			if len(lines) > start && strings.TrimSpace(lines[len(lines)-1]) == "" {
				end = len(lines) - 1
			}
			rawLines := lines[start:end]
			for i, l := range rawLines {
				rawLines[i] = strings.TrimRightFunc(l, unicode.IsSpace)
			}
			return strings.Join(dedent(rawLines), "\n")
		}
	}
	return strings.TrimSpace(content)
}

func isCommentStandalone(sourceContent []byte, startByte int) bool {
	// Find the start of the current line
	lineStart := bytes.LastIndexByte(sourceContent[:startByte], '\n') + 1
	// Check if everything from line start to comment start is whitespace
	prefix := string(sourceContent[lineStart:startByte])
	return strings.TrimSpace(prefix) == ""
}

func (h *CommentAnalyzer) Analyze(sourcePath string) ([]CommentInfo, error) {
	lang, _, err := GetLanguageAndQuery(h.language)
	if err != nil {
		return nil, err
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read source file %s", sourcePath)
	}

	q, err := getCommentQuery(h.language)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get comment query for %s", h.language)
	}
	query, queryErr := sitter.NewQuery(lang, q)
	if queryErr != nil {
		return nil, errors.Wrapf(queryErr, "comment analyzer failed to create query for %s", h.language)
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	tree := parser.Parse(sourceContent, nil)
	defer tree.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(query, tree.RootNode(), sourceContent)

	var tokens []CommentInfo

	for match := matches.Next(); match != nil; match = matches.Next() {
		if len(match.Captures) == 0 {
			continue
		}

		firstNode := match.Captures[0].Node
		startByte := int(firstNode.StartByte())

		if !isCommentStandalone(sourceContent, startByte) {
			continue
		}

		var contentParts []string
		var start scip.Position
		var end scip.Position
		first := true

		for _, capture := range match.Captures {
			node := capture.Node

			// Capture range
			s := scip.Position{
				Line:      int32(node.StartPosition().Row),
				Character: int32(node.StartPosition().Column),
			}
			e := scip.Position{
				Line:      int32(node.EndPosition().Row),
				Character: int32(node.EndPosition().Column),
			}

			if first {
				start = s
				first = false
			}
			end = e

			startByte := node.StartByte()
			endByte := node.EndByte()
			nodeContent := string(sourceContent[startByte:endByte])
			cleaned := cleanNodeContent(nodeContent, h.language)
			contentParts = append(contentParts, cleaned)
		}

		if !first {
			token := CommentInfo{
				Content: strings.Join(contentParts, "\n"),
				Span: scip.Range{
					Start: start,
					End:   end,
				},
			}
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}
