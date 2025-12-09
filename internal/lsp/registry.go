package lsp

type LanguageConfig struct {
	Command string
	Args    []string
}

// LanguageRegistry maps language IDs (as used in Tree-sitter or file extensions) to LSP commands.
var LanguageRegistry = map[string]LanguageConfig{
	"go":         {Command: "gopls", Args: []string{}},
	"python":     {Command: "pylsp", Args: []string{}},
	"typescript": {Command: "typescript-language-server", Args: []string{"--stdio"}},
	"javascript": {Command: "typescript-language-server", Args: []string{"--stdio"}},
	"rust":       {Command: "rust-analyzer", Args: []string{}},
	"cpp":        {Command: "clangd", Args: []string{}},
	"c":          {Command: "clangd", Args: []string{}},
	"haskell":    {Command: "haskell-language-server-wrapper", Args: []string{"--lsp"}},
}

func GetConfig(lang string) (LanguageConfig, bool) {
	cfg, ok := LanguageRegistry[lang]
	return cfg, ok
}
