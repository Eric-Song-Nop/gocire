package lsp

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"github.com/sourcegraph/jsonrpc2"
)

type Client struct {
	process *exec.Cmd
	conn    *jsonrpc2.Conn
	ctx     context.Context
	cancel  context.CancelFunc
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

	handler := jsonrpc2.HandlerWithError(func(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) (interface{}, error) {
		return nil, nil
	})

	conn := jsonrpc2.NewConn(ctx, stream, handler)

	clientCtx, cancel := context.WithCancel(ctx)

	return &Client{
		process: cmd,
		conn:    conn,
		ctx:     clientCtx,
		cancel:  cancel,
	}, nil
}

func (c *Client) Initialize(rootPath string) error {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return err
	}

	params := &InitializeParams{
		RootURI: ToURI(absPath),
		Capabilities: ClientCapabilities{
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

	var result Hover
	println("Called Hover")
	if err := c.conn.Call(c.ctx, MethodTextDocumentHover, params, &result); err != nil {
		println("Called Hover Failed")
		return nil, errors.Wrap(err, "hover request failed")
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

	var result []Location
	if err := json.Unmarshal(raw, &result); err == nil {
		return result, nil
	}

	var single Location
	if err := json.Unmarshal(raw, &single); err == nil {
		return []Location{single}, nil
	}

	println("GOto Def gives", raw)

	return nil, errors.New("failed to unmarshal definition result for ")
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
