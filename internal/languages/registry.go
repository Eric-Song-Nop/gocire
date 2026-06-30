package languages

import (
	"path/filepath"
	"slices"
	"strings"

	dartsitter "github.com/UserNobody14/tree-sitter-dart/bindings/go"
	"github.com/cockroachdb/errors"
	sitter "github.com/tree-sitter/go-tree-sitter"
	csharpsitter "github.com/tree-sitter/tree-sitter-c-sharp/bindings/go"
	csitter "github.com/tree-sitter/tree-sitter-c/bindings/go"
	cppsitter "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	golangsitter "github.com/tree-sitter/tree-sitter-go/bindings/go"
	haskellsitter "github.com/tree-sitter/tree-sitter-haskell/bindings/go"
	javasitter "github.com/tree-sitter/tree-sitter-java/bindings/go"
	javascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	phpsitter "github.com/tree-sitter/tree-sitter-php/bindings/go"
	pythonsitter "github.com/tree-sitter/tree-sitter-python/bindings/go"
	rubysitter "github.com/tree-sitter/tree-sitter-ruby/bindings/go"
	rustsitter "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	typescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

type LanguageConfig struct {
	SitterLanguage            *sitter.Language
	QueryFileName             string
	LSPCommand                string
	LSPArgs                   []string
	LSPInitializationOptions  map[string]interface{}
	LSPWorkspaceConfiguration map[string]interface{}
	IgnoredCaptures           []string
	Extensions                []string
}

var defaultIgnoredCaptures = []string{"punctuation", "keyword", "operator", "comment", "string"}

var goplsInitializationOptions = map[string]interface{}{
	"hints": map[string]bool{
		"assignVariableTypes":    true,
		"compositeLiteralFields": true,
		"compositeLiteralTypes":  true,
		"constantValues":         true,
		"functionTypeParameters": true,
		"ignoredError":           true,
		"parameterNames":         true,
		"rangeVariableTypes":     true,
	},
}

var typescriptInitializationOptions = map[string]interface{}{
	"preferences": map[string]interface{}{
		"includeInlayEnumMemberValueHints":                      true,
		"includeInlayFunctionLikeReturnTypeHints":               true,
		"includeInlayFunctionParameterTypeHints":                true,
		"includeInlayParameterNameHints":                        "all",
		"includeInlayParameterNameHintsWhenArgumentMatchesName": true,
		"includeInlayPropertyDeclarationTypeHints":              true,
		"includeInlayVariableTypeHints":                         true,
		"includeInlayVariableTypeHintsWhenTypeMatchesName":      true,
	},
}

var typescriptWorkspaceConfiguration = map[string]interface{}{
	"typescript": map[string]interface{}{
		"inlayHints": typescriptInitializationOptions["preferences"],
	},
	"javascript": map[string]interface{}{
		"inlayHints": typescriptInitializationOptions["preferences"],
	},
}

var rustAnalyzerInitializationOptions = map[string]interface{}{
	"files": map[string]interface{}{
		"excludeDirs": []string{".gocire"},
	},
	"inlayHints": map[string]interface{}{
		"bindingModeHints": map[string]bool{
			"enable": true,
		},
		"chainingHints": map[string]bool{
			"enable": true,
		},
		"closureReturnTypeHints": map[string]interface{}{
			"enable": "always",
		},
		"genericParameterHints": map[string]interface{}{
			"const": map[string]bool{
				"enable": true,
			},
			"lifetime": map[string]bool{
				"enable": true,
			},
			"type": map[string]bool{
				"enable": true,
			},
		},
		"lifetimeElisionHints": map[string]interface{}{
			"enable":            "always",
			"useParameterNames": true,
		},
		"parameterHints": map[string]interface{}{
			"enable": true,
		},
		"typeHints": map[string]interface{}{
			"enable": true,
		},
	},
}

var goplsWorkspaceConfiguration = map[string]interface{}{
	"gopls": goplsInitializationOptions,
}

var rustAnalyzerWorkspaceConfiguration = map[string]interface{}{
	"rust-analyzer": rustAnalyzerInitializationOptions,
}

var registry = map[string]LanguageConfig{
	"go": {
		SitterLanguage:            sitter.NewLanguage(golangsitter.Language()),
		QueryFileName:             "go.scm",
		LSPCommand:                "gopls",
		LSPArgs:                   []string{},
		LSPInitializationOptions:  goplsInitializationOptions,
		LSPWorkspaceConfiguration: goplsWorkspaceConfiguration,
		IgnoredCaptures:           defaultIgnoredCaptures,
		Extensions:                []string{".go"},
	},
	"python": {
		SitterLanguage:  sitter.NewLanguage(pythonsitter.Language()),
		QueryFileName:   "python.scm",
		LSPCommand:      "pylsp",
		LSPArgs:         []string{},
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".py"},
	},
	"typescript": {
		SitterLanguage:            sitter.NewLanguage(typescript.LanguageTypescript()),
		QueryFileName:             "typescript.scm",
		LSPCommand:                "typescript-language-server",
		LSPArgs:                   []string{"--stdio"},
		LSPInitializationOptions:  typescriptInitializationOptions,
		LSPWorkspaceConfiguration: typescriptWorkspaceConfiguration,
		IgnoredCaptures:           defaultIgnoredCaptures,
		Extensions:                []string{".ts", ".tsx"},
	},
	"javascript": {
		SitterLanguage:            sitter.NewLanguage(javascript.Language()),
		QueryFileName:             "javascript.scm",
		LSPCommand:                "typescript-language-server",
		LSPArgs:                   []string{"--stdio"},
		LSPInitializationOptions:  typescriptInitializationOptions,
		LSPWorkspaceConfiguration: typescriptWorkspaceConfiguration,
		IgnoredCaptures:           defaultIgnoredCaptures,
		Extensions:                []string{".js", ".jsx"},
	},
	"rust": {
		SitterLanguage:            sitter.NewLanguage(rustsitter.Language()),
		QueryFileName:             "rust.scm",
		LSPCommand:                "rust-analyzer",
		LSPArgs:                   []string{},
		LSPInitializationOptions:  rustAnalyzerInitializationOptions,
		LSPWorkspaceConfiguration: rustAnalyzerWorkspaceConfiguration,
		IgnoredCaptures:           defaultIgnoredCaptures,
		Extensions:                []string{".rs"},
	},
	"cpp": {
		SitterLanguage:  sitter.NewLanguage(cppsitter.Language()),
		QueryFileName:   "cpp.scm",
		LSPCommand:      "clangd",
		LSPArgs:         []string{},
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".cpp", ".cxx", ".cc", ".hpp"},
	},
	"c": {
		SitterLanguage:  sitter.NewLanguage(csitter.Language()),
		QueryFileName:   "c.scm",
		LSPCommand:      "clangd",
		LSPArgs:         []string{},
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".c", ".h"},
	},
	"haskell": {
		SitterLanguage:  sitter.NewLanguage(haskellsitter.Language()),
		QueryFileName:   "haskell.scm",
		LSPCommand:      "haskell-language-server-wrapper",
		LSPArgs:         []string{"--lsp"},
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".hs"},
	},
	"java": {
		SitterLanguage:  sitter.NewLanguage(javasitter.Language()),
		QueryFileName:   "java.scm",
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".java"},
	},
	"ruby": {
		SitterLanguage:  sitter.NewLanguage(rubysitter.Language()),
		QueryFileName:   "ruby.scm",
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".rb"},
	},
	"csharp": {
		SitterLanguage:  sitter.NewLanguage(csharpsitter.Language()),
		QueryFileName:   "c_sharp.scm",
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".cs"},
	},
	"php": {
		SitterLanguage:  sitter.NewLanguage(phpsitter.LanguagePHP()),
		QueryFileName:   "php.scm",
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".php"},
	},
	"dart": {
		SitterLanguage:  sitter.NewLanguage(dartsitter.Language()),
		QueryFileName:   "dart.scm",
		IgnoredCaptures: defaultIgnoredCaptures,
		Extensions:      []string{".dart"},
	},
}

