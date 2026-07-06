package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var expectedAstroAssetFiles = []string{
	"package.json",
	"astro.config.mjs",
	"src/pages/rss.xml.ts",
	"src/pages/sitemap.xml.ts",
	"src/layouts/SiteLayout.astro",
	"src/components/CodePage.astro",
	"src/components/NavigationRail.astro",
	"src/components/Sidebar.astro",
	"src/components/SidebarBody.astro",
	"src/components/SidebarItems.astro",
	"src/styles/global.css",
	"src/scripts/code-copy.js",
	"src/scripts/navigation-rail.js",
	"src/scripts/tooltip.js",
	"src/scripts/theme.js",
}

func TestAstroTemplateOutputFilesMatchExpectedAssetContract(t *testing.T) {
	got := astroTemplateOutputFiles()
	assertAstroStringSetsEqual(t, got, expectedAstroAssetFiles)
}

func TestWriteAstroSiteAssetsWritesExpectedFiles(t *testing.T) {
	outputDir := t.TempDir()

	if err := WriteAstroSiteAssets(outputDir, "Example Docs"); err != nil {
		t.Fatalf("WriteAstroSiteAssets returned error: %v", err)
	}

	for _, relPath := range expectedAstroAssetFiles {
		if _, err := os.Stat(filepath.Join(outputDir, filepath.FromSlash(relPath))); err != nil {
			t.Fatalf("expected Astro asset %q: %v", relPath, err)
		}
	}

	rssEndpoint := readAstroAssetFile(t, outputDir, "src/pages/rss.xml.ts")
	sitemapEndpoint := readAstroAssetFile(t, outputDir, "src/pages/sitemap.xml.ts")
	for _, endpoint := range []struct {
		name     string
		contents string
	}{
		{name: "rss.xml.ts", contents: rssEndpoint},
		{name: "sitemap.xml.ts", contents: sitemapEndpoint},
	} {
		assertAstroAssetContains(t, endpoint.contents, `import { siteData } from "../generated/site-data";`)
		if strings.Contains(endpoint.contents, "import.meta.glob") {
			t.Fatalf("%s should use generated site-data instead of import.meta.glob", endpoint.name)
		}
	}
	assertAstroAssetContains(t, rssEndpoint, `kind === "blog"`)
	assertAstroAssetContains(t, rssEndpoint, "site.url is required to generate rss.xml")
	assertAstroAssetContains(t, sitemapEndpoint, "siteData.pages")
	assertAstroAssetContains(t, sitemapEndpoint, "site.url is required to generate sitemap.xml")

	packageJSON := readAstroAssetFile(t, outputDir, "package.json")
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(packageJSON), &pkg); err != nil {
		t.Fatalf("package.json is not valid JSON: %v", err)
	}
	if pkg.Dependencies["astro"] == "" {
		t.Fatal("package.json dependencies missing astro")
	}
	if pkg.Dependencies["katex"] == "" {
		t.Fatal("package.json dependencies missing katex")
	}
	if pkg.Dependencies["@floating-ui/dom"] == "" {
		t.Fatal("package.json dependencies missing @floating-ui/dom")
	}
	if pkg.Dependencies["lucide-astro"] == "" {
		t.Fatal("package.json dependencies missing lucide-astro")
	}

	astroConfig := readAstroAssetFile(t, outputDir, "astro.config.mjs")
	for _, want := range []string{
		`output: "static"`,
		`trailingSlash: "always"`,
		`build: {`,
		`format: "directory"`,
	} {
		assertAstroAssetContains(t, astroConfig, want)
	}

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	assertAstroAssetContains(t, layout, "Example Docs")
	for _, want := range []string{
		`import Moon from "lucide-astro/Moon";`,
		`import Sun from "lucide-astro/Sun";`,
		`import { siteData } from "../generated/site-data";`,
		`import "katex/dist/katex.min.css";`,
		`const fallbackDescription = siteData.site.description;`,
		`description = fallbackDescription`,
		"theme-toggle",
		"data-theme",
		"../scripts/code-copy.js",
		"../scripts/navigation-rail.js",
		"../scripts/theme.js",
	} {
		assertAstroAssetContains(t, layout, want)
	}
	assertAstroAssetNotContains(t, layout, "../scripts/sidebar.js")

	codePage := readAstroAssetFile(t, outputDir, "src/components/CodePage.astro")
	for _, want := range []string{
		`import Check from "lucide-astro/Check";`,
		`import Copy from "lucide-astro/Copy";`,
		"title: string",
		"kind?: string",
		"language?: string",
		"sourcePath?: string",
		"date?: string",
		"tags?: string[]",
		"author?: string",
		"toc?: TableOfContentsItem[]",
		`import NavigationRail from "./NavigationRail.astro";`,
		"code-page--has-toc",
		"<NavigationRail",
		"<slot />",
		`<template id="gocire-code-copy-icon">`,
		`<template id="gocire-code-copy-success-icon">`,
	} {
		assertAstroAssetContains(t, codePage, want)
	}
	for _, unwanted := range []string{
		`<header class="page-header">`,
		`<p class="page-kicker">`,
		`<dl class="page-meta">`,
		"<dt>Date</dt>",
		"<dt>Author</dt>",
		"<dt>Tags</dt>",
		"metadata-tags",
		`<dt>Language</dt>`,
		`<dt>Path</dt>`,
		`<dd><code>{sourcePath}</code></dd>`,
	} {
		assertAstroAssetNotContains(t, codePage, unwanted)
	}

	codeCopy := readAstroAssetFile(t, outputDir, "src/scripts/code-copy.js")
	for _, want := range []string{
		"[data-code-block]",
		".page-content .cire-prose > pre, .page-content .cire-prose > .chroma",
		"wrapProseCodeSurface",
		"data-code-copy",
		"Copy code",
		"Copied",
		"Copy failed",
		"navigator.clipboard",
		"window.isSecureContext",
		`querySelectorAll("[data-inlay-hint]")`,
		"hint.remove()",
		"document.execCommand",
		"HTMLTemplateElement",
		"gocire-code-copy-icon",
		"gocire-code-copy-success-icon",
		"replaceChildren",
		"window.setTimeout",
		"window.clearTimeout",
	} {
		assertAstroAssetContains(t, codeCopy, want)
	}

	tooltip := readAstroAssetFile(t, outputDir, "src/scripts/tooltip.js")
	for _, want := range []string{
		"@floating-ui/dom",
		"[data-hover-html], [data-hover]",
		"data-hover-html",
		"data-hover",
		"innerHTML",
		"textContent",
		"TextDecoder",
		`setAttribute("role", "dialog")`,
		`setAttribute("aria-modal", "false")`,
		"aria-describedby",
		"aria-controls",
		"aria-expanded",
		"tabindex",
		"Escape",
		"pointerdown",
		"pointermove",
		"pointerup",
		"pointercancel",
		`document.addEventListener(
    "click"`,
		"autoUpdate",
		`strategy: "fixed"`,
		"const hideDelayMs = 120",
		"const tapMoveThreshold = 8",
		"const focusableSelector",
		"touchPinned",
		"keyboardPinned",
		"suppressNextClick",
		"handleKeyboardActivation",
		`event.key !== "Enter" && event.key !== " "`,
		`token.addEventListener("keydown"`,
		"tooltipFocusableElements",
		"focusFirstTooltipItem",
		"handleTooltipTab",
		`event.key !== "Tab"`,
		"focusPageElementAdjacentToToken",
		`focus({ preventScroll: true })`,
		"document.activeElement === element",
		"event.preventDefault()",
		"event.stopPropagation()",
		"window.setTimeout",
		"window.clearTimeout",
		`tooltip.addEventListener("mouseenter", cancelHide)`,
		`tooltip.addEventListener("mouseleave", scheduleHide)`,
		`tooltip.addEventListener("focusin", cancelHide)`,
		`tooltip.addEventListener("focusout", scheduleHide)`,
		"tooltip.contains(target)",
		`hideTooltip(activeToken, { force: true })`,
		".gocire-tooltip__actions",
		".gocire-tooltip__action",
		"tokenHref",
	} {
		assertAstroAssetContains(t, tooltip, want)
	}

	theme := readAstroAssetFile(t, outputDir, "src/scripts/theme.js")
	themeRuntime := layout + "\n" + theme
	for _, want := range []string{
		"localStorage",
		"prefers-color-scheme",
		"data-theme",
		"theme-toggle",
	} {
		assertAstroAssetContains(t, themeRuntime, want)
	}

	navigationRail := readAstroAssetFile(t, outputDir, "src/components/NavigationRail.astro")
	for _, want := range []string{
		"data-toc-rail",
		"data-toc-link",
		"data-toc-target",
		"data-toc-marker-item",
		"navigation-rail__markers",
		"navigation-rail__marker",
		"navigation-rail__tick",
		"navigation-rail__label",
		`aria-label="On this page"`,
		"Jump to",
		"title={item.title}",
		`href={"#" + item.id}`,
	} {
		assertAstroAssetContains(t, navigationRail, want)
	}
	for _, unwanted := range []string{
		`import ListTree from "lucide-astro/ListTree";`,
		"navigation-rail-mobile",
		"data-toc-mobile",
	} {
		assertAstroAssetNotContains(t, navigationRail, unwanted)
	}

	navigationRailScript := readAstroAssetFile(t, outputDir, "src/scripts/navigation-rail.js")
	for _, want := range []string{
		"[data-toc-link]",
		"data-toc-target",
		"aria-current",
		"location",
		"requestAnimationFrame",
		"getBoundingClientRect",
		"scrollMarginTop",
		"updateTargetPositions",
		`style.setProperty("--toc-progress"`,
		"document.documentElement.scrollHeight - window.innerHeight",
		"hashchange",
	} {
		assertAstroAssetContains(t, navigationRailScript, want)
	}
	for _, unwanted := range []string{
		"[data-toc-mobile]",
		"Escape",
		`removeAttribute("open")`,
	} {
		assertAstroAssetNotContains(t, navigationRailScript, unwanted)
	}

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	assertAstroAssetContainsAny(t, globalCSS, []string{
		`html[data-theme="dark"]`,
		`:root[data-theme="dark"]`,
		`[data-theme="dark"]`,
	})
	assertAstroAssetContains(t, globalCSS, ".theme-toggle")
	for _, want := range []string{
		".cire-code-block",
		".cire-code-copy",
		".cire-code-copy[data-copy-state=\"copied\"]",
		".cire-code-copy[data-copy-state=\"failed\"]",
		".cire-prose .cire-code-block > .chroma",
		"padding-right: 3.75rem",
		"position: absolute",
		"top: 9px",
		"right: 9px",
		"place-items: center",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
	for _, want := range []string{
		".code-page--has-toc",
		".navigation-rail__markers",
		".navigation-rail__marker",
		".navigation-rail__tick",
		".navigation-rail__label",
		".navigation-rail__marker[aria-current=\"location\"] .navigation-rail__tick",
		"grid-template-columns: minmax(160px, 220px) minmax(0, 1fr)",
		"top: 50%",
		"right: max(12px, env(safe-area-inset-right))",
		"height: min(72dvh, 520px)",
		"justify-items: end",
		"place-items: center end",
		"top: calc(var(--toc-progress, 0) * 100%)",
		"transform-origin: right center",
		"width: 12px",
		"width: 24px",
		"right: max(8px, env(safe-area-inset-right))",
		"transform: translateY(-50%)",
		".page-content",
		"--anchor-scroll-offset: 86px",
		".code-page--has-toc .page-content",
		"padding-bottom: max(64px, calc(100dvh - var(--anchor-scroll-offset)))",
		".cire-prose :is(h1, h2, h3, h4)[id]",
		"scroll-margin-top: var(--anchor-scroll-offset, 86px)",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
	for _, unwanted := range []string{
		".navigation-rail-mobile",
		".navigation-rail--desktop",
		".navigation-rail__marker--level-1 .navigation-rail__tick",
		".navigation-rail__marker--level-3 .navigation-rail__tick",
		".navigation-rail__marker--level-4 .navigation-rail__tick",
		"navigation-rail-mobile__panel",
		".page-header",
		".page-kicker",
		".page-meta",
		".metadata-tags",
		".metadata-tag",
		".code-page--has-toc .cire-prose",
	} {
		assertAstroAssetNotContains(t, globalCSS, unwanted)
	}
	assertAstroAssetNotContains(t, globalCSS, "min-height: 240px")
	assertAstroAssetNotContains(t, globalCSS, "min-height: 180px")
	for _, want := range []string{
		"--code-keyword",
		"--code-string",
		"--code-function",
		"--code-type",
		"--code-comment",
		"--code-definition",
		"--code-reference-border",
		"--code-inlay-hint-text",
		"--code-inlay-hint-bg",
		"--code-inlay-hint-border",
		"var(--code-keyword)",
		"var(--code-string)",
		"var(--code-function)",
		"var(--code-type)",
		"var(--code-comment)",
		"var(--code-definition)",
		"var(--code-reference-border)",
		"var(--code-inlay-hint-text)",
		"var(--code-inlay-hint-bg)",
		"var(--code-inlay-hint-border)",
		"--tooltip-link",
		"--tooltip-inline-code-bg",
		"--tooltip-code-bg",
		"--tooltip-code-border",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
	for _, want := range []string{
		".gocire-tooltip__content",
		".gocire-tooltip p",
		".gocire-tooltip ul",
		".gocire-tooltip ol",
		".gocire-tooltip a",
		".gocire-tooltip code",
		".gocire-tooltip pre",
		".gocire-tooltip table",
		".gocire-tooltip .chroma",
		"max-height",
		"overflow: auto",
		"overflow-x: auto",
		"position: fixed",
		"100dvh",
		"-webkit-overflow-scrolling: touch",
		"overscroll-behavior: contain",
		"pointer-events: auto",
		"touch-action: manipulation",
		".gocire-tooltip__actions",
		".gocire-tooltip__action",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
	assertAstroAssetNotContains(t, globalCSS, "pointer-events: none")
}

func TestAstroSiteAssetsIncludeKatexRenderingSupport(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Math Docs")

	packageJSON := readAstroAssetFile(t, outputDir, "package.json")
	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if err := json.Unmarshal([]byte(packageJSON), &pkg); err != nil {
		t.Fatalf("package.json is not valid JSON: %v", err)
	}
	if pkg.Dependencies["katex"] == "" {
		t.Fatal("package.json dependencies missing katex")
	}

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	katexImport := `import "katex/dist/katex.min.css";`
	globalImport := `import "../styles/global.css";`
	assertAstroAssetContains(t, layout, katexImport)
	assertAstroAssetContains(t, layout, globalImport)
	if strings.Index(layout, katexImport) > strings.Index(layout, globalImport) {
		t.Fatal("SiteLayout should import KaTeX CSS before global.css so local overrides win")
	}

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	displayMathRule := extractAstroCSSRuleBlock(t, globalCSS, ".cire-prose .katex-display")
	for _, want := range []string{
		"max-width: 100%",
		"margin: 1.25rem 0",
		"overflow-x: auto",
		"overflow-y: hidden",
	} {
		if !strings.Contains(displayMathRule, want) {
			t.Fatalf("display math CSS rule missing %q\nGot:\n%s", want, displayMathRule)
		}
	}
	assertAstroAssetContains(t, globalCSS, ".cire-prose .katex-display > .katex")
}

func TestAstroGlobalCSSIncludesProseMarkdownStyles(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Markdown Docs")

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	codeSurfaceSelector := ":is(.cire-prose > pre, .cire-prose > .chroma, .cire-prose .cire-code-block > pre, .cire-prose .cire-code-block > .chroma, .cire-code, .source-code)"
	for _, selector := range []string{
		".cire-prose a",
		".cire-prose blockquote",
		".cire-prose table",
		".cire-prose th",
		".cire-prose td",
		".cire-prose > :not(pre):not(.chroma):not(.cire-code-block)",
		".cire-prose > pre",
		".cire-prose > .chroma",
		".cire-prose .chroma pre",
		".cire-prose img",
		".cire-prose li + li",
	} {
		extractAstroCSSRuleBlock(t, globalCSS, selector)
	}

	for _, check := range []struct {
		selector string
		want     string
	}{
		{".cire-prose a", "color: var(--accent)"},
		{".cire-prose blockquote", "border-left: 3px solid var(--line)"},
		{".cire-prose", "min-width: 0"},
		{".cire-prose > :not(pre):not(.chroma):not(.cire-code-block)", "max-width: 760px"},
		{".cire-prose table", "overflow-x: auto"},
		{codeSurfaceSelector, "background: var(--code-bg)"},
		{codeSurfaceSelector, "box-shadow: 0 18px 44px var(--code-shadow)"},
		{codeSurfaceSelector, "line-height: 1.7"},
		{".cire-prose > .chroma", "overflow-x: auto"},
		{".cire-prose img", "max-width: 100%"},
	} {
		ruleBlock := extractAstroCSSRuleBlock(t, globalCSS, check.selector)
		if !strings.Contains(ruleBlock, check.want) {
			t.Fatalf("CSS selector %q missing %q\nGot:\n%s", check.selector, check.want, ruleBlock)
		}
	}
}

func TestAstroSiteLayoutUsesGeneratedNavigation(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Navigation Docs")

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	for _, want := range []string{
		`import { navigation } from "../generated/navigation";`,
	} {
		assertAstroAssetContains(t, layout, want)
	}
	assertAstroAssetContainsAny(t, layout, []string{
		`navigation.docs?.firstHref || "/"`,
		`navigation.docs.firstHref || "/"`,
		`navigation.docs?.firstHref ?? "/"`,
		`navigation.docs.firstHref ?? "/"`,
		`navigation.docs?.firstHref ? navigation.docs?.firstHref : "/"`,
		`navigation.docs.firstHref ? navigation.docs.firstHref : "/"`,
	})
	assertAstroAssetContainsAny(t, layout, []string{
		`navigation.blog?.firstHref || "/"`,
		`navigation.blog.firstHref || "/"`,
		`navigation.blog?.firstHref ?? "/"`,
		`navigation.blog.firstHref ?? "/"`,
		`navigation.blog?.firstHref ? navigation.blog?.firstHref : "/"`,
		`navigation.blog.firstHref ? navigation.blog.firstHref : "/"`,
	})
	for _, unwanted := range []string{
		`/#docs`,
		`/#blog`,
	} {
		assertAstroAssetNotContains(t, layout, unwanted)
	}
}

func TestAstroCodePageUsesSidebarComponent(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Sidebar Docs")

	codePage := readAstroAssetFile(t, outputDir, "src/components/CodePage.astro")
	for _, want := range []string{
		`import Sidebar from "./Sidebar.astro";`,
		`import NavigationRail from "./NavigationRail.astro";`,
		"<Sidebar",
		"<NavigationRail",
		"currentPath={currentPath}",
		"sourcePath={sourcePath}",
		"language={language}",
		"toc?: TableOfContentsItem[]",
		"tocItems",
		"hasToc",
		"code-page--has-toc",
	} {
		assertAstroAssetContains(t, codePage, want)
	}
	for _, unwanted := range []string{
		`<header class="page-header">`,
		`<p class="page-kicker">`,
		`<dl class="page-meta">`,
		"pageDate",
		"pageAuthor",
		"pageTags",
		"metadata-tags",
		"Date",
		"Author",
		"Tags",
		"kindLabel",
		`<dt>Language</dt>`,
		`<dt>Path</dt>`,
		`<aside class="page-sidebar" aria-label="Page context">`,
		`<p class="sidebar-label">Kind</p>`,
		`<p class="sidebar-label">Path</p>`,
		`<p class="sidebar-label">Language</p>`,
	} {
		assertAstroAssetNotContains(t, codePage, unwanted)
	}
}

func TestAstroSidebarUsesNavigationSectionsAndSourceMetadata(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Sidebar Docs")

	sidebar := readAstroAssetFile(t, outputDir, "src/components/Sidebar.astro")
	for _, want := range []string{
		`import { navigation } from "../generated/navigation";`,
		`import SidebarBody from "./SidebarBody.astro";`,
		"currentPath?: string",
		"sourcePath?: string",
		"language?: string",
		"kind?: string",
		"sourcePath",
		"language",
		"Docs navigation",
		"Blog navigation",
		"Source context",
		`<div class="sidebar-desktop">`,
		`<details class="sidebar-disclosure">`,
		`<summary class="sidebar-summary">`,
		"sidebar-summary__label",
		"{sidebarLabel}",
		"<SidebarBody",
	} {
		assertAstroAssetContains(t, sidebar, want)
	}
	assertAstroAssetContainsAny(t, sidebar, []string{
		"navigation.docs.items",
		"navigation.docs?.items",
	})
	assertAstroAssetContainsAny(t, sidebar, []string{
		"navigation.blog.items",
		"navigation.blog?.items",
	})
	for _, unwanted := range []string{
		`import SidebarItems from "./SidebarItems.astro";`,
		"open data-sidebar-disclosure",
		"data-sidebar-disclosure",
		"item.date",
		"item.author",
		"item.tags",
	} {
		assertAstroAssetNotContains(t, sidebar, unwanted)
	}
}

func TestAstroSidebarBodyRendersNavigationSectionsAndSourceMetadata(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Sidebar Docs")

	sidebarBody := readAstroAssetFile(t, outputDir, "src/components/SidebarBody.astro")
	for _, want := range []string{
		`import SidebarItems from "./SidebarItems.astro";`,
		"currentPath?: string",
		"sourcePath?: string",
		"language?: string",
		"date?: string",
		"tags?: string[]",
		"author?: string",
		"Docs",
		"Blog",
		"item.date",
		"item.author",
		"item.tags",
		"sidebar-blog-meta",
		"sidebar-blog-author",
		"sidebar-blog-tags",
		"sidebar-blog-date",
		"Source context",
		"sourcePath",
		"language",
		"Path",
		"Language",
		"sidebar-body",
		"<SidebarItems",
		"aria-current",
	} {
		assertAstroAssetContains(t, sidebarBody, want)
	}
}

func TestAstroSidebarItemsRendersRecursiveNavigation(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Sidebar Docs")

	sidebarItems := readAstroAssetFile(t, outputDir, "src/components/SidebarItems.astro")
	for _, want := range []string{
		"currentPath",
		`item.type === "category"`,
		"href={item.href}",
		"aria-current",
		`"page"`,
		"sidebar-link",
		"tags?: string[]",
		"author?: string",
		"sidebar-link__meta",
		"sidebar-link__tags",
		"sidebar-category",
	} {
		assertAstroAssetContains(t, sidebarItems, want)
	}
	assertAstroAssetContainsAny(t, sidebarItems, []string{
		"items:",
		"items?:",
	})
	assertAstroAssetContainsAny(t, sidebarItems, []string{
		"<SidebarItems",
		"<Astro.self",
	})
}

func TestAstroGlobalCSSIncludesSidebarNavigationClasses(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Sidebar Docs")

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	for _, want := range []string{
		".page-sidebar",
		".sidebar-link",
		".sidebar-category",
		".sidebar-blog-date",
		".sidebar-blog-meta",
		".sidebar-blog-author",
		".sidebar-blog-tags",
		".sidebar-link__meta",
		".sidebar-context",
		".sidebar-desktop",
		".sidebar-disclosure",
		".sidebar-summary",
		".sidebar-summary::after",
		".sidebar-summary__label",
		".sidebar-body",
		".sidebar-disclosure:not([open]) > .sidebar-body",
		".sidebar-disclosure[open] > .sidebar-body",
		".sidebar-summary::-webkit-details-marker",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
	for _, want := range []string{
		"position: sticky",
		"position: static",
		"border-bottom: 1px solid var(--line)",
		"max-height: min(55dvh, 360px)",
		"padding: 12px",
		"background: var(--surface)",
		"box-shadow: var(--shadow)",
		"--shadow:",
		"overscroll-behavior: contain",
		"-webkit-overflow-scrolling: touch",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
	assertAstroAssetContainsAny(t, globalCSS, []string{
		`.sidebar-link[aria-current="page"]`,
		".sidebar-link.is-active",
		".sidebar-link--active",
		".sidebar-active",
	})
}

func TestAstroGlobalCSSIncludesThemeAwareCodeHighlighting(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Code Theme Docs")

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	lightThemeBlock := extractAstroCSSRuleBlock(t, globalCSS, `html[data-theme="light"]`)
	darkThemeBlock := extractAstroCSSRuleBlock(t, globalCSS, `html[data-theme="dark"]`)

	codeVariables := []string{
		"--code-bg",
		"--code-text",
		"--code-muted",
		"--code-border",
		"--code-shadow",
		"--code-keyword",
		"--code-string",
		"--code-function",
		"--code-type",
		"--code-comment",
		"--code-definition",
		"--code-variable",
		"--code-constant",
		"--code-number",
		"--code-operator",
		"--code-punctuation",
		"--code-reference-border",
		"--code-inlay-hint-text",
		"--code-inlay-hint-bg",
		"--code-inlay-hint-border",
	}
	for _, variable := range codeVariables {
		extractAstroCSSVariableValue(t, lightThemeBlock, variable)
		extractAstroCSSVariableValue(t, darkThemeBlock, variable)
	}

	lightCodeBG := extractAstroCSSVariableValue(t, lightThemeBlock, "--code-bg")
	darkCodeBG := extractAstroCSSVariableValue(t, darkThemeBlock, "--code-bg")
	if lightCodeBG == darkCodeBG {
		t.Fatalf("expected light and dark --code-bg values to differ, both were %q", lightCodeBG)
	}

	for _, want := range []string{
		"var(--code-variable)",
		"var(--code-constant)",
		"var(--code-number)",
		"var(--code-operator)",
		"var(--code-punctuation)",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}

	for _, mapping := range []struct {
		selector string
		variable string
	}{
		{`.cire .function\.method`, "--code-function"},
		{`.cire .type\.builtin`, "--code-type"},
		{`.cire .variable\.parameter`, "--code-variable"},
		{`.cire .constant\.builtin`, "--code-constant"},
		{`.cire .punctuation\.delimiter`, "--code-punctuation"},
	} {
		assertAstroCSSSelectorUsesVariable(t, globalCSS, mapping.selector, mapping.variable)
	}

	for _, mapping := range []struct {
		selector string
		variable string
	}{
		{":is(.cire-prose, .gocire-tooltip) .chroma .m", "--code-number"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .mi", "--code-number"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .mf", "--code-number"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .nv", "--code-variable"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .n", "--code-variable"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .o", "--code-operator"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .p", "--code-punctuation"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .k", "--code-keyword"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .s", "--code-string"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .nf", "--code-function"},
		{":is(.cire-prose, .gocire-tooltip) .chroma .c", "--code-comment"},
	} {
		assertAstroCSSSelectorUsesVariable(t, globalCSS, mapping.selector, mapping.variable)
	}
}

func TestAstroGlobalCSSIncludesInlayHintStyles(t *testing.T) {
	outputDir := writeAstroAssetsForTest(t, "Inlay Docs")

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	ruleBlock := extractAstroCSSRuleBlock(t, globalCSS, ".cire .inlay-hint")
	for _, want := range []string{
		"var(--code-inlay-hint-text)",
		"var(--code-inlay-hint-bg)",
		"var(--code-inlay-hint-border)",
		"user-select: none",
		"white-space: pre",
	} {
		if !strings.Contains(ruleBlock, want) {
			t.Fatalf("inlay hint CSS rule missing %q\nGot:\n%s", want, ruleBlock)
		}
	}

	lightThemeBlock := extractAstroCSSRuleBlock(t, globalCSS, `html[data-theme="light"]`)
	darkThemeBlock := extractAstroCSSRuleBlock(t, globalCSS, `html[data-theme="dark"]`)
	if extractAstroCSSVariableValue(t, lightThemeBlock, "--code-inlay-hint-bg") ==
		extractAstroCSSVariableValue(t, darkThemeBlock, "--code-inlay-hint-bg") {
		t.Fatal("expected light and dark inlay hint backgrounds to differ")
	}
}

func TestWriteAstroSiteAssetsRepeatedCallOverwritesStable(t *testing.T) {
	outputDir := t.TempDir()

	if err := WriteAstroSiteAssets(outputDir, "Stable Docs"); err != nil {
		t.Fatalf("first WriteAstroSiteAssets returned error: %v", err)
	}
	firstSnapshot := readAstroAssetSnapshot(t, outputDir)

	packagePath := filepath.Join(outputDir, "package.json")
	if err := os.WriteFile(packagePath, []byte("{}\n"), 0o644); err != nil {
		t.Fatalf("corrupt package.json: %v", err)
	}
	layoutPath := filepath.Join(outputDir, "src", "layouts", "SiteLayout.astro")
	if err := os.WriteFile(layoutPath, []byte("---\nconst broken = true;\n---\n"), 0o644); err != nil {
		t.Fatalf("corrupt SiteLayout.astro: %v", err)
	}

	if err := WriteAstroSiteAssets(outputDir, "Stable Docs"); err != nil {
		t.Fatalf("second WriteAstroSiteAssets returned error: %v", err)
	}
	secondSnapshot := readAstroAssetSnapshot(t, outputDir)
	assertAstroAssetSnapshotsEqual(t, secondSnapshot, firstSnapshot)

	if err := WriteAstroSiteAssets(outputDir, "Stable Docs"); err != nil {
		t.Fatalf("third WriteAstroSiteAssets returned error: %v", err)
	}
	thirdSnapshot := readAstroAssetSnapshot(t, outputDir)
	assertAstroAssetSnapshotsEqual(t, thirdSnapshot, firstSnapshot)
}

func TestAstroSiteLayoutQuotesFallbackSiteTitleAsStringLiteral(t *testing.T) {
	siteTitle := "Docs \"Quotes\" <unsafe>\nsecond line and backslash \\"
	outputDir := writeAstroAssetsForTest(t, siteTitle)

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	fallbackSiteTitle := extractFallbackSiteTitleLiteral(t, layout)
	unquoted, err := strconv.Unquote(fallbackSiteTitle)
	if err != nil {
		t.Fatalf("fallbackSiteTitle is not a valid quoted string literal: %v\nGot: %s", err, fallbackSiteTitle)
	}
	if unquoted != siteTitle {
		t.Fatalf("fallbackSiteTitle = %q, want %q", unquoted, siteTitle)
	}
}

func TestAstroSiteAssetsUseTemplateDirOverrides(t *testing.T) {
	outputDir := t.TempDir()
	templateDir := t.TempDir()
	writeAstroTemplateOverrideFile(t, templateDir, "src/styles/global.css", "/* custom global css */\nbody { color: rebeccapurple; }\n")

	if err := (AstroSiteAssets{
		OutputDir:   outputDir,
		SiteTitle:   "Custom Docs",
		TemplateDir: templateDir,
	}).Write(); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	if globalCSS != "/* custom global css */\nbody { color: rebeccapurple; }\n" {
		t.Fatalf("global.css = %q, want custom override", globalCSS)
	}

	themeJS := readAstroAssetFile(t, outputDir, "src/scripts/theme.js")
	assertAstroAssetContains(t, themeJS, "gocire-theme")
}

func TestAstroSiteAssetsRenderTemplateDirLayoutOverride(t *testing.T) {
	outputDir := t.TempDir()
	templateDir := t.TempDir()
	writeAstroTemplateOverrideFile(t, templateDir, "src/layouts/SiteLayout.astro.tmpl", `---
const fallbackSiteTitle = {{ .FallbackSiteTitle }};
---

<main>{fallbackSiteTitle}</main>
`)

	siteTitle := "Custom \"Layout\""
	if err := (AstroSiteAssets{
		OutputDir:   outputDir,
		SiteTitle:   siteTitle,
		TemplateDir: templateDir,
	}).Write(); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	fallbackSiteTitle := extractFallbackSiteTitleLiteral(t, layout)
	unquoted, err := strconv.Unquote(fallbackSiteTitle)
	if err != nil {
		t.Fatalf("fallbackSiteTitle is not a valid quoted string literal: %v\nGot: %s", err, fallbackSiteTitle)
	}
	if unquoted != siteTitle {
		t.Fatalf("fallbackSiteTitle = %q, want %q", unquoted, siteTitle)
	}
	if _, err := os.Stat(filepath.Join(outputDir, "src", "layouts", "SiteLayout.astro.tmpl")); !os.IsNotExist(err) {
		t.Fatalf("SiteLayout.astro.tmpl output err = %v, want not exist", err)
	}
}

func TestAstroSiteAssetsRejectMissingTemplateDir(t *testing.T) {
	err := (AstroSiteAssets{
		OutputDir:   t.TempDir(),
		SiteTitle:   "Docs",
		TemplateDir: filepath.Join(t.TempDir(), "missing"),
	}).Write()
	if err == nil {
		t.Fatal("Write returned nil error for missing templateDir")
	}
	if !strings.Contains(err.Error(), "Astro template directory") {
		t.Fatalf("error = %q, want template directory context", err.Error())
	}
}

func TestAstroSiteAssetsBuildSmoke(t *testing.T) {
	if os.Getenv("GOCIRE_ASTRO_BUILD_TEST") != "1" {
		t.Skip("set GOCIRE_ASTRO_BUILD_TEST=1 to run the Astro build smoke test")
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skipf("pnpm is not available: %v", err)
	}

	outputDir := writeAstroAssetsForTest(t, "Smoke Docs")
	writeAstroBuildSmokeFile(t, outputDir, "src/generated/navigation.ts", `export { navigation } from "./site-data";
`)
	writeAstroBuildSmokeFile(t, outputDir, "src/generated/site-data.ts", `export const siteData = {
  site: {
    title: "Smoke Docs",
    description: "Smoke Docs description",
    url: "https://example.com/docs",
    trailingSlash: "always"
  },
  pages: [
    {
      routeParam: "blog/second",
      title: "Second Blog",
      href: "/blog/second/",
      module: "../generated/pages/blog/second.astro",
      kind: "blog",
      sourcePath: "content/blog/second.md",
      language: "markdown",
      date: "2026-06-30",
      author: "Ada Lovelace",
      tags: ["astro", "rss"]
    },
    {
      routeParam: "blog/first",
      title: "First Blog",
      href: "/blog/first/",
      module: "../generated/pages/blog/first.astro",
      kind: "blog",
      sourcePath: "content/blog/first.md",
      language: "markdown",
      date: "2026-06-29",
      author: "Grace Hopper",
      tags: ["go"]
    },
    {
      routeParam: "guides/setup",
      title: "Setup Guide",
      href: "/guides/setup/",
      module: "../generated/pages/guides/setup.astro",
      kind: "docs",
      sourcePath: "docs/setup.md",
      language: "markdown",
      date: "",
      author: "",
      tags: []
    },
    {
      routeParam: "_source/main.go.html",
      title: "main.go",
      href: "/_source/main.go.html/",
      module: "../generated/pages/_source/main.go.html.astro",
      kind: "source",
      sourcePath: "main.go",
      language: "go",
      date: "",
      author: "",
      tags: []
    }
  ],
  navigation: {
    docs: { firstHref: "/", items: [] },
    blog: { firstHref: "/", items: [] }
  }
} as const;

export const pages = siteData.pages;
export const navigation = siteData.navigation;
`)
	writeAstroBuildSmokeFile(t, outputDir, "src/pages/index.astro", `---
import SiteLayout from "../layouts/SiteLayout.astro";
---

<SiteLayout title="Smoke Docs">
  <main class="page-shell">
    <h1>Smoke Docs</h1>
  </main>
</SiteLayout>
`)

	runAstroBuildSmokeInstall(t, outputDir)
	runAstroBuildSmokeCommand(t, outputDir, "pnpm", "build")

	rssXML := readAstroBuildSmokeDistFile(t, outputDir, "rss.xml")
	sitemapXML := readAstroBuildSmokeDistFile(t, outputDir, "sitemap.xml")
	blogURLs := []string{
		"https://example.com/docs/blog/second/",
		"https://example.com/docs/blog/first/",
	}
	nonBlogURLs := []string{
		"https://example.com/docs/guides/setup/",
		"https://example.com/docs/_source/main.go.html/",
	}
	for _, want := range blogURLs {
		assertAstroAssetContains(t, rssXML, want)
		assertAstroAssetContains(t, sitemapXML, want)
	}
	for _, want := range nonBlogURLs {
		assertAstroAssetNotContains(t, rssXML, want)
		assertAstroAssetContains(t, sitemapXML, want)
	}
	for _, unwanted := range []string{
		"https://example.com/blog/second/",
		"https://example.com/blog/first/",
		"https://example.com/guides/setup/",
		"https://example.com/_source/main.go.html/",
	} {
		assertAstroAssetNotContains(t, rssXML, unwanted)
		assertAstroAssetNotContains(t, sitemapXML, unwanted)
	}
}

func writeAstroAssetsForTest(t *testing.T, siteTitle string) string {
	t.Helper()

	outputDir := t.TempDir()
	if err := WriteAstroSiteAssets(outputDir, siteTitle); err != nil {
		t.Fatalf("WriteAstroSiteAssets returned error: %v", err)
	}
	return outputDir
}

func writeAstroTemplateOverrideFile(t *testing.T, templateDir, slashRelPath, contents string) {
	t.Helper()

	outPath := filepath.Join(templateDir, filepath.FromSlash(slashRelPath))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(outPath), err)
	}
	if err := os.WriteFile(outPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", slashRelPath, err)
	}
}

func extractFallbackSiteTitleLiteral(t *testing.T, layout string) string {
	t.Helper()

	re := regexp.MustCompile(`(?m)^\s*const\s+fallbackSiteTitle\s*=\s*(.+);\s*$`)
	matches := re.FindStringSubmatch(layout)
	if matches == nil {
		t.Fatal("SiteLayout.astro does not define const fallbackSiteTitle")
	}
	return strings.TrimSpace(matches[1])
}

func writeAstroBuildSmokeFile(t *testing.T, outputDir, slashRelPath, contents string) {
	t.Helper()

	outPath := filepath.Join(outputDir, filepath.FromSlash(slashRelPath))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(outPath), err)
	}
	if err := os.WriteFile(outPath, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", slashRelPath, err)
	}
}

func readAstroBuildSmokeDistFile(t *testing.T, outputDir, slashRelPath string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(outputDir, "dist", filepath.FromSlash(slashRelPath)))
	if err != nil {
		t.Fatalf("ReadFile(dist/%s): %v", slashRelPath, err)
	}
	return string(contents)
}

func runAstroBuildSmokeCommand(t *testing.T, workDir, name string, args ...string) {
	t.Helper()

	output := runAstroBuildSmokeCommandOutput(t, workDir, name, args...)
	if output.err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), output.err, output.contents)
	}
}

