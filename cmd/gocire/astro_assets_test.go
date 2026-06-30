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
	if pkg.Dependencies["katex"] == "" {
		t.Fatal("package.json dependencies missing katex")
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
		`import "katex/dist/katex.min.css";`,
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
		"date?: string",
		"tags?: string[]",
		"author?: string",
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
		"overflow-x: auto",
	} {
		assertAstroAssetContains(t, globalCSS, want)
	}
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
	for _, selector := range []string{
		".cire-prose a",
		".cire-prose blockquote",
		".cire-prose table",
		".cire-prose th",
		".cire-prose td",
		".cire-prose pre",
		".cire-prose .chroma",
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
		{".cire-prose table", "overflow-x: auto"},
		{".cire-prose pre", "background: var(--code-bg)"},
		{".cire-prose .chroma", "overflow-x: auto"},
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
		"<Sidebar",
		"currentPath={currentPath}",
		"sourcePath={sourcePath}",
		"language={language}",
		"pageDate",
		"pageAuthor",
		"pageTags",
		"metadata-tags",
		"Date",
		"Author",
		"Tags",
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
		".metadata-tags",
		".metadata-tag",
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
		{".gocire-tooltip .chroma .m", "--code-number"},
		{".gocire-tooltip .chroma .mi", "--code-number"},
		{".gocire-tooltip .chroma .mf", "--code-number"},
		{".gocire-tooltip .chroma .nv", "--code-variable"},
		{".gocire-tooltip .chroma .n", "--code-variable"},
		{".gocire-tooltip .chroma .o", "--code-operator"},
		{".gocire-tooltip .chroma .p", "--code-punctuation"},
		{".gocire-tooltip .chroma .k", "--code-keyword"},
		{".gocire-tooltip .chroma .s", "--code-string"},
		{".gocire-tooltip .chroma .nf", "--code-function"},
		{".gocire-tooltip .chroma .c", "--code-comment"},
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
		for _, candidate := range strings.Split(contents[ruleStart:ruleOpen], ",") {
			if strings.TrimSpace(candidate) == selector {
				return contents[ruleOpen+1 : ruleClose]
			}
		}
		offset = ruleClose + 1
	}
	t.Fatalf("CSS asset does not contain selector %q", selector)
	return ""
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

func assertAstroAssetSnapshotsEqual(t *testing.T, got, want map[string]string) {
	t.Helper()

	for _, relPath := range expectedAstroAssetFiles {
		if got[relPath] != want[relPath] {
			t.Fatalf("asset %q changed after repeated write", relPath)
		}
	}
}
