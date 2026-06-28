package internal

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Eric-Song-Nop/gocire/internal/languages"
	"github.com/Eric-Song-Nop/gocire/internal/lsp"
	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/scip/bindings/go/scip"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type LSPAnalyzer struct {
	language      string
	sourcePath    string
	workspaceRoot string
}

// LSPSession owns one initialized language-server process and reuses it across files.
type LSPSession struct {
	language string
	cfg      *languages.LanguageConfig
	client   *lsp.Client

	requestMu sync.Mutex
	closeMu   sync.Mutex
	closed    bool
}

func NewLSPAnalyzer(language, sourcePath string, workspaceRoot string) *LSPAnalyzer {
	return &LSPAnalyzer{
		language:      language,
		sourcePath:    sourcePath,
		workspaceRoot: workspaceRoot,
	}
}

func (l *LSPAnalyzer) Analyze(sourceContent []byte) ([]TokenInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	session, err := NewLSPSession(ctx, l.language, resolveLSPWorkspaceRoot(l.workspaceRoot, l.sourcePath))
	if err != nil {
		return nil, err
	}
	defer session.Close()

	return session.AnalyzeFile(l.sourcePath, sourceContent)
}

// NewLSPSession starts and initializes one language server for a workspace.
func NewLSPSession(ctx context.Context, language, workspaceRoot string) (*LSPSession, error) {
	cfg, err := languages.GetConfig(language)
	if err != nil {
		return nil, err
	}
	if cfg.LSPCommand == "" {
		return nil, errors.Newf("no lsp server configured for language %s", language)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	rootDir := resolveLSPWorkspaceRoot(workspaceRoot, "")

	client, err := lsp.NewClient(ctx, cfg.LSPCommand, cfg.LSPArgs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to start lsp client")
	}

	var initErr error
	if cfg.LSPInitializationOptions != nil {
		initErr = client.Initialize(rootDir, cfg.LSPInitializationOptions)
	} else {
		initErr = client.Initialize(rootDir)
	}
	if initErr != nil {
		_ = client.Shutdown()
		return nil, errors.Wrap(initErr, "lsp initialize failed")
	}

	// Wait for server to finish indexing (up to 10 seconds)
	// This helps avoid empty results if the server is still parsing the workspace.
	_ = client.WaitForIndexing(10 * time.Second)

	return &LSPSession{
		language: language,
		cfg:      cfg,
		client:   client,
	}, nil
}

func resolveLSPWorkspaceRoot(workspaceRoot, sourcePath string) string {
	if workspaceRoot != "" {
		return workspaceRoot
	}
	if sourcePath != "" {
		return filepath.Dir(sourcePath)
	}
	return "."
}

// AnalyzeFile analyzes one file through the session's language server.
func (s *LSPSession) AnalyzeFile(sourcePath string, sourceContent []byte) ([]TokenInfo, error) {
	if err := s.withLSPClient(func(client *lsp.Client) error {
		return client.DidOpen(sourcePath, s.language, string(sourceContent))
	}); err != nil {
		return nil, errors.Wrap(err, "lsp didOpen failed")
	}

	// Give the server a moment to process the file open event
	time.Sleep(1 * time.Second)

	inlayHintTokens := s.fetchInlayHintTokens(sourcePath, sourceContent)

	tokens, err := analyzeLSPTokens(s.language, sourcePath, sourceContent, s.cfg, func(line, char int) (*lsp.Hover, []lsp.Location) {
		var hover *lsp.Hover
		var defs []lsp.Location
		_ = s.withLSPClient(func(client *lsp.Client) error {
			hover, _ = client.Hover(sourcePath, line, char)
			defs, _ = client.Definition(sourcePath, line, char)
			return nil
		})
		return hover, defs
	})
	if err != nil {
		return nil, err
	}

	return append(tokens, inlayHintTokens...), nil
}

// Close shuts down the language-server process owned by the session.
func (s *LSPSession) Close() error {
	s.closeMu.Lock()
	if s.closed {
		s.closeMu.Unlock()
		return nil
	}
	s.closed = true
	s.closeMu.Unlock()

	s.requestMu.Lock()
	defer s.requestMu.Unlock()

	return s.client.Shutdown()
}

func (s *LSPSession) withLSPClient(fn func(*lsp.Client) error) error {
	s.closeMu.Lock()
	closed := s.closed
	s.closeMu.Unlock()
	if closed {
		return errors.New("lsp session is closed")
	}

	s.requestMu.Lock()
	defer s.requestMu.Unlock()

	s.closeMu.Lock()
	closed = s.closed
	s.closeMu.Unlock()
	if closed {
		return errors.New("lsp session is closed")
	}

	return fn(s.client)
}

type lspTokenQuery func(line, char int) (*lsp.Hover, []lsp.Location)

func (s *LSPSession) fetchInlayHintTokens(sourcePath string, sourceContent []byte) []TokenInfo {
	hintRange := fullDocumentInlayHintRange(sourceContent)

	var hints []lsp.InlayHint
	var requestErr error
	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		requestErr = s.withLSPClient(func(client *lsp.Client) error {
			var err error
			hints, err = client.InlayHint(
				sourcePath,
				hintRange.Start.Line,
				hintRange.Start.Character,
				hintRange.End.Line,
				hintRange.End.Character,
			)
			return err
		})
		if requestErr == nil {
			return tokenInfosFromInlayHints(hints)
		}
		if !isContentModifiedError(requestErr) || attempt == maxRetries-1 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "Warning: Failed to fetch inlay hints for %s: %v\n", sourcePath, requestErr)
	return nil
}

