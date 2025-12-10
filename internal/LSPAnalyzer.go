package internal

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/Eric-Song-Nop/gocire/internal/languages"
	"github.com/Eric-Song-Nop/gocire/internal/lsp"
	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type LSPAnalyzer struct {
	language   string
	sourcePath string
}

func NewLSPAnalyzer(language, sourcePath string) *LSPAnalyzer {
	return &LSPAnalyzer{
		language:   language,
		sourcePath: sourcePath,
	}
}

func (l *LSPAnalyzer) Analyze(sourceContent []byte) ([]TokenInfo, error) {
	// 1. Get Config
	cfg, err := languages.GetConfig(l.language)
	if err != nil {
		return nil, err
	}

	if cfg.LSPCommand == "" {
		return nil, errors.Newf("no lsp server configured for language %s", l.language)
	}

	// 2. Start Client
	// Use a generous timeout for the entire analysis session
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Determine root. Use file dir as a simple fallback for now.
	rootDir := filepath.Dir(l.sourcePath)

	client, err := lsp.NewClient(ctx, cfg.LSPCommand, cfg.LSPArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start lsp client")
	}
	defer client.Shutdown()

	if err := client.Initialize(rootDir); err != nil {
		return nil, errors.Wrap(err, "lsp initialize failed")
	}

	if err := client.DidOpen(l.sourcePath, l.language, string(sourceContent)); err != nil {
		return nil, errors.Wrap(err, "lsp didOpen failed")
	}

	// 3. Find Tokens using Tree-sitter
	parser := sitter.NewParser()
	defer parser.Close()
	parser.SetLanguage(cfg.SitterLanguage)

	tree := parser.Parse(sourceContent, nil)
	defer tree.Close()

	queryContent, err := queryFS.ReadFile("queries/" + cfg.QueryFileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read query file %s", cfg.QueryFileName)
	}

	query, qErr := sitter.NewQuery(cfg.SitterLanguage, string(queryContent))
	if qErr != nil {
		return nil, errors.Wrapf(qErr, "failed to create query for %s", l.language)
	}
	defer query.Close()

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	matches := qc.Matches(query, tree.RootNode(), sourceContent)

	var tokens []TokenInfo
	type posKey struct {
		line, char int32
	}
	seen := make(map[posKey]bool)

	for match := matches.Next(); match != nil; match = matches.Next() {
		for _, capture := range match.Captures {
			node := capture.Node
			start := node.StartPosition()
			end := node.EndPosition()

			// Deduplication check
			key := posKey{int32(start.Row), int32(start.Column)}
			if seen[key] {
				continue
			}
			seen[key] = true

			captureName := query.CaptureNames()[capture.Index]

			var docs []string
			var symbolID string           // Initialize empty symbol ID
			var isDefinition bool = false // Initialize to false
			var isReference bool = false  // Initialize to false

			// Only query LSP if the capture is not ignored
			if !isIgnoredCapture(captureName, cfg.IgnoredCaptures) {
				// Query LSP
				// We query at the start of the token
				hover, _ := client.Hover(l.sourcePath, int(start.Row), int(start.Column))
				defs, _ := client.Definition(l.sourcePath, int(start.Row), int(start.Column))

				// Process hover results
				if hover != nil && hover.Contents.Value != "" {
					docs = append(docs, hover.Contents.Value)
				}

				// Process definition results
				if len(defs) > 0 {
					d := defs[0]
					// Generate a unique symbol ID based on the definition location
					// This allows references to link to this specific definition
					// We use the first definition if multiple are returned
					uriStr := string(d.URI)
					// Use 0-based indexing for the ID to match internal logic,
					// though LSP uses 0-based too.
					symID := getSymbolID(uriStr, int(d.Range.Start.Line), int(d.Range.Start.Character))

					// If we have a valid definition location, we assign the symbol ID
					// This effectively links this token (reference or def) to that ID.
					if symID != "" {
						symbolID = symID
					}

					// Check if this token IS the definition
					// We compare the returned definition location with the current token's location
					// We must check if the file matches.
					// d.URI is usually file://...
					// l.sourcePath is usually an absolute path /Users/...
					// We do a loose check: if d.URI ends with sourcePath (handling protocol prefix)
					defPath := strings.TrimPrefix(uriStr, "file://")
					isCurrentFile := false

					// Simple check: do they refer to the same file?
					// l.sourcePath should be absolute.
					if defPath == l.sourcePath || strings.HasSuffix(defPath, l.sourcePath) || strings.HasSuffix(l.sourcePath, defPath) {
						isCurrentFile = true
					}

					if isCurrentFile &&
						int(d.Range.Start.Line) == int(start.Row) &&
						int(d.Range.Start.Character) == int(start.Column) {
						isDefinition = true
					} else if isCurrentFile { // Only mark as reference if definition is in current file
						isReference = true
					}
				}
			}

			// Always create a TokenInfo for syntax highlighting and any available LSP data
			token := TokenInfo{
				Symbol:         symbolID,
				IsReference:    isReference,
				IsDefinition:   isDefinition,
				HighlightClass: captureName,
				Document:       docs,
				Span: scip.Range{
					Start: scip.Position{
						Line:      int32(start.Row),
						Character: int32(start.Column),
					},
					End: scip.Position{
						Line:      int32(end.Row),
						Character: int32(end.Column),
					},
				},
			}
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}

func getSymbolID(uriStr string, line, col int) string {
	// Create a safe ID string from the URI and position
	// Remove file:// prefix
	path := strings.TrimPrefix(uriStr, "file://")

	// Sanitize path to be ID-safe (alphanumeric + underscores)
	// We just replace common separators.
	safePath := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, path)

	// Format: def_path_line_col
	return fmt.Sprintf("def_%s_%d_%d", safePath, line, col)
}

func isIgnoredCapture(name string, ignoreList []string) bool {
	// Ignore punctuation, brackets, operators, and basic keywords from expensive LSP queries
	// unless we really want them.
	for _, i := range ignoreList {
		if strings.Contains(name, i) {
			return true
		}
	}
	return false
}
