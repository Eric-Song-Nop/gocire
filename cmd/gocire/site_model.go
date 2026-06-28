package main

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

type SiteModel struct {
	Pages      []SitePage
	Routes     internal.SourceRouteManifest
	Navigation SiteNavigation
}

type SitePage struct {
	File       project.SourceFile
	Route      string
	Href       string
	Title      string
	Kind       project.PageKind
	Language   string
	SourcePath string
}

type SiteNavigation struct {
	Docs SiteNavigationSection
	Blog SiteNavigationSection
}

type SiteNavigationSection struct {
	FirstHref string
	Items     []SiteNavigationItem
}

type SiteNavigationItem struct {
	Type       SiteNavigationItemType
	Title      string
	Href       string
	SourcePath string
	Date       string
	Items      []SiteNavigationItem
}

type SiteNavigationItemType string

const (
	SiteNavigationItemLink     SiteNavigationItemType = "link"
	SiteNavigationItemCategory SiteNavigationItemType = "category"
)

func BuildSiteModel(cfg projectconfig.ProjectConfig, files []project.SourceFile) (SiteModel, error) {
	routes, err := SourceRouteManifestForProject(cfg, files)
	if err != nil {
		return SiteModel{}, err
	}

	pages, err := SitePagesForProject(files, routes)
	if err != nil {
		return SiteModel{}, err
	}

	navigation := SiteNavigationForPages(cfg, pages)
	return SiteModel{
		Pages:      pages,
		Routes:     routes,
		Navigation: navigation,
	}, nil
}

func SitePagesForProject(files []project.SourceFile, routes internal.SourceRouteManifest) ([]SitePage, error) {
	pages := make([]SitePage, 0, len(files))
	for _, file := range files {
		if strings.TrimSpace(file.RelPath) == "" {
			return nil, fmt.Errorf("source file has no relative path")
		}
		route, err := sitePageRoute(routes, file)
		if err != nil {
			return nil, err
		}
		pages = append(pages, SitePage{
			File:       file,
			Route:      route,
			Href:       astroRouteHref(route),
			Title:      sitePageTitle(file),
			Kind:       file.Kind,
			Language:   file.Language,
			SourcePath: file.RelPath,
		})
	}

	sort.SliceStable(pages, func(i, j int) bool {
		return pages[i].SourcePath < pages[j].SourcePath
	})
	return pages, nil
}

func SiteNavigationForPages(cfg projectconfig.ProjectConfig, pages []SitePage) SiteNavigation {
	docsPrefix := siteContentPrefix(cfg.Project.Root, cfg.Content.Docs, "docs")
	blogPrefix := siteContentPrefix(cfg.Project.Root, cfg.Content.Blogs, "blogs")
	return SiteNavigation{
		Docs: siteDocsNavigation(pages, docsPrefix),
		Blog: siteBlogNavigation(pages, blogPrefix),
	}
}

func (m SiteModel) PageForFile(file project.SourceFile) (SitePage, bool) {
	target := cleanSiteAbsPath(file.AbsPath)
	for _, page := range m.Pages {
		if cleanSiteAbsPath(page.File.AbsPath) == target {
			return page, true
		}
	}
	return SitePage{}, false
}

func sitePageRoute(routes internal.SourceRouteManifest, file project.SourceFile) (string, error) {
	if route, _, ok := routes.RouteForSourcePath(file.AbsPath); ok {
		return route, nil
	}
	if strings.TrimSpace(file.RelPath) == "" {
		return "", fmt.Errorf("source file has no relative path")
	}
	return routes.SourceRoute(file.RelPath), nil
}

func sitePageTitle(file project.SourceFile) string {
	if strings.TrimSpace(file.RelPath) != "" {
		return file.RelPath
	}
	return filepath.Base(file.AbsPath)
}

