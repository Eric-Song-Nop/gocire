package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Eric-Song-Nop/gocire/internal"
	"golang.org/x/sync/errgroup"
)

// TokenAnalyzer is a common interface for anything that produces tokens from source code.
type TokenAnalyzer interface {
	Analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, error)
}

// DocumentGenerator is a common interface for generating output.
type DocumentGenerator interface {
	Generate(tokens []internal.TokenInfo, comments []internal.CommentInfo) string
}

// Pipeline orchestrates the analysis and generation process.
type Pipeline struct {
	cfg       *Config
	analyzers []TokenAnalyzer
	comments  *internal.CommentAnalyzer
	generator DocumentGenerator
}

// NewPipeline assembles the pipeline based on configuration.
func NewPipeline(cfg *Config) (*Pipeline, error) {
	p := &Pipeline{
		cfg: cfg,
	}

	sourceLines := readSourceLines(cfg.AbsSrcPath)

	// 1. Configure Analyzers
	if cfg.UseLSP {
		// LSP Mode: Exclusive
		if cfg.Lang == "" {
			return nil, fmt.Errorf("language (--lang) is required for LSP analysis")
		}
		fmt.Printf("Starting LSP analysis for %s...\n", cfg.Lang)
		p.analyzers = append(p.analyzers, &LSPWrapper{
			inner: internal.NewLSPAnalyzer(cfg.Lang, cfg.AbsSrcPath, cfg.LSPRoot),
		})
	} else {
		// Static Mode: SCIP + Highlight
		if cfg.AbsIndexPath != "" {
			scipAnalyzer, err := internal.NewSCIPAnalyzer(cfg.AbsIndexPath)
			if err == nil {
				fmt.Printf("Index path: %s\n", cfg.AbsIndexPath)
				p.analyzers = append(p.analyzers, &SCIPWrapper{
					inner:      scipAnalyzer,
					sourcePath: cfg.AbsSrcPath,
				})
			} else {
				fmt.Fprintf(os.Stderr, "Warning: Load SCIP index file failed: %v. SCIP analysis will be skipped.\n", err)
			}
		}

		if cfg.Lang != "" {
			p.analyzers = append(p.analyzers, &HighlightWrapper{
				inner: internal.NewHighlightAnalyzer(cfg.Lang),
			})
		}
	}

	// Comment analysis (if language provided)
	if cfg.Lang != "" {
		p.comments = internal.NewCommentAnalyzer(cfg.Lang)
	}

	// 2. Configure Generator
	if cfg.Format == "mdx" {
		gen := internal.NewMDXGenerator(sourceLines)
		if cfg.CodeWrapperStart != "" {
			gen.CodeWrapperStart = cfg.CodeWrapperStart
		}
		if cfg.CodeWrapperEnd != "" {
			gen.CodeWrapperEnd = cfg.CodeWrapperEnd
		}
		p.generator = &MDXWrapper{inner: gen}
	} else {
		gen := internal.NewMarkdownGenerator(sourceLines)
		if cfg.CodeWrapperStart != "" {
			gen.CodeWrapperStart = cfg.CodeWrapperStart
		}
		if cfg.CodeWrapperEnd != "" {
			gen.CodeWrapperEnd = cfg.CodeWrapperEnd
		}
		p.generator = &MarkdownWrapper{inner: gen}
	}

	return p, nil
}

func (p *Pipeline) Run() error {
	fmt.Printf("Source path: %s\n", p.cfg.AbsSrcPath)

	content, err := os.ReadFile(p.cfg.AbsSrcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file: %w", err)
	}

	g, ctx := errgroup.WithContext(context.Background())

	// Run Token Analyzers
	results := make([][]internal.TokenInfo, len(p.analyzers))
	for i, analyzer := range p.analyzers {
		i, analyzer := i, analyzer
		g.Go(func() error {
			tokens, err := analyzer.Analyze(ctx, content)
			if err != nil {
				return err
			}
			results[i] = tokens
			return nil
		})
	}

	// Run Comment Analyzer
	var comments []internal.CommentInfo
	if p.comments != nil {
		g.Go(func() error {
			var err error
			// CommentAnalyzer doesn't support context cancellation yet, but that's fine for now
			comments, err = p.comments.Analyze(content)
			return err
		})
	}

	if err := g.Wait(); err != nil {
		return fmt.Errorf("analysis failed: %w", err)
	}

	// Merge Results
	var allTokens []internal.TokenInfo
	for _, res := range results {
		allTokens = append(allTokens, res...)
	}

	// Post-Processing
	internal.SortBySpan(allTokens)
	internal.SortBySpan(comments)

	allTokens, err = internal.MergeSplitTokens(allTokens)
	if err != nil {
		return fmt.Errorf("merge split tokens failed: %w", err)
	}

	// Generate Output
	output := p.generator.Generate(allTokens, comments)

	// Write File
	outPath := p.cfg.ResolveOutputPath()
	if err := os.WriteFile(outPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	fmt.Printf("%s generated at: %s\n", p.cfg.Format, outPath)
	return nil
}

// --- Wrappers ---

type LSPWrapper struct {
	inner *internal.LSPAnalyzer
}

func (w *LSPWrapper) Analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, error) {
	// LSPAnalyzer doesn't take context yet, but we conform to the interface
	return w.inner.Analyze(content)
}

type HighlightWrapper struct {
	inner *internal.HighlightAnalyzer
}

func (w *HighlightWrapper) Analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, error) {
	return w.inner.Analyze(content)
}

type SCIPWrapper struct {
	inner      *internal.SCIPAnalyzer
	sourcePath string
}

func (w *SCIPWrapper) Analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, error) {
	// SCIP uses path, not content
	return w.inner.Analyze(w.sourcePath), nil
}

type MarkdownWrapper struct {
	inner *internal.MarkdownGenerator
}

func (w *MarkdownWrapper) Generate(tokens []internal.TokenInfo, comments []internal.CommentInfo) string {
	// Markdown generator ignores comments
	return w.inner.GenerateMarkdown(tokens)
}

type MDXWrapper struct {
	inner *internal.MDXGenerator
}

func (w *MDXWrapper) Generate(tokens []internal.TokenInfo, comments []internal.CommentInfo) string {
	return w.inner.GenerateMDX(tokens, comments)
}

// Helper to read source lines (needed for generators)
// In a real scenario, we might want to pass content directly to generators instead of re-reading or splitting again.
// For now, we mimic the existing logic.
func readSourceLines(path string) []string {
	content, _ := os.ReadFile(path) // Error handled in Run(), here we just need lines for init
	return strings.Split(string(content), "\n")
}
