package internal

import (
	"embed"
	"os"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
	sitter "github.com/tree-sitter/go-tree-sitter"

	// Language bindings - The package names are often derived or explicitly set within the module.
	// I will use explicit aliases for clarity and to avoid conflicts with 'go' keyword.
	dartsitter "github.com/UserNobody14/tree-sitter-dart/bindings/go"
	csharpsitter "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	csitter "github.com/tree-sitter/tree-sitter-c/bindings/go"
	cppsitter "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	golangsitter "github.com/tree-sitter/tree-sitter-go/bindings/go"
	haskellsitter "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	javasitter "github.com/tree-sitter/tree-sitter-java/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	phpsitter "github.com/tree-sitter/tree-sitter-php/bindings/go"
	pythonsitter "github.com/tree-sitter/tree-sitter-python/bindings/go"
	rubysitter "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	rustsitter "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
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

func (h *HighlightAnalyzer) Analyze(sourcePath string) ([]TokenInfo, error) {
	lang, queryFileName, err := h.getLanguageAndQuery()
	if err != nil {
		return nil, err
	}

	sourceContent, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read source file %s", sourcePath)
	}

	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(lang)

	// Fix 1: parser.Parse signature
	tree := parser.Parse(sourceContent, nil)
	defer tree.Close()

	queryContent, err := queryFS.ReadFile("queries/" + queryFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read query file %s", queryFileName)
	}

	query, queryErr := sitter.NewQuery(lang, string(queryContent))
	if queryErr != nil {
		return nil, errors.Wrapf(queryErr, "failed to create query for %s", h.language)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	// Fix 2 & 3: QueryCursor iteration
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
				InlayText:      []string{},
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

func (h *HighlightAnalyzer) getLanguageAndQuery() (*sitter.Language, string, error) {
	switch strings.ToLower(h.language) {
	case "go", "golang":
		return sitter.NewLanguage(golangsitter.Language()), "go.scm", nil
	case "java":
		return sitter.NewLanguage(javasitter.Language()), "java.scm", nil
	case "js", "javascript":
		return sitter.NewLanguage(javascript.Language()), "javascript.scm", nil
	case "ts", "typescript":
		return sitter.NewLanguage(typescript.LanguageTypescript()), "typescript.scm", nil
	case "rust":
		return sitter.NewLanguage(rustsitter.Language()), "rust.scm", nil
	case "c":
		return sitter.NewLanguage(csitter.Language()), "c.scm", nil
	case "cpp", "c++":
		return sitter.NewLanguage(cppsitter.Language()), "cpp.scm", nil
	case "ruby":
		return sitter.NewLanguage(rubysitter.Language()), "ruby.scm", nil
	case "python", "py":
		return sitter.NewLanguage(pythonsitter.Language()), "python.scm", nil
	case "csharp", "c#", "cs":
		return sitter.NewLanguage(csharpsitter.Language()), "c_sharp.scm", nil
	case "php":
		return sitter.NewLanguage(phpsitter.LanguagePHP()), "php.scm", nil
	case "haskell":
		return sitter.NewLanguage(haskellsitter.Language()), "haskell.scm", nil
	case "dart":
		return sitter.NewLanguage(dartsitter.Language()), "dart.scm", nil
	default:
		return nil, "", errors.Newf("unsupported language: %s", h.language)
	}
}
