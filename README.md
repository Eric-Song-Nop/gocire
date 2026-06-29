# gocire

[中文文档](README.zh.md)

`gocire` generates code-centered documentation from source files. It can export
single files to Markdown or MDX, and it can export a whole project as an Astro
static documentation site.

The main product direction is the project docsite:

- files under `docs` become narrative documentation pages,
- files under `blogs` become blog posts,
- other supported source files become source context pages,
- LSP data powers hover cards, inlay hints, and jump-to-definition links,
- project export reuses one language-server session while processing files
  concurrently.

## Generate This Project's Docsite

```bash
gocire -project -format astro -lsp -lang go -lsp-root .
cd .gocire/site
pnpm install
pnpm build
pnpm dev -- --host 127.0.0.1
```

The generated Astro project is written to `.gocire/site` by default.

## Single-File Export

```bash
gocire -src internal/AstroGenerator.go -lang go -format mdx
gocire -src internal/TokenInfo.go -lang go -format markdown
```

## Documentation Source

The real project documentation lives in source files under `docs` and `blogs`.
Those files are rendered by `gocire` itself, so examples can point at real APIs
and generated pages can preserve syntax highlighting, hover text, and definition
navigation across the repository.

Start with:

- `docs/01_overview.go`
- `docs/02_usage.go`
- `docs/03_architecture.go`
- `docs/04_docsite_generator.go`
- `docs/plans/01_cross_file_jump_to_definition.go`

## Requirements

- Go
- Node.js and pnpm for the generated Astro site
- `gopls` when using `-lsp -lang go`
