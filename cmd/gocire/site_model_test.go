package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

func TestBuildSiteModelBuildsPagesRoutesAndNavigation(t *testing.T) {
	root := t.TempDir()
	files := []project.SourceFile{
		siteModelTestFile(root, "docs/intro.go", project.PageKindDocs),
		siteModelTestFile(root, "docs/plans/cross-file.go", project.PageKindDocs),
		siteModelTestFile(root, "blogs/2026_06_28_astro_backend.go", project.PageKindBlog),
		siteModelTestFile(root, "blogs/2026_06_20_project_export.go", project.PageKindBlog),
		siteModelTestFile(root, "internal/model.go", project.PageKindSource),
	}

	model, err := BuildSiteModel(projectconfig.ProjectConfig{
		Project: projectconfig.ProjectSection{Root: root},
		Source:  projectconfig.SourceConfig{RoutePrefix: "/_source"},
	}, files)
	if err != nil {
		t.Fatalf("BuildSiteModel returned error: %v", err)
	}

	if len(model.Pages) != len(files) {
		t.Fatalf("len(model.Pages) = %d, want %d", len(model.Pages), len(files))
	}

	page, ok := model.PageForFile(files[0])
	if !ok {
		t.Fatal("PageForFile returned ok=false")
	}
	if page.Route != "/_source/docs/intro.go.html" {
		t.Fatalf("page route = %q, want /_source/docs/intro.go.html", page.Route)
	}
	if page.Href != "/_source/docs/intro.go.html/" {
		t.Fatalf("page href = %q, want /_source/docs/intro.go.html/", page.Href)
	}
	if page.Title != "Intro" {
		t.Fatalf("docs page title = %q, want Intro", page.Title)
	}

	blogPage, ok := model.PageForFile(files[2])
	if !ok {
		t.Fatal("PageForFile returned ok=false for blog page")
	}
	if blogPage.Title != "Astro backend" {
		t.Fatalf("blog page title = %q, want Astro backend", blogPage.Title)
	}

	sourcePage, ok := model.PageForFile(files[4])
	if !ok {
		t.Fatal("PageForFile returned ok=false for source page")
	}
	if sourcePage.Title != "internal/model.go" {
		t.Fatalf("source page title = %q, want internal/model.go", sourcePage.Title)
	}

	if model.Navigation.Docs.FirstHref != "/_source/docs/intro.go.html/" {
		t.Fatalf("docs first href = %q, want intro href", model.Navigation.Docs.FirstHref)
	}
	if len(model.Navigation.Docs.Items) != 2 {
		t.Fatalf("len(docs items) = %d, want 2", len(model.Navigation.Docs.Items))
	}
	if model.Navigation.Docs.Items[0].Title != "Intro" {
		t.Fatalf("first docs title = %q, want Intro", model.Navigation.Docs.Items[0].Title)
	}
	if model.Navigation.Docs.Items[1].Type != SiteNavigationItemCategory || model.Navigation.Docs.Items[1].Title != "Plans" {
		t.Fatalf("second docs item = %#v, want Plans category", model.Navigation.Docs.Items[1])
	}
	if len(model.Navigation.Docs.Items[1].Items) != 1 || model.Navigation.Docs.Items[1].Items[0].Title != "Cross file" {
		t.Fatalf("plans children = %#v, want Cross file child", model.Navigation.Docs.Items[1].Items)
	}

	if model.Navigation.Blog.FirstHref != "/_source/blogs/2026_06_28_astro_backend.go.html/" {
		t.Fatalf("blog first href = %q, want latest blog href", model.Navigation.Blog.FirstHref)
	}
	if len(model.Navigation.Blog.Items) != 2 {
		t.Fatalf("len(blog items) = %d, want 2", len(model.Navigation.Blog.Items))
	}
	if got := model.Navigation.Blog.Items[0].Date; got != "2026-06-28" {
		t.Fatalf("latest blog date = %q, want 2026-06-28", got)
	}
	if got := model.Navigation.Blog.Items[0].Title; got != "Astro backend" {
		t.Fatalf("latest blog title = %q, want Astro backend", got)
	}
}

