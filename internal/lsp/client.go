package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type Client struct {
	process         *exec.Cmd
	conn            *jsonrpc2.Conn
	ctx             context.Context
	cancel          context.CancelFunc
	configMu        sync.RWMutex
	workspaceConfig interface{}

	// Synchronization for work done progress
	mu           sync.Mutex
	activeWork   map[string]bool
	workDoneCond *sync.Cond
}

type readWriteCloser struct {
	r io.ReadCloser
	w io.WriteCloser
}

func (c *readWriteCloser) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *readWriteCloser) Write(p []byte) (n int, err error) {
	return c.w.Write(p)
}

func (c *readWriteCloser) Close() error {
	err := c.r.Close()
	wErr := c.w.Close()
	if err != nil {
		return err
	}
	return wErr
}

func NewClient(ctx context.Context, cmdName string, args []string) (*Client, error) {
	cmd := exec.Command(cmdName, args...)
	lspDebugf("start command=%s args=%v", cmdName, args)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stdin pipe")
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get stdout pipe")
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrapf(err, "failed to start command %s", cmdName)
	}

	rwc := &readWriteCloser{
		r: stdout,
		w: stdin,
	}

	stream := jsonrpc2.NewBufferedStream(rwc, jsonrpc2.VSCodeObjectCodec{})

	clientCtx, cancel := context.WithCancel(ctx)

	c := &Client{
		process:    cmd,
		ctx:        clientCtx,
		cancel:     cancel,
		activeWork: make(map[string]bool),
	}
	c.workDoneCond = sync.NewCond(&c.mu)

	handler := jsonrpc2.HandlerWithError(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
		lspDebugf("server request method=%s", req.Method)
		if req.Method == MethodWindowWorkDoneProgressCreate {
			return nil, nil
		}
		if req.Method == MethodWorkspaceConfiguration {
			return c.workspaceConfiguration(req.Params)
		}
		if req.Method == MethodProgress {
			var params ProgressParams
			if err := json.Unmarshal(*req.Params, &params); err == nil {
				c.handleProgress(params)
			}
		}
		return nil, nil
	})

	conn := jsonrpc2.NewConn(ctx, stream, handler)
	c.conn = conn

	return c, nil
}

func (c *Client) handleProgress(params ProgressParams) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Convert token to string key
	tokenStr := ""
	switch t := params.Token.(type) {
	case string:
		tokenStr = t
	case float64: // JSON numbers are float64
		tokenStr = strconv.Itoa(int(t))
	default:
		return
	}

	// Parse value
	// We need to re-marshal interface{} to handle the specific WorkDoneProgressValue struct
	// or blindly cast map[string]interface{}
	valBytes, _ := json.Marshal(params.Value)
	var workVal WorkDoneProgressValue
	json.Unmarshal(valBytes, &workVal)

	switch workVal.Kind {
	case "begin":
		c.activeWork[tokenStr] = true
	case "end":
		delete(c.activeWork, tokenStr)
		c.workDoneCond.Broadcast()
	default:
		// Check for implicit completion in report messages
		// e.g. "15/15", "100%"
		if strings.Contains(workVal.Message, "15/15") || strings.Contains(workVal.Message, "100%") {
			delete(c.activeWork, tokenStr)
			c.workDoneCond.Broadcast()
		}
	}
}

func (c *Client) WaitForIndexing(timeout time.Duration) error {
	// Give the server a moment to start reporting progress
	time.Sleep(500 * time.Millisecond)

	// Wait until no active work, or timeout
	// Simple implementation: wait loop with condition
	done := make(chan struct{})

	go func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		for len(c.activeWork) > 0 {
			c.workDoneCond.Wait()
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		// It's okay if we timeout, we just proceed.
		// Some servers might never send "end" or start something else.
		// We just want to give it a chance to finish initial indexing.
		return errors.New("timeout waiting for indexing")
	case <-c.ctx.Done():
		return c.ctx.Err()
	}
}

func (c *Client) Initialize(rootPath string, initializationOptions ...interface{}) error {
	var options interface{}
	if len(initializationOptions) > 0 {
		options = initializationOptions[0]
	}
	return c.InitializeWithOptions(rootPath, options, nil)
}

