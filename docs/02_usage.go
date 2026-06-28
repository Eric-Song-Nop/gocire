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
// npm install
// npm run build
// npm run dev -- --host 127.0.0.1
// ```
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
// source, resolve links, then render through the selected backend.
func RecommendedProjectJobs() int {
	return 4
}
