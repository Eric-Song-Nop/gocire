package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var expectedAstroAssetFiles = []string{
	"package.json",
	"astro.config.mjs",
	"src/layouts/SiteLayout.astro",
	"src/components/CodePage.astro",
	"src/components/Sidebar.astro",
	"src/components/SidebarItems.astro",
	"src/styles/global.css",
	"src/scripts/tooltip.js",
	"src/scripts/theme.js",
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
	if pkg.Dependencies["@floating-ui/dom"] == "" {
		t.Fatal("package.json dependencies missing @floating-ui/dom")
	}
	if pkg.Dependencies["lucide-astro"] == "" {
		t.Fatal("package.json dependencies missing lucide-astro")
	}

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	assertAstroAssetContains(t, layout, "Example Docs")
	for _, want := range []string{
		`import Moon from "lucide-astro/Moon";`,
		`import Sun from "lucide-astro/Sun";`,
		"theme-toggle",
		"data-theme",
		"../scripts/theme.js",
	} {
		assertAstroAssetContains(t, layout, want)
	}

	codePage := readAstroAssetFile(t, outputDir, "src/components/CodePage.astro")
	for _, want := range []string{
		"title: string",
		"kind?: string",
		"language?: string",
		"sourcePath?: string",
		"<slot />",
	} {
		assertAstroAssetContains(t, codePage, want)
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
		`setAttribute("role", "tooltip")`,
		"aria-describedby",
		"tabindex",
		"Escape",
		"autoUpdate",
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

	globalCSS := readAstroAssetFile(t, outputDir, "src/styles/global.css")
	assertAstroAssetContainsAny(t, globalCSS, []string{
		`html[data-theme="dark"]`,
		`:root[data-theme="dark"]`,
		`[data-theme="dark"]`,
	})
	assertAstroAssetContains(t, globalCSS, ".theme-toggle")
	for _, want := range []string{
		"--code-keyword",
		"--code-string",
		"--code-function",
		"--code-type",
		"--code-comment",
		"--code-definition",
		"--code-reference-border",
		"var(--code-keyword)",
		"var(--code-string)",
		"var(--code-function)",
		"var(--code-type)",
		"var(--code-comment)",
		"var(--code-definition)",
		"var(--code-reference-border)",
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
		"overflow-x: auto",
	} {
		assertAstroAssetContains(t, globalCSS, want)
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
		"<Sidebar",
		"currentPath={currentPath}",
		"sourcePath={sourcePath}",
		"language={language}",
	} {
		assertAstroAssetContains(t, codePage, want)
	}
	assertAstroAssetContainsAny(t, codePage, []string{
		"kind={kind}",
		"kind={kindLabel}",
	})
	for _, unwanted := range []string{
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
		`import SidebarItems from "./SidebarItems.astro";`,
		"currentPath?: string",
		"sourcePath?: string",
		"language?: string",
		"kind?: string",
		"Docs",
		"Blog",
		"item.date",
		"sidebar-blog-date",
		"Source",
		"sourcePath",
		"language",
		"Path",
		"Language",
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
		".sidebar-context",
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

func writeAstroAssetsForTest(t *testing.T, siteTitle string) string {
	t.Helper()

	outputDir := t.TempDir()
	if err := WriteAstroSiteAssets(outputDir, siteTitle); err != nil {
		t.Fatalf("WriteAstroSiteAssets returned error: %v", err)
	}
	return outputDir
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

func assertAstroAssetSnapshotsEqual(t *testing.T, got, want map[string]string) {
	t.Helper()

	for _, relPath := range expectedAstroAssetFiles {
		if got[relPath] != want[relPath] {
			t.Fatalf("asset %q changed after repeated write", relPath)
		}
	}
}
