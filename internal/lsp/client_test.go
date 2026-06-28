package lsp

import (
	"encoding/json"
	"path/filepath"
	"testing"
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