func fullDocumentInlayHintRange(sourceContent []byte) lsp.Range {
	source := string(sourceContent)
	lines := strings.Split(source, "\n")
	lineCount := len(lines)
	if lineCount > 0 && lines[lineCount-1] == "" && strings.HasSuffix(source, "\n") {
		lineCount--
	}

	lastLineIdx := lineCount - 1
	if lastLineIdx < 0 {
		lastLineIdx = 0
	}

	lastLineLen := 0
	if lineCount > 0 {
		lastLineLen = len([]rune(lines[lastLineIdx]))
	}

	return lsp.Range{
		Start: lsp.Position{Line: 0, Character: 0},
		End:   lsp.Position{Line: lastLineIdx, Character: lastLineLen},
	}
}

func tokenInfosFromInlayHints(hints []lsp.InlayHint) []TokenInfo {
	tokens := make([]TokenInfo, 0, len(hints))
	for _, hint := range hints {
		label := inlayHintLabel(hint)
		if label == "" {
			continue
		}

		pos := scip.Position{
			Line:      int32(hint.Position.Line),
			Character: int32(hint.Position.Character),
		}
		tokens = append(tokens, TokenInfo{
			InlayHintLabel: label,
			Span: scip.Range{
				Start: pos,
				End:   pos,
			},
		})
	}
	return tokens
}

func inlayHintLabel(hint lsp.InlayHint) string {
	label := inlayHintBaseLabel(hint.Label)
	if label == "" {
		return ""
	}
	if hint.PaddingLeft {
		label = " " + label
	}
	if hint.PaddingRight {
		label += " "
	}
	return label
}

func inlayHintBaseLabel(label interface{}) string {
	switch v := label.(type) {
	case string:
		return v
	case []lsp.InlayHintLabelPart:
		var sb strings.Builder
		for _, part := range v {
			sb.WriteString(part.Value)
		}
		return sb.String()
	case []interface{}:
		var sb strings.Builder
		for _, part := range v {
			switch p := part.(type) {
			case lsp.InlayHintLabelPart:
				sb.WriteString(p.Value)
			case map[string]interface{}:
				if value, ok := p["value"].(string); ok {
					sb.WriteString(value)
				}
			}
		}
		return sb.String()
	default:
		return ""
	}
}

func isContentModifiedError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "content modified") || strings.Contains(msg, "-32801")
}

