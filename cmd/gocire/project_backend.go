package main

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/Eric-Song-Nop/gocire/internal"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

type ProjectBackend interface {
	Prepare(ctx context.Context, plan *ProjectExportPlan) error
	ExportFile(ctx context.Context, req ProjectFileExport) error
	Finish(ctx context.Context) error
}

type ProjectFileExport struct {
	File     project.SourceFile
	Pipeline *Pipeline
}

func NewProjectBackend(format string, plan *ProjectExportPlan) (ProjectBackend, error) {
	switch format {
	case "markdown", "mdx":
		return &documentProjectBackend{
			format: format,
			plan:   plan,
		}, nil
	case "astro":
		return &astroProjectBackend{
			plan: plan,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported project format %q", format)
	}
}

type documentProjectBackend struct {
	format string
	plan   *ProjectExportPlan
}

func (b *documentProjectBackend) Prepare(ctx context.Context, plan *ProjectExportPlan) error {
	return nil
}

func (b *documentProjectBackend) ExportFile(ctx context.Context, req ProjectFileExport) error {
	outPath, err := ProjectOutputPath(b.plan.Config.Output.Dir, b.plan.Manifest, req.File, b.format)
	if err != nil {
		return err
	}

	return req.Pipeline.RunFile(PipelineRunOptions{
		Context:    ctx,
		Manifest:   &b.plan.Manifest,
		OutputPath: outPath,
	})
}

func (b *documentProjectBackend) Finish(ctx context.Context) error {
	return nil
}

type astroProjectBackend struct {
	plan  *ProjectExportPlan
	mu    sync.Mutex
	pages []astroGeneratedPage
}

type astroGeneratedPage struct {
	Route      string
	Module     string
	Kind       project.PageKind
	Title      string
	SourcePath string
}

func (b *astroProjectBackend) Prepare(ctx context.Context, plan *ProjectExportPlan) error {
	return WriteAstroSiteAssets(plan.Config.Output.Dir, plan.Config.Site.Title)
}

func (b *astroProjectBackend) ExportFile(ctx context.Context, req ProjectFileExport) error {
	linkManifest := astroLinkManifest(b.plan.Manifest)
	analysis, err := req.Pipeline.AnalyzeFile(PipelineRunOptions{
		Context:  ctx,
		Manifest: &linkManifest,
	})
	if err != nil {
		return err
	}

	route, err := AstroProjectRoute(b.plan.Manifest, req.File)
	if err != nil {
		return err
	}

	outPath, modulePath, err := astroGeneratedPagePathForRoute(b.plan.Config.Output.Dir, route)
	if err != nil {
		return err
	}

	gen := internal.NewAstroGenerator(analysis.SourceLines)
	output := gen.GenerateAstro(analysis.Tokens, analysis.Comments, internal.AstroPageOptions{
		Title:          astroPageTitle(req.File),
		Kind:           string(req.File.Kind),
		Language:       req.File.Language,
		SourcePath:     req.File.RelPath,
		RenderMode:     astroRenderModeForKind(req.File.Kind),
		CodePageImport: astroCodePageImportForGeneratedRoute(route),
	})

	if err := writeOutputFile(outPath, output); err != nil {
		return err
	}

	b.mu.Lock()
	b.pages = append(b.pages, astroGeneratedPage{
		Route:      strings.TrimLeft(route, "/"),
		Module:     modulePath,
		Kind:       req.File.Kind,
		Title:      astroPageTitle(req.File),
		SourcePath: req.File.RelPath,
	})
	b.mu.Unlock()

	return nil
}

func (b *astroProjectBackend) Finish(ctx context.Context) error {
	b.mu.Lock()
	pages := append([]astroGeneratedPage(nil), b.pages...)
	b.mu.Unlock()

	if err := writeAstroRouteIndex(b.plan.Config.Output.Dir, pages); err != nil {
		return err
	}
	return writeAstroHomePage(b.plan.Config.Output.Dir, b.plan.Config.Site.Title, pages)
}

func AstroProjectRoute(manifest internal.SourceRouteManifest, file project.SourceFile) (string, error) {
	if route, _, ok := manifest.RouteForSourcePath(file.AbsPath); ok {
		return route, nil
	}
	if strings.TrimSpace(file.RelPath) == "" {
		return "", fmt.Errorf("source file has no relative path")
	}
	return manifest.SourceRoute(file.RelPath), nil
}

func AstroProjectOutputPath(outputDir string, manifest internal.SourceRouteManifest, file project.SourceFile) (string, error) {
	route, err := AstroProjectRoute(manifest, file)
	if err != nil {
		return "", err
	}
	outPath, _, err := astroGeneratedPagePathForRoute(outputDir, route)
	return outPath, err
}

func astroLinkManifest(manifest internal.SourceRouteManifest) internal.SourceRouteManifest {
	routes := make(map[string]string, len(manifest.Routes))
	for relPath, route := range manifest.Routes {
		routes[relPath] = astroRouteHref(route)
	}
	manifest.Routes = routes
	return manifest
}

func astroRouteHref(route string) string {
	route = strings.TrimSpace(route)
	if route == "" || strings.HasSuffix(route, "/") {
		return route
	}
	return route + "/"
}

func astroGeneratedPagePathForRoute(outputDir, route string) (outPath string, modulePath string, err error) {
	routePath := strings.TrimSpace(route)
	if routePath == "" {
		return "", "", fmt.Errorf("astro route is required")
	}
	routePath = strings.TrimLeft(routePath, "/")
	routePath = strings.TrimRight(routePath, "/")

	generatedPagesDir := filepath.Join(outputDir, "src", "generated", "pages")
	outPath, err = safeOutputPath(generatedPagesDir, routePath+".astro")
	if err != nil {
		return "", "", err
	}
	return outPath, "../generated/pages/" + routePath + ".astro", nil
}

func astroRenderModeForKind(kind project.PageKind) internal.AstroRenderMode {
	switch kind {
	case project.PageKindDocs, project.PageKindBlog:
		return internal.AstroRenderModeNarrative
	default:
		return internal.AstroRenderModeSource
	}
}

func astroCodePageImportForGeneratedRoute(route string) string {
	routePath := strings.TrimLeft(strings.TrimSpace(route), "/")
	routeDir := path.Dir(routePath)
	depth := 0
	if routeDir != "." && routeDir != "" {
		depth = len(strings.Split(routeDir, "/"))
	}
	return strings.Repeat("../", depth+2) + "components/CodePage.astro"
}

func astroPageTitle(file project.SourceFile) string {
	if strings.TrimSpace(file.RelPath) != "" {
		return file.RelPath
	}
	return filepath.Base(file.AbsPath)
}

func writeAstroRouteIndex(outputDir string, pages []astroGeneratedPage) error {
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].Route < pages[j].Route
	})

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(`const pageModules = import.meta.glob("../generated/pages/**/*.astro");

export function getStaticPaths() {
  const generatedPages = [
`)
	for _, page := range pages {
		fmt.Fprintf(
			&sb,
			"    { route: %s, module: %s },\n",
			strconv.Quote(page.Route),
			strconv.Quote(page.Module),
		)
	}
	sb.WriteString(`  ];

  return generatedPages.map((page) => ({
    params: { gocire: page.route },
    props: { module: page.module },
  }));
}

const modulePath = Astro.props.module;
const loadPage = pageModules[modulePath];
if (!loadPage) {
  throw new Error("Missing generated page module: " + modulePath);
}
const { default: Page } = await loadPage();
---

<Page />
`)

	return writeOutputFile(filepath.Join(outputDir, "src", "pages", "[...gocire].astro"), sb.String())
}

