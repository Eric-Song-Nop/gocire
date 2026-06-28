package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
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

type PipelineLSPRequest struct {
	SourcePath    string
	Language      string
	WorkspaceRoot string
}

type LSPAnalyzerFactory func(ctx context.Context, req PipelineLSPRequest) (TokenAnalyzer, error)

type PipelineOptions struct {
	Context            context.Context
	LSPAnalyzerFactory LSPAnalyzerFactory
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
	return NewPipelineWithOptions(cfg, PipelineOptions{})
}

func NewPipelineWithOptions(cfg *Config, options PipelineOptions) (*Pipeline, error) {
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
		if options.LSPAnalyzerFactory != nil {
			optionsCtx := options.Context
			if optionsCtx == nil {
				optionsCtx = context.Background()
			}
			analyzer, err := options.LSPAnalyzerFactory(optionsCtx, PipelineLSPRequest{
				SourcePath:    cfg.AbsSrcPath,
				Language:      cfg.Lang,
				WorkspaceRoot: cfg.LSPRoot,
			})
			if err != nil {
				return nil, err
			}
			if analyzer == nil {
				return nil, fmt.Errorf("lsp analyzer factory returned nil")
			}
			p.analyzers = append(p.analyzers, analyzer)
		} else {
			p.analyzers = append(p.analyzers, &LSPWrapper{
				inner: internal.NewLSPAnalyzer(cfg.Lang, cfg.AbsSrcPath, cfg.LSPRoot),
			})
		}
		p.analyzers = append(p.analyzers, &HighlightWrapper{
			inner: internal.NewHighlightAnalyzer(cfg.Lang),
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
	switch cfg.Format {
	case "mdx":
		gen := internal.NewMDXGenerator(sourceLines)
		if cfg.CodeWrapperStart != "" {
			gen.CodeWrapperStart = cfg.CodeWrapperStart
		}
		if cfg.CodeWrapperEnd != "" {
			gen.CodeWrapperEnd = cfg.CodeWrapperEnd
		}
		p.generator = &MDXWrapper{inner: gen}
	case "markdown":
		gen := internal.NewMarkdownGenerator(sourceLines)
		if cfg.CodeWrapperStart != "" {
			gen.CodeWrapperStart = cfg.CodeWrapperStart
		}
		if cfg.CodeWrapperEnd != "" {
			gen.CodeWrapperEnd = cfg.CodeWrapperEnd
		}
		p.generator = &MarkdownWrapper{inner: gen}
	case "astro":
		// Project backends reuse Pipeline.AnalyzeFile and generate Astro pages outside
		// the single-file pipeline.
	default:
		return nil, fmt.Errorf("unsupported format %q", cfg.Format)
	}

	return p, nil
}

type PipelineRunOptions struct {
	Context    context.Context
	Manifest   *internal.SourceRouteManifest
	OutputPath string
}

type PipelineAnalysis struct {
	SourceLines []string
	Tokens      []internal.TokenInfo
	Comments    []internal.CommentInfo
}

func (p *Pipeline) Run() error {
	manifest, err := p.sourceRouteManifest()
	var manifestPtr *internal.SourceRouteManifest
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: build source route manifest failed: %v\n", err)
	} else {
		manifestPtr = &manifest
	}

	return p.RunFile(PipelineRunOptions{
		Context:    context.Background(),
		Manifest:   manifestPtr,
		OutputPath: p.cfg.ResolveOutputPath(),
	})
}

func (p *Pipeline) RunFile(opts PipelineRunOptions) error {
	analysis, err := p.AnalyzeFile(opts)
	if err != nil {
		return err
	}

	if p.generator == nil {
		return fmt.Errorf("format %q does not support single-file pipeline generation", p.cfg.Format)
	}

	output := p.generator.Generate(analysis.Tokens, analysis.Comments)

	outPath := opts.OutputPath
	if outPath == "" {
		outPath = p.cfg.ResolveOutputPath()
	}
	if err := writeOutputFile(outPath, output); err != nil {
		return err
	}

	fmt.Printf("%s generated at: %s\n", p.cfg.Format, outPath)
	return nil
}

func (p *Pipeline) AnalyzeFile(opts PipelineRunOptions) (*PipelineAnalysis, error) {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	fmt.Printf("Source path: %s\n", p.cfg.AbsSrcPath)

	content, err := os.ReadFile(p.cfg.AbsSrcPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read source file: %w", err)
	}

	allTokens, comments, err := p.analyze(ctx, content)
	if err != nil {
		return nil, err
	}

	allTokens, err = p.mergeSortSplit(allTokens, comments)
	if err != nil {
		return nil, err
	}

	if opts.Manifest != nil {
		p.resolveTokenLinksWithManifest(allTokens, *opts.Manifest)
	}

	return &PipelineAnalysis{
		SourceLines: strings.Split(string(content), "\n"),
		Tokens:      allTokens,
		Comments:    comments,
	}, nil
}

func (p *Pipeline) analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, []internal.CommentInfo, error) {
	g, ctx := errgroup.WithContext(ctx)

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
		return nil, nil, fmt.Errorf("analysis failed: %w", err)
	}

	// Merge Results
	var allTokens []internal.TokenInfo
	for _, res := range results {
		allTokens = append(allTokens, res...)
	}

	return allTokens, comments, nil
}

func (p *Pipeline) mergeSortSplit(allTokens []internal.TokenInfo, comments []internal.CommentInfo) ([]internal.TokenInfo, error) {
	internal.SortBySpan(allTokens)
	internal.SortBySpan(comments)

	var err error
	allTokens, err = internal.MergeSplitTokens(allTokens)
	if err != nil {
		return nil, fmt.Errorf("merge split tokens failed: %w", err)
	}

	return allTokens, nil
}

func (p *Pipeline) sourceRouteManifest() (internal.SourceRouteManifest, error) {
	cfg, err := projectconfig.Load("")
	if err != nil {
		return internal.SourceRouteManifest{}, err
	}

	if p.cfg.LSPRoot != "" {
		root, err := filepath.Abs(p.cfg.LSPRoot)
		if err != nil {
			return internal.SourceRouteManifest{}, err
		}
		cfg.Project.Root = root
	}

	files, err := project.Scan(*cfg)
	if err != nil {
		return internal.SourceRouteManifest{}, err
	}

	sourcePaths := make([]string, 0, len(files))
	for _, file := range files {
		sourcePaths = append(sourcePaths, file.AbsPath)
	}

	return internal.NewSourceRouteManifestWithPrefix(cfg.Project.Root, cfg.Source.RoutePrefix, sourcePaths)
}

func (p *Pipeline) resolveTokenLinksWithManifest(tokens []internal.TokenInfo, manifest internal.SourceRouteManifest) {
	for _, warning := range internal.ResolveTokenLinks(p.cfg.AbsSrcPath, tokens, manifest) {
		fmt.Fprintf(os.Stderr, "Warning: definition link not resolved: %s\n", warning.String())
	}
}

func writeOutputFile(outPath string, output string) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	if err := os.WriteFile(outPath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
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
