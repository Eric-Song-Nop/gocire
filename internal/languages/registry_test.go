package languages

import "testing"

func TestGoConfigEnablesGoplsInlayHints(t *testing.T) {
	cfg, err := GetConfig("go")
	if err != nil {
		t.Fatalf("GetConfig(go) returned error: %v", err)
	}

	hints, ok := cfg.LSPInitializationOptions["hints"].(map[string]bool)
	if !ok {
		t.Fatalf("go LSP initialization options missing hints map: %#v", cfg.LSPInitializationOptions)
	}

	for _, hint := range []string{
		"assignVariableTypes",
		"compositeLiteralFields",
		"compositeLiteralTypes",
		"constantValues",
		"functionTypeParameters",
		"ignoredError",
		"parameterNames",
		"rangeVariableTypes",
	} {
		if !hints[hint] {
			t.Fatalf("gopls hint %q is not enabled: %#v", hint, hints)
		}
	}
}
