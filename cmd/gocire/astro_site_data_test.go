package main

import (
	"reflect"
	"testing"

	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

func TestNewAstroSiteDataUsesStableContract(t *testing.T) {
	data := newAstroSiteData(projectconfig.SiteConfig{
		Title: " Docs ",
		URL:   "https://example.com/docs",
	}, []astroGeneratedPage{
		{
			Route:      "source/z.go.html",
			Href:       "/source/z.go.html/",
			Module:     "../generated/pages/source/z.go.html.astro",
			Kind:       project.PageKindSource,
			Title:      "z.go",
			Language:   "go",
			SourcePath: "z.go",
		},
		{
			Route:      "docs/intro.html",
			Module:     "../generated/pages/docs/intro.html.astro",
			Kind:       project.PageKindDocs,
			Title:      "Intro",
			Date:       " 2026-06-30 ",
			Tags:       []string{"docs", "intro"},
			Author:     " Ada ",
			SourcePath: "docs/intro.go",
		},
	}, SiteNavigation{
		Docs: SiteNavigationSection{
			FirstHref: "/docs/intro.html/",
			Items: []SiteNavigationItem{
				{
					Type:       SiteNavigationItemLink,
					Title:      "Intro",
					Href:       "/docs/intro.html/",
					SourcePath: "docs/intro.go",
					Tags:       []string{"docs"},
				},
			},
		},
	})

	if data.Site.Title != "Docs" {
		t.Fatalf("site title = %q, want Docs", data.Site.Title)
	}
	if data.Site.Description != defaultAstroSiteDescription {
		t.Fatalf("site description = %q, want default", data.Site.Description)
	}
	if data.Site.URL != "https://example.com/docs" {
		t.Fatalf("site url = %q, want configured URL", data.Site.URL)
	}
	if data.Site.TrailingSlash != "always" {
		t.Fatalf("trailing slash = %q, want always", data.Site.TrailingSlash)
	}

	if len(data.Pages) != 2 {
		t.Fatalf("len(pages) = %d, want 2", len(data.Pages))
	}
	if data.Pages[0].RouteParam != "docs/intro.html" {
		t.Fatalf("first routeParam = %q, want sorted docs page", data.Pages[0].RouteParam)
	}
	if data.Pages[0].Href != "/docs/intro.html/" {
		t.Fatalf("empty href fallback = %q, want trailing slash route href", data.Pages[0].Href)
	}
	if data.Pages[0].Date != "2026-06-30" {
		t.Fatalf("page date = %q, want trimmed date", data.Pages[0].Date)
	}
	if data.Pages[0].Author != "Ada" {
		t.Fatalf("page author = %q, want trimmed author", data.Pages[0].Author)
	}
	if !reflect.DeepEqual(data.Pages[0].Tags, []string{"docs", "intro"}) {
		t.Fatalf("page tags = %#v, want docs/intro tags", data.Pages[0].Tags)
	}
	if data.Pages[1].Href != "/source/z.go.html/" {
		t.Fatalf("configured href = %q, want preserved href", data.Pages[1].Href)
	}

	if data.Navigation.Docs.FirstHref != "/docs/intro.html/" {
		t.Fatalf("docs first href = %q, want navigation href", data.Navigation.Docs.FirstHref)
	}
	if len(data.Navigation.Docs.Items) != 1 || data.Navigation.Docs.Items[0].Title != "Intro" {
		t.Fatalf("docs navigation items = %#v, want Intro item", data.Navigation.Docs.Items)
	}
}
