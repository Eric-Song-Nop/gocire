package internal

import (
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

func GetLanguageAndQuery(language string) (*sitter.Language, string, error) {
	switch strings.ToLower(language) {
	case "go", "golang":
		return sitter.NewLanguage(golangsitter.Language()), "go.scm", nil
	case "java":
		return sitter.NewLanguage(javasitter.Language()), "java.scm", nil
	case "js", "javascript":
		return sitter.NewLanguage(javascript.Language()), "javascript.scm", nil
	case "ts", "typescript":
		return sitter.NewLanguage(typescript.LanguageTypescript()), "typescript.scm", nil
	case "rust":
		return sitter.NewLanguage(rustsitter.Language()), "rust.scm", nil
	case "c":
		return sitter.NewLanguage(csitter.Language()), "c.scm", nil
	case "cpp", "c++":
		return sitter.NewLanguage(cppsitter.Language()), "cpp.scm", nil
	case "ruby":
		return sitter.NewLanguage(rubysitter.Language()), "ruby.scm", nil
	case "python", "py":
		return sitter.NewLanguage(pythonsitter.Language()), "python.scm", nil
	case "csharp", "c#", "cs":
		return sitter.NewLanguage(csharpsitter.Language()), "c_sharp.scm", nil
	case "php":
		return sitter.NewLanguage(phpsitter.LanguagePHP()), "php.scm", nil
	case "haskell":
		return sitter.NewLanguage(haskellsitter.Language()), "haskell.scm", nil
	case "dart":
		return sitter.NewLanguage(dartsitter.Language()), "dart.scm", nil
	default:
		return nil, "", errors.Newf("unsupported language: %s", language)
	}
}
