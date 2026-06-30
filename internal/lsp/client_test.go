package lsp

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sourcegraph/jsonrpc2"
)

func TestInitializeParamsIncludeInlayHintCapabilityAndOptions(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")
	options := map[string]interface{}{
		"hints": map[string]bool{
			"parameterNames": true,
		},
	}

	params := newInitializeParams(root, options)
	payload, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("marshal initialize params: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal initialize params: %v", err)
	}

	capabilities, ok := got["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatalf("initialize params missing capabilities: %s", payload)
	}
	textDocument, ok := capabilities["textDocument"].(map[string]interface{})
	if !ok {
		t.Fatalf("initialize params missing textDocument capabilities: %s", payload)
	}
	if _, ok := textDocument["inlayHint"].(map[string]interface{}); !ok {
		t.Fatalf("initialize params missing inlayHint capability: %s", payload)
	}
	workspace, ok := capabilities["workspace"].(map[string]interface{})
	if !ok {
		t.Fatalf("initialize params missing workspace capabilities: %s", payload)
	}
	if workspace["configuration"] != true {
		t.Fatalf("workspace.configuration = %#v, want true in %s", workspace["configuration"], payload)
	}

	initializationOptions, ok := got["initializationOptions"].(map[string]interface{})
	if !ok {
		t.Fatalf("initialize params missing initializationOptions: %s", payload)
	}
	hints, ok := initializationOptions["hints"].(map[string]interface{})
	if !ok {
		t.Fatalf("initialize params missing hints options: %s", payload)
	}
	if hints["parameterNames"] != true {
		t.Fatalf("parameterNames hint = %#v, want true", hints["parameterNames"])
	}
}

func TestInitializeParamsOmitNilInitializationOptions(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "repo")

	payload, err := json.Marshal(newInitializeParams(root, nil))
	if err != nil {
		t.Fatalf("marshal initialize params: %v", err)
	}

	var got map[string]interface{}
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("unmarshal initialize params: %v", err)
	}
	if _, ok := got["initializationOptions"]; ok {
		t.Fatalf("nil initializationOptions should be omitted: %s", payload)
	}
}

func TestConfigurationForSectionReturnsRustAnalyzerSettings(t *testing.T) {
	rustAnalyzerConfig := map[string]interface{}{
		"inlayHints": map[string]interface{}{
			"bindingModeHints": map[string]bool{
				"enable": true,
			},
			"typeHints": map[string]interface{}{
				"enable": true,
			},
		},
	}
	config := map[string]interface{}{
		"rust-analyzer": rustAnalyzerConfig,
	}

	got := configurationForSection(config, "rust-analyzer")
	asMap, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("rust-analyzer section returned %#v, want map", got)
	}
	if asMap["inlayHints"] == nil {
		t.Fatalf("rust-analyzer section missing inlayHints: %#v", asMap)
	}

	got = configurationForSection(config, "rust-analyzer.inlayHints.typeHints.enable")
	if got != true {
		t.Fatalf("nested rust-analyzer setting = %#v, want true", got)
	}

	got = configurationForSection(config, "rust-analyzer.inlayHints.bindingModeHints.enable")
	if got != true {
		t.Fatalf("nested rust-analyzer setting through typed bool map = %#v, want true", got)
	}
}

func TestConfigurationForSectionReturnsTypeScriptInlayHintValues(t *testing.T) {
	preferences := map[string]interface{}{
		"includeInlayParameterNameHints":                   "all",
		"includeInlayFunctionLikeReturnTypeHints":          true,
		"includeInlayVariableTypeHintsWhenTypeMatchesName": true,
	}
	config := map[string]interface{}{
		"typescript": map[string]interface{}{"inlayHints": preferences},
		"javascript": map[string]interface{}{"inlayHints": preferences},
	}

	for _, section := range []string{
		"typescript.inlayHints.includeInlayParameterNameHints",
		"javascript.inlayHints.includeInlayParameterNameHints",
	} {
		got := configurationForSection(config, section)
		if got != "all" {
			t.Fatalf("%s section = %#v, want all", section, got)
		}
	}
}

func TestConfigurationForSectionRequiresWorkspaceConfigurationShape(t *testing.T) {
	for _, tc := range []struct {
		name    string
		config  map[string]interface{}
		section string
	}{
		{
			name: "typescript initialization preferences",
			config: map[string]interface{}{
				"preferences": map[string]interface{}{
					"includeInlayParameterNameHints": "all",
				},
			},
			section: "typescript.inlayHints.includeInlayParameterNameHints",
		},
		{
			name: "rust analyzer initialization options",
			config: map[string]interface{}{
				"inlayHints": map[string]interface{}{
					"typeHints": map[string]interface{}{"enable": true},
				},
			},
			section: "rust-analyzer.inlayHints.typeHints.enable",
		},
		{
			name: "gopls initialization options",
			config: map[string]interface{}{
				"hints": map[string]bool{"parameterNames": true},
			},
			section: "gopls.hints.parameterNames",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := configurationForSection(tc.config, tc.section)
			if got != nil {
				t.Fatalf("%s section = %#v, want nil", tc.section, got)
			}
		})
	}
}

