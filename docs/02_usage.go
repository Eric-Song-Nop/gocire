package docs

// # Usage
//
// Generate the Astro docsite for this repository:
//
// ```bash
// gocire -project -format astro -lsp -lang go -lsp-root .
// ```
//
// The default output directory is `.gocire/site`. The generated directory is an
// Astro project, so it can be built or served with normal Astro commands:
//
// ```bash
// cd .gocire/site
// pnpm install
// pnpm build
// pnpm dev -- --host 127.0.0.1
// ```
//
// A project can customize the generated Astro shell with `.gocire.yml`. The
// built-in template remains the base, and `site.templateDir` overlays files by
// the same relative path:
//
// ```yaml
// site:
//
//	title: My Docs
//	templateDir: .gocire/template
//
// ```
//
// For example, `.gocire/template/src/styles/global.css` replaces the built-in
// `src/styles/global.css`. Missing files fall back to the embedded template.
// Template files such as `src/layouts/SiteLayout.astro.tmpl` are still rendered
// by `gocire`, so values like the site title remain available.
//
// Single-file export still exists for Markdown and MDX:
//
// ```bash
// gocire -src internal/AstroGenerator.go -lang go -format mdx
// gocire -src internal/TokenInfo.go -lang go -format markdown
// ```
func ProjectExportCommand() []string {
	return []string{
		"gocire",
		"-project",
		"-format",
		"astro",
		"-lsp",
		"-lang",
		"go",
		"-lsp-root",
		".",
	}
}

// Project export can run files concurrently while sharing one language server
// session for the project. The pipeline for each file remains linear: analyze
// source, resolve links and inlay hints, then render through the selected
// backend.
func RecommendedProjectJobs() int {
	return 4
}
