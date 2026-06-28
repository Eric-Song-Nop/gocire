package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const defaultAstroSiteTitle = "gocire docs"

type AstroSiteAssets struct {
	OutputDir string
	SiteTitle string
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
	packageJSON, err := astroPackageJSON()
	if err != nil {
		return nil, err
	}

	siteTitle := normalizedAstroSiteTitle(a.SiteTitle)
	return map[string]string{
		"package.json":                  packageJSON,
		"astro.config.mjs":              astroConfigMJS(),
		"src/layouts/SiteLayout.astro":  astroSiteLayout(siteTitle),
		"src/components/CodePage.astro": astroCodePage(),
		"src/styles/global.css":         astroGlobalCSS(),
		"src/scripts/theme.js":          astroThemeJS(),
		"src/scripts/tooltip.js":        astroTooltipJS(),
	}, nil
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

func astroPackageJSON() (string, error) {
	type packageJSON struct {
		Name         string            `json:"name"`
		Private      bool              `json:"private"`
		Type         string            `json:"type"`
		Scripts      map[string]string `json:"scripts"`
		Dependencies map[string]string `json:"dependencies"`
	}

	contents, err := json.MarshalIndent(packageJSON{
		Name:    "gocire-docsite",
		Private: true,
		Type:    "module",
		Scripts: map[string]string{
			"dev":     "astro dev",
			"build":   "astro build",
			"preview": "astro preview",
		},
		Dependencies: map[string]string{
			"@floating-ui/dom": "latest",
			"astro":            "latest",
		},
	}, "", "  ")
	if err != nil {
		return "", fmt.Errorf("build Astro package.json: %w", err)
	}
	return string(contents) + "\n", nil
}

func astroConfigMJS() string {
	return `import { defineConfig } from "astro/config";

export default defineConfig({});
`
}

func astroSiteLayout(siteTitle string) string {
	return fmt.Sprintf(`---
import "../styles/global.css";

interface Props {
  title?: string;
  siteTitle?: string;
  description?: string;
}

const fallbackSiteTitle = %s;
const {
  title = fallbackSiteTitle,
  siteTitle = fallbackSiteTitle,
  description = "Generated source documentation.",
} = Astro.props;
const pageTitle = title === siteTitle ? siteTitle : title + " | " + siteTitle;
---

<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <meta name="description" content={description} />
    <title>{pageTitle}</title>
    <script is:inline>
      (() => {
        const storageKey = "gocire-theme";
        const validThemes = new Set(["light", "dark"]);

        const readStoredTheme = () => {
          try {
            return localStorage.getItem(storageKey);
          } catch {
            return null;
          }
        };

        const preferredTheme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
        const storedTheme = readStoredTheme();
        document.documentElement.setAttribute("data-theme", validThemes.has(storedTheme) ? storedTheme : preferredTheme);
      })();
    </script>
  </head>
  <body>
    <header class="site-header">
      <div class="site-header__inner">
        <a class="site-brand" href="/">{siteTitle}</a>
        <div class="site-actions">
          <nav class="site-nav" aria-label="Main navigation">
            <a href="/">Home</a>
            <a href="/#docs">Docs</a>
            <a href="/#blog">Blog</a>
          </nav>
          <button class="theme-toggle" type="button" data-theme-toggle aria-label="Toggle color theme" title="Toggle color theme">
            <span class="theme-toggle__icon theme-toggle__icon--sun" aria-hidden="true"></span>
            <span class="theme-toggle__icon theme-toggle__icon--moon" aria-hidden="true"></span>
          </button>
        </div>
      </div>
    </header>
    <slot />
    <footer class="site-footer">
      <span>{siteTitle}</span>
    </footer>
    <script>
      import "../scripts/theme.js";
      import "../scripts/tooltip.js";
    </script>
  </body>
</html>
`, strconv.Quote(siteTitle))
}

func astroCodePage() string {
	return `---
import SiteLayout from "../layouts/SiteLayout.astro";

interface Props {
  title: string;
  kind?: string;
  language?: string;
  sourcePath?: string;
  renderMode?: string;
}

const {
  title,
  kind = "Source",
  language,
  sourcePath,
  renderMode = "source",
} = Astro.props;
const pageClass = "site-shell code-page code-page--" + renderMode;
const kindLabel = String(kind || "Source");
---

<SiteLayout title={title}>
  <main class={pageClass}>
    <aside class="page-sidebar" aria-label="Page context">
      <a class="sidebar-home" href="/">Home</a>
      <div class="sidebar-section">
        <p class="sidebar-label">Kind</p>
        <p class="sidebar-value">{kindLabel}</p>
      </div>
      {sourcePath && (
        <div class="sidebar-section">
          <p class="sidebar-label">Path</p>
          <p class="sidebar-value sidebar-path">{sourcePath}</p>
        </div>
      )}
      {language && (
        <div class="sidebar-section">
          <p class="sidebar-label">Language</p>
          <p class="sidebar-value">{language}</p>
        </div>
      )}
    </aside>

    <div class="page-main">
      <header class="page-header">
        <p class="page-kicker">{kindLabel}</p>
        <h1>{title}</h1>
        {(language || sourcePath) && (
          <dl class="page-meta">
            {language && (
              <>
                <dt>Language</dt>
                <dd>{language}</dd>
              </>
            )}
            {sourcePath && (
              <>
                <dt>Path</dt>
                <dd><code>{sourcePath}</code></dd>
              </>
            )}
          </dl>
        )}
      </header>

      <section class="page-content" aria-label={"Content for " + title}>
        <slot />
      </section>
    </div>
  </main>
</SiteLayout>
`
}

func astroGlobalCSS() string {
	return `:root,
html[data-theme="light"] {
  color-scheme: light;
  --page-bg: #f7f8fa;
  --surface: #ffffff;
  --surface-muted: #f1f4f7;
  --text: #1f242b;
  --muted: #68717d;
  --line: #d9dee6;
  --accent: #2f6f8f;
  --link-hover: #245b78;
  --accent-warm: #8a5b2e;
  --focus: #c87822;
  --inline-code-text: #28313a;
  --meta-text: #4d5662;
  --body-gradient-start: rgba(255, 255, 255, 0.92);
  --body-gradient-end: rgba(247, 248, 250, 0.98);
  --header-bg: rgba(255, 255, 255, 0.86);
  --code-bg: #151a21;
  --code-text: #edf2f7;
  --code-muted: #aeb8c6;
  --code-border: #cfd6df;
  --code-shadow: rgba(31, 36, 43, 0.08);
  --code-keyword: #f3c969;
  --code-string: #9ed6a3;
  --code-function: #8ecae6;
  --code-type: #c4b5fd;
  --code-comment: var(--code-muted);
  --code-definition: #f5d28c;
  --code-reference-border: rgba(142, 202, 230, 0.45);
  --hover-underline: rgba(47, 111, 143, 0.58);
  --tooltip-bg: #20262e;
  --tooltip-text: #f7fafc;
  --tooltip-border: rgba(255, 255, 255, 0.14);
  --tooltip-link: #a7dff4;
  --tooltip-inline-code-bg: rgba(255, 255, 255, 0.1);
  --tooltip-code-bg: rgba(0, 0, 0, 0.2);
  --tooltip-code-border: var(--tooltip-border);
  --radius: 8px;
  --mono: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  --sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
}

html[data-theme="dark"] {
  color-scheme: dark;
  --page-bg: #111418;
  --surface: #171b21;
  --surface-muted: #222831;
  --text: #e6eaf0;
  --muted: #a4adba;
  --line: #303842;
  --accent: #7fc8ea;
  --link-hover: #a7dff4;
  --accent-warm: #dfb16d;
  --focus: #f0a642;
  --inline-code-text: #e6eaf0;
  --meta-text: #c4cbd5;
  --body-gradient-start: rgba(23, 27, 33, 0.96);
  --body-gradient-end: rgba(17, 20, 24, 0.98);
  --header-bg: rgba(17, 20, 24, 0.84);
  --code-bg: #0d1117;
  --code-text: #e7edf5;
  --code-muted: #909aa8;
  --code-border: #2d3540;
  --code-shadow: rgba(0, 0, 0, 0.24);
  --code-keyword: #f3c969;
  --code-string: #9ed6a3;
  --code-function: #8ecae6;
  --code-type: #c4b5fd;
  --code-comment: var(--code-muted);
  --code-definition: #f5d28c;
  --code-reference-border: rgba(142, 202, 230, 0.45);
  --hover-underline: rgba(127, 200, 234, 0.66);
  --tooltip-bg: #1d232c;
  --tooltip-text: #f5f8fb;
  --tooltip-border: rgba(255, 255, 255, 0.12);
  --tooltip-link: #a7dff4;
  --tooltip-inline-code-bg: rgba(255, 255, 255, 0.1);
  --tooltip-code-bg: rgba(0, 0, 0, 0.2);
  --tooltip-code-border: var(--tooltip-border);
}

* {
  box-sizing: border-box;
}

html {
  background: var(--page-bg);
  color: var(--text);
  font-family: var(--sans);
  font-size: 16px;
  line-height: 1.6;
}

body {
  min-width: 320px;
  margin: 0;
  background:
    linear-gradient(180deg, var(--body-gradient-start), var(--body-gradient-end) 260px),
    var(--page-bg);
}

a {
  color: var(--accent);
  text-decoration-thickness: 1px;
  text-underline-offset: 0.18em;
}

a:hover {
  color: var(--link-hover);
}

:focus-visible {
  outline: 3px solid var(--focus);
  outline: 3px solid color-mix(in srgb, var(--focus), transparent 35%);
  outline-offset: 3px;
}

.site-header {
  border-bottom: 1px solid var(--line);
  background: var(--header-bg);
  backdrop-filter: blur(12px);
}

.site-header__inner,
.site-shell,
.site-footer {
  width: min(calc(100% - 32px), 1120px);
  margin-inline: auto;
}

.site-header__inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  min-height: 58px;
}

.site-brand {
  color: var(--text);
  font-size: 0.98rem;
  font-weight: 700;
  text-decoration: none;
}

.site-actions {
  display: flex;
  align-items: center;
  gap: 14px;
  min-width: 0;
}

.site-nav {
  display: flex;
  align-items: center;
  gap: 18px;
  color: var(--muted);
  font-size: 0.9rem;
  font-weight: 650;
}

.site-nav a {
  color: inherit;
  text-decoration: none;
}

.site-nav a:hover {
  color: var(--text);
}

.theme-toggle {
  position: relative;
  display: inline-grid;
  place-items: center;
  width: 34px;
  height: 34px;
  flex: 0 0 auto;
  border: 1px solid var(--line);
  border-radius: 999px;
  background: var(--surface);
  color: var(--text);
  cursor: pointer;
}

.theme-toggle:hover {
  border-color: var(--accent);
}

.theme-toggle__icon {
  position: absolute;
  display: block;
  width: 15px;
  height: 15px;
  opacity: 0;
  transform: scale(0.72) rotate(-18deg);
  transition: opacity 140ms ease, transform 140ms ease;
}

.theme-toggle__icon--sun {
  border: 2px solid currentColor;
  border-radius: 999px;
  box-shadow:
    0 -8px 0 -5px currentColor,
    0 8px 0 -5px currentColor,
    8px 0 0 -5px currentColor,
    -8px 0 0 -5px currentColor,
    5.7px 5.7px 0 -5px currentColor,
    -5.7px 5.7px 0 -5px currentColor,
    5.7px -5.7px 0 -5px currentColor,
    -5.7px -5.7px 0 -5px currentColor;
}

.theme-toggle__icon--moon {
  border-radius: 999px;
  box-shadow: inset 5px -4px 0 0 currentColor;
}

html[data-theme="light"] .theme-toggle__icon--sun,
html:not([data-theme]) .theme-toggle__icon--sun,
html[data-theme="dark"] .theme-toggle__icon--moon {
  opacity: 1;
  transform: scale(1) rotate(0deg);
}

.site-shell {
  padding-block: 42px 64px;
}

.code-page {
  display: grid;
  grid-template-columns: minmax(160px, 240px) minmax(0, 1fr);
  gap: 36px;
  align-items: start;
}

.page-sidebar {
  position: sticky;
  top: 82px;
  display: grid;
  gap: 18px;
  min-width: 0;
  padding: 18px 0 18px 16px;
  border-left: 2px solid var(--line);
  color: var(--muted);
}

.sidebar-home {
  width: fit-content;
  color: var(--text);
  font-size: 0.92rem;
  font-weight: 750;
  text-decoration: none;
}

.sidebar-section {
  display: grid;
  gap: 4px;
}

.sidebar-label,
.sidebar-value {
  margin: 0;
}

.sidebar-label {
  color: var(--accent-warm);
  font-size: 0.75rem;
  font-weight: 750;
  text-transform: uppercase;
}

.sidebar-value {
  color: var(--text);
  font-size: 0.9rem;
}

.sidebar-path {
  overflow-wrap: anywhere;
  font-family: var(--mono);
  font-size: 0.82rem;
}

.page-main {
  min-width: 0;
}

.page-header {
  display: grid;
  gap: 12px;
  padding-bottom: 22px;
  border-bottom: 1px solid var(--line);
}

.page-kicker {
  margin: 0;
  color: var(--accent-warm);
  font-size: 0.84rem;
  font-weight: 700;
}

.page-header h1 {
  max-width: 880px;
  margin: 0;
  color: var(--text);
  font-size: 3rem;
  line-height: 1.08;
}

.page-meta {
  display: grid;
  grid-template-columns: max-content minmax(0, 1fr);
  gap: 4px 14px;
  max-width: 880px;
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 0.92rem;
}

.page-meta dt {
  color: var(--meta-text);
  font-weight: 700;
}

.page-meta dd {
  min-width: 0;
  margin: 0;
}

.page-meta code {
  overflow-wrap: anywhere;
  color: var(--text);
  font-family: var(--mono);
  font-size: 0.9em;
}

.page-content {
  margin-top: 28px;
}

.home-page {
  display: grid;
  gap: 46px;
}

.home-hero {
  display: grid;
  gap: 16px;
  max-width: 880px;
  padding-bottom: 28px;
  border-bottom: 1px solid var(--line);
}

.home-hero h1 {
  margin: 0;
  font-size: 3.2rem;
  line-height: 1.06;
}

.home-hero p {
  max-width: 680px;
  margin: 0;
  color: var(--muted);
  font-size: 1.04rem;
}

.home-section {
  display: grid;
  gap: 14px;
}

.home-section h2 {
  margin: 0;
  font-size: 1.1rem;
}

.home-list {
  display: grid;
  gap: 8px;
  max-width: 760px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.home-list a {
  display: block;
  padding: 10px 0;
  border-bottom: 1px solid var(--line);
  color: var(--text);
  font-family: var(--mono);
  font-size: 0.92rem;
  text-decoration: none;
  overflow-wrap: anywhere;
}

.home-empty {
  max-width: 680px;
  margin: 0;
  color: var(--muted);
}

.cire-page {
  display: grid;
  gap: 22px;
}

.cire-prose {
  max-width: 760px;
  color: var(--text);
}

.cire-prose > :first-child {
  margin-top: 0;
}

.cire-prose > :last-child {
  margin-bottom: 0;
}

.cire-prose p,
.cire-prose ul,
.cire-prose ol {
  margin: 0 0 1rem;
}

.cire-prose code {
  padding: 0.12em 0.28em;
  border-radius: 4px;
  background: var(--surface-muted);
  color: var(--inline-code-text);
  font-family: var(--mono);
  font-size: 0.92em;
}

.source-panel {
  margin-top: 28px;
}

.cire-code,
.source-code {
  min-height: 240px;
  margin: 0;
  padding: 22px;
  overflow: hidden;
  border: 1px solid var(--code-border);
  border-radius: var(--radius);
  background: var(--code-bg);
  box-shadow: 0 18px 44px var(--code-shadow);
  color: var(--code-text);
  font-family: var(--mono);
  font-size: 0.92rem;
  line-height: 1.7;
  tab-size: 2;
}

.cire-code {
  overflow: auto;
}

.source-code {
  overflow: auto;
}

.cire-code code,
.source-code code,
.cire {
  font-family: inherit;
}

.cire a {
  color: inherit;
  text-decoration: none;
}

.cire .keyword {
  color: var(--code-keyword);
}

.cire .string {
  color: var(--code-string);
}

.cire .function,
.cire .function\.method,
.cire .function\.builtin {
  color: var(--code-function);
}

.cire .type,
.cire .property {
  color: var(--code-type);
}

.cire .comment {
  color: var(--code-comment);
}

.cire .definition {
  color: var(--code-definition);
}

.cire .reference {
  border-bottom: 1px solid var(--code-reference-border);
}

.page-content [data-hover],
.page-content [data-hover-html] {
  border-bottom: 1px dotted var(--hover-underline);
  cursor: help;
}

.page-content [data-hover]:focus-visible,
.page-content [data-hover-html]:focus-visible {
  border-bottom-color: var(--focus);
}

.gocire-tooltip {
  position: absolute;
  z-index: 50;
  max-width: min(560px, calc(100vw - 32px));
  max-height: min(420px, calc(100vh - 32px));
  overflow: auto;
  padding: 10px 12px;
  border: 1px solid var(--tooltip-border);
  border-radius: 6px;
  background: var(--tooltip-bg);
  box-shadow: 0 16px 38px rgba(9, 13, 18, 0.28);
  color: var(--tooltip-text);
  font-family: var(--sans);
  font-size: 0.88rem;
  line-height: 1.5;
  pointer-events: none;
  white-space: normal;
}

.gocire-tooltip[hidden] {
  display: none;
}

.gocire-tooltip__content > :first-child {
  margin-top: 0;
}

.gocire-tooltip__content > :last-child {
  margin-bottom: 0;
}

.gocire-tooltip p {
  margin: 0.45rem 0;
}

.gocire-tooltip ul,
.gocire-tooltip ol {
  margin: 0.45rem 0;
  padding-left: 1.2rem;
}

.gocire-tooltip li + li {
  margin-top: 0.18rem;
}

.gocire-tooltip a {
  color: var(--tooltip-link);
  text-decoration-thickness: 1px;
  text-underline-offset: 0.18em;
}

.gocire-tooltip code {
  border-radius: 4px;
  background: var(--tooltip-inline-code-bg);
  color: var(--tooltip-text);
  font-family: var(--mono);
  font-size: 0.88em;
  padding: 0.08em 0.28em;
}

.gocire-tooltip pre {
  max-width: 100%;
  margin: 0.65rem 0;
  overflow-x: auto;
  border: 1px solid var(--tooltip-code-border);
  border-radius: 6px;
  background: var(--tooltip-code-bg);
  padding: 0.75rem;
}

.gocire-tooltip pre code,
.gocire-tooltip .chroma code {
  display: block;
  overflow-x: auto;
  background: transparent;
  padding: 0;
}

.gocire-tooltip table {
  display: block;
  max-width: 100%;
  margin: 0.65rem 0;
  overflow-x: auto;
  border-collapse: collapse;
}

.gocire-tooltip th,
.gocire-tooltip td {
  border: 1px solid var(--tooltip-code-border);
  padding: 0.25rem 0.45rem;
}

.gocire-tooltip .chroma {
  overflow-x: auto;
  background: transparent;
  color: var(--tooltip-text);
}

.gocire-tooltip .chroma .k,
.gocire-tooltip .chroma .kc,
.gocire-tooltip .chroma .kd,
.gocire-tooltip .chroma .kn,
.gocire-tooltip .chroma .kp,
.gocire-tooltip .chroma .kr {
  color: var(--code-keyword);
}

.gocire-tooltip .chroma .kt {
  color: var(--code-type);
}

.gocire-tooltip .chroma .s,
.gocire-tooltip .chroma .s1,
.gocire-tooltip .chroma .s2,
.gocire-tooltip .chroma .se,
.gocire-tooltip .chroma .sh,
.gocire-tooltip .chroma .si,
.gocire-tooltip .chroma .sx {
  color: var(--code-string);
}

.gocire-tooltip .chroma .nf,
.gocire-tooltip .chroma .nx,
.gocire-tooltip .chroma .na {
  color: var(--code-function);
}

.gocire-tooltip .chroma .c,
.gocire-tooltip .chroma .c1,
.gocire-tooltip .chroma .cm {
  color: var(--code-comment);
}

.site-footer {
  padding-block: 22px 34px;
  border-top: 1px solid var(--line);
  color: var(--muted);
  font-size: 0.88rem;
}

@media (max-width: 720px) {
  .site-header__inner,
  .site-shell,
  .site-footer {
    width: min(calc(100% - 24px), 1120px);
  }

  .site-shell {
    padding-block: 28px 44px;
  }

  .site-header__inner {
    align-items: flex-start;
    flex-direction: column;
    padding-block: 12px;
  }

  .site-actions {
    width: 100%;
    justify-content: space-between;
  }

  .site-nav {
    gap: 14px;
  }

  .code-page {
    grid-template-columns: 1fr;
    gap: 20px;
  }

  .page-sidebar {
    position: static;
    padding: 0 0 16px;
    border-left: 0;
    border-bottom: 1px solid var(--line);
  }

  .page-meta {
    grid-template-columns: 1fr;
    gap: 2px;
  }

  .page-meta dd + dt {
    margin-top: 8px;
  }

  .cire-code,
  .source-code {
    min-height: 180px;
    padding: 16px;
    font-size: 0.86rem;
  }

  .page-header h1 {
    font-size: 2rem;
  }

  .home-hero h1 {
    font-size: 2.15rem;
  }
}

@media (prefers-reduced-motion: no-preference) {
  .page-content [data-hover],
  .page-content [data-hover-html] {
    transition: border-color 140ms ease, color 140ms ease;
  }
}
`
}

func astroThemeJS() string {
	return `const storageKey = "gocire-theme";
const validThemes = new Set(["light", "dark"]);
const root = document.documentElement;
const buttons = Array.from(document.querySelectorAll("[data-theme-toggle]"));

const readStoredTheme = () => {
  try {
    return localStorage.getItem(storageKey);
  } catch {
    return null;
  }
};

const writeStoredTheme = (theme) => {
  try {
    localStorage.setItem(storageKey, theme);
  } catch {
    // Theme persistence is optional; the current page still updates.
  }
};

const systemTheme = () => {
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
};

const currentTheme = () => {
  const theme = root.getAttribute("data-theme");
  if (validThemes.has(theme)) {
    return theme;
  }
  const storedTheme = readStoredTheme();
  return validThemes.has(storedTheme) ? storedTheme : systemTheme();
};

const updateToggleLabels = (theme) => {
  const nextTheme = theme === "dark" ? "light" : "dark";
  const label = nextTheme === "dark" ? "Switch to dark theme" : "Switch to light theme";
  for (const button of buttons) {
    button.setAttribute("aria-label", label);
    button.setAttribute("title", label);
  }
};

const applyTheme = (theme, options = {}) => {
  const nextTheme = validThemes.has(theme) ? theme : systemTheme();
  root.setAttribute("data-theme", nextTheme);
  updateToggleLabels(nextTheme);

  if (options.persist) {
    writeStoredTheme(nextTheme);
  }
};

applyTheme(currentTheme());

for (const button of buttons) {
  button.addEventListener("click", () => {
    applyTheme(currentTheme() === "dark" ? "light" : "dark", { persist: true });
  });
}
`
}

func astroTooltipJS() string {
	return `import { autoUpdate, computePosition, flip, offset, shift } from "@floating-ui/dom";

const hoverSelector = "[data-hover-html], [data-hover]";
const tokens = Array.from(document.querySelectorAll(hoverSelector));

if (tokens.length > 0) {
  let tooltip;
  let cleanupPosition;
  let activeToken;

  const ensureTooltip = () => {
    if (tooltip) {
      return tooltip;
    }

    tooltip = document.createElement("div");
    tooltip.id = "gocire-tooltip";
    tooltip.className = "gocire-tooltip";
    tooltip.setAttribute("role", "tooltip");
    tooltip.hidden = true;
    const content = document.createElement("div");
    content.className = "gocire-tooltip__content";
    tooltip.appendChild(content);
    document.body.appendChild(tooltip);
    return tooltip;
  };

  const decodeBase64 = (encoded) => {
    if (!encoded) {
      return "";
    }

    try {
      return new TextDecoder().decode(Uint8Array.from(atob(encoded), (char) => char.charCodeAt(0)));
    } catch {
      return "";
    }
  };

  const setTooltipContent = (floating, token) => {
    const content = floating.querySelector(".gocire-tooltip__content");
    if (!content) {
      return false;
    }

    const html = decodeBase64(token.getAttribute("data-hover-html"));
    if (html) {
      content.innerHTML = html;
      return true;
    }

    const text = decodeBase64(token.getAttribute("data-hover"));
    if (text) {
      content.textContent = text;
      return true;
    }

    content.textContent = "";
    return false;
  };

  const updatePosition = async (token) => {
    const floating = ensureTooltip();
    const { x, y } = await computePosition(token, floating, {
      placement: "top-start",
      middleware: [offset(8), flip(), shift({ padding: 12 })],
    });

    Object.assign(floating.style, {
      left: x + "px",
      top: y + "px",
    });
  };

  const stopPositionUpdates = () => {
    if (cleanupPosition) {
      cleanupPosition();
      cleanupPosition = undefined;
    }
  };

  const hideTooltip = (token) => {
    if (token && activeToken && token !== activeToken) {
      return;
    }

    stopPositionUpdates();
    if (activeToken) {
      activeToken.removeAttribute("aria-describedby");
    }
    activeToken = undefined;

    if (tooltip) {
      tooltip.hidden = true;
    }
  };

  const showTooltip = (token) => {
    const floating = ensureTooltip();
    if (!setTooltipContent(floating, token)) {
      hideTooltip();
      return;
    }

    floating.hidden = false;
    activeToken = token;
    token.setAttribute("aria-describedby", floating.id);

    stopPositionUpdates();
    cleanupPosition = autoUpdate(token, floating, () => {
      updatePosition(token);
    });
    updatePosition(token);
  };

  for (const token of tokens) {
    if (!token.hasAttribute("tabindex")) {
      token.setAttribute("tabindex", "0");
    }

    token.addEventListener("mouseenter", () => showTooltip(token));
    token.addEventListener("mouseleave", () => hideTooltip(token));
    token.addEventListener("focus", () => showTooltip(token));
    token.addEventListener("blur", () => hideTooltip(token));
  }

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      hideTooltip(activeToken);
    }
  });

  document.addEventListener(
    "pointerdown",
    (event) => {
      const target = event.target instanceof Element ? event.target.closest(hoverSelector) : null;
      if (!target) {
        hideTooltip(activeToken);
      }
    },
    true,
  );
}
`
}