func TestWorkspaceConfigurationReturnsOneResultPerItem(t *testing.T) {
	rustAnalyzerConfig := map[string]interface{}{
		"inlayHints": map[string]interface{}{
			"typeHints": map[string]interface{}{
				"enable": true,
			},
		},
	}
	config := map[string]interface{}{"rust-analyzer": rustAnalyzerConfig}
	client := &Client{workspaceConfig: config}
	raw := json.RawMessage(`{
		"items": [
			{"section": "rust-analyzer"},
			{"section": "rust-analyzer.inlayHints.typeHints.enable"}
		]
	}`)

	got, err := client.workspaceConfiguration(&raw)
	if err != nil {
		t.Fatalf("workspaceConfiguration returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("workspaceConfiguration returned %d items, want 2: %#v", len(got), got)
	}
	if !reflect.DeepEqual(got[0], rustAnalyzerConfig) {
		t.Fatalf("first workspace configuration item = %#v, want %#v", got[0], rustAnalyzerConfig)
	}
	if got[1] != true {
		t.Fatalf("second workspace configuration item = %#v, want true", got[1])
	}
}

func TestWorkspaceConfigurationReturnsEmptyResultForEmptyItems(t *testing.T) {
	client := &Client{workspaceConfig: map[string]interface{}{"hints": map[string]bool{"parameterNames": true}}}
	raw := json.RawMessage(`{"items": []}`)

	got, err := client.workspaceConfiguration(&raw)
	if err != nil {
		t.Fatalf("workspaceConfiguration returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("workspaceConfiguration returned %#v, want empty result for empty items", got)
	}
}

func TestWorkspaceConfigurationRejectsMalformedParams(t *testing.T) {
	client := &Client{}
	raw := json.RawMessage(`{"items":`)

	got, err := client.workspaceConfiguration(&raw)
	if err == nil {
		t.Fatalf("workspaceConfiguration returned result %#v, want invalid params error", got)
	}
	rpcErr, ok := err.(*jsonrpc2.Error)
	if !ok {
		t.Fatalf("workspaceConfiguration error = %T %[1]v, want *jsonrpc2.Error", err)
	}
	if rpcErr.Code != jsonrpc2.CodeInvalidParams {
		t.Fatalf("workspaceConfiguration error code = %d, want %d", rpcErr.Code, jsonrpc2.CodeInvalidParams)
	}
}

func TestWorkspaceConfigurationReturnsNilForUnknownSection(t *testing.T) {
	client := &Client{
		workspaceConfig: map[string]interface{}{
			"typescript": map[string]interface{}{
				"inlayHints": map[string]interface{}{
					"includeInlayParameterNameHints": "all",
				},
			},
		},
	}
	raw := json.RawMessage(`{
		"items": [
			{"section": "typescript.unknown"},
			{"section": "rust-analyzer"}
		]
	}`)

	got, err := client.workspaceConfiguration(&raw)
	if err != nil {
		t.Fatalf("workspaceConfiguration returned error: %v", err)
	}
	if len(got) != 2 || got[0] != nil || got[1] != nil {
		t.Fatalf("workspaceConfiguration returned %#v, want [nil nil]", got)
	}
}

func TestWorkspaceConfigurationResolvesConfiguredServerSections(t *testing.T) {
	client := &Client{
		workspaceConfig: map[string]interface{}{
			"gopls": map[string]interface{}{
				"hints": map[string]bool{
					"parameterNames": true,
				},
			},
			"rust-analyzer": map[string]interface{}{
				"inlayHints": map[string]interface{}{
					"bindingModeHints": map[string]bool{
						"enable": true,
					},
				},
			},
			"typescript": map[string]interface{}{
				"inlayHints": map[string]interface{}{
					"includeInlayParameterNameHints": "all",
				},
			},
			"javascript": map[string]interface{}{
				"inlayHints": map[string]interface{}{
					"includeInlayParameterNameHints": "all",
				},
			},
		},
	}
	raw := json.RawMessage(`{
		"items": [
			{"section": "gopls.hints.parameterNames"},
			{"section": "rust-analyzer.inlayHints.bindingModeHints.enable"},
			{"section": "typescript.inlayHints.includeInlayParameterNameHints"},
			{"section": "javascript.inlayHints.includeInlayParameterNameHints"}
		]
	}`)

	got, err := client.workspaceConfiguration(&raw)
	if err != nil {
		t.Fatalf("workspaceConfiguration returned error: %v", err)
	}
	want := []interface{}{true, true, "all", "all"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("workspaceConfiguration returned %#v, want %#v", got, want)
	}
}
