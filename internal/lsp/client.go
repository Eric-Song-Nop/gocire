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
		println("RX Method:", req.Method) // Log every method received
		if req.Method == MethodWindowWorkDoneProgressCreate {
			println("RX Progress Create")
			return nil, nil // Accept the request to create progress
		}
		if req.Method == MethodProgress {
			var params ProgressParams
			if err := json.Unmarshal(*req.Params, &params); err == nil {
				c.handleProgress(params)
			} else {
				println("Failed to unmarshal progress params:", err.Error())
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
		println("Unknown token type:", t)
		return // Unknown token type
	}

	// Parse value
	// We need to re-marshal interface{} to handle the specific WorkDoneProgressValue struct
	// or blindly cast map[string]interface{}
	valBytes, _ := json.Marshal(params.Value)
	var workVal WorkDoneProgressValue
	json.Unmarshal(valBytes, &workVal)

	println("Progress Update:", tokenStr, workVal.Kind, workVal.Message, workVal.Title)

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
			println("Force completing task due to message:", workVal.Message)
			delete(c.activeWork, tokenStr)
			c.workDoneCond.Broadcast()
		}
	}
	println("Active Work Count:", len(c.activeWork))
}

func (c *Client) WaitForIndexing(timeout time.Duration) error {
	println("WaitForIndexing: Starting sleep (500ms)")
	// Give the server a moment to start reporting progress
	time.Sleep(500 * time.Millisecond)

	c.mu.Lock()
	initialWork := len(c.activeWork)
	c.mu.Unlock()
	println("WaitForIndexing: Sleep done. Active work items:", initialWork)

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
		println("WaitForIndexing: All work finished.")
		return nil
	case <-time.After(timeout):
		// Log which tasks are stuck
		c.mu.Lock()
		keys := make([]string, 0, len(c.activeWork))
		for k := range c.activeWork {
			keys = append(keys, k)
		}
		c.mu.Unlock()
		println("WaitForIndexing: Timeout reached. Stuck tasks:", strings.Join(keys, ", "))

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
	println("Called Hover")
	if err := c.conn.Call(c.ctx, MethodTextDocumentHover, params, &raw); err != nil {
		println("Called Hover Failed")
		return nil, errors.Wrap(err, "hover request failed")
	}
	// Print raw response for debugging
	println("Hover Raw Response:", string(raw))

	var result Hover
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal hover result")
	}
	println("Called Hover Success with Kind:", result.Contents.Kind)

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

	// Print raw response for debugging
	println("Definition Raw Response:", string(raw))

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

func (c *Client) Shutdown() error {
	c.conn.Call(c.ctx, MethodShutdown, nil, nil)
	c.conn.Notify(c.ctx, MethodExit, nil)
	c.cancel()
	c.conn.Close()
	return c.process.Wait()
}

// status Get status of the language server
func (c *Client) status() error {
	return nil
}
