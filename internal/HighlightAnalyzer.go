package internal

import (
	"embed"

	"github.com/Eric-Song-Nop/gocire/internal/languages"
	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

//go:embed queries/*.scm
var queryFS embed.FS

type HighlightAnalyzer struct {
	language string
}

func NewHighlightAnalyzer(language string) *HighlightAnalyzer {
	return &HighlightAnalyzer{
		language: language,
	}
}

func (h *HighlightAnalyzer) Analyze(sourceContent []byte) ([]TokenInfo, error) {
	cfg, err := languages.GetConfig(h.language)
	if err != nil {
		return nil, err
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cfg.SitterLanguage)

	tree := parser.Parse(sourceContent, nil)
	defer tree.Close()

	queryContent, err := queryFS.ReadFile("queries/" + cfg.QueryFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read query file %s", cfg.QueryFileName)
	}

	query, queryErr := sitter.NewQuery(cfg.SitterLanguage, string(queryContent))
	if queryErr != nil {
		return nil, errors.Wrapf(queryErr, "failed to create query for %s", h.language)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(query, tree.RootNode(), sourceContent)

	var tokens []TokenInfo

	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			node := capture.Node
			token := TokenInfo{
				Symbol:         "",
				IsReference:    false,
				IsDefinition:   false,
				HighlightClass: query.CaptureNames()[capture.Index],
				Document:       []string{},
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
