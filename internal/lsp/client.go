package lsp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type Client struct {
	process *exec.Cmd
	conn    *jsonrpc2.Conn
	ctx     context.Context
	cancel  context.CancelFunc

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
		if req.Method == MethodWindowWorkDoneProgressCreate {
			return nil, nil
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

func (c *Client) Initialize(rootPath string) error {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return err
	}

	params := &InitializeParams{
		RootURI: ToURI(absPath),
		WorkspaceFolders: []WorkspaceFolder{
			{
				URI:  ToURI(absPath),
				Name: filepath.Base(absPath),
			},
		},
		Capabilities: ClientCapabilities{
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
	}

	var result InitializeResult
	if err := c.conn.Call(c.ctx, MethodInitialize, params, &result); err != nil {
		return errors.Wrap(err, "initialize request failed")
	}

	if err := c.conn.Notify(c.ctx, MethodInitialized, &InitializedParams{}); err != nil {
		return errors.Wrap(err, "initialized notification failed")
	}

	return nil
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

	var result Hover
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal hover result")
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

	var result []Location
	if err := json.Unmarshal(raw, &result); err == nil {
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
	if err := c.conn.Call(c.ctx, MethodTextDocumentInlayHint, params, &result); err != nil {
		return nil, errors.Wrap(err, "inlayHint request failed")
	}

	return result, nil
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