func TestBuildSiteModelInfersNarrativeTitleFromOpeningH1(t *testing.T) {
	root := t.TempDir()
	doc := siteModelTestWriteFile(t, root, "docs/intro.go", project.PageKindDocs, `package docs

// # Getting Started
// Intro body.
`)
	blog := siteModelTestWriteFile(t, root, "blogs/2026_06_30_release_notes.go", project.PageKindBlog, `package blogs

/*
# Release Notes

Body.
*/
`)

	model, err := BuildSiteModel(siteModelTestConfig(root), []project.SourceFile{doc, blog})
	if err != nil {
		t.Fatalf("BuildSiteModel returned error: %v", err)
	}

	docPage, ok := model.PageForFile(doc)
	if !ok {
		t.Fatal("PageForFile returned ok=false for docs page")
	}
	if docPage.Title != "Getting Started" {
		t.Fatalf("docs page title = %q, want Getting Started", docPage.Title)
	}
	if got := model.Navigation.Docs.Items[0].Title; got != "Getting Started" {
		t.Fatalf("docs nav title = %q, want Getting Started", got)
	}

	blogPage, ok := model.PageForFile(blog)
	if !ok {
		t.Fatal("PageForFile returned ok=false for blog page")
	}
	if blogPage.Title != "Release Notes" {
		t.Fatalf("blog page title = %q, want Release Notes", blogPage.Title)
	}
	if got := model.Navigation.Blog.Items[0].Title; got != "Release Notes" {
		t.Fatalf("blog nav title = %q, want Release Notes", got)
	}
}

func TestBuildSiteModelAppliesConfiguredMetadataOverFallbacks(t *testing.T) {
	root := t.TempDir()
	doc := siteModelTestWriteFile(t, root, "docs/intro.go", project.PageKindDocs, `// # Inferred Docs
package docs
`)
	blog := siteModelTestWriteFile(t, root, "blogs/2026_06_30_release_notes.go", project.PageKindBlog, `// # Inferred Blog
package blogs
`)
	cfg := siteModelTestConfig(root)
	cfg.Content.Metadata = map[string]projectconfig.ContentMetadata{
		"docs/intro.go": {
			Title:  "Configured Docs",
			Date:   "2026-01-02",
			Tags:   []string{"guide", "api"},
			Author: "Ada",
		},
		"blogs/2026_06_30_release_notes.go": {
			Title:  "Configured Blog",
			Date:   "2026-07-01",
			Tags:   []string{"release"},
			Author: "Grace",
		},
	}

	model, err := BuildSiteModel(cfg, []project.SourceFile{doc, blog})
	if err != nil {
		t.Fatalf("BuildSiteModel returned error: %v", err)
	}

	docPage, ok := model.PageForFile(doc)
	if !ok {
		t.Fatal("PageForFile returned ok=false for docs page")
	}
	if docPage.Title != "Configured Docs" {
		t.Fatalf("docs title = %q, want Configured Docs", docPage.Title)
	}
	if docPage.Date != "2026-01-02" {
		t.Fatalf("docs date = %q, want 2026-01-02", docPage.Date)
	}
	assertSiteModelStrings(t, docPage.Tags, []string{"guide", "api"})
	if docPage.Author != "Ada" {
		t.Fatalf("docs author = %q, want Ada", docPage.Author)
	}

	docNav := model.Navigation.Docs.Items[0]
	if docNav.Title != "Configured Docs" {
		t.Fatalf("docs nav title = %q, want Configured Docs", docNav.Title)
	}
	assertSiteModelStrings(t, docNav.Tags, []string{"guide", "api"})
	if docNav.Author != "Ada" {
		t.Fatalf("docs nav author = %q, want Ada", docNav.Author)
	}

	blogNav := model.Navigation.Blog.Items[0]
	if blogNav.Title != "Configured Blog" {
		t.Fatalf("blog nav title = %q, want Configured Blog", blogNav.Title)
	}
	if blogNav.Date != "2026-07-01" {
		t.Fatalf("blog nav date = %q, want configured date", blogNav.Date)
	}
	assertSiteModelStrings(t, blogNav.Tags, []string{"release"})
	if blogNav.Author != "Grace" {
		t.Fatalf("blog nav author = %q, want Grace", blogNav.Author)
	}
}

