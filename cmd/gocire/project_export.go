package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
	"golang.org/x/sync/errgroup"
)

type ProjectExportPlan struct {
	Config   *projectconfig.ProjectConfig
	Files    []project.SourceFile
	Manifest internal.SourceRouteManifest
}

type ProjectExportRunner struct {
	cfg  *Config
	plan *ProjectExportPlan
}

func NewProjectExportRunner(cfg *Config) (*ProjectExportRunner, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	runnerCfg := *cfg
	if runnerCfg.Jobs < 1 {
		runnerCfg.Jobs = runtime.NumCPU()
	}
	if runnerCfg.AbsIndexPath != "" {
		if _, err := os.Stat(runnerCfg.AbsIndexPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Load SCIP index file failed: %v. SCIP analysis will be skipped.\n", err)
			runnerCfg.IndexPath = ""
			runnerCfg.AbsIndexPath = ""
		}
	}

	plan, err := NewProjectExportPlan(&runnerCfg)
	if err != nil {
		return nil, err
	}

	return &ProjectExportRunner{
		cfg:  &runnerCfg,
		plan: plan,
	}, nil
}

func NewProjectExportPlan(cfg *Config) (*ProjectExportPlan, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	projectCfg, err := projectconfig.Load(cfg.ConfigPath)
	if err != nil {
		return nil, err
	}

	if cfg.OutPath != "" {
		outputDir, err := filepath.Abs(cfg.OutPath)
		if err != nil {
			return nil, fmt.Errorf("resolve project output directory: %w", err)
		}
		projectCfg.Output.Dir = filepath.Clean(outputDir)
	}

	files, err := project.Scan(*projectCfg)
	if err != nil {
		return nil, err
	}

	manifest, err := SourceRouteManifestForProject(*projectCfg, files)
	if err != nil {
		return nil, err
	}

	return &ProjectExportPlan{
		Config:   projectCfg,
		Files:    files,
		Manifest: manifest,
	}, nil
}

func SourceRouteManifestForProject(cfg projectconfig.ProjectConfig, files []project.SourceFile) (internal.SourceRouteManifest, error) {
	sourcePaths := make([]string, 0, len(files))
	for _, file := range files {
		sourcePaths = append(sourcePaths, file.AbsPath)
	}
	return internal.NewSourceRouteManifestWithPrefix(cfg.Project.Root, cfg.Source.RoutePrefix, sourcePaths)
}

func (r *ProjectExportRunner) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	fmt.Printf("Project root: %s\n", r.plan.Config.Project.Root)
	fmt.Printf("Project files: %d\n", len(r.plan.Files))

	backend, err := NewProjectBackend(r.cfg.Format, r.plan)
	if err != nil {
		return err
	}
	if err := backend.Prepare(ctx, r.plan); err != nil {
		return fmt.Errorf("prepare project backend: %w", err)
	}

	if len(r.plan.Files) == 0 {
		return backend.Finish(ctx)
	}

	lspFactory, closeLSP, err := r.projectLSPAnalyzerFactory(ctx)
	if err != nil {
		return err
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(r.cfg.Jobs)

	for _, file := range r.plan.Files {
		file := file
		g.Go(func() error {
			return r.exportFile(ctx, file, lspFactory, backend)
		})
	}

	runErr := g.Wait()
	if closeLSP != nil {
		if closeErr := closeLSP(); runErr == nil && closeErr != nil {
			runErr = closeErr
		}
	}
	if runErr != nil {
		return runErr
	}
	if err := backend.Finish(ctx); err != nil {
		return err
	}

	fmt.Printf("Project export completed: %d files, output dir: %s\n", len(r.plan.Files), r.plan.Config.Output.Dir)
	return nil
}

