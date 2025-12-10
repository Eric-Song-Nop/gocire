package main

import (
	"flag"
	"fmt"
	"path/filepath"
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
	Format           string
	PrefixDate       bool
	CodeWrapperStart string
	CodeWrapperEnd   string
}

func ParseConfig() (*Config, error) {
	cfg := &Config{}

	flag.StringVar(&cfg.SrcPath, "src", "", "source file path")
	flag.StringVar(&cfg.IndexPath, "index", "./index.scip", "SCIP Index File Path")
	flag.StringVar(&cfg.OutPath, "output", "", "Output file path (optional). Defaults to source file path with appropriate extension")
	flag.StringVar(&cfg.Lang, "lang", "", "Language for syntax highlighting (optional)")
	flag.BoolVar(&cfg.UseLSP, "lsp", false, "Use LSP for analysis (requires language server installed)")
	flag.StringVar(&cfg.Format, "format", "mdx", "Output format: markdown or mdx")
	flag.BoolVar(&cfg.PrefixDate, "date", false, "Prefix output file with current date")
	flag.StringVar(&cfg.CodeWrapperStart, "code-wrapper-start", `<details open="true">
<summary>Expand to view code</summary>
<pre className="cire"><code>`, "Custom opening HTML/JSX for code blocks")
	flag.StringVar(&cfg.CodeWrapperEnd, "code-wrapper-end", `</code></pre>
</details>`, "Custom closing HTML/JSX for code blocks")

	flag.Parse()

	if cfg.SrcPath == "" {
		flag.Usage()
		return nil, fmt.Errorf("source file path is required")
	}

	if cfg.Format != "markdown" && cfg.Format != "mdx" {
		flag.Usage()
		return nil, fmt.Errorf("format must be 'markdown' or 'mdx'")
	}

	var err error
	cfg.AbsSrcPath, err = filepath.Abs(cfg.SrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}

	// Index path is optional but we resolve it if present
	if cfg.IndexPath != "" {
		cfg.AbsIndexPath, err = filepath.Abs(cfg.IndexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve index path: %w", err)
		}
	}

	return cfg, nil
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
		prefix = prefix + "-"
	}

	ext := ".md"
	if c.Format == "mdx" {
		ext = ".mdx"
	}

	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", prefix, base, ext))
}
