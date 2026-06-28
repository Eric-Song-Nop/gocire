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
  </head>
  <body>
    <header class="site-header">
      <div class="site-header__inner">
        <a class="site-brand" href="/">{siteTitle}</a>
        <nav class="site-nav" aria-label="Main navigation">
          <a href="/">Home</a>
          <a href="/#docs">Docs</a>
          <a href="/#blog">Blog</a>
        </nav>
      </div>
    </header>
    <slot />
    <footer class="site-footer">
      <span>{siteTitle}</span>
    </footer>
    <script>
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
	return `:root {
  color-scheme: light;
  --page-bg: #f7f8fa;
  --surface: #ffffff;
  --surface-muted: #f1f4f7;
  --text: #1f242b;
  --muted: #68717d;
  --line: #d9dee6;
  --accent: #2f6f8f;
  --accent-warm: #8a5b2e;
  --focus: #c87822;
  --code-bg: #151a21;
  --code-text: #edf2f7;
  --code-muted: #aeb8c6;
  --tooltip-bg: #20262e;
  --tooltip-text: #f7fafc;
  --radius: 8px;
  --mono: "SFMono-Regular", Consolas, "Liberation Mono", Menlo, monospace;
  --sans: Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
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
    linear-gradient(180deg, rgba(255, 255, 255, 0.92), rgba(247, 248, 250, 0.98) 260px),
    var(--page-bg);
}

a {
  color: var(--accent);
  text-decoration-thickness: 1px;
  text-underline-offset: 0.18em;
}

a:hover {
  color: #245b78;
}

:focus-visible {
  outline: 3px solid var(--focus);
  outline: 3px solid color-mix(in srgb, var(--focus), transparent 35%);
  outline-offset: 3px;
}

.site-header {
  border-bottom: 1px solid var(--line);
  background: rgba(255, 255, 255, 0.86);
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
  color: #4d5662;
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
  color: #28313a;
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
  border: 1px solid #cfd6df;
  border-radius: var(--radius);
  background: var(--code-bg);
  box-shadow: 0 18px 44px rgba(31, 36, 43, 0.08);
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
  color: #f3c969;
}

.cire .string {
  color: #9ed6a3;
}

.cire .function,
.cire .function\.method,
.cire .function\.builtin {
  color: #8ecae6;
}

.cire .type,
.cire .property {
  color: #c4b5fd;
}

.cire .comment {
  color: var(--code-muted);
}

.cire .definition {
  color: #f5d28c;
}

.cire .reference {
  border-bottom: 1px solid rgba(142, 202, 230, 0.45);
}

.page-content [data-hover] {
  border-bottom: 1px dotted rgba(237, 242, 247, 0.52);
  cursor: help;
}

.page-content [data-hover]:focus-visible {
  border-bottom-color: var(--focus);
}

.gocire-tooltip {
  position: absolute;
  z-index: 50;
  max-width: min(420px, calc(100vw - 32px));
  padding: 10px 12px;
  border: 1px solid rgba(255, 255, 255, 0.14);
  border-radius: 6px;
  background: var(--tooltip-bg);
  box-shadow: 0 16px 38px rgba(9, 13, 18, 0.28);
  color: var(--tooltip-text);
  font-family: var(--sans);
  font-size: 0.88rem;
  line-height: 1.5;
  pointer-events: none;
  white-space: pre-wrap;
}

.gocire-tooltip[hidden] {
  display: none;
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
  .page-content [data-hover] {
    transition: border-color 140ms ease, color 140ms ease;
  }
}
`
}

func astroTooltipJS() string {
	return `import { autoUpdate, computePosition, flip, offset, shift } from "@floating-ui/dom";

const hoverSelector = "[data-hover]";
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
    document.body.appendChild(tooltip);
    return tooltip;
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
    const encoded = token.getAttribute("data-hover");
    const text = encoded ? new TextDecoder().decode(Uint8Array.from(atob(encoded), (char) => char.charCodeAt(0))) : "";
    if (!text) {
      hideTooltip(token);
      return;
    }

    const floating = ensureTooltip();
    floating.textContent = text;
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