// Aliases
var aliases = map[string]string{
	"golang": "go",
	"js":     "javascript",
	"ts":     "typescript",
	"c++":    "cpp",
	"py":     "python",
	"c#":     "csharp",
	"cs":     "csharp",
	"hs":     "haskell",
}

func GetConfig(language string) (*LanguageConfig, error) {
	lang, err := CanonicalName(language)
	if err != nil {
		return nil, err
	}
	cfg, ok := registry[lang]
	if !ok {
		return nil, errors.Newf("unsupported language: %s", language)
	}
	return &cfg, nil
}

func CanonicalName(language string) (string, error) {
	lang := strings.ToLower(language)
	if canonical, ok := aliases[lang]; ok {
		lang = canonical
	}
	if _, ok := registry[lang]; !ok {
		return "", errors.Newf("unsupported language: %s", language)
	}
	return lang, nil
}

// DetectLanguage attempts to determine the language from the file extension.
// It returns the language name (key in registry) and an error if not found.
func DetectLanguage(filename string) (string, error) {
	ext := strings.ToLower(filepath.Ext(filename))
	for lang, config := range registry {
		if slices.Contains(config.Extensions, ext) {
			return lang, nil
		}
	}
	return "", errors.Newf("could not detect language for extension: %s", ext)
}