func cleanSiteAbsPath(path string) string {
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func siteDocsNavigation(pages []SitePage, docsPrefix string) SiteNavigationSection {
	docs := filterSitePagesByKind(pages, project.PageKindDocs)
	sort.SliceStable(docs, func(i, j int) bool {
		return docs[i].SourcePath < docs[j].SourcePath
	})

	items := make([]SiteNavigationItem, 0, len(docs))
	for _, page := range docs {
		insertDocsNavigationItem(&items, docsNavigationParts(page.SourcePath, docsPrefix), SiteNavigationItem{
			Type:       SiteNavigationItemLink,
			Title:      siteNavigationTitle(page, docsPrefix, ""),
			Href:       page.Href,
			SourcePath: page.SourcePath,
		})
	}

	return SiteNavigationSection{
		FirstHref: firstSiteNavigationHref(items),
		Items:     items,
	}
}

func siteBlogNavigation(pages []SitePage, blogPrefix string) SiteNavigationSection {
	blogs := filterSitePagesByKind(pages, project.PageKindBlog)
	sort.SliceStable(blogs, func(i, j int) bool {
		leftDate := siteBlogDate(blogs[i])
		rightDate := siteBlogDate(blogs[j])
		if leftDate != rightDate {
			return leftDate > rightDate
		}
		return blogs[i].SourcePath > blogs[j].SourcePath
	})

	items := make([]SiteNavigationItem, 0, len(blogs))
	for _, page := range blogs {
		items = append(items, SiteNavigationItem{
			Type:       SiteNavigationItemLink,
			Title:      siteNavigationTitle(page, "", blogPrefix),
			Href:       page.Href,
			SourcePath: page.SourcePath,
			Date:       siteBlogDate(page),
		})
	}

	return SiteNavigationSection{
		FirstHref: firstSiteNavigationHref(items),
		Items:     items,
	}
}

func filterSitePagesByKind(pages []SitePage, kind project.PageKind) []SitePage {
	filtered := make([]SitePage, 0)
	for _, page := range pages {
		if page.Kind == kind {
			filtered = append(filtered, page)
		}
	}
	return filtered
}

func docsNavigationParts(sourcePath string, docsPrefix string) []string {
	sourcePath = trimSiteContentPrefix(sourcePath, docsPrefix)
	sourcePath = path.Clean(sourcePath)
	if sourcePath == "." || sourcePath == "" {
		return nil
	}
	return strings.Split(sourcePath, "/")
}

func insertDocsNavigationItem(items *[]SiteNavigationItem, parts []string, link SiteNavigationItem) {
	if len(parts) <= 1 {
		*items = append(*items, link)
		return
	}

	categoryTitle := siteTitleFromPathPart(parts[0])
	for i := range *items {
		item := &(*items)[i]
		if item.Type == SiteNavigationItemCategory && item.Title == categoryTitle {
			insertDocsNavigationItem(&item.Items, parts[1:], link)
			return
		}
	}

	category := SiteNavigationItem{
		Type:  SiteNavigationItemCategory,
		Title: categoryTitle,
	}
	insertDocsNavigationItem(&category.Items, parts[1:], link)
	*items = append(*items, category)
}

func firstSiteNavigationHref(items []SiteNavigationItem) string {
	for _, item := range items {
		if item.Type == SiteNavigationItemLink && item.Href != "" {
			return item.Href
		}
		if href := firstSiteNavigationHref(item.Items); href != "" {
			return href
		}
	}
	return ""
}

func siteNavigationTitle(page SitePage, docsPrefix string, blogPrefix string) string {
	rel := page.SourcePath
	switch page.Kind {
	case project.PageKindDocs:
		rel = trimSiteContentPrefix(rel, docsPrefix)
	case project.PageKindBlog:
		rel = trimSiteContentPrefix(rel, blogPrefix)
	}
	return siteTitleFromPathPart(rel)
}

func siteContentPrefix(root string, contentPath string, fallback string) string {
	if strings.TrimSpace(contentPath) == "" {
		return fallback
	}
	if filepath.IsAbs(contentPath) {
		if rel, err := filepath.Rel(root, contentPath); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(contentPath)
}

func trimSiteContentPrefix(sourcePath string, prefix string) string {
	prefix = strings.Trim(path.Clean(prefix), "/")
	if prefix == "." || prefix == "" {
		return sourcePath
	}
	return strings.TrimPrefix(sourcePath, prefix+"/")
}

func siteTitleFromPathPart(value string) string {
	value = path.Base(strings.TrimSpace(value))
	ext := path.Ext(value)
	value = strings.TrimSuffix(value, ext)
	value = strings.TrimLeft(value, "0123456789-_")
	value = strings.ReplaceAll(value, "_", " ")
	value = strings.ReplaceAll(value, "-", " ")
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return "Untitled"
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func siteBlogDate(page SitePage) string {
	base := path.Base(page.SourcePath)
	if len(base) < len("2006_01_02") {
		return ""
	}
	prefix := base[:len("2006_01_02")]
	if isDateLike(prefix, '_') {
		return strings.ReplaceAll(prefix, "_", "-")
	}
	if isDateLike(prefix, '-') {
		return prefix
	}
	return ""
}

func isDateLike(value string, sep byte) bool {
	if len(value) != len("2006_01_02") {
		return false
	}
	for i := range value {
		switch i {
		case 4, 7:
			if value[i] != sep {
				return false
			}
		default:
			if value[i] < '0' || value[i] > '9' {
				return false
			}
		}
	}
	return true
}
