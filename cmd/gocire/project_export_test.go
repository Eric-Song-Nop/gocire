package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

func TestProjectOutputPathUsesManifestRoute(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "pkg", "main.go")
	writeProjectTestFile(t, srcPath, "package main\n")

	manifest, err := internal.NewSourceRouteManifestWithPrefix(root, "/code", []string{srcPath})
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "site")
	got, err := ProjectOutputPath(outDir, manifest, project.SourceFile{
		AbsPath: srcPath,
		RelPath: "pkg/main.go",
	}, "mdx")
	if err != nil {
		t.Fatalf("ProjectOutputPath returned error: %v", err)
	}

	want := filepath.Join(outDir, "code", "pkg", "main.go.mdx")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
	if got == srcPath {
		t.Fatal("output path overwrote source path")
	}
}

func TestAstroProjectOutputPathUsesManifestRoute(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "foo.go")
	writeProjectTestFile(t, srcPath, "package main\n")

	manifest, err := internal.NewSourceRouteManifestWithPrefix(root, "/_source", []string{srcPath})
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "site")
	got, err := AstroProjectOutputPath(outDir, manifest, project.SourceFile{
		AbsPath: srcPath,
		RelPath: "foo.go",
	})
	if err != nil {
		t.Fatalf("AstroProjectOutputPath returned error: %v", err)
	}

	want := filepath.Join(outDir, "src", "generated", "pages", "_source", "foo.go.html.astro")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

func TestAstroLinkManifestUsesTrailingSlashRoutes(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "foo.go")
	writeProjectTestFile(t, srcPath, "package main\n")

	manifest, err := internal.NewSourceRouteManifestWithPrefix(root, "/_source", []string{srcPath})
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	astroManifest := astroLinkManifest(manifest)
	route, ok := astroManifest.RouteForRelPath("foo.go")
	if !ok {
		t.Fatal("RouteForRelPath returned ok=false")
	}
	if route != "/_source/foo.go.html/" {
		t.Fatalf("Astro route = %q, want trailing slash route", route)
	}

	originalRoute, ok := manifest.RouteForRelPath("foo.go")
	if !ok {
		t.Fatal("original RouteForRelPath returned ok=false")
	}
	if originalRoute != "/_source/foo.go.html" {
		t.Fatalf("original route = %q, want unchanged route", originalRoute)
	}
}

