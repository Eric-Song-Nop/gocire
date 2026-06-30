# gocire

[English](README.md)

`gocire` 从源码文件生成以代码为中心的文档。它可以把单个文件导出成
Markdown 或 MDX，也可以把整个项目导出成 Astro 静态文档站。

现在的主要产品方向是项目级 docsite：

- `docs` 下的源码文件生成 narrative documentation pages，
- `blogs` 下的源码文件生成 blog posts，
- 其他支持的源码文件生成 source context pages，
- LSP 信息用于 hover card、inlay hints 和 jump-to-definition 链接，
- 项目级导出会复用同一个语言服务器 session，并发处理多个文件。

## 生成本项目的文档站

```bash
gocire -project -format astro -lsp -lang go -lsp-root .
cd .gocire/site
pnpm install
pnpm build
pnpm dev -- --host 127.0.0.1
```

默认输出目录是 `.gocire/site`。

如果要自定义生成的 Astro 外壳，可以在 `.gocire.yml` 里设置
`site.templateDir`。这个目录里的文件会按相同相对路径覆盖内置模板，缺失的文件
继续回退到内置模板：

```yaml
site:
  title: My Docs
  templateDir: .gocire/template
```

例如，`.gocire/template/src/styles/global.css` 会替换生成站点里的默认
`src/styles/global.css`。

## 单文件导出

```bash
gocire -src internal/AstroGenerator.go -lang go -format mdx
gocire -src internal/TokenInfo.go -lang go -format markdown
```

## 文档源码

真正的项目文档在 `docs` 和 `blogs` 下的源码文件里。这些文件由 `gocire`
自己渲染，所以示例可以引用真实 API，生成页面也可以保留语法高亮、hover 文档和
跨文件 definition 跳转。

可以从这些页面开始：

- `docs/01_overview.go`
- `docs/02_usage.go`
- `docs/03_architecture.go`
- `docs/04_docsite_generator.go`
- `docs/plans/01_cross_file_jump_to_definition.go`

## 依赖

- Go
- Node.js 和 pnpm，用于生成的 Astro 站点
- 使用 `-lsp -lang go` 时需要 `gopls`