func writeAstroHomePage(outputDir string, siteTitle string, pages []astroGeneratedPage) error {
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].SourcePath < pages[j].SourcePath
	})

	title := normalizedAstroSiteTitle(siteTitle)
	docs := filterAstroPagesByKind(pages, project.PageKindDocs)
	blogs := filterAstroPagesByKind(pages, project.PageKindBlog)
	sourceCount := 0
	for _, page := range pages {
		if page.Kind == project.PageKindSource {
			sourceCount++
		}
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(`import SiteLayout from "../layouts/SiteLayout.astro";
`)
	fmt.Fprintf(&sb, "const siteTitle = %s;\n", strconv.Quote(title))
	fmt.Fprintf(&sb, "const docs = %s;\n", astroPageListLiteral(docs))
	fmt.Fprintf(&sb, "const blogs = %s;\n", astroPageListLiteral(blogs))
	fmt.Fprintf(&sb, "const sourceCount = %d;\n", sourceCount)
	sb.WriteString(`---

<SiteLayout title={siteTitle}>
  <main class="site-shell home-page">
    <section class="home-hero">
      <p class="page-kicker">Generated docsite</p>
      <h1>{siteTitle}</h1>
      <p>{sourceCount} source pages were generated for semantic navigation and stable code locations.</p>
    </section>

    <section id="docs" class="home-section" aria-labelledby="docs-title">
      <h2 id="docs-title">Docs</h2>
      {docs.length > 0 ? (
        <ul class="home-list">
          {docs.map((page) => (
            <li><a href={page.href}>{page.title}</a></li>
          ))}
        </ul>
      ) : (
        <p class="home-empty">No generated docs pages yet.</p>
      )}
    </section>

    <section id="blog" class="home-section" aria-labelledby="blog-title">
      <h2 id="blog-title">Blog</h2>
      {blogs.length > 0 ? (
        <ul class="home-list">
          {blogs.map((page) => (
            <li><a href={page.href}>{page.title}</a></li>
          ))}
        </ul>
      ) : (
        <p class="home-empty">No generated blog posts yet.</p>
      )}
    </section>
  </main>
</SiteLayout>
`)

	return writeOutputFile(filepath.Join(outputDir, "src", "pages", "index.astro"), sb.String())
}

func filterAstroPagesByKind(pages []astroGeneratedPage, kind project.PageKind) []astroGeneratedPage {
	filtered := make([]astroGeneratedPage, 0)
	for _, page := range pages {
		if page.Kind == kind {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

func astroPageListLiteral(pages []astroGeneratedPage) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, page := range pages {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(
			&sb,
			"{ title: %s, href: %s }",
			strconv.Quote(page.Title),
			strconv.Quote("/"+page.Route+"/"),
		)
	}
	sb.WriteString("]")
	return sb.String()
}