func TestAstroNavigationItemLiteralIncludesMetadata(t *testing.T) {
	got := astroNavigationItemLiteral(SiteNavigationItem{
		Type:   SiteNavigationItemLink,
		Title:  "Post",
		Href:   "/blog/post/",
		Date:   " 2026-06-30 ",
		Tags:   []string{"go", " ", "astro"},
		Author: " Ada Lovelace ",
	})
	for _, want := range []string{
		`type: "link"`,
		`title: "Post"`,
		`, date: "2026-06-30"`,
		`, tags: ["go", "astro"]`,
		`, author: "Ada Lovelace"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("metadata literal missing %q\nGot:\n%s", want, got)
		}
	}
	if strings.Contains(got, `" "`) {
		t.Fatalf("metadata literal should omit blank tags\nGot:\n%s", got)
	}

	empty := astroNavigationItemLiteral(SiteNavigationItem{
		Type:   SiteNavigationItemLink,
		Tags:   []string{" "},
		Author: " ",
	})
	for _, unwanted := range []string{"date:", "tags:", "author:"} {
		if strings.Contains(empty, unwanted) {
			t.Fatalf("empty metadata should not contain %q\nGot:\n%s", unwanted, empty)
		}
	}
}

func TestAstroRenderModeForPageKind(t *testing.T) {
	for _, tt := range []struct {
		name string
		kind project.PageKind
		want internal.AstroRenderMode
	}{
		{name: "docs", kind: project.PageKindDocs, want: internal.AstroRenderModeNarrative},
		{name: "blog", kind: project.PageKindBlog, want: internal.AstroRenderModeNarrative},
		{name: "source", kind: project.PageKindSource, want: internal.AstroRenderModeSource},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := astroRenderModeForKind(tt.kind); got != tt.want {
				t.Fatalf("render mode = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectOutputPathFallsBackToRelPath(t *testing.T) {
	manifestRoot := t.TempDir()
	sourceRoot := t.TempDir()
	srcPath := filepath.Join(sourceRoot, "lib", "app.go")

	manifest, err := internal.NewSourceRouteManifestWithPrefix(manifestRoot, "/code", nil)
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "site")
	got, err := ProjectOutputPath(outDir, manifest, project.SourceFile{
		AbsPath: srcPath,
		RelPath: "lib/app.go",
	}, "markdown")
	if err != nil {
		t.Fatalf("ProjectOutputPath returned error: %v", err)
	}

	want := filepath.Join(outDir, "lib", "app.go.md")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

func TestOutputPathRejectsEscapingRelPath(t *testing.T) {
	_, err := outputPathForRelPath(t.TempDir(), "../outside.go", "mdx")
	if err == nil {
		t.Fatal("outputPathForRelPath returned nil error for escaping path")
	}
}

func TestNewProjectExportPlanScansAndBuildsManifest(t *testing.T) {
	root := t.TempDir()
	writeProjectTestFile(t, filepath.Join(root, "repo", "pkg", "app.go"), "package app\n")
	writeProjectTestFile(t, filepath.Join(root, "repo", "README.md"), "# docs\n")

	configPath := filepath.Join(root, ".gocire.yml")
	writeProjectTestFile(t, configPath, `
project:
  root: repo
source:
  routePrefix: code
  include:
    - "**/*.go"
output:
  dir: public
`)

	plan, err := NewProjectExportPlan(&Config{
		Project:    true,
		ConfigPath: configPath,
		Format:     "mdx",
	})
	if err != nil {
		t.Fatalf("NewProjectExportPlan returned error: %v", err)
	}

	if len(plan.Files) != 1 {
		t.Fatalf("len(plan.Files) = %d, want 1", len(plan.Files))
	}
	if plan.Files[0].RelPath != "pkg/app.go" {
		t.Fatalf("RelPath = %q, want pkg/app.go", plan.Files[0].RelPath)
	}
	if plan.Config.Output.Dir != filepath.Join(root, "public") {
		t.Fatalf("output dir = %q, want %q", plan.Config.Output.Dir, filepath.Join(root, "public"))
	}

	route, ok := plan.Site.Routes.RouteForRelPath("pkg/app.go")
	if !ok {
		t.Fatal("RouteForRelPath returned ok=false")
	}
	if route != "/code/pkg/app.go.html" {
		t.Fatalf("route = %q, want /code/pkg/app.go.html", route)
	}
	if len(plan.Site.Pages) != 1 {
		t.Fatalf("len(plan.Site.Pages) = %d, want 1", len(plan.Site.Pages))
	}
	if plan.Site.Pages[0].Href != "/code/pkg/app.go.html/" {
		t.Fatalf("page href = %q, want /code/pkg/app.go.html/", plan.Site.Pages[0].Href)
	}
}

func TestProjectExportRunnerWritesProjectFiles(t *testing.T) {
	root := t.TempDir()
	writeProjectTestFile(t, filepath.Join(root, "repo", "main.go"), "package main\n\nfunc main() {}\n")
	writeProjectTestFile(t, filepath.Join(root, "repo", "pkg", "util.go"), "package pkg\n\nfunc Util() {}\n")

	configPath := filepath.Join(root, ".gocire.yml")
	writeProjectTestFile(t, configPath, `
project:
  root: repo
source:
  include:
    - "**/*.go"
output:
  dir: site
`)

	runner, err := NewProjectExportRunner(&Config{
		Project:    true,
		ConfigPath: configPath,
		Jobs:       2,
		Format:     "markdown",
	})
	if err != nil {
		t.Fatalf("NewProjectExportRunner returned error: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, relPath := range []string{
		filepath.Join("_source", "main.go.md"),
		filepath.Join("_source", "pkg", "util.go.md"),
	} {
		outPath := filepath.Join(root, "site", relPath)
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("expected output file %q: %v", outPath, err)
		}
	}
}

func TestProjectExportRunnerWritesAstroProject(t *testing.T) {
	root := t.TempDir()
	writeProjectTestFile(t, filepath.Join(root, "repo", "main.go"), "package main\n\nfunc main() {}\n")
	writeProjectTestFile(t, filepath.Join(root, "theme", "src", "scripts", "theme.js"), "export const customThemeMarker = true;\n")

	configPath := filepath.Join(root, ".gocire.yml")
	writeProjectTestFile(t, configPath, `
site:
  title: Test Site
  description: Test site description
  url: https://example.com/docs
  templateDir: theme
content:
  metadata:
    main.go:
      date: "2026-06-30"
      tags:
        - source
        - runtime
      author: Ada Lovelace
project:
  root: repo
source:
  include:
    - "**/*.go"
output:
  dir: site
`)

	runner, err := NewProjectExportRunner(&Config{
		Project:    true,
		ConfigPath: configPath,
		Jobs:       1,
		Format:     "astro",
	})
	if err != nil {
		t.Fatalf("NewProjectExportRunner returned error: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, relPath := range []string{
		"package.json",
		filepath.Join("src", "generated", "site-data.ts"),
		filepath.Join("src", "generated", "navigation.ts"),
		filepath.Join("src", "generated", "pages", "_source", "main.go.html.astro"),
		filepath.Join("src", "pages", "[...gocire].astro"),
		filepath.Join("src", "pages", "rss.xml.ts"),
		filepath.Join("src", "pages", "sitemap.xml.ts"),
	} {
		outPath := filepath.Join(root, "site", relPath)
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("expected output file %q: %v", outPath, err)
		}
	}

	themePath := filepath.Join(root, "site", "src", "scripts", "theme.js")
	theme, err := os.ReadFile(themePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", themePath, err)
	}
	if !strings.Contains(string(theme), "customThemeMarker") {
		t.Fatalf("Astro theme script did not use templateDir override\nGot:\n%s", string(theme))
	}

	pagePath := filepath.Join(root, "site", "src", "generated", "pages", "_source", "main.go.html.astro")
	page, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", pagePath, err)
	}
	for _, want := range []string{
		`import CodePage from "../../../components/CodePage.astro";`,
		`renderMode="source"`,
		`sourcePath="main.go"`,
		`date="2026-06-30"`,
		`author="Ada Lovelace"`,
		`tags={["source", "runtime"]}`,
	} {
		if !strings.Contains(string(page), want) {
			t.Fatalf("Astro page missing %q\nGot:\n%s", want, string(page))
		}
	}

	routeIndexPath := filepath.Join(root, "site", "src", "pages", "[...gocire].astro")
	routeIndex, err := os.ReadFile(routeIndexPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", routeIndexPath, err)
	}
	for _, want := range []string{
		`route: "_source/main.go.html"`,
		`module: "../generated/pages/_source/main.go.html.astro"`,
		`getStaticPaths`,
	} {
		if !strings.Contains(string(routeIndex), want) {
			t.Fatalf("Astro route index missing %q\nGot:\n%s", want, string(routeIndex))
		}
	}

	navigationPath := filepath.Join(root, "site", "src", "generated", "navigation.ts")
	navigation, err := os.ReadFile(navigationPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", navigationPath, err)
	}
	if strings.TrimSpace(string(navigation)) != `export { navigation } from "./site-data";` {
		t.Fatalf("Astro navigation should re-export site-data navigation\nGot:\n%s", string(navigation))
	}

	siteDataPath := filepath.Join(root, "site", "src", "generated", "site-data.ts")
	siteData, err := os.ReadFile(siteDataPath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", siteDataPath, err)
	}
	for _, want := range []string{
		`export const siteData =`,
		`"title": "Test Site"`,
		`"description": "Test site description"`,
		`"url": "https://example.com/docs"`,
		`"trailingSlash": "always"`,
		`"routeParam": "_source/main.go.html"`,
		`"href": "/_source/main.go.html/"`,
		`"module": "../generated/pages/_source/main.go.html.astro"`,
		`"kind": "source"`,
		`"title": "main.go"`,
		`"sourcePath": "main.go"`,
		`"language": "go"`,
		`"date": "2026-06-30"`,
		`"tags": [`,
		`"source"`,
		`"runtime"`,
		`"author": "Ada Lovelace"`,
		`export const pages = siteData.pages;`,
		`export const navigation = siteData.navigation;`,
	} {
		if !strings.Contains(string(siteData), want) {
			t.Fatalf("Astro site data missing %q\nGot:\n%s", want, string(siteData))
		}
	}
}

func TestProjectExportRunnerWritesMixedLanguageAstroProject(t *testing.T) {
	root := t.TempDir()
	writeProjectTestFile(t, filepath.Join(root, "repo", "main.go"), "package main\n\nfunc main() {}\n")
	writeProjectTestFile(t, filepath.Join(root, "repo", "web", "app.ts"), "export function render(): string {\n  return 'ok';\n}\n")

	configPath := filepath.Join(root, ".gocire.yml")
	writeProjectTestFile(t, configPath, `
project:
  root: repo
source:
  include:
    - "**/*.go"
    - "**/*.ts"
output:
  dir: site
`)

	runner, err := NewProjectExportRunner(&Config{
		Project:    true,
		ConfigPath: configPath,
		Jobs:       2,
		Format:     "astro",
	})
	if err != nil {
		t.Fatalf("NewProjectExportRunner returned error: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	assertGeneratedPageLanguage(t, filepath.Join(root, "site", "src", "generated", "pages", "_source", "main.go.html.astro"), "go")
	assertGeneratedPageLanguage(t, filepath.Join(root, "site", "src", "generated", "pages", "_source", "web", "app.ts.html.astro"), "typescript")
}

func TestPipelineConfigForProjectFileUsesDetectedLanguage(t *testing.T) {
	root := t.TempDir()
	runner := &ProjectExportRunner{
		cfg: &Config{
			Lang:   "go",
			UseLSP: true,
			Format: "astro",
		},
		plan: &ProjectExportPlan{
			Config: &projectconfig.ProjectConfig{
				Project: projectconfig.ProjectSection{
					Root: root,
				},
			},
		},
	}

	tsFile := project.SourceFile{
		AbsPath:  filepath.Join(root, "web", "app.ts"),
		RelPath:  "web/app.ts",
		Language: "typescript",
		Kind:     project.PageKindSource,
	}
	tsCfg := runner.pipelineConfigForProjectFile(tsFile, "out")
	if tsCfg.Lang != "typescript" {
		t.Fatalf("Lang = %q, want detected file language typescript", tsCfg.Lang)
	}
	if tsCfg.UseLSP {
		t.Fatal("UseLSP = true for non-selected project language, want false")
	}
	if tsCfg.LSPRoot != root {
		t.Fatalf("LSPRoot = %q, want project root %q", tsCfg.LSPRoot, root)
	}

	goFile := project.SourceFile{
		AbsPath:  filepath.Join(root, "main.go"),
		RelPath:  "main.go",
		Language: "go",
		Kind:     project.PageKindSource,
	}
	goCfg := runner.pipelineConfigForProjectFile(goFile, "")
	if goCfg.Lang != "go" {
		t.Fatalf("Lang = %q, want go", goCfg.Lang)
	}
	if !goCfg.UseLSP {
		t.Fatal("UseLSP = false for selected project language, want true")
	}

	runner.cfg.Lang = "ts"
	tsAliasCfg := runner.pipelineConfigForProjectFile(tsFile, "")
	if !tsAliasCfg.UseLSP {
		t.Fatal("UseLSP = false for aliased selected project language, want true")
	}
}

func assertGeneratedPageLanguage(t *testing.T, pagePath string, language string) {
	t.Helper()

	page, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", pagePath, err)
	}
	want := `language="` + language + `"`
	if !strings.Contains(string(page), want) {
		t.Fatalf("generated page %s missing %q\nGot:\n%s", pagePath, want, string(page))
	}
}

func writeProjectTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
