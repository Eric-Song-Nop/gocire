package main

import (
	"path/filepath"
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

func siteModelTestFile(root string, relPath string, kind project.PageKind) project.SourceFile {
	return project.SourceFile{
		AbsPath:  filepath.Join(root, filepath.FromSlash(relPath)),
		RelPath:  relPath,
		Language: "go",
		Kind:     kind,
	}
}