func runAstroBuildSmokeInstall(t *testing.T, workDir string) {
	t.Helper()

	output := runAstroBuildSmokeCommandOutput(t, workDir, "pnpm", "install")
	if output.err == nil {
		return
	}
	if !strings.Contains(string(output.contents), "ERR_PNPM_IGNORED_BUILDS") {
		t.Fatalf("pnpm install failed: %v\n%s", output.err, output.contents)
	}

	runAstroBuildSmokeCommand(t, workDir, "pnpm", "approve-builds", "--all")
	runAstroBuildSmokeCommand(t, workDir, "pnpm", "install")
}

type astroBuildSmokeCommandOutput struct {
	contents []byte
	err      error
}

func runAstroBuildSmokeCommandOutput(t *testing.T, workDir, name string, args ...string) astroBuildSmokeCommandOutput {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("%s %s timed out after 2 minutes\n%s", name, strings.Join(args, " "), output)
	}
	return astroBuildSmokeCommandOutput{contents: output, err: err}
}

func readAstroAssetSnapshot(t *testing.T, outputDir string) map[string]string {
	t.Helper()

	snapshot := make(map[string]string, len(expectedAstroAssetFiles))
	for _, relPath := range expectedAstroAssetFiles {
		snapshot[relPath] = readAstroAssetFile(t, outputDir, relPath)
	}
	return snapshot
}

