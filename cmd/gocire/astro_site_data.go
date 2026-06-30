package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
)

const (
	defaultAstroSiteDescription = "Generated source documentation."
	astroTrailingSlashPolicy    = "always"
)

type astroSiteData struct {
	Site       astroSiteDataSite       `json:"site"`
	Pages      []astroSiteDataPage     `json:"pages"`
	Navigation astroSiteDataNavigation `json:"navigation"`
}

type astroSiteDataSite struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	URL           string `json:"url"`
	TrailingSlash string `json:"trailingSlash"`
}

type astroSiteDataPage struct {
	RouteParam string   `json:"routeParam"`
	Href       string   `json:"href"`
	Module     string   `json:"module"`
	Kind       string   `json:"kind"`
	Title      string   `json:"title"`
	SourcePath string   `json:"sourcePath"`
	Language   string   `json:"language"`
	Date       string   `json:"date"`
	Tags       []string `json:"tags"`
	Author     string   `json:"author"`
}

type astroSiteDataNavigation struct {
	Docs astroSiteDataNavigationSection `json:"docs"`
	Blog astroSiteDataNavigationSection `json:"blog"`
}

type astroSiteDataNavigationSection struct {
	FirstHref string                        `json:"firstHref"`
	Items     []astroSiteDataNavigationItem `json:"items"`
}

type astroSiteDataNavigationItem struct {
	Type       string                        `json:"type"`
	Title      string                        `json:"title,omitempty"`
	Href       string                        `json:"href,omitempty"`
	SourcePath string                        `json:"sourcePath,omitempty"`
	Date       string                        `json:"date,omitempty"`
	Tags       []string                      `json:"tags,omitempty"`
	Author     string                        `json:"author,omitempty"`
	Items      []astroSiteDataNavigationItem `json:"items,omitempty"`
}

func writeAstroSiteData(outputDir string, siteConfig projectconfig.SiteConfig, pages []astroGeneratedPage, navigation SiteNavigation) error {
	siteData := newAstroSiteData(siteConfig, pages, navigation)
	data, err := json.MarshalIndent(siteData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal Astro site data: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("export const siteData = ")
	sb.Write(data)
	sb.WriteString(" as const;\n\n")
	sb.WriteString("export const pages = siteData.pages;\n")
	sb.WriteString("export const navigation = siteData.navigation;\n")

	return writeOutputFile(filepath.Join(outputDir, "src", "generated", "site-data.ts"), sb.String())
}

func newAstroSiteData(siteConfig projectconfig.SiteConfig, pages []astroGeneratedPage, navigation SiteNavigation) astroSiteData {
	return astroSiteData{
		Site: astroSiteDataSite{
			Title:         normalizedAstroSiteTitle(siteConfig.Title),
			Description:   normalizedAstroSiteDescription(siteConfig.Description),
			URL:           strings.TrimSpace(siteConfig.URL),
			TrailingSlash: astroTrailingSlashPolicy,
		},
		Pages:      astroSiteDataPages(pages),
		Navigation: astroSiteDataNavigationFromSiteNavigation(navigation),
	}
}

func astroSiteDataPages(pages []astroGeneratedPage) []astroSiteDataPage {
	sortedPages := append([]astroGeneratedPage(nil), pages...)
	sort.SliceStable(sortedPages, func(i, j int) bool {
		return sortedPages[i].Route < sortedPages[j].Route
	})

	sitePages := make([]astroSiteDataPage, 0, len(sortedPages))
	for _, page := range sortedPages {
		href := strings.TrimSpace(page.Href)
		if href == "" {
			href = astroRouteHref("/" + strings.TrimLeft(page.Route, "/"))
		}
		sitePages = append(sitePages, astroSiteDataPage{
			RouteParam: page.Route,
			Href:       href,
			Module:     page.Module,
			Kind:       string(page.Kind),
			Title:      page.Title,
			SourcePath: page.SourcePath,
			Language:   page.Language,
			Date:       strings.TrimSpace(page.Date),
			Tags:       cloneAstroSiteDataStrings(page.Tags),
			Author:     strings.TrimSpace(page.Author),
		})
	}
	return sitePages
}

func astroSiteDataNavigationFromSiteNavigation(navigation SiteNavigation) astroSiteDataNavigation {
	return astroSiteDataNavigation{
		Docs: astroSiteDataNavigationSectionFromSiteNavigationSection(navigation.Docs),
		Blog: astroSiteDataNavigationSectionFromSiteNavigationSection(navigation.Blog),
	}
}

func astroSiteDataNavigationSectionFromSiteNavigationSection(section SiteNavigationSection) astroSiteDataNavigationSection {
	return astroSiteDataNavigationSection{
		FirstHref: section.FirstHref,
		Items:     astroSiteDataNavigationItems(section.Items),
	}
}

func astroSiteDataNavigationItems(items []SiteNavigationItem) []astroSiteDataNavigationItem {
	siteItems := make([]astroSiteDataNavigationItem, 0, len(items))
	for _, item := range items {
		siteItems = append(siteItems, astroSiteDataNavigationItem{
			Type:       string(item.Type),
			Title:      item.Title,
			Href:       item.Href,
			SourcePath: item.SourcePath,
			Date:       strings.TrimSpace(item.Date),
			Tags:       cloneAstroSiteDataStrings(item.Tags),
			Author:     strings.TrimSpace(item.Author),
			Items:      astroSiteDataNavigationItems(item.Items),
		})
	}
	return siteItems
}

func normalizedAstroSiteDescription(description string) string {
	if description := strings.TrimSpace(description); description != "" {
		return description
	}
	return defaultAstroSiteDescription
}

func cloneAstroSiteDataStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	return append([]string(nil), values...)
}
