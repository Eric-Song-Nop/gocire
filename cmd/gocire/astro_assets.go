package main

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

const defaultAstroSiteTitle = "gocire docs"

//go:embed astro_template
var astroTemplateFS embed.FS

type AstroSiteAssets struct {
	OutputDir   string
	SiteTitle   string
	TemplateDir string
}

type astroSiteAssetTemplate struct {
	templatePath     string
	outputPath       string
	renderSiteLayout bool
}

var astroSiteAssetTemplates = []astroSiteAssetTemplate{
	{templatePath: "astro_template/package.json", outputPath: "package.json"},
	{templatePath: "astro_template/astro.config.mjs", outputPath: "astro.config.mjs"},
	{templatePath: "astro_template/src/layouts/SiteLayout.astro.tmpl", outputPath: "src/layouts/SiteLayout.astro", renderSiteLayout: true},
	{templatePath: "astro_template/src/components/CodePage.astro", outputPath: "src/components/CodePage.astro"},
	{templatePath: "astro_template/src/components/NavigationRail.astro", outputPath: "src/components/NavigationRail.astro"},
	{templatePath: "astro_template/src/components/Sidebar.astro", outputPath: "src/components/Sidebar.astro"},
	{templatePath: "astro_template/src/components/SidebarItems.astro", outputPath: "src/components/SidebarItems.astro"},
	{templatePath: "astro_template/src/pages/rss.xml.ts", outputPath: "src/pages/rss.xml.ts"},
	{templatePath: "astro_template/src/pages/sitemap.xml.ts", outputPath: "src/pages/sitemap.xml.ts"},
	{templatePath: "astro_template/src/styles/global.css", outputPath: "src/styles/global.css"},
	{templatePath: "astro_template/src/scripts/navigation-rail.js", outputPath: "src/scripts/navigation-rail.js"},
	{templatePath: "astro_template/src/scripts/sidebar.js", outputPath: "src/scripts/sidebar.js"},
	{templatePath: "astro_template/src/scripts/theme.js", outputPath: "src/scripts/theme.js"},
	{templatePath: "astro_template/src/scripts/tooltip.js", outputPath: "src/scripts/tooltip.js"},
}

func astroTemplateOutputFiles() []string {
	files := make([]string, 0, len(astroSiteAssetTemplates))
	for _, asset := range astroSiteAssetTemplates {
		files = append(files, asset.outputPath)
	}
	return files
}

func WriteAstroSiteAssets(outputDir string, siteTitle string) error {
	return AstroSiteAssets{
		OutputDir: outputDir,
		SiteTitle: siteTitle,
	}.Write()
}

func (a AstroSiteAssets) Write() error {
	if strings.TrimSpace(a.OutputDir) == "" {
		return fmt.Errorf("output directory is required")
	}

	assets, err := a.files()
	if err != nil {
		return err
	}

	for relPath, contents := range assets {
		if err := writeAstroSiteAsset(a.OutputDir, relPath, contents); err != nil {
			return err
		}
	}
	return nil
}

func (a AstroSiteAssets) files() (map[string]string, error) {
	siteTitle := normalizedAstroSiteTitle(a.SiteTitle)
	templateDir, err := normalizedAstroTemplateDir(a.TemplateDir)
	if err != nil {
		return nil, err
	}
	assets := make(map[string]string, len(astroSiteAssetTemplates))

	for _, asset := range astroSiteAssetTemplates {
		contents, err := a.readAstroSiteAssetTemplate(asset, templateDir)
		if err != nil {
			return nil, err
		}

		if asset.renderSiteLayout {
			contents, err = renderAstroSiteLayoutTemplate(asset.templatePath, contents, siteTitle)
			if err != nil {
				return nil, err
			}
		}

		assets[asset.outputPath] = contents
	}

	return assets, nil
}

func (a AstroSiteAssets) readAstroSiteAssetTemplate(asset astroSiteAssetTemplate, templateDir string) (string, error) {
	if templateDir != "" {
		contents, ok, err := readAstroTemplateDirAsset(templateDir, asset)
		if err != nil {
			return "", err
		}
		if ok {
			return contents, nil
		}
	}
	return readEmbeddedAstroSiteAssetTemplate(asset.templatePath)
}

func readAstroTemplateDirAsset(templateDir string, asset astroSiteAssetTemplate) (contents string, ok bool, err error) {
	relPath := asset.templateRelPath()
	templatePath := filepath.Join(templateDir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(templatePath)
	if err == nil {
		return string(data), true, nil
	}
	if os.IsNotExist(err) {
		return "", false, nil
	}
	return "", false, fmt.Errorf("read Astro asset template override %s: %w", templatePath, err)
}

func readEmbeddedAstroSiteAssetTemplate(templatePath string) (string, error) {
	contents, err := astroTemplateFS.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("read Astro asset template %s: %w", templatePath, err)
	}
	return string(contents), nil
}

func renderAstroSiteLayoutTemplate(templatePath, contents, siteTitle string) (string, error) {
	tmpl, err := template.New(filepath.Base(templatePath)).Option("missingkey=error").Parse(contents)
	if err != nil {
		return "", fmt.Errorf("parse Astro asset template %s: %w", templatePath, err)
	}

	var rendered bytes.Buffer
	data := struct {
		FallbackSiteTitle string
	}{
		FallbackSiteTitle: strconv.Quote(siteTitle),
	}
	if err := tmpl.Execute(&rendered, data); err != nil {
		return "", fmt.Errorf("render Astro asset template %s: %w", templatePath, err)
	}
	return rendered.String(), nil
}

func (a astroSiteAssetTemplate) templateRelPath() string {
	return strings.TrimPrefix(a.templatePath, "astro_template/")
}

func writeAstroSiteAsset(outputDir, slashRelPath, contents string) error {
	outPath := filepath.Join(outputDir, filepath.FromSlash(slashRelPath))
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return fmt.Errorf("create Astro asset directory %s: %w", filepath.Dir(outPath), err)
	}
	if err := os.WriteFile(outPath, []byte(contents), 0o644); err != nil {
		return fmt.Errorf("write Astro asset %s: %w", slashRelPath, err)
	}
	return nil
}

func normalizedAstroSiteTitle(siteTitle string) string {
	if title := strings.TrimSpace(siteTitle); title != "" {
		return title
	}
	return defaultAstroSiteTitle
}

func normalizedAstroTemplateDir(templateDir string) (string, error) {
	templateDir = strings.TrimSpace(templateDir)
	if templateDir == "" {
		return "", nil
	}
	info, err := os.Stat(templateDir)
	if err != nil {
		return "", fmt.Errorf("read Astro template directory %s: %w", templateDir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("Astro template directory %s is not a directory", templateDir)
	}
	return templateDir, nil
}