func analyzeLSPTokens(language, sourcePath string, sourceContent []byte, cfg *languages.LanguageConfig, queryLSP lspTokenQuery) ([]TokenInfo, error) {
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
		return nil, errors.Wrapf(qErr, "failed to create query for %s", language)
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
			span := scip.Range{
				Start: scip.Position{
					Line:      int32(start.Row),
					Character: int32(start.Column),
				},
				End: scip.Position{
					Line:      int32(end.Row),
					Character: int32(end.Column),
				},
			}

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
			var definition *SourceLocation

			// Only query LSP if the capture is not ignored
			if !isIgnoredCapture(captureName, cfg.IgnoredCaptures) {
				// Query LSP
				// We query at the start of the token
				hover, defs := queryLSP(int(start.Row), int(start.Column))

				// Process hover results
				if hover != nil && hover.Contents.Value != "" {
					docs = append(docs, hover.Contents.Value)
				}

				// Process definition results
				if len(defs) > 0 {
					d := defs[0]
					definition = sourceLocationFromLSP(d)
					// Generate a unique symbol ID based on the definition location
					// This allows references to link to this specific definition
					// We use the first definition if multiple are returned
					// Use 0-based indexing for the ID to match internal logic,
					// though LSP uses 0-based too.
					symID := getSymbolID(definition.URI, int(definition.Range.Start.Line), int(definition.Range.Start.Character))

					// If we have a valid definition location, we assign the symbol ID
					// This effectively links this token (reference or def) to that ID.
					if symID != "" {
						symbolID = symID
					}

					// Check if this token IS the definition
					// We compare the returned definition location with the current token's location
					// across normalized file paths.
					isDefinition = isDefinitionLocation(definition, sourcePath, span)
					isReference = !isDefinition
				}
			}

			token := TokenInfo{
				Symbol:         symbolID,
				IsReference:    isReference,
				IsDefinition:   isDefinition,
				HighlightClass: "",
				Document:       docs,
				Span:           span,
				Definition:     definition,
			}
			tokens = append(tokens, token)
		}
	}

	return tokens, nil
}

func sourceLocationFromLSP(location lsp.Location) *SourceLocation {
	uriStr := string(location.URI)
	return &SourceLocation{
		URI:   uriStr,
		Path:  normalizedPathFromURI(uriStr),
		Range: lspRangeToSCIP(location.Range),
	}
}

func lspRangeToSCIP(r lsp.Range) scip.Range {
	return scip.Range{
		Start: scip.Position{
			Line:      int32(r.Start.Line),
			Character: int32(r.Start.Character),
		},
		End: scip.Position{
			Line:      int32(r.End.Line),
			Character: int32(r.End.Character),
		},
	}
}

func isDefinitionLocation(definition *SourceLocation, sourcePath string, span scip.Range) bool {
	if definition == nil || !samePath(definition.Path, sourcePath) {
		return false
	}

	return scip.Position.Compare(definition.Range.Start, span.Start) == 0
}

func samePath(left, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return normalizePath(left) == normalizePath(right)
}

func normalizedPathFromURI(uriStr string) string {
	if uriStr == "" {
		return ""
	}

	parsed, err := url.Parse(uriStr)
	if err == nil && parsed.Scheme == "file" {
		path := parsed.Path
		if parsed.Host != "" {
			path = "//" + parsed.Host + path
		}
		return normalizePath(path)
	}

	if strings.HasPrefix(uriStr, "file://") {
		return normalizePath(strings.TrimPrefix(uriStr, "file://"))
	}

	if err == nil && parsed.Scheme == "" {
		return normalizePath(uriStr)
	}

	return ""
}

func normalizePath(path string) string {
	if path == "" {
		return ""
	}

	if decoded, err := url.PathUnescape(path); err == nil {
		path = decoded
	}

	path = filepath.FromSlash(path)
	if absPath, err := filepath.Abs(path); err == nil {
		path = absPath
	}

	return filepath.Clean(path)
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