func (c *Client) InitializeWithOptions(rootPath string, initializationOptions, workspaceConfiguration interface{}) error {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return err
	}

	c.configMu.Lock()
	c.workspaceConfig = workspaceConfiguration
	c.configMu.Unlock()

	params := newInitializeParams(absPath, initializationOptions)
	lspDebugf("initialize root=%s initOptions=%#v workspaceConfig=%#v", absPath, initializationOptions, workspaceConfiguration)

	var result InitializeResult
	if err := c.conn.Call(c.ctx, MethodInitialize, params, &result); err != nil {
		return errors.Wrap(err, "initialize request failed")
	}
	lspDebugf("initialize result capabilities=%#v", result.Capabilities)

	if err := c.conn.Notify(c.ctx, MethodInitialized, &InitializedParams{}); err != nil {
		return errors.Wrap(err, "initialized notification failed")
	}

	return nil
}

func (c *Client) workspaceConfiguration(raw *json.RawMessage) ([]interface{}, error) {
	var params WorkspaceConfigurationParams
	if raw == nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: "workspace/configuration params are required",
		}
	}
	if err := json.Unmarshal(*raw, &params); err != nil {
		return nil, &jsonrpc2.Error{
			Code:    jsonrpc2.CodeInvalidParams,
			Message: fmt.Sprintf("invalid workspace/configuration params: %v", err),
		}
	}

	if len(params.Items) == 0 {
		return []interface{}{}, nil
	}

	result := make([]interface{}, len(params.Items))
	config := c.currentWorkspaceConfiguration()
	for i, item := range params.Items {
		result[i] = configurationForSection(config, item.Section)
		lspDebugf("workspace/configuration item=%d section=%q scope=%q result=%#v", i, item.Section, item.ScopeURI, result[i])
	}
	return result, nil
}

func (c *Client) currentWorkspaceConfiguration() interface{} {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return c.workspaceConfig
}

func configurationForSection(config interface{}, section string) interface{} {
	if section == "" {
		return config
	}

	if value, ok := mapValue(config, section); ok {
		return value
	}
	if value, ok := nestedConfigurationValue(config, section); ok {
		return value
	}

	return nil
}

func nestedConfigurationValue(root interface{}, path string) (interface{}, bool) {
	if path == "" {
		return root, true
	}

	var current interface{} = root
	for _, key := range strings.Split(path, ".") {
		value, ok := mapValue(current, key)
		if !ok {
			return nil, false
		}
		current = value
	}
	return current, true
}

func mapValue(value interface{}, key string) (interface{}, bool) {
	switch typed := value.(type) {
	case map[string]interface{}:
		v, ok := typed[key]
		return v, ok
	case map[string]bool:
		v, ok := typed[key]
		if !ok {
			return nil, false
		}
		return v, true
	}

	rv := reflect.ValueOf(value)
	if !rv.IsValid() || rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	keyValue := reflect.ValueOf(key)
	if !keyValue.Type().AssignableTo(rv.Type().Key()) {
		keyValue = keyValue.Convert(rv.Type().Key())
	}
	found := rv.MapIndex(keyValue)
	if !found.IsValid() {
		return nil, false
	}
	return found.Interface(), true
}

func newInitializeParams(absPath string, initializationOptions interface{}) *InitializeParams {
	return &InitializeParams{
		RootURI: ToURI(absPath),
		WorkspaceFolders: []WorkspaceFolder{
			{
				URI:  ToURI(absPath),
				Name: filepath.Base(absPath),
			},
		},
		Capabilities: ClientCapabilities{
			Workspace: &WorkspaceClientCapabilities{
				Configuration: true,
			},
			Window: &WindowClientCapabilities{
				WorkDoneProgress: true,
			},
			TextDocument: &TextDocumentClientCapabilities{
				Hover: &HoverTextDocumentClientCapabilities{
					ContentFormat: []string{Markdown},
				},
				Definition: &DefinitionTextDocumentClientCapabilities{},
				InlayHint:  &InlayHintTextDocumentClientCapabilities{},
			},
		},
		InitializationOptions: initializationOptions,
	}
}

func (c *Client) DidOpen(filePath string, languageID string, content string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	params := &DidOpenTextDocumentParams{
		TextDocument: TextDocumentItem{
			URI:        ToURI(absPath),
			LanguageID: languageID,
			Version:    1,
			Text:       content,
		},
	}

	err = c.conn.Notify(c.ctx, MethodTextDocumentDidOpen, params)
	return errors.Wrap(err, "textDocument/didOpen failed")
}

