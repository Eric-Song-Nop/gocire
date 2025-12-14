package lsp

import (
	"path/filepath"
	"strings"
)

// JSON-RPC Method Constants
const (
	MethodInitialize             = "initialize"
	MethodInitialized            = "initialized"
	MethodTextDocumentDidOpen    = "textDocument/didOpen"
	MethodTextDocumentHover      = "textDocument/hover"
	MethodTextDocumentDefinition = "textDocument/definition"
	MethodStatus                 = "status"
	MethodShutdown               = "shutdown"
	MethodExit                   = "exit"
)

// MarkupKind Constants
const (
	Markdown  = "markdown"
	PlainText = "plaintext"
)

// Basic Types
type DocumentURI string

func ToURI(path string) DocumentURI {
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	path = filepath.ToSlash(path)
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return DocumentURI("file://" + path)
}

type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

type MarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Params & Results

type InitializeParams struct {
	ProcessID    int                `json:"processId,omitempty"`
	RootURI      DocumentURI        `json:"rootUri,omitempty"`
	Capabilities ClientCapabilities `json:"capabilities"`
}

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Hover      *HoverTextDocumentClientCapabilities      `json:"hover,omitempty"`
	Definition *DefinitionTextDocumentClientCapabilities `json:"definition,omitempty"`
}

type HoverTextDocumentClientCapabilities struct {
	ContentFormat []string `json:"contentFormat,omitempty"`
}

type DefinitionTextDocumentClientCapabilities struct {
	// Empty in original
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

type ServerCapabilities struct {
	// Add fields if needed
}

type InitializedParams struct{}

type DidOpenTextDocumentParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

type TextDocumentItem struct {
	URI        DocumentURI `json:"uri"`
	LanguageID string      `json:"languageId"`
	Version    int         `json:"version"`
	Text       string      `json:"text"`
}

type HoverParams struct {
	TextDocumentPositionParams
}

type TextDocumentPositionParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Position     Position               `json:"position"`
}

type TextDocumentIdentifier struct {
	URI DocumentURI `json:"uri"`
}

type Hover struct {
	Contents MarkupContent `json:"contents"`
	Range    *Range        `json:"range,omitempty"`
}

type DefinitionParams struct {
	TextDocumentPositionParams
}