func readAstroAssetFile(t *testing.T, outputDir, relPath string) string {
	t.Helper()

	contents, err := os.ReadFile(filepath.Join(outputDir, filepath.FromSlash(relPath)))
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", relPath, err)
	}
	return string(contents)
}

func assertAstroAssetContains(t *testing.T, contents, want string) {
	t.Helper()

	if !strings.Contains(contents, want) {
		t.Fatalf("asset does not contain %q", want)
	}
}

func assertAstroAssetNotContains(t *testing.T, contents, unwanted string) {
	t.Helper()

	if strings.Contains(contents, unwanted) {
		t.Fatalf("asset unexpectedly contains %q", unwanted)
	}
}

func assertAstroAssetContainsAny(t *testing.T, contents string, wants []string) {
	t.Helper()

	for _, want := range wants {
		if strings.Contains(contents, want) {
			return
		}
	}
	t.Fatalf("asset does not contain any of %q", wants)
}

func assertAstroCSSSelectorUsesVariable(t *testing.T, contents, selector, variable string) {
	t.Helper()

	ruleBlock := extractAstroCSSRuleBlock(t, contents, selector)
	want := "var(" + variable + ")"
	if !strings.Contains(ruleBlock, want) {
		t.Fatalf("CSS selector %q does not use %q", selector, want)
	}
}

