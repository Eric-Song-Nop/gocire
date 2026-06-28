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

	layout := readAstroAssetFile(t, outputDir, "src/layouts/SiteLayout.astro")
	assertAstroAssetContains(t, layout, "Example Docs")
	for _, want := range []string{
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
	assertAstroAssetContains(t, tooltip, "@floating-ui/dom")
	assertAstroAssetContains(t, tooltip, "[data-hover]")
	assertAstroAssetContains(t, tooltip, "TextDecoder")

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
