package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

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
	Date       string
	Tags       []string
	Author     string
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
	Tags       []string
	Author     string
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
	pages = mergeSitePageMetadata(cfg, pages)

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
			Tags:       []string{},
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

type sitePageMetadata struct {
	Title  string
	Date   string
	Tags   []string
	Author string
}

func mergeSitePageMetadata(cfg projectconfig.ProjectConfig, pages []SitePage) []SitePage {
	docsPrefix := siteContentPrefix(cfg.Project.Root, cfg.Content.Docs, "docs")
	blogPrefix := siteContentPrefix(cfg.Project.Root, cfg.Content.Blogs, "blogs")
	metadataByPath := siteConfiguredMetadataByPath(cfg)

	for i := range pages {
		page := &pages[i]
		page.Title = siteInferredPageTitle(*page, docsPrefix, blogPrefix)
		page.Date = siteInferredPageDate(*page)
		page.Tags = []string{}
		page.Author = ""

		metadata, ok := siteConfiguredMetadataForPage(metadataByPath, cfg.Project.Root, *page)
		if !ok {
			continue
		}
		if metadata.Title != "" {
			page.Title = metadata.Title
		}
		if metadata.Date != "" {
			page.Date = metadata.Date
		}
		page.Tags = cloneSiteStrings(metadata.Tags)
		if metadata.Author != "" {
			page.Author = metadata.Author
		}
	}

	return pages
}

func siteInferredPageTitle(page SitePage, docsPrefix string, blogPrefix string) string {
	switch page.Kind {
	case project.PageKindDocs, project.PageKindBlog:
		if title := inferSitePageH1Title(page.File); title != "" {
			return title
		}
		return siteNavigationTitle(page, docsPrefix, blogPrefix)
	default:
		return sitePageTitle(page.File)
	}
}

func siteInferredPageDate(page SitePage) string {
	if page.Kind != project.PageKindBlog {
		return ""
	}
	return siteBlogDate(page)
}

func siteConfiguredMetadataByPath(cfg projectconfig.ProjectConfig) map[string]sitePageMetadata {
	if len(cfg.Content.Metadata) == 0 {
		return nil
	}

	metadataByPath := make(map[string]sitePageMetadata, len(cfg.Content.Metadata))
	for key, metadata := range cfg.Content.Metadata {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		siteMetadata := sitePageMetadataFromConfig(metadata)
		for _, normalizedKey := range siteMetadataKeyVariants(cfg.Project.Root, key) {
			metadataByPath[normalizedKey] = siteMetadata
		}
	}
	return metadataByPath
}

func siteConfiguredMetadataForPage(metadataByPath map[string]sitePageMetadata, root string, page SitePage) (sitePageMetadata, bool) {
	if len(metadataByPath) == 0 {
		return sitePageMetadata{}, false
	}
	for _, key := range sitePageMetadataLookupKeys(root, page) {
		if metadata, ok := metadataByPath[key]; ok {
			return metadata, true
		}
	}
	return sitePageMetadata{}, false
}

func sitePageMetadataFromConfig(metadata projectconfig.ContentMetadata) sitePageMetadata {
	return sitePageMetadata{
		Title:  strings.TrimSpace(metadata.Title),
		Date:   strings.TrimSpace(metadata.Date),
		Tags:   normalizeSiteStrings(metadata.Tags),
		Author: strings.TrimSpace(metadata.Author),
	}
}

func sitePageMetadataLookupKeys(root string, page SitePage) []string {
	keys := make([]string, 0, 6)
	keys = appendSiteMetadataKeyVariants(keys, root, page.SourcePath)
	keys = appendSiteMetadataKeyVariants(keys, root, page.File.RelPath)
	keys = appendSiteMetadataKeyVariants(keys, root, page.File.AbsPath)
	return keys
}

func siteMetadataKeyVariants(root string, value string) []string {
	return appendSiteMetadataKeyVariants(nil, root, value)
}

func appendSiteMetadataKeyVariants(keys []string, root string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return keys
	}

	add := func(candidate string) {
		candidate = cleanSiteMetadataPath(candidate)
		if candidate == "" {
			return
		}
		for _, existing := range keys {
			if existing == candidate {
				return
			}
		}
		keys = append(keys, candidate)
	}

	add(value)

	filePath := filepath.FromSlash(value)
	if filepath.IsAbs(filePath) {
		if rel, err := filepath.Rel(root, filePath); err == nil && !strings.HasPrefix(rel, "..") {
			add(rel)
		}
	} else if strings.TrimSpace(root) != "" {
		add(filepath.Join(root, filePath))
	}

	return keys
}

func cleanSiteMetadataPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = filepath.ToSlash(filepath.Clean(value))
	value = strings.TrimPrefix(value, "./")
	return value
}

func normalizeSiteStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value := strings.TrimSpace(value); value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func cloneSiteStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func inferSitePageH1Title(file project.SourceFile) string {
	if strings.TrimSpace(file.AbsPath) == "" {
		return ""
	}
	data, err := os.ReadFile(file.AbsPath)
	if err != nil {
		return ""
	}
	const maxTitleScanBytes = 64 * 1024
	if len(data) > maxTitleScanBytes {
		data = data[:maxTitleScanBytes]
	}
	return firstStandaloneCommentH1Title(string(data), file.Language)
}

func firstMarkdownH1Title(markdown string) string {
	for _, line := range strings.Split(markdown, "\n") {
		line = strings.TrimSpace(line)
		if len(line) < 2 || line[0] != '#' {
			continue
		}
		if len(line) > 1 && line[1] == '#' {
			continue
		}
		if line[1] != ' ' && line[1] != '\t' {
			continue
		}
		if title := strings.TrimSpace(line[1:]); title != "" {
			return title
		}
	}
	return ""
}

func firstStandaloneCommentH1Title(source string, language string) string {
	linePrefixes, blockComments := siteCommentSyntax(language)
	lines := strings.Split(source, "\n")

	for i := 0; i < len(lines); {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		matched := false
		for _, prefix := range linePrefixes {
			if strings.HasPrefix(line, prefix) {
				commentLines := make([]string, 0)
				for i < len(lines) {
					trimmed := strings.TrimSpace(lines[i])
					if !strings.HasPrefix(trimmed, prefix) {
						break
					}
					commentLines = append(commentLines, cleanSiteLineComment(trimmed, prefix))
					i++
				}
				if title := firstMarkdownH1Title(strings.Join(commentLines, "\n")); title != "" {
					return title
				}
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		for _, block := range blockComments {
			if strings.HasPrefix(line, block.start) {
				blockLines := []string{line}
				i++
				for !strings.Contains(blockLines[len(blockLines)-1], block.end) && i < len(lines) {
					blockLines = append(blockLines, lines[i])
					i++
				}
				if title := firstMarkdownH1Title(cleanSiteBlockComment(strings.Join(blockLines, "\n"), block.start, block.end)); title != "" {
					return title
				}
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		i++
	}

	return ""
}

type siteBlockCommentSyntax struct {
	start string
	end   string
}

func siteCommentSyntax(language string) ([]string, []siteBlockCommentSyntax) {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python", "py", "ruby":
		return []string{"#"}, nil
	case "haskell":
		return []string{"--"}, []siteBlockCommentSyntax{{start: "{-", end: "-}"}}
	default:
		return []string{"//"}, []siteBlockCommentSyntax{{start: "/*", end: "*/"}}
	}
}

func cleanSiteLineComment(line string, prefix string) string {
	line = strings.TrimPrefix(strings.TrimSpace(line), prefix)
	if strings.HasPrefix(line, " ") {
		line = line[1:]
	}
	return strings.TrimRight(line, " \t\r")
}

func cleanSiteBlockComment(text string, start string, end string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, start)
	text = strings.TrimSuffix(text, end)
	lines := strings.Split(text, "\n")

	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	for i, line := range lines {
		line = strings.TrimRight(line, " \t\r")
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "*") {
			trimmed = strings.TrimPrefix(trimmed, "*")
			if strings.HasPrefix(trimmed, " ") {
				trimmed = trimmed[1:]
			}
			line = trimmed
		}
		lines[i] = line
	}

	return strings.Join(lines, "\n")
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
			Title:      siteNavigationPageTitle(page, docsPrefix, ""),
			Href:       page.Href,
			SourcePath: page.SourcePath,
			Tags:       cloneSiteStrings(page.Tags),
			Author:     page.Author,
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
		leftDate := blogs[i].Date
		rightDate := blogs[j].Date
		if leftDate != rightDate {
			return leftDate > rightDate
		}
		return blogs[i].SourcePath > blogs[j].SourcePath
	})

	items := make([]SiteNavigationItem, 0, len(blogs))
	for _, page := range blogs {
		items = append(items, SiteNavigationItem{
			Type:       SiteNavigationItemLink,
			Title:      siteNavigationPageTitle(page, "", blogPrefix),
			Href:       page.Href,
			SourcePath: page.SourcePath,
			Date:       page.Date,
			Tags:       cloneSiteStrings(page.Tags),
			Author:     page.Author,
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

func siteNavigationPageTitle(page SitePage, docsPrefix string, blogPrefix string) string {
	if title := strings.TrimSpace(page.Title); title != "" {
		return title
	}
	return siteNavigationTitle(page, docsPrefix, blogPrefix)
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
		return validSiteDate(strings.ReplaceAll(prefix, "_", "-"))
	}
	if isDateLike(prefix, '-') {
		return validSiteDate(prefix)
	}
	return ""
}

func validSiteDate(value string) string {
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return ""
	}
	return value
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