func TestBuildSiteModelSortsBlogNavigationByPageDate(t *testing.T) {
	root := t.TempDir()
	files := []project.SourceFile{
		siteModelTestWriteFile(t, root, "blogs/2026_06_10_old.go", project.PageKindBlog, "package blogs\n"),
		siteModelTestWriteFile(t, root, "blogs/2026-07-01-new.go", project.PageKindBlog, "package blogs\n"),
		siteModelTestWriteFile(t, root, "blogs/2026_99_01_invalid.go", project.PageKindBlog, "package blogs\n"),
	}

	model, err := BuildSiteModel(siteModelTestConfig(root), files)
	if err != nil {
		t.Fatalf("BuildSiteModel returned error: %v", err)
	}

	items := model.Navigation.Blog.Items
	if len(items) != 3 {
		t.Fatalf("len(blog items) = %d, want 3", len(items))
	}
	if got := items[0].SourcePath; got != "blogs/2026-07-01-new.go" {
		t.Fatalf("first blog source path = %q, want newest dated post", got)
	}
	if got := items[0].Date; got != "2026-07-01" {
		t.Fatalf("first blog date = %q, want 2026-07-01", got)
	}
	if got := items[1].SourcePath; got != "blogs/2026_06_10_old.go" {
		t.Fatalf("second blog source path = %q, want older dated post", got)
	}
	if got := items[2].Date; got != "" {
		t.Fatalf("invalid date fallback = %q, want empty date", got)
	}
}

func TestBuildSiteModelDoesNotInferSourceTitleFromH1(t *testing.T) {
	root := t.TempDir()
	source := siteModelTestWriteFile(t, root, "internal/model.go", project.PageKindSource, `// # Source Narrative Title
package internal
`)

	model, err := BuildSiteModel(siteModelTestConfig(root), []project.SourceFile{source})
	if err != nil {
		t.Fatalf("BuildSiteModel returned error: %v", err)
	}

	page, ok := model.PageForFile(source)
	if !ok {
		t.Fatal("PageForFile returned ok=false")
	}
	if page.Title != "internal/model.go" {
		t.Fatalf("source title = %q, want path fallback", page.Title)
	}
}

func TestBuildSiteModelRejectsFileWithoutRelPath(t *testing.T) {
	root := t.TempDir()
	_, err := BuildSiteModel(projectconfig.ProjectConfig{
		Project: projectconfig.ProjectSection{Root: root},
		Source:  projectconfig.SourceConfig{RoutePrefix: "/_source"},
	}, []project.SourceFile{
		{
			AbsPath:  filepath.Join(root, "main.go"),
			Language: "go",
			Kind:     project.PageKindSource,
		},
	})
	if err == nil {
		t.Fatal("BuildSiteModel returned nil error for missing rel path")
	}
}

func siteModelTestConfig(root string) projectconfig.ProjectConfig {
	return projectconfig.ProjectConfig{
		Project: projectconfig.ProjectSection{Root: root},
		Content: projectconfig.ContentConfig{
			Docs:  "docs",
			Blogs: "blogs",
		},
		Source: projectconfig.SourceConfig{RoutePrefix: "/_source"},
	}
}

func assertSiteModelStrings(t *testing.T, got []string, want []string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("strings = %#v, want %#v", got, want)
	}
}

func siteModelTestWriteFile(t *testing.T, root string, relPath string, kind project.PageKind, content string) project.SourceFile {
	t.Helper()
	file := siteModelTestFile(root, relPath, kind)
	if err := os.MkdirAll(filepath.Dir(file.AbsPath), 0755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(file.AbsPath, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return file
}

func siteModelTestFile(root string, relPath string, kind project.PageKind) project.SourceFile {
	return project.SourceFile{
		AbsPath:  filepath.Join(root, filepath.FromSlash(relPath)),
		RelPath:  relPath,
		Language: "go",
		Kind:     kind,
	}
}
