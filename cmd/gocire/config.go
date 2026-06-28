package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"runtime"
	"time"
)

type Config struct {
	SrcPath          string
	AbsSrcPath       string
	IndexPath        string
	AbsIndexPath     string
	OutPath          string
	Lang             string
	UseLSP           bool
	LSPRoot          string
	Project          bool
	Site             bool
	ConfigPath       string
	Jobs             int
	Format           string
	PrefixDate       bool
	CodeWrapperStart string
	CodeWrapperEnd   string
}

func ParseConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.SrcPath, "src", "", "source file path")
	flag.StringVar(&cfg.IndexPath, "index", "./index.scip", "SCIP Index File Path")
	flag.StringVar(&cfg.OutPath, "output", "", "Output file path for single-file mode, or output directory override for project mode")
	flag.StringVar(&cfg.Lang, "lang", "", "Language for syntax highlighting (optional)")
	flag.BoolVar(&cfg.UseLSP, "lsp", false, "Use LSP for analysis (requires language server installed)")
	flag.StringVar(&cfg.LSPRoot, "lsp-root", "", "Workspace root for lsp")
	flag.BoolVar(&cfg.Project, "project", false, "Export all project files using .gocire.yml")
	flag.BoolVar(&cfg.Site, "site", false, "Alias for -project")
	flag.StringVar(&cfg.ConfigPath, "config", "", "Project config file path (defaults to .gocire.yml)")
	flag.IntVar(&cfg.Jobs, "jobs", runtime.NumCPU(), "Project export concurrency")
	flag.StringVar(&cfg.Format, "format", "mdx", "Output format: markdown, mdx, or astro")
	flag.BoolVar(&cfg.PrefixDate, "date", false, "Prefix output file with current date")
	flag.StringVar(&cfg.CodeWrapperStart, "code-wrapper-start", `<details open="true">
<summary>Expand to view code</summary>
<pre className="cire"><code>`, "Custom opening HTML/JSX for code blocks")
	flag.StringVar(&cfg.CodeWrapperEnd, "code-wrapper-end", `</code></pre>
</details>`, "Custom closing HTML/JSX for code blocks")

	flag.Parse()

	if cfg.Site {
		cfg.Project = true
	}

	if cfg.Format != "markdown" && cfg.Format != "mdx" && cfg.Format != "astro" {
		flag.Usage()
		return nil, fmt.Errorf("format must be 'markdown', 'mdx', or 'astro'")
	}
	if cfg.Format == "astro" && !cfg.ProjectMode() {
		flag.Usage()
		return nil, fmt.Errorf("format 'astro' is only supported with -project")
	}

	if cfg.Jobs < 1 {
		flag.Usage()
		return nil, fmt.Errorf("jobs must be greater than 0")
	}

	var err error
	if cfg.IndexPath != "" {
		cfg.AbsIndexPath, err = filepath.Abs(cfg.IndexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve index path: %w", err)
		}
	}

	if cfg.Project {
		return cfg, nil
	}

	if cfg.SrcPath == "" {
		flag.Usage()
		return nil, fmt.Errorf("source file path is required")
	}

	cfg.AbsSrcPath, err = filepath.Abs(cfg.SrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}

	return cfg, nil
}

func (c *Config) ProjectMode() bool {
	return c != nil && (c.Project || c.Site)
}

// ResolveOutputPath calculates the final output path.
// If OutPath is set, it returns it.
// Otherwise, it generates a path based on the source filename and current date.
func (c *Config) ResolveOutputPath() string {
	if c.OutPath != "" {
		return c.OutPath
	}

	dir := filepath.Dir(c.AbsSrcPath)
	base := filepath.Base(c.AbsSrcPath)
	var prefix string
	if c.PrefixDate {
		prefix = time.Now().Format("2006-01-02")
	}

	ext := ".md"
	if c.Format == "mdx" {
		ext = ".mdx"
	} else if c.Format == "astro" {
		ext = ".astro"
	}

	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, base, ext))
}