func extractAstroCSSRuleBlock(t *testing.T, contents, selector string) string {
	t.Helper()

	offset := 0
	for offset < len(contents) {
		openIndex := strings.Index(contents[offset:], "{")
		if openIndex == -1 {
			break
		}
		ruleStart := offset
		ruleOpen := offset + openIndex
		closeIndex := strings.Index(contents[ruleOpen+1:], "}")
		if closeIndex == -1 {
			t.Fatalf("CSS selector list %q is missing a closing rule block", strings.TrimSpace(contents[ruleStart:ruleOpen]))
		}
		ruleClose := ruleOpen + 1 + closeIndex
		for _, candidate := range splitAstroCSSSelectorList(contents[ruleStart:ruleOpen]) {
			if strings.TrimSpace(candidate) == selector {
				return contents[ruleOpen+1 : ruleClose]
			}
		}
		offset = ruleClose + 1
	}
	t.Fatalf("CSS asset does not contain selector %q", selector)
	return ""
}

func splitAstroCSSSelectorList(selectors string) []string {
	var result []string
	start := 0
	depth := 0
	for i, r := range selectors {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				result = append(result, selectors[start:i])
				start = i + 1
			}
		}
	}
	return append(result, selectors[start:])
}

func extractAstroCSSVariableValue(t *testing.T, contents, variable string) string {
	t.Helper()

	prefix := variable + ":"
	for _, line := range strings.Split(contents, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			value := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, prefix), ";"))
			if value == "" {
				t.Fatalf("CSS variable %q has an empty value", variable)
			}
			return value
		}
	}
	t.Fatalf("CSS block does not define variable %q", variable)
	return ""
}

func assertAstroStringSetsEqual(t *testing.T, got, want []string) {
	t.Helper()

	gotSorted := append([]string(nil), got...)
	wantSorted := append([]string(nil), want...)
	sort.Strings(gotSorted)
	sort.Strings(wantSorted)

	if len(gotSorted) != len(wantSorted) {
		t.Fatalf("len(got) = %d, want %d\ngot:  %q\nwant: %q", len(gotSorted), len(wantSorted), gotSorted, wantSorted)
	}
	for i := range wantSorted {
		if gotSorted[i] != wantSorted[i] {
			t.Fatalf("got[%d] = %q, want %q\ngot:  %q\nwant: %q", i, gotSorted[i], wantSorted[i], gotSorted, wantSorted)
		}
	}
}

func assertAstroAssetSnapshotsEqual(t *testing.T, got, want map[string]string) {
	t.Helper()

	for _, relPath := range expectedAstroAssetFiles {
		if got[relPath] != want[relPath] {
			t.Fatalf("asset %q changed after repeated write", relPath)
		}
	}
}