func (c *Client) Hover(filePath string, line, char int) (*Hover, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	params := &HoverParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: ToURI(absPath),
			},
			Position: Position{
				Line:      line,
				Character: char,
			},
		},
	}

	var raw json.RawMessage
	if err := c.conn.Call(c.ctx, MethodTextDocumentHover, params, &raw); err != nil {
		return nil, errors.Wrap(err, "hover request failed")
	}
	if isJSONNull(raw) {
		lspDebugf("hover null file=%s line=%d char=%d", absPath, line, char)
		return nil, nil
	}

	var result Hover
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal hover result")
	}
	if result.Contents.Value == "" {
		lspDebugf("hover empty file=%s line=%d char=%d raw=%s", absPath, line, char, string(raw))
	}

	return &result, nil
}

func (c *Client) Definition(filePath string, line, char int) ([]Location, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	params := &DefinitionParams{
		TextDocumentPositionParams: TextDocumentPositionParams{
			TextDocument: TextDocumentIdentifier{
				URI: ToURI(absPath),
			},
			Position: Position{
				Line:      line,
				Character: char,
			},
		},
	}

	var raw json.RawMessage
	if err := c.conn.Call(c.ctx, MethodTextDocumentDefinition, params, &raw); err != nil {
		return nil, errors.Wrap(err, "definition request failed")
	}
	if isJSONNull(raw) {
		lspDebugf("definition null file=%s line=%d char=%d", absPath, line, char)
		return nil, nil
	}

	var result []Location
	if err := json.Unmarshal(raw, &result); err == nil {
		if len(result) == 0 {
			lspDebugf("definition empty file=%s line=%d char=%d raw=%s", absPath, line, char, string(raw))
		}
		return result, nil
	}

	var single Location
	if err := json.Unmarshal(raw, &single); err == nil {
		return []Location{single}, nil
	}

	return nil, errors.New("failed to unmarshal definition result")
}

func (c *Client) InlayHint(filePath string, startLine, startChar, endLine, endChar int) ([]InlayHint, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	params := &InlayHintParams{
		TextDocument: TextDocumentIdentifier{
			URI: ToURI(absPath),
		},
		Range: Range{
			Start: Position{Line: startLine, Character: startChar},
			End:   Position{Line: endLine, Character: endChar},
		},
	}

	var result []InlayHint
	var raw json.RawMessage
	if err := c.conn.Call(c.ctx, MethodTextDocumentInlayHint, params, &raw); err != nil {
		return nil, errors.Wrap(err, "inlayHint request failed")
	}
	if isJSONNull(raw) {
		lspDebugf("inlayHint null file=%s range=%d:%d-%d:%d", absPath, startLine, startChar, endLine, endChar)
		return nil, nil
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal inlayHint result")
	}
	lspDebugf("inlayHint result file=%s range=%d:%d-%d:%d count=%d", absPath, startLine, startChar, endLine, endChar, len(result))

	return result, nil
}

func isJSONNull(raw json.RawMessage) bool {
	return strings.TrimSpace(string(raw)) == "null"
}

func (c *Client) Shutdown() error {
	var firstErr error
	recordErr := func(err error) {
		if firstErr == nil && !isExpectedShutdownError(err) {
			firstErr = err
		}
	}

	if c.conn != nil {
		recordErr(c.conn.Call(c.ctx, MethodShutdown, nil, nil))
		recordErr(c.conn.Notify(c.ctx, MethodExit, nil))
	}
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		recordErr(c.conn.Close())
	}
	if c.process != nil {
		recordErr(c.process.Wait())
	}
	return firstErr
}

func isExpectedShutdownError(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrProcessDone) || errors.Is(err, io.ErrClosedPipe) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "file already closed") ||
		strings.Contains(msg, "use of closed file")
}

// status Get status of the language server
func (c *Client) status() error {
	return nil
}

func lspDebugf(format string, args ...interface{}) {
	if os.Getenv("GOCIRE_LSP_DEBUG") == "" {
		return
	}
	fmt.Fprintf(os.Stderr, "[gocire:lsp-client] "+format+"\n", args...)
}
