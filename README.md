# gocire

[中文文档](README.zh.md)

`gocire` is a CLI tool that turns one source file into MDX or Markdown from the
source code itself. It is designed for code-oriented documentation, especially
when the generated MDX is used inside a [Docusaurus](https://docusaurus.io/)
site.

The current implementation is single-file oriented: it analyzes one source file,
extracts language information, and renders one output file.

## What It Does

- Generates MDX or Markdown from a source file.
- Uses Tree-sitter for syntax highlighting and comment extraction.
- Turns standalone source comments into prose in MDX output.
- Interleaves prose and semantic code blocks in source order.
- Optionally reads a SCIP index for symbol roles and hover documentation.
- Optionally starts an LSP server for hover and definition information.
- Supports custom code block wrappers for Docusaurus or other renderers.

## Current Scope

`gocire` currently generates documentation for one source file at a time.

It does not yet generate a complete static website, route map, sidebar, or
site-aware cross-file navigation. SCIP and LSP can provide semantic information,
but the current renderers produce links for the generated file rather than a
full repository-wide documentation site.

## Installation

```bash
go install github.com/Eric-Song-Nop/gocire/cmd/gocire@latest
```

## Usage

Generate MDX for a source file:

```bash
gocire -src cmd/gocire/main.go
```

Specify the language explicitly:

```bash
gocire -src internal/LSPAnalyzer.go -lang go
```

Generate Markdown instead of MDX:

```bash
gocire -src internal/TokenInfo.go -format markdown -output TokenInfo.md
```

Use an LSP server:

```bash
gocire -src internal/LSPAnalyzer.go -lang go -lsp -lsp-root .
```

Use a SCIP index:

```bash
scip-go
gocire -src internal/LSPAnalyzer.go -lang go -index index.scip
```

## How Source Becomes Documentation

For MDX output, `gocire` follows the current source-order rendering model:

- Standalone comments become prose.
- Inline comments remain inside code.
- Source code between standalone comments becomes semantic code blocks.
- Code blocks keep syntax highlighting and available symbol information.

Example source:

```go
// This paragraph becomes prose.
func main() {
    println("hello") // this comment stays in code
}
```

The generated MDX contains prose followed by a rendered code block.

Markdown output currently renders a code block with syntax and symbol markup. It
does not interleave extracted comments as prose.

## Analysis Modes

### Tree-sitter

Tree-sitter is used for:

- language parsing,
- syntax highlighting,
- comment extraction,
- finding candidate tokens for LSP queries.

### SCIP Mode

By default, `gocire` tries to load the SCIP index at `./index.scip`.

If the index loads successfully, SCIP occurrences are used to add:

- symbol IDs,
- definition/reference roles,
- hover documentation from SCIP symbol information.

If the index cannot be loaded, `gocire` prints a warning and continues with the
other available analyzers.

### LSP Mode

When `-lsp` is set, `gocire` starts the configured language server for the
selected language.

LSP mode currently uses:

- `textDocument/hover`,
- `textDocument/definition`.

The LSP server is started during generation only. The generated output is still
static.

## CLI Flags

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-src` | Source file to analyze. | Required |
| `-lang` | Language ID. If omitted, `gocire` tries to detect it from the file extension. | Auto-detected when possible |
| `-index` | SCIP index path. Used when not running with `-lsp`. | `./index.scip` |
| `-output` | Output file path. | Generated next to the source file |
| `-format` | Output format: `mdx` or `markdown`. | `mdx` |
| `-lsp` | Use the configured language server instead of SCIP mode. | `false` |
| `-lsp-root` | Workspace root passed to the language server. | Source file directory |
| `-date` | Prefix generated output with the current date. | `false` |
| `-code-wrapper-start` | Opening HTML/JSX wrapper for generated code blocks. | `<details ...><pre className="cire"><code>` |
| `-code-wrapper-end` | Closing HTML/JSX wrapper for generated code blocks. | `</code></pre></details>` |

When `-output` is omitted, the current implementation writes a generated file
next to the source file. Use `-output` when you need a specific path.

## Supported Languages

`gocire` supports these language IDs and aliases:

| Language | IDs / aliases | Extensions | LSP command when configured |
| :--- | :--- | :--- | :--- |
| C | `c` | `.c`, `.h` | `clangd` |
| C++ | `cpp`, `c++` | `.cpp`, `.cxx`, `.cc`, `.hpp` | `clangd` |
| C# | `csharp`, `c#`, `cs` | `.cs` | - |
| Dart | `dart` | `.dart` | - |
| Go | `go`, `golang` | `.go` | `gopls` |
| Haskell | `haskell`, `hs` | `.hs` | `haskell-language-server-wrapper --lsp` |
| Java | `java` | `.java` | - |
| JavaScript | `javascript`, `js` | `.js`, `.jsx` | `typescript-language-server --stdio` |
| PHP | `php` | `.php` | - |
| Python | `python`, `py` | `.py` | `pylsp` |
| Ruby | `ruby` | `.rb` | - |
| Rust | `rust` | `.rs` | `rust-analyzer` |
| TypeScript | `typescript`, `ts` | `.ts`, `.tsx` | `typescript-language-server --stdio` |

Languages without a configured LSP command can still use Tree-sitter syntax
highlighting and comment extraction.

## Docusaurus Integration

MDX output can be placed inside a Docusaurus docs directory.

```bash
gocire -src internal/LSPAnalyzer.go -lang go -output docs/LSPAnalyzer.mdx
```

The default CLI wrapper emits code blocks with the `.cire` class:

```html
<pre className="cire"><code>
```

You can use `examples/gruvbox.css` as a starting point for styling generated
code blocks in Docusaurus.

For hover cards in generated MDX, install `@rc-component/tooltip` and expose it
as an MDX component:

```bash
pnpm i @rc-component/tooltip
```

```ts
import Tooltip from "@rc-component/tooltip";
import MDXComponents from "@theme-original/MDXComponents";

export default {
  ...MDXComponents,
  Tooltip,
};
```

Hover documentation is rendered from Markdown to HTML. If your hover content
uses math, configure KaTeX in Docusaurus as usual.

## Current Limitations

- The CLI is single-file oriented.
- Generated links are not yet aware of a full static site route map.
- Markdown output does not currently turn comments into prose.
- LSP mode requires the relevant language server to be installed.
- LSP mode currently uses hover and definition requests; inlay hints are not
  implemented.
- Cross-file definition locations returned by LSP are not yet rendered as
  complete site-aware links.
