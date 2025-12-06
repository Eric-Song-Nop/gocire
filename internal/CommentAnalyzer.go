package internal

import (
	"os"
	"strings"

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
	case "go", "golang":
		return "(comment)+ @comment", nil
	default:
		return "", errors.New("unsupported language")
	}
}

func (h *CommentAnalyzer) Analyze(sourcePath string) ([]CommentInfo, error) {
	lang, queryFileName, err := GetLanguageAndQuery(h.language)
	if err != nil {
		return nil, err
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read source file %s", sourcePath)
	}

	query, queryErr := sitter.NewQuery(lang, queryFileName)
	if queryErr != nil {
		return nil, errors.Wrapf(queryErr, "failed to create query for %s", h.language)
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
		for _, capture := range match.Captures {
			node := capture.Node
			token := CommentInfo{
				Content: "",
				Span: scip.Range{
					Start: scip.Position{
						Line:      int32(node.StartPosition().Row),
						Character: int32(node.StartPosition().Column),
					},
					End: scip.Position{
						Line:      int32(node.EndPosition().Row),
						Character: int32(node.EndPosition().Column),
					},
				},
			}
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}
