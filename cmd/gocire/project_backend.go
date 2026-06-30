package main

import (
	"context"
	"fmt"
	"os"
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
	Page     SitePage
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
	outPath, err := ProjectOutputPath(b.plan.Config.Output.Dir, b.plan.Site.Routes, req.File, b.format)
	if err != nil {
		return err
	}

	return req.Pipeline.RunFile(PipelineRunOptions{
		Context:    ctx,
		Manifest:   &b.plan.Site.Routes,
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
	Href       string
	Module     string
	Kind       project.PageKind
	Title      string
	Date       string
	Tags       []string
	Author     string
	Language   string
	SourcePath string
}

func (b *astroProjectBackend) Prepare(ctx context.Context, plan *ProjectExportPlan) error {
	return AstroSiteAssets{
		OutputDir:   plan.Config.Output.Dir,
		SiteTitle:   plan.Config.Site.Title,
		TemplateDir: plan.Config.Site.TemplateDir,
	}.Write()
}

func (b *astroProjectBackend) ExportFile(ctx context.Context, req ProjectFileExport) error {
	linkManifest := astroLinkManifest(b.plan.Site.Routes)
	analysis, err := req.Pipeline.AnalyzeFile(PipelineRunOptions{
		Context:  ctx,
		Manifest: &linkManifest,
	})
	if err != nil {
		return err
	}

	route := req.Page.Route

	outPath, modulePath, err := astroGeneratedPagePathForRoute(b.plan.Config.Output.Dir, route)
	if err != nil {
		return err
	}

	gen := internal.NewAstroGenerator(analysis.SourceLines)
	output := gen.GenerateAstro(analysis.Tokens, analysis.Comments, internal.AstroPageOptions{
		Title:          req.Page.Title,
		Kind:           string(req.Page.Kind),
		Language:       req.Page.Language,
		SourcePath:     req.Page.SourcePath,
		Date:           req.Page.Date,
		Tags:           req.Page.Tags,
		Author:         req.Page.Author,
		RenderMode:     astroRenderModeForKind(req.Page.Kind),
		CodePageImport: astroCodePageImportForGeneratedRoute(route),
	})

	if err := writeOutputFile(outPath, output); err != nil {
		return err
	}

	b.mu.Lock()
	b.pages = append(b.pages, astroGeneratedPage{
		Route:      strings.TrimLeft(route, "/"),
		Href:       req.Page.Href,
		Module:     modulePath,
		Kind:       req.Page.Kind,
		Title:      req.Page.Title,
		Date:       req.Page.Date,
		Tags:       cloneAstroSiteDataStrings(req.Page.Tags),
		Author:     req.Page.Author,
		Language:   req.Page.Language,
		SourcePath: req.Page.SourcePath,
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
	if err := writeAstroSiteData(b.plan.Config.Output.Dir, b.plan.Config.Site, pages, b.plan.Site.Navigation); err != nil {
		return err
	}
	if err := writeAstroNavigation(b.plan.Config.Output.Dir, b.plan.Site.Navigation); err != nil {
		return err
	}
	return removeAstroHomePage(b.plan.Config.Output.Dir)
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

func removeAstroHomePage(outputDir string) error {
	indexPath := filepath.Join(outputDir, "src", "pages", "index.astro")
	if err := os.Remove(indexPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove Astro home page %s: %w", indexPath, err)
	}
	return nil
}

func writeAstroNavigation(outputDir string, navigation SiteNavigation) error {
	return writeOutputFile(
		filepath.Join(outputDir, "src", "generated", "navigation.ts"),
		"export { navigation } from \"./site-data\";\n",
	)
}

func astroNavigationLiteral(navigation SiteNavigation) string {
	var sb strings.Builder
	sb.WriteString("{")
	fmt.Fprintf(&sb, " docs: %s,", astroNavigationSectionLiteral(navigation.Docs))
	fmt.Fprintf(&sb, " blog: %s", astroNavigationSectionLiteral(navigation.Blog))
	sb.WriteString(" }")
	return sb.String()
}

func astroNavigationSectionLiteral(section SiteNavigationSection) string {
	var sb strings.Builder
	sb.WriteString("{")
	fmt.Fprintf(&sb, " firstHref: %s,", strconv.Quote(section.FirstHref))
	fmt.Fprintf(&sb, " items: %s", astroNavigationItemsLiteral(section.Items))
	sb.WriteString(" }")
	return sb.String()
}

func astroNavigationItemsLiteral(items []SiteNavigationItem) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, item := range items {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(astroNavigationItemLiteral(item))
	}
	sb.WriteString("]")
	return sb.String()
}

func astroNavigationItemLiteral(item SiteNavigationItem) string {
	var sb strings.Builder
	sb.WriteString("{")
	fmt.Fprintf(&sb, " type: %s", strconv.Quote(string(item.Type)))
	if item.Title != "" {
		fmt.Fprintf(&sb, ", title: %s", strconv.Quote(item.Title))
	}
	if item.Href != "" {
		fmt.Fprintf(&sb, ", href: %s", strconv.Quote(item.Href))
	}
	if item.SourcePath != "" {
		fmt.Fprintf(&sb, ", sourcePath: %s", strconv.Quote(item.SourcePath))
	}
	astroWriteMetadataLiteral(&sb, item)
	if len(item.Items) > 0 {
		fmt.Fprintf(&sb, ", items: %s", astroNavigationItemsLiteral(item.Items))
	}
	sb.WriteString(" }")
	return sb.String()
}

func astroWriteMetadataLiteral(sb *strings.Builder, item SiteNavigationItem) {
	if date := strings.TrimSpace(item.Date); date != "" {
		fmt.Fprintf(sb, ", date: %s", strconv.Quote(date))
	}
	if tags := astroNormalizeStringList(item.Tags); len(tags) > 0 {
		fmt.Fprintf(sb, ", tags: %s", astroStringArrayLiteral(tags))
	}
	if author := strings.TrimSpace(item.Author); author != "" {
		fmt.Fprintf(sb, ", author: %s", strconv.Quote(author))
	}
}

func astroNormalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func astroStringArrayLiteral(values []string) string {
	var sb strings.Builder
	sb.WriteString("[")
	for i, value := range values {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(strconv.Quote(value))
	}
	sb.WriteString("]")
	return sb.String()
}
