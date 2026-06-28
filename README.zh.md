# gocire

[English](README.md)

`gocire` 从源码文件生成以代码为中心的文档。它可以把单个文件导出成
Markdown 或 MDX，也可以把整个项目导出成 Astro 静态文档站。

现在的主要产品方向是项目级 docsite：

- `docs` 下的源码文件生成 narrative documentation pages，
- `blogs` 下的源码文件生成 blog posts，
- 其他支持的源码文件生成 source context pages，
- LSP 信息用于 hover card 和 jump-to-definition 链接。

## 生成本项目的文档站

```bash
gocire -project -format astro -lsp -lang go -lsp-root .
cd .gocire/site
npm install
npm run build
npm run dev -- --host 127.0.0.1
```

默认输出目录是 `.gocire/site`。

## 单文件导出

```bash
gocire -src internal/AstroGenerator.go -lang go -format mdx
gocire -src internal/TokenInfo.go -lang go -format markdown
```

## 文档源码

真正的项目文档在 `docs` 和 `blogs` 下的源码文件里。这些文件由 `gocire`
自己渲染，所以示例可以引用真实 API，生成页面也可以保留语法高亮、hover 文档和
definition 跳转。

可以从这些页面开始：

- `docs/01_overview.go`
- `docs/02_usage.go`
- `docs/03_architecture.go`
- `docs/04_docsite_generator.go`
- `docs/plans/01_cross_file_jump_to_definition.go`

## 依赖

- Go
- Node.js 和 npm，用于生成的 Astro 站点
- 使用 `-lsp -lang go` 时需要 `gopls`
