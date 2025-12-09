package lsp

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cockroachdb/errors"
	"go.lsp.dev/jsonrpc2"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type Client struct {
	process *exec.Cmd
	conn    jsonrpc2.Conn
	ctx     context.Context
	cancel  context.CancelFunc
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

	// We redirect stderr to the parent's stderr for debugging purposes,
	// or we could capture it. For now, inheriting is useful.
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, errors.Wrapf(err, "failed to start command %s", cmdName)
	}

	// Create a read-write closer that combines stdout (read) and stdin (write)
	rwc := struct {
		io.ReadCloser
		io.Writer
	}{
		ReadCloser: stdout,
		Writer:     stdin,
	}

	stream := jsonrpc2.NewStream(rwc)
	conn := jsonrpc2.NewConn(stream)

	clientCtx, cancel := context.WithCancel(ctx)

	// Start the connection handling loop in the background
	conn.Go(clientCtx, jsonrpc2.MethodNotFoundHandler)

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

	params := &protocol.InitializeParams{
		RootURI: uri.File(absPath),
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				Hover: &protocol.HoverTextDocumentClientCapabilities{
					ContentFormat: []protocol.MarkupKind{protocol.Markdown},
				},
				Definition: &protocol.DefinitionTextDocumentClientCapabilities{},
			},
		},
	}

	var result protocol.InitializeResult
	_, err = c.conn.Call(c.ctx, protocol.MethodInitialize, params, &result)
	if err != nil {
		return errors.Wrap(err, "initialize request failed")
	}

	if err := c.conn.Notify(c.ctx, protocol.MethodInitialized, &protocol.InitializedParams{}); err != nil {
		return errors.Wrap(err, "initialized notification failed")
	}

	return nil
}

func (c *Client) DidOpen(filePath string, content string) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return err
	}

	params := &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri.File(absPath),
			LanguageID: "go", // This might need to be dynamic later
			Version:    1,
			Text:       content,
		},
	}

	err = c.conn.Notify(c.ctx, protocol.MethodTextDocumentDidOpen, params)
	return errors.Wrap(err, "textDocument/didOpen failed")
}

func (c *Client) Hover(filePath string, line, char int) (*protocol.Hover, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	params := &protocol.HoverParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri.File(absPath),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(char),
			},
		},
	}

	var result protocol.Hover
	_, err = c.conn.Call(c.ctx, protocol.MethodTextDocumentHover, params, &result)
	if err != nil {
		return nil, errors.Wrap(err, "hover request failed")
	}

	return &result, nil
}

func (c *Client) Definition(filePath string, line, char int) ([]protocol.Location, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, err
	}

	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri.File(absPath),
			},
			Position: protocol.Position{
				Line:      uint32(line),
				Character: uint32(char),
			},
		},
	}

	// Definition can return Location or []Location.
	// The library might handle unmarshaling into a slice, but sometimes it returns a single object.
	// Let's try unmarshaling into a slice first.
	var result []protocol.Location
	_, err = c.conn.Call(c.ctx, protocol.MethodTextDocumentDefinition, params, &result)

	// Handle single Location case if unmarshal fails or result is empty (some servers return null or empty array)
	// But strictly speaking, the response can be Location | Location[] | null.
	// We might need a raw json.RawMessage if this fails often.
	// For now, let's assume standard behavior or try to catch it.
	if err != nil {
		// Try single location
		var singleRes protocol.Location
		_, errSingle := c.conn.Call(c.ctx, protocol.MethodTextDocumentDefinition, params, &singleRes)
		if errSingle == nil {
			return []protocol.Location{singleRes}, nil
		}
		return nil, errors.Wrap(err, "definition request failed")
	}

	return result, nil
}

func (c *Client) Shutdown() error {
	// Ignore errors on shutdown as we are exiting anyway
	c.conn.Call(c.ctx, protocol.MethodShutdown, nil, nil)
	c.conn.Notify(c.ctx, protocol.MethodExit, nil)
	c.cancel()
	c.conn.Close()
	return c.process.Wait()
}
