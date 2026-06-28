# gocire

[English](README.md)

`gocire` 是一个 CLI 工具，用来直接从源码文件生成 MDX 或 Markdown。它面向
代码导向的文档写作，尤其适合把生成的 MDX 放进
[Docusaurus](https://docusaurus.io/) 站点。

当前实现以单文件为中心：分析一个源码文件，提取语言信息，然后生成一个输出文件。

## 它做什么

- 从源码文件生成 MDX 或 Markdown。
- 使用 Tree-sitter 做语法高亮和注释提取。
- 在 MDX 输出中，把独立成行的源码注释转换成正文。
- 按源码顺序交织正文和语义代码块。
- 可选读取 SCIP 索引，获取符号角色和 hover 文档。
- 可选启动 LSP 服务器，获取 hover 和 definition 信息。
- 支持自定义代码块 wrapper，方便接入 Docusaurus 或其他渲染环境。

## 当前范围

`gocire` 当前一次只为一个源码文件生成文档。

它还不是完整静态站点生成器；目前不会生成全站路由、sidebar、站点地图，也不会
生成完整的跨文件站点级跳转。SCIP 和 LSP 可以提供语义信息，但当前 renderer 输出
的是单个生成文件内可使用的链接，而不是完整代码库文档站的链接系统。

## 安装

```bash
go install github.com/Eric-Song-Nop/gocire/cmd/gocire@latest
```

## 用法

为源码文件生成 MDX：

```bash
gocire -src cmd/gocire/main.go
```

显式指定语言：

```bash
gocire -src internal/LSPAnalyzer.go -lang go
```

生成 Markdown：

```bash
gocire -src internal/TokenInfo.go -format markdown -output TokenInfo.md
```

使用 LSP：

```bash
gocire -src internal/LSPAnalyzer.go -lang go -lsp -lsp-root .
```

使用 SCIP 索引：

```bash
scip-go
gocire -src internal/LSPAnalyzer.go -lang go -index index.scip
```

## 源码如何变成文档

对 MDX 输出，`gocire` 沿用当前的源码顺序渲染模型：

- 独立成行的注释变成正文。
- 行内注释仍然留在代码里。
- 独立注释之间的源码变成语义代码块。
- 代码块保留语法高亮和可用的符号信息。

示例源码：

```go
// This paragraph becomes prose.
func main() {
    println("hello") // this comment stays in code
}
```

生成的 MDX 会包含正文，然后跟随一个渲染后的代码块。

Markdown 输出当前主要生成带语法和符号标记的代码块，不会把提取出的注释交织成正文。

## 分析模式

### Tree-sitter

Tree-sitter 用于：

- 语言解析，
- 语法高亮，
- 注释提取，
- 在 LSP 模式下寻找候选 token。

### SCIP 模式

默认情况下，`gocire` 会尝试读取 `./index.scip`。

如果索引加载成功，SCIP occurrence 会用于添加：

- symbol ID，
- definition/reference 角色，
- SCIP symbol information 中的 hover 文档。

如果索引加载失败，`gocire` 会打印 warning，然后继续使用其他可用分析器。

### LSP 模式

设置 `-lsp` 时，`gocire` 会启动当前语言配置的 language server。

当前 LSP 模式使用：

- `textDocument/hover`，
- `textDocument/definition`。

语言服务器只在生成期间运行；生成结果仍然是静态文件。

## CLI 参数

| 参数 | 说明 | 默认值 |
| :--- | :--- | :--- |
| `-src` | 要分析的源码文件。 | 必填 |
| `-lang` | 语言 ID。如果不提供，会尝试根据文件扩展名自动识别。 | 可自动识别 |
| `-index` | SCIP 索引路径。非 `-lsp` 模式下使用。 | `./index.scip` |
| `-output` | 输出文件路径。 | 在源码文件旁生成 |
| `-format` | 输出格式：`mdx` 或 `markdown`。 | `mdx` |
| `-lsp` | 使用配置好的 language server，而不是 SCIP 模式。 | `false` |
| `-lsp-root` | 传给 language server 的 workspace root。 | 源文件所在目录 |
| `-date` | 给生成文件加当前日期前缀。 | `false` |
| `-code-wrapper-start` | 生成代码块的开头 HTML/JSX wrapper。 | `<details ...><pre className="cire"><code>` |
| `-code-wrapper-end` | 生成代码块的结尾 HTML/JSX wrapper。 | `</code></pre></details>` |

如果没有设置 `-output`，当前实现会在源码文件旁边生成输出文件。需要稳定路径时建议
显式使用 `-output`。

## 支持语言

`gocire` 支持以下语言 ID 和别名：

| 语言 | ID / 别名 | 扩展名 | 已配置的 LSP 命令 |
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

没有配置 LSP 命令的语言仍然可以使用 Tree-sitter 语法高亮和注释提取。

## Docusaurus 集成

MDX 输出可以直接放入 Docusaurus 的 docs 目录。

```bash
gocire -src internal/LSPAnalyzer.go -lang go -output docs/LSPAnalyzer.mdx
```

默认 CLI wrapper 会生成带 `.cire` class 的代码块：

```html
<pre className="cire"><code>
```

可以使用 `examples/gruvbox.css` 作为 Docusaurus 中生成代码块的样式起点。

如果要让生成的 MDX hover card 正常工作，需要安装 `@rc-component/tooltip` 并暴露为
MDX component：

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

Hover 文档会先从 Markdown 渲染成 HTML。如果 hover 内容使用数学公式，请按
Docusaurus 的常规方式配置 KaTeX。

## 当前限制

- CLI 仍然以单文件为中心。
- 生成链接还不了解完整静态站点的路由结构。
- Markdown 输出目前不会把注释转换成正文。
- LSP 模式要求本机已安装对应 language server。
- LSP 模式当前使用 hover 和 definition 请求；尚未实现 inlay hints。
- LSP 返回的跨文件 definition 位置目前还没有渲染成完整的站点级链接。