func (r *ProjectExportRunner) exportFile(ctx context.Context, file project.SourceFile, lspFactory func(sourcePath string) (TokenAnalyzer, error), backend ProjectBackend) error {
	fileCfg := r.pipelineConfigForFile(file, "", lspFactory)
	pipeline, err := NewPipelineWithOptions(fileCfg, PipelineOptions{
		LSPAnalyzerFactory: lspFactory,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", file.RelPath, err)
	}

	if err := backend.ExportFile(ctx, ProjectFileExport{
		File:     file,
		Pipeline: pipeline,
	}); err != nil {
		return fmt.Errorf("%s: %w", file.RelPath, err)
	}
	return nil
}

func (r *ProjectExportRunner) pipelineConfigForFile(file project.SourceFile, outPath string, lspFactory func(sourcePath string) (TokenAnalyzer, error)) *Config {
	fileCfg := *r.cfg
	fileCfg.SrcPath = file.AbsPath
	fileCfg.AbsSrcPath = file.AbsPath
	fileCfg.OutPath = outPath
	if fileCfg.Lang == "" {
		fileCfg.Lang = file.Language
	}
	if fileCfg.LSPRoot == "" {
		fileCfg.LSPRoot = r.plan.Config.Project.Root
	}
	return &fileCfg
}

func (r *ProjectExportRunner) projectLSPAnalyzerFactory(ctx context.Context) (func(sourcePath string) (TokenAnalyzer, error), func() error, error) {
	if !r.cfg.UseLSP {
		return nil, nil, nil
	}

	language, err := projectLanguage(r.cfg.Lang, r.plan.Files)
	if err != nil {
		return nil, nil, err
	}

	workspaceRoot := r.plan.Config.Project.Root
	if r.cfg.LSPRoot != "" {
		workspaceRoot, err = filepath.Abs(r.cfg.LSPRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve lsp root: %w", err)
		}
	}

	session, err := internal.NewLSPSession(ctx, language, workspaceRoot)
	if err != nil {
		return nil, nil, err
	}

	return func(sourcePath string) (TokenAnalyzer, error) {
		return &ProjectLSPAnalyzer{
			session:    session,
			sourcePath: sourcePath,
		}, nil
	}, session.Close, nil
}

type ProjectLSPAnalyzer struct {
	session    *internal.LSPSession
	sourcePath string
}

func (a *ProjectLSPAnalyzer) Analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return a.session.AnalyzeFile(a.sourcePath, content)
}

func projectLanguage(cliLang string, files []project.SourceFile) (string, error) {
	if cliLang != "" {
		return cliLang, nil
	}
	if len(files) == 0 {
		return "", fmt.Errorf("project contains no source files")
	}

	language := files[0].Language
	for _, file := range files[1:] {
		if file.Language != language {
			return "", fmt.Errorf("project contains multiple languages (%s and %s); pass -lang to choose one LSP language", language, file.Language)
		}
	}
	return language, nil
}

func ProjectOutputPath(outputDir string, manifest internal.SourceRouteManifest, file project.SourceFile, format string) (string, error) {
	if route, _, ok := manifest.RouteForSourcePath(file.AbsPath); ok {
		return outputPathForRoute(outputDir, route, format)
	}
	if file.RelPath == "" {
		return "", fmt.Errorf("source file has no relative path")
	}
	return outputPathForRelPath(outputDir, file.RelPath, format)
}

func outputPathForRoute(outputDir, route, format string) (string, error) {
	routePath := strings.TrimSpace(route)
	routePath = strings.TrimSuffix(routePath, ".html")
	return safeOutputPath(outputDir, routePath+outputExtension(format))
}

func outputPathForRelPath(outputDir, relPath, format string) (string, error) {
	return safeOutputPath(outputDir, relPath+outputExtension(format))
}

func outputExtension(format string) string {
	if format == "markdown" {
		return ".md"
	}
	return ".mdx"
}

func safeOutputPath(outputDir, slashRelPath string) (string, error) {
	if strings.TrimSpace(outputDir) == "" {
		return "", fmt.Errorf("output directory is required")
	}

	slashRelPath = strings.ReplaceAll(filepath.ToSlash(slashRelPath), "\\", "/")
	slashRelPath = strings.TrimLeft(slashRelPath, "/")
	slashRelPath = path.Clean(slashRelPath)
	if slashRelPath == "." || slashRelPath == "" {
		return "", fmt.Errorf("output relative path is required")
	}
	if slashRelPath == ".." || strings.HasPrefix(slashRelPath, "../") {
		return "", fmt.Errorf("output path escapes output directory: %s", slashRelPath)
	}

	outputRoot, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output directory: %w", err)
	}

	outPath := filepath.Join(outputRoot, filepath.FromSlash(slashRelPath))
	rel, err := filepath.Rel(outputRoot, outPath)
	if err != nil {
		return "", fmt.Errorf("validate output path: %w", err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("output path escapes output directory: %s", outPath)
	}

	return outPath, nil
}
