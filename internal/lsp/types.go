package lsp

import (
	"path/filepath"
	"strings"
)

// JSON-RPC Method Constants
const (
	MethodInitialize                   = "initialize"
	MethodInitialized                  = "initialized"
	MethodTextDocumentDidOpen          = "textDocument/didOpen"
	MethodTextDocumentHover            = "textDocument/hover"
	MethodTextDocumentDefinition       = "textDocument/definition"
	MethodTextDocumentInlayHint        = "textDocument/inlayHint"
	MethodWorkspaceConfiguration       = "workspace/configuration"
	MethodStatus                       = "status"
	MethodShutdown                     = "shutdown"
	MethodExit                         = "exit"
	MethodProgress                     = "$/progress"
	MethodWindowWorkDoneProgressCreate = "window/workDoneProgress/create"
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
	ProcessID             int                `json:"processId,omitempty"`
	RootURI               DocumentURI        `json:"rootUri,omitempty"`
	WorkspaceFolders      []WorkspaceFolder  `json:"workspaceFolders,omitempty"`
	Capabilities          ClientCapabilities `json:"capabilities"`
	InitializationOptions interface{}        `json:"initializationOptions,omitempty"`
}

type WorkspaceFolder struct {
	URI  DocumentURI `json:"uri"`
	Name string      `json:"name"`
}

type ClientCapabilities struct {
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	Window       *WindowClientCapabilities       `json:"window,omitempty"`
	Workspace    *WorkspaceClientCapabilities    `json:"workspace,omitempty"`
}

type WorkspaceClientCapabilities struct {
	Configuration bool `json:"configuration,omitempty"`
}

type WindowClientCapabilities struct {
	WorkDoneProgress bool `json:"workDoneProgress,omitempty"`
}

type TextDocumentClientCapabilities struct {
	Hover      *HoverTextDocumentClientCapabilities      `json:"hover,omitempty"`
	Definition *DefinitionTextDocumentClientCapabilities `json:"definition,omitempty"`
	InlayHint  *InlayHintTextDocumentClientCapabilities  `json:"inlayHint,omitempty"`
}

type HoverTextDocumentClientCapabilities struct {
	ContentFormat []string `json:"contentFormat,omitempty"`
}

type DefinitionTextDocumentClientCapabilities struct {
	// Empty in original
}

type InlayHintTextDocumentClientCapabilities struct {
	DynamicRegistration bool                     `json:"dynamicRegistration,omitempty"`
	ResolveSupport      *InlayHintResolveSupport `json:"resolveSupport,omitempty"`
}

type InlayHintResolveSupport struct {
	Properties []string `json:"properties"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

type ServerCapabilities struct {
	// Add fields if needed
}

type InitializedParams struct{}

type WorkspaceConfigurationParams struct {
	Items []ConfigurationItem `json:"items"`
}

type ConfigurationItem struct {
	ScopeURI string `json:"scopeUri,omitempty"`
	Section  string `json:"section,omitempty"`
}

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

type InlayHintParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
	Range        Range                  `json:"range"`
}

type InlayHint struct {
	Position     Position    `json:"position"`
	Label        interface{} `json:"label"`
	Kind         int         `json:"kind,omitempty"`
	PaddingLeft  bool        `json:"paddingLeft,omitempty"`
	PaddingRight bool        `json:"paddingRight,omitempty"`
}

type InlayHintLabelPart struct {
	Value string `json:"value"`
}

const (
	InlayHintKindType      = 1
	InlayHintKindParameter = 2
)

// Progress Types
type ProgressParams struct {
	Token interface{} `json:"token"`
	Value interface{} `json:"value"`
}

type WorkDoneProgressValue struct {
	Kind       string `json:"kind"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Percentage int    `json:"percentage,omitempty"`
}

type WorkDoneProgressCreateParams struct {
	Token interface{} `json:"token"`
}
