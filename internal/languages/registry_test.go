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

func TestTypeScriptConfigEnablesInlayHints(t *testing.T) {
	cfg, err := GetConfig("typescript")
	if err != nil {
		t.Fatalf("GetConfig(typescript) returned error: %v", err)
	}

	preferences, ok := cfg.LSPInitializationOptions["preferences"].(map[string]interface{})
	if !ok {
		t.Fatalf("typescript LSP initialization options missing preferences: %#v", cfg.LSPInitializationOptions)
	}

	if preferences["includeInlayParameterNameHints"] != "all" {
		t.Fatalf("includeInlayParameterNameHints = %#v, want all", preferences["includeInlayParameterNameHints"])
	}
	for _, hint := range []string{
		"includeInlayEnumMemberValueHints",
		"includeInlayFunctionLikeReturnTypeHints",
		"includeInlayFunctionParameterTypeHints",
		"includeInlayParameterNameHintsWhenArgumentMatchesName",
		"includeInlayPropertyDeclarationTypeHints",
		"includeInlayVariableTypeHints",
		"includeInlayVariableTypeHintsWhenTypeMatchesName",
	} {
		if preferences[hint] != true {
			t.Fatalf("typescript hint %q is not enabled: %#v", hint, preferences)
		}
	}
}

func TestRustConfigEnablesRustAnalyzerInlayHints(t *testing.T) {
	cfg, err := GetConfig("rust")
	if err != nil {
		t.Fatalf("GetConfig(rust) returned error: %v", err)
	}

	inlayHints, ok := cfg.LSPInitializationOptions["inlayHints"].(map[string]interface{})
	if !ok {
		t.Fatalf("rust LSP initialization options missing inlayHints: %#v", cfg.LSPInitializationOptions)
	}

	for _, path := range [][]string{
		{"bindingModeHints", "enable"},
		{"chainingHints", "enable"},
		{"genericParameterHints", "const", "enable"},
		{"genericParameterHints", "lifetime", "enable"},
		{"genericParameterHints", "type", "enable"},
		{"parameterHints", "enable"},
		{"typeHints", "enable"},
	} {
		if got := nestedBool(t, inlayHints, path...); !got {
			t.Fatalf("rust hint %v is not enabled: %#v", path, inlayHints)
		}
	}
}

func TestWorkspaceConfigurationsExposeServerSections(t *testing.T) {
	goCfg, err := GetConfig("go")
	if err != nil {
		t.Fatalf("GetConfig(go) returned error: %v", err)
	}
	gopls, ok := goCfg.LSPWorkspaceConfiguration["gopls"].(map[string]interface{})
	if !ok {
		t.Fatalf("go workspace configuration missing gopls section: %#v", goCfg.LSPWorkspaceConfiguration)
	}
	goHints, ok := gopls["hints"].(map[string]bool)
	if !ok || !goHints["parameterNames"] {
		t.Fatalf("go workspace configuration missing parameterNames hint: %#v", gopls)
	}

	tsCfg, err := GetConfig("typescript")
	if err != nil {
		t.Fatalf("GetConfig(typescript) returned error: %v", err)
	}
	for _, language := range []string{"typescript", "javascript"} {
		languageConfig, ok := tsCfg.LSPWorkspaceConfiguration[language].(map[string]interface{})
		if !ok {
			t.Fatalf("typescript workspace configuration missing %s section: %#v", language, tsCfg.LSPWorkspaceConfiguration)
		}
		inlayHints, ok := languageConfig["inlayHints"].(map[string]interface{})
		if !ok {
			t.Fatalf("typescript workspace configuration missing %s.inlayHints: %#v", language, languageConfig)
		}
		if inlayHints["includeInlayParameterNameHints"] != "all" {
			t.Fatalf("%s.inlayHints.includeInlayParameterNameHints = %#v, want all", language, inlayHints["includeInlayParameterNameHints"])
		}
	}

	rustCfg, err := GetConfig("rust")
	if err != nil {
		t.Fatalf("GetConfig(rust) returned error: %v", err)
	}
	rustAnalyzer, ok := rustCfg.LSPWorkspaceConfiguration["rust-analyzer"].(map[string]interface{})
	if !ok {
		t.Fatalf("rust workspace configuration missing rust-analyzer section: %#v", rustCfg.LSPWorkspaceConfiguration)
	}
	inlayHints, ok := rustAnalyzer["inlayHints"].(map[string]interface{})
	if !ok {
		t.Fatalf("rust workspace configuration missing rust-analyzer.inlayHints: %#v", rustAnalyzer)
	}
	if got := nestedBool(t, inlayHints, "bindingModeHints", "enable"); !got {
		t.Fatalf("rust workspace configuration missing binding mode hints: %#v", inlayHints)
	}

	files, ok := rustAnalyzer["files"].(map[string]interface{})
	if !ok {
		t.Fatalf("rust workspace configuration missing rust-analyzer.files: %#v", rustAnalyzer)
	}
	excludeDirs, ok := files["excludeDirs"].([]string)
	if !ok {
		t.Fatalf("rust workspace configuration missing files.excludeDirs: %#v", files)
	}
	if len(excludeDirs) != 1 || excludeDirs[0] != ".gocire" {
		t.Fatalf("rust files.excludeDirs = %#v, want [.gocire]", excludeDirs)
	}
}

func nestedBool(t *testing.T, root map[string]interface{}, path ...string) bool {
	t.Helper()

	var current interface{} = root
	for _, key := range path {
		m, ok := current.(map[string]interface{})
		if !ok {
			if typed, typedOK := current.(map[string]bool); typedOK {
				return typed[key]
			}
			t.Fatalf("path %v reached non-map value %#v", path, current)
		}
		current = m[key]
	}

	value, ok := current.(bool)
	if !ok {
		t.Fatalf("path %v reached non-bool value %#v", path, current)
	}
	return value
}
