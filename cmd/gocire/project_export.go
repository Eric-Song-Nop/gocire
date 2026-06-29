package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/languages"
	"github.com/Eric-Song-Nop/gocire/internal/project"
	"golang.org/x/sync/errgroup"
)

type ProjectExportPlan struct {
	Config *projectconfig.ProjectConfig
	Files  []project.SourceFile
	Site   SiteModel
}

type ProjectExportRunner struct {
	cfg  *Config
	plan *ProjectExportPlan
}

type projectLSPSessionKey struct {
	language      string
	workspaceRoot string
}

type projectLSPAnalyzerProvider struct {
	ctx           context.Context
	workspaceRoot string
	sessions      map[projectLSPSessionKey]*internal.LSPSession
	mu            sync.Mutex
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

	site, err := BuildSiteModel(*projectCfg, files)
	if err != nil {
		return nil, err
	}

	return &ProjectExportPlan{
		Config: projectCfg,
		Files:  files,
		Site:   site,
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

	g, runCtx := errgroup.WithContext(ctx)
	g.SetLimit(r.cfg.Jobs)

	for _, file := range r.plan.Files {
		file := file
		g.Go(func() error {
			return r.exportFile(runCtx, file, lspFactory, backend)
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

func (r *ProjectExportRunner) exportFile(ctx context.Context, file project.SourceFile, lspFactory LSPAnalyzerFactory, backend ProjectBackend) error {
	page, ok := r.plan.Site.PageForFile(file)
	if !ok {
		return fmt.Errorf("%s: site page not found", file.RelPath)
	}

	fileCfg := r.pipelineConfigForProjectFile(file, "")
	pipeline, err := NewPipelineWithOptions(fileCfg, PipelineOptions{
		Context:            ctx,
		LSPAnalyzerFactory: lspFactory,
	})
	if err != nil {
		return fmt.Errorf("%s: %w", file.RelPath, err)
	}

	if err := backend.ExportFile(ctx, ProjectFileExport{
		File:     file,
		Page:     page,
		Pipeline: pipeline,
	}); err != nil {
		return fmt.Errorf("%s: %w", file.RelPath, err)
	}
	return nil
}

func (r *ProjectExportRunner) pipelineConfigForProjectFile(file project.SourceFile, outPath string) *Config {
	fileCfg := *r.cfg
	fileCfg.SrcPath = file.AbsPath
	fileCfg.AbsSrcPath = file.AbsPath
	fileCfg.OutPath = outPath
	fileCfg.Lang = file.Language
	fileCfg.UseLSP = r.useLSPForProjectFile(file)
	if fileCfg.LSPRoot == "" {
		fileCfg.LSPRoot = r.plan.Config.Project.Root
	}
	return &fileCfg
}

func (r *ProjectExportRunner) useLSPForProjectFile(file project.SourceFile) bool {
	if !r.cfg.UseLSP {
		return false
	}
	if r.cfg.Lang == "" {
		return true
	}
	selectedLanguage, err := languages.CanonicalName(r.cfg.Lang)
	if err != nil {
		return false
	}
	return selectedLanguage == file.Language
}

func (r *ProjectExportRunner) projectLSPAnalyzerFactory(ctx context.Context) (LSPAnalyzerFactory, func() error, error) {
	if !r.cfg.UseLSP {
		return nil, nil, nil
	}
	if r.cfg.Lang != "" {
		if _, err := languages.CanonicalName(r.cfg.Lang); err != nil {
			return nil, nil, err
		}
	}

	workspaceRoot := r.plan.Config.Project.Root
	if r.cfg.LSPRoot != "" {
		absWorkspaceRoot, err := filepath.Abs(r.cfg.LSPRoot)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve lsp root: %w", err)
		}
		workspaceRoot = absWorkspaceRoot
	}

	provider := &projectLSPAnalyzerProvider{
		ctx:           ctx,
		workspaceRoot: workspaceRoot,
		sessions:      make(map[projectLSPSessionKey]*internal.LSPSession),
	}
	return provider.AnalyzerFor, provider.Close, nil
}

func (p *projectLSPAnalyzerProvider) AnalyzerFor(ctx context.Context, req PipelineLSPRequest) (TokenAnalyzer, error) {
	workspaceRoot := req.WorkspaceRoot
	if workspaceRoot == "" {
		workspaceRoot = p.workspaceRoot
	}
	sessionCtx := p.ctx
	if sessionCtx == nil {
		sessionCtx = ctx
	}
	session, err := p.session(sessionCtx, req.Language, workspaceRoot)
	if err != nil {
		return nil, err
	}
	return &ProjectLSPAnalyzer{
		session:    session,
		sourcePath: req.SourcePath,
	}, nil
}

func (p *projectLSPAnalyzerProvider) session(ctx context.Context, language, workspaceRoot string) (*internal.LSPSession, error) {
	key := projectLSPSessionKey{
		language:      language,
		workspaceRoot: workspaceRoot,
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if session, ok := p.sessions[key]; ok {
		return session, nil
	}

	session, err := internal.NewLSPSession(ctx, language, workspaceRoot)
	if err != nil {
		return nil, err
	}
	p.sessions[key] = session
	return session, nil
}

func (p *projectLSPAnalyzerProvider) Close() error {
	p.mu.Lock()
	sessions := make([]*internal.LSPSession, 0, len(p.sessions))
	for _, session := range p.sessions {
		sessions = append(sessions, session)
	}
	p.mu.Unlock()

	var firstErr error
	for _, session := range sessions {
		if err := session.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
