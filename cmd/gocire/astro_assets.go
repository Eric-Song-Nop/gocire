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
		"package.json":                      packageJSON,
		"astro.config.mjs":                  astroConfigMJS(),
		"src/layouts/SiteLayout.astro":      astroSiteLayout(siteTitle),
		"src/components/CodePage.astro":     astroCodePage(),
		"src/components/Sidebar.astro":      astroSidebar(),
		"src/components/SidebarItems.astro": astroSidebarItems(),
		"src/styles/global.css":             astroGlobalCSS(),
		"src/scripts/theme.js":              astroThemeJS(),
		"src/scripts/tooltip.js":            astroTooltipJS(),
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
			"katex":            "0.17.0",
			"lucide-astro":     "latest",
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
import Moon from "lucide-astro/Moon";
import Sun from "lucide-astro/Sun";
import { navigation } from "../generated/navigation";
import "katex/dist/katex.min.css";
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
const docsHref = navigation.docs.firstHref || "/";
const blogHref = navigation.blog.firstHref || "/";
const primaryHref = navigation.docs.firstHref || navigation.blog.firstHref || "/";
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
        <a class="site-brand" href={primaryHref}>{siteTitle}</a>
        <div class="site-actions">
          <nav class="site-nav" aria-label="Main navigation">
            <a href={docsHref}>Docs</a>
            <a href={blogHref}>Blog</a>
          </nav>
          <button class="theme-toggle" type="button" data-theme-toggle aria-label="Toggle color theme" title="Toggle color theme">
            <Sun class="theme-toggle__icon theme-toggle__icon--sun" size={18} aria-hidden="true" />
            <Moon class="theme-toggle__icon theme-toggle__icon--moon" size={18} aria-hidden="true" />
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
import Sidebar from "./Sidebar.astro";

interface Props {
  title: string;
  kind?: string;
  language?: string;
  sourcePath?: string;
  date?: string;
  tags?: string[];
  author?: string;
  renderMode?: string;
}

const {
  title,
  kind = "Source",
  language,
  sourcePath,
  date,
  tags = [],
  author,
  renderMode = "source",
} = Astro.props;
const pageClass = "site-shell code-page code-page--" + renderMode;
const kindLabel = String(kind || "Source");
const currentPath = Astro.url.pathname;
const pageDate = String(date || "").trim();
const pageAuthor = String(author || "").trim();
const pageTags = compactTags(tags);
const hasPageMeta = Boolean(pageDate || pageAuthor || pageTags.length > 0);

function compactTags(values?: string[]) {
  return (values ?? []).map((value) => String(value).trim()).filter(Boolean);
}
---

<SiteLayout title={title}>
  <main class={pageClass}>
    <Sidebar kind={kind} sourcePath={sourcePath} language={language} currentPath={currentPath} />

    <div class="page-main">
      <header class="page-header">
        <p class="page-kicker">{kindLabel}</p>
        <h1>{title}</h1>
        {hasPageMeta && (
          <dl class="page-meta">
            {pageDate && (
              <>
                <dt>Date</dt>
                <dd><time datetime={pageDate}>{pageDate}</time></dd>
              </>
            )}
            {pageAuthor && (
              <>
                <dt>Author</dt>
                <dd>{pageAuthor}</dd>
              </>
            )}
            {pageTags.length > 0 && (
              <>
                <dt>Tags</dt>
                <dd>
                  <ul class="metadata-tags" aria-label="Tags">
                    {pageTags.map((tag) => <li class="metadata-tag">{tag}</li>)}
                  </ul>
                </dd>
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

func astroSidebar() string {
	return `---
import { navigation } from "../generated/navigation";
import SidebarItems from "./SidebarItems.astro";

interface NavigationItem {
  type?: string;
  title?: string;
  href?: string;
  sourcePath?: string;
  date?: string;
  tags?: string[];
  author?: string;
  items?: NavigationItem[];
}

interface Props {
  kind?: string;
  sourcePath?: string;
  language?: string;
  currentPath?: string;
}

const {
  kind = "source",
  sourcePath,
  language,
  currentPath = "/",
} = Astro.props;

const normalizedKind = String(kind || "source").toLowerCase();
const normalizedCurrentPath = normalizePath(currentPath);
const docsItems = (navigation.docs.items ?? []) as NavigationItem[];
const blogItems = (navigation.blog.items ?? []) as NavigationItem[];
const showDocs = normalizedKind === "docs";
const showBlog = normalizedKind === "blog";
const showSource = !showDocs && !showBlog;
const sidebarLabel = showDocs ? "Docs navigation" : showBlog ? "Blog navigation" : "Source context";

function normalizePath(value?: string) {
  const pathname = String(value || "/").split(/[?#]/)[0] || "/";
  if (pathname === "/") {
    return "/";
  }
  return pathname.endsWith("/") ? pathname : pathname + "/";
}

function isActive(href?: string) {
  return href ? normalizePath(href) === normalizedCurrentPath : false;
}

function compactText(value?: string) {
  return String(value || "").trim();
}

function compactTags(values?: string[]) {
  return (values ?? []).map((value) => String(value).trim()).filter(Boolean);
}
---

<aside class="page-sidebar" aria-label={sidebarLabel}>
  {showDocs && (
    <nav class="sidebar-nav" aria-label="Docs">
      <p class="sidebar-heading">Docs</p>
      {docsItems.length > 0 ? (
        <SidebarItems items={docsItems} currentPath={normalizedCurrentPath} />
      ) : (
        <p class="sidebar-empty">No docs pages yet.</p>
      )}
    </nav>
  )}

  {showBlog && (
    <nav class="sidebar-nav" aria-label="Blog">
      <p class="sidebar-heading">Blog</p>
      {blogItems.length > 0 ? (
        <ul class="sidebar-blog-list">
          {blogItems.map((item) => {
            const itemDate = compactText(item.date);
            const itemAuthor = compactText(item.author);
            const itemTags = compactTags(item.tags);
            return (
              <li class:list={["sidebar-blog-item", { "is-active": isActive(item.href) }]}>
                <a class="sidebar-blog-link" href={item.href || "#"} aria-current={isActive(item.href) ? "page" : undefined}>
                  <span class="sidebar-blog-title">{item.title || item.sourcePath || item.href}</span>
                  {(itemDate || itemAuthor || itemTags.length > 0) && (
                    <span class="sidebar-blog-meta">
                      {itemDate && <time class="sidebar-blog-date sidebar-date" datetime={itemDate}>{itemDate}</time>}
                      {itemAuthor && <span class="sidebar-blog-author">{itemAuthor}</span>}
                      {itemTags.length > 0 && <span class="sidebar-blog-tags">{itemTags.join(", ")}</span>}
                    </span>
                  )}
                </a>
              </li>
            );
          })}
        </ul>
      ) : (
        <p class="sidebar-empty">No blog posts yet.</p>
      )}
    </nav>
  )}

  {showSource && (
    <div class="sidebar-context">
      <p class="sidebar-heading">Source context</p>
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
    </div>
  )}
</aside>
`
}

func astroSidebarItems() string {
	return `---
interface NavigationItem {
  type?: string;
  title?: string;
  href?: string;
  sourcePath?: string;
  date?: string;
  tags?: string[];
  author?: string;
  items?: NavigationItem[];
}

interface Props {
  items: NavigationItem[];
  currentPath?: string;
  depth?: number;
}

const {
  items = [],
  currentPath = "/",
  depth = 0,
} = Astro.props;

const normalizedCurrentPath = normalizePath(currentPath);
const listClass = depth === 0 ? "sidebar-items" : "sidebar-items sidebar-items--nested";

function normalizePath(value?: string) {
  const pathname = String(value || "/").split(/[?#]/)[0] || "/";
  if (pathname === "/") {
    return "/";
  }
  return pathname.endsWith("/") ? pathname : pathname + "/";
}

function isActive(href?: string) {
  return href ? normalizePath(href) === normalizedCurrentPath : false;
}

function compactText(value?: string) {
  return String(value || "").trim();
}

function compactTags(values?: string[]) {
  return (values ?? []).map((value) => String(value).trim()).filter(Boolean);
}
---

<ul class={listClass}>
  {items.map((item) => (
    <li class:list={["sidebar-item", { "sidebar-item--category": item.type === "category" }]}>
      {item.type === "category" ? (
        <>
        <span class="sidebar-category">{item.title}</span>
        {(item.items?.length ?? 0) > 0 && (
          <Astro.self items={item.items} currentPath={normalizedCurrentPath} depth={depth + 1} />
        )}
        </>
      ) : (
        <a class:list={["sidebar-link", { "is-active": isActive(item.href) }]} href={item.href} aria-current={isActive(item.href) ? "page" : undefined}>
          <span class="sidebar-link__title">{item.title || item.sourcePath || item.href}</span>
          {(compactText(item.date) || compactText(item.author) || compactTags(item.tags).length > 0) && (
            <span class="sidebar-link__meta">
              {compactText(item.date) && <time class="sidebar-date" datetime={compactText(item.date)}>{compactText(item.date)}</time>}
              {compactText(item.author) && <span>{compactText(item.author)}</span>}
              {compactTags(item.tags).length > 0 && <span class="sidebar-link__tags">{compactTags(item.tags).join(", ")}</span>}
            </span>
          )}
        </a>
      )}
    </li>
  ))}
</ul>
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
  --code-bg: #f4f6f9;
  --code-text: #202833;
  --code-muted: #6a7380;
  --code-border: #cfd6df;
  --code-shadow: rgba(31, 36, 43, 0.08);
  --code-keyword: #7a4e00;
  --code-string: #2f6f4f;
  --code-function: #0f6680;
  --code-type: #6f4e99;
  --code-variable: #27313d;
  --code-constant: #8a4b73;
  --code-number: #9a5a1f;
  --code-operator: #5f6874;
  --code-punctuation: #788290;
  --code-attribute: #8a5b00;
  --code-module: #526d2a;
  --code-constructor: #7a4f7d;
  --code-label: #8a5b2e;
  --code-escape: #b25a16;
  --code-comment: var(--code-muted);
  --code-definition: #8a5b2e;
  --code-error: #b42318;
  --code-deleted: #b42318;
  --code-inserted: #1f7a4d;
  --code-reference-border: rgba(15, 102, 128, 0.38);
  --code-inlay-hint-text: #5f6874;
  --code-inlay-hint-bg: rgba(47, 111, 143, 0.1);
  --code-inlay-hint-border: rgba(47, 111, 143, 0.18);
  --hover-underline: rgba(47, 111, 143, 0.58);
  --tooltip-bg: #ffffff;
  --tooltip-text: #202833;
  --tooltip-border: rgba(31, 36, 43, 0.16);
  --tooltip-link: #245b78;
  --tooltip-inline-code-bg: #edf1f5;
  --tooltip-code-bg: var(--code-bg);
  --tooltip-code-border: var(--code-border);
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
  --code-variable: #e7edf5;
  --code-constant: #f6b6c8;
  --code-number: #f2b47d;
  --code-operator: #b8c0ca;
  --code-punctuation: #858e9c;
  --code-attribute: #f0c674;
  --code-module: #b9d781;
  --code-constructor: #e5b3f2;
  --code-label: #dfb16d;
  --code-escape: #f2a66f;
  --code-comment: var(--code-muted);
  --code-definition: #f5d28c;
  --code-error: #ff7b72;
  --code-deleted: #ff9b9b;
  --code-inserted: #9ed6a3;
  --code-reference-border: rgba(142, 202, 230, 0.45);
  --code-inlay-hint-text: #a4adba;
  --code-inlay-hint-bg: rgba(127, 200, 234, 0.12);
  --code-inlay-hint-border: rgba(127, 200, 234, 0.2);
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
  width: 18px;
  height: 18px;
  fill: none;
  opacity: 0;
  stroke: currentColor;
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-width: 2;
  transform: scale(0.72) rotate(-18deg);
  transition: opacity 140ms ease, transform 140ms ease;
}

html[data-theme="light"] .theme-toggle__icon--moon,
html:not([data-theme]) .theme-toggle__icon--moon,
html[data-theme="dark"] .theme-toggle__icon--sun {
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
  gap: 16px;
  min-width: 0;
  max-width: 100%;
  padding: 18px 0 18px 16px;
  border-left: 2px solid var(--line);
  color: var(--muted);
  overflow: hidden;
}

.sidebar-nav,
.sidebar-context {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.sidebar-heading {
  margin: 0;
  color: var(--accent-warm);
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0;
  text-transform: uppercase;
}

.sidebar-items,
.sidebar-blog-list {
  display: grid;
  gap: 2px;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.sidebar-items--nested {
  margin-top: 4px;
  padding-left: 12px;
  border-left: 1px solid var(--line);
}

.sidebar-item,
.sidebar-blog-item {
  min-width: 0;
}

.sidebar-category {
  display: block;
  padding: 7px 0 4px;
  color: var(--meta-text);
  font-size: 0.78rem;
  font-weight: 750;
}

.sidebar-link,
.sidebar-blog-link {
  display: grid;
  min-width: 0;
  gap: 2px;
  padding: 6px 8px;
  border-radius: 6px;
  color: var(--muted);
  font-size: 0.88rem;
  line-height: 1.35;
  text-decoration: none;
}

.sidebar-link:hover,
.sidebar-blog-link:hover {
  background: var(--surface-muted);
  color: var(--text);
}

.sidebar-link.is-active,
.sidebar-blog-link[aria-current="page"] {
  background: color-mix(in srgb, var(--accent), transparent 88%);
  color: var(--text);
  font-weight: 720;
}

.sidebar-link__title,
.sidebar-blog-title {
  min-width: 0;
  overflow-wrap: anywhere;
}

.sidebar-blog-meta,
.sidebar-link__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 8px;
  min-width: 0;
  color: var(--muted);
  font-size: 0.76rem;
  font-weight: 500;
}

.sidebar-blog-author,
.sidebar-blog-tags,
.sidebar-link__tags {
  min-width: 0;
  overflow-wrap: anywhere;
}

.sidebar-blog-date,
.sidebar-date {
  color: var(--muted);
  font-family: var(--mono);
  font-size: 0.76rem;
}

.sidebar-empty {
  margin: 0;
  color: var(--muted);
  font-size: 0.88rem;
}

.sidebar-section {
  display: grid;
  gap: 4px;
  min-width: 0;
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

.metadata-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 2px 10px;
  min-width: 0;
  margin: 0;
  padding: 0;
  list-style: none;
}

.metadata-tag {
  min-width: 0;
  color: var(--text);
  overflow-wrap: anywhere;
}

.metadata-tag::before {
  content: "#";
  margin-right: 1px;
  color: var(--accent-warm);
}

.page-content {
  margin-top: 28px;
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

.cire-prose ul,
.cire-prose ol {
  padding-left: 1.45rem;
}

.cire-prose li + li {
  margin-top: 0.28rem;
}

.cire-prose a {
  color: var(--accent);
  font-weight: 600;
  text-decoration-thickness: 1px;
  text-underline-offset: 0.18em;
}

.cire-prose a:hover {
  color: var(--link-hover);
}

.cire-prose blockquote {
  margin: 1.1rem 0;
  padding: 0.1rem 0 0.1rem 1rem;
  border-left: 3px solid var(--line);
  color: var(--meta-text);
}

.cire-prose blockquote > :first-child {
  margin-top: 0;
}

.cire-prose blockquote > :last-child {
  margin-bottom: 0;
}

.cire-prose code {
  padding: 0.12em 0.28em;
  border-radius: 4px;
  background: var(--surface-muted);
  color: var(--inline-code-text);
  font-family: var(--mono);
  font-size: 0.92em;
}

.cire-prose pre,
.cire-prose .chroma {
  max-width: 100%;
  margin: 1rem 0;
  overflow-x: auto;
  border: 1px solid var(--code-border);
  border-radius: 6px;
  background: var(--code-bg);
  color: var(--code-text);
  padding: 1rem;
}

.cire-prose .chroma pre {
  margin: 0;
  overflow: visible;
  border: 0;
  background: transparent;
  padding: 0;
}

.cire-prose pre code,
.cire-prose .chroma code {
  display: block;
  overflow-x: auto;
  border-radius: 0;
  background: transparent;
  color: var(--code-text);
  padding: 0;
}

.cire-prose table {
  display: block;
  max-width: 100%;
  margin: 1.25rem 0;
  overflow-x: auto;
  border-collapse: collapse;
  font-size: 0.95em;
}

.cire-prose th,
.cire-prose td {
  border: 1px solid var(--line);
  padding: 0.45rem 0.65rem;
  text-align: left;
  vertical-align: top;
}

.cire-prose th {
  background: var(--surface-muted);
  color: var(--meta-text);
  font-weight: 700;
}

.cire-prose img {
  display: block;
  max-width: 100%;
  height: auto;
  margin: 1rem auto;
  border-radius: 6px;
}

.cire-prose .katex-display {
  max-width: 100%;
  margin: 1.25rem 0;
  overflow-x: auto;
  overflow-y: hidden;
  padding: 0.15rem 0 0.25rem;
}

.cire-prose .katex-display > .katex {
  white-space: nowrap;
}

.source-panel {
  margin-top: 28px;
}

.cire-code,
.source-code {
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

.cire .keyword,
.cire .keyword\.conditional,
.cire .keyword\.debug,
.cire .keyword\.directive,
.cire .keyword\.exception,
.cire .keyword\.import,
.cire .keyword\.repeat {
  color: var(--code-keyword);
}

.cire .string,
.cire .character,
.cire .string\.special,
.cire .string\.special\.regex,
.cire .string\.special\.symbol {
  color: var(--code-string);
}

.cire .function,
.cire .function\.builtin,
.cire .function\.call,
.cire .function\.macro,
.cire .function\.method,
.cire .function\.method\.builtin,
.cire .function\.special {
  color: var(--code-function);
}

.cire .type,
.cire .type\.builtin,
.cire .identifier\.constant,
.cire .property\.definition,
.cire .property {
  color: var(--code-type);
}

.cire .variable,
.cire .variable\.builtin,
.cire .variable\.member,
.cire .variable\.parameter,
.cire .identifier,
.cire .identifier\.parameter,
.cire ._name {
  color: var(--code-variable);
}

.cire .constant,
.cire .constant\.builtin,
.cire .constant\.null,
.cire .boolean,
.cire .builtin {
  color: var(--code-constant);
}

.cire .number,
.cire .number\.float {
  color: var(--code-number);
}

.cire .operator,
.cire ._op {
  color: var(--code-operator);
}

.cire .punctuation,
.cire .punctuation\.bracket,
.cire .punctuation\.delimiter,
.cire .punctuation\.special,
.cire .delimiter {
  color: var(--code-punctuation);
}

.cire .attribute {
  color: var(--code-attribute);
}

.cire .module,
.cire .module\.builtin,
.cire .tag {
  color: var(--code-module);
}

.cire .constructor,
.cire ._type {
  color: var(--code-constructor);
}

.cire .label {
  color: var(--code-label);
}

.cire .escape,
.cire .string\.escape,
.cire .embedded {
  color: var(--code-escape);
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

.cire .inlay-hint {
  display: inline-block;
  margin-inline: 0.12em;
  padding: 0 0.28em;
  border: 1px solid var(--code-inlay-hint-border);
  border-radius: 4px;
  background: var(--code-inlay-hint-bg);
  color: var(--code-inlay-hint-text);
  font-size: 0.78em;
  font-style: normal;
  line-height: 1.35;
  text-decoration: none;
  user-select: none;
  vertical-align: baseline;
  white-space: pre;
}

.page-content [data-hover],
.page-content [data-hover-html] {
  border-bottom: 1px dotted var(--hover-underline);
  cursor: help;
  touch-action: manipulation;
}

.page-content [data-hover]:focus-visible,
.page-content [data-hover-html]:focus-visible {
  border-bottom-color: var(--focus);
}

.gocire-tooltip {
  position: fixed;
  z-index: 50;
  max-width: min(560px, calc(100vw - 24px));
  max-height: min(420px, calc(100dvh - 24px));
  overflow: auto;
  -webkit-overflow-scrolling: touch;
  padding: 10px 12px;
  border: 1px solid var(--tooltip-border);
  border-radius: 6px;
  background: var(--tooltip-bg);
  box-shadow: 0 16px 38px rgba(9, 13, 18, 0.28);
  color: var(--tooltip-text);
  font-family: var(--sans);
  font-size: 0.88rem;
  line-height: 1.5;
  overscroll-behavior: contain;
  pointer-events: auto;
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
  color: var(--code-text);
  padding: 0.75rem;
}

.gocire-tooltip pre code,
.gocire-tooltip .chroma code {
  display: block;
  overflow-x: auto;
  background: transparent;
  color: var(--code-text);
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
  color: var(--code-text);
}

.gocire-tooltip .chroma .k,
.gocire-tooltip .chroma .kc,
.gocire-tooltip .chroma .kd,
.gocire-tooltip .chroma .kn,
.gocire-tooltip .chroma .kp,
.gocire-tooltip .chroma .kr,
.gocire-tooltip .chroma .ow {
  color: var(--code-keyword);
}

.gocire-tooltip .chroma .kt,
.gocire-tooltip .chroma .nc,
.gocire-tooltip .chroma .nt {
  color: var(--code-type);
}

.gocire-tooltip .chroma .s,
.gocire-tooltip .chroma .sa,
.gocire-tooltip .chroma .sb,
.gocire-tooltip .chroma .sc,
.gocire-tooltip .chroma .dl,
.gocire-tooltip .chroma .sd,
.gocire-tooltip .chroma .s1,
.gocire-tooltip .chroma .s2,
.gocire-tooltip .chroma .sh,
.gocire-tooltip .chroma .si,
.gocire-tooltip .chroma .sx,
.gocire-tooltip .chroma .sr,
.gocire-tooltip .chroma .ss {
  color: var(--code-string);
}

.gocire-tooltip .chroma .se {
  color: var(--code-escape);
}

.gocire-tooltip .chroma .m,
.gocire-tooltip .chroma .mb,
.gocire-tooltip .chroma .mf,
.gocire-tooltip .chroma .mh,
.gocire-tooltip .chroma .mi,
.gocire-tooltip .chroma .il,
.gocire-tooltip .chroma .mo {
  color: var(--code-number);
}

.gocire-tooltip .chroma .nf,
.gocire-tooltip .chroma .fm,
.gocire-tooltip .chroma .nb {
  color: var(--code-function);
}

.gocire-tooltip .chroma .n,
.gocire-tooltip .chroma .nx,
.gocire-tooltip .chroma .nv,
.gocire-tooltip .chroma .vc,
.gocire-tooltip .chroma .vg,
.gocire-tooltip .chroma .vi,
.gocire-tooltip .chroma .vm,
.gocire-tooltip .chroma .py,
.gocire-tooltip .chroma .bp {
  color: var(--code-variable);
}

.gocire-tooltip .chroma .no,
.gocire-tooltip .chroma .l,
.gocire-tooltip .chroma .ld {
  color: var(--code-constant);
}

.gocire-tooltip .chroma .na,
.gocire-tooltip .chroma .nd,
.gocire-tooltip .chroma .ni {
  color: var(--code-attribute);
}

.gocire-tooltip .chroma .nn {
  color: var(--code-module);
}

.gocire-tooltip .chroma .ne {
  color: var(--code-constructor);
}

.gocire-tooltip .chroma .nl {
  color: var(--code-label);
}

.gocire-tooltip .chroma .o {
  color: var(--code-operator);
}

.gocire-tooltip .chroma .p {
  color: var(--code-punctuation);
}

.gocire-tooltip .chroma .c,
.gocire-tooltip .chroma .ch,
.gocire-tooltip .chroma .c1,
.gocire-tooltip .chroma .cm,
.gocire-tooltip .chroma .cpf,
.gocire-tooltip .chroma .cs {
  color: var(--code-comment);
}

.gocire-tooltip .chroma .err,
.gocire-tooltip .chroma .gr,
.gocire-tooltip .chroma .gt {
  color: var(--code-error);
}

.gocire-tooltip .chroma .gd {
  color: var(--code-deleted);
}

.gocire-tooltip .chroma .gi {
  color: var(--code-inserted);
}

.gocire-tooltip .chroma .gh,
.gocire-tooltip .chroma .go,
.gocire-tooltip .chroma .gp,
.gocire-tooltip .chroma .gu {
  color: var(--code-muted);
}

.gocire-tooltip .chroma .ge {
  font-style: italic;
}

.gocire-tooltip .chroma .gs {
  font-weight: 600;
}

.gocire-tooltip__actions {
  display: flex;
  gap: 8px;
  margin-top: 0.7rem;
  padding-top: 0.6rem;
  border-top: 1px solid var(--tooltip-border);
}

.gocire-tooltip__action {
  color: var(--tooltip-link);
  font-size: 0.82rem;
  font-weight: 700;
  text-decoration-thickness: 1px;
  text-underline-offset: 0.18em;
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

  .page-content [data-hover],
  .page-content [data-hover-html] {
    cursor: pointer;
  }

  .gocire-tooltip {
    max-height: min(60dvh, calc(100dvh - 24px));
  }

  .cire-code,
  .source-code {
    padding: 16px;
    font-size: 0.86rem;
  }

  .page-header h1 {
    font-size: 2rem;
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
  const hideDelayMs = 120;
  const tapMoveThreshold = 8;
  const focusableSelector = [
    "a[href]",
    "button:not([disabled])",
    "input:not([disabled])",
    "select:not([disabled])",
    "textarea:not([disabled])",
    "[tabindex]:not([tabindex=\"-1\"])",
  ].join(", ");
  let tooltip;
  let cleanupPosition;
  let activeToken;
  let hideTimer;
  let mode = "closed";
  let pointerCandidate;
  let suppressClickToken;
  let suppressClickTimer;

  const ensureTooltip = () => {
    if (tooltip) {
      return tooltip;
    }

    tooltip = document.createElement("div");
    tooltip.id = "gocire-tooltip";
    tooltip.className = "gocire-tooltip";
    tooltip.setAttribute("role", "dialog");
    tooltip.setAttribute("aria-label", "Symbol information");
    tooltip.setAttribute("aria-modal", "false");
    tooltip.setAttribute("tabindex", "-1");
    tooltip.hidden = true;

    const content = document.createElement("div");
    content.className = "gocire-tooltip__content";
    tooltip.appendChild(content);

    const actions = document.createElement("div");
    actions.className = "gocire-tooltip__actions";
    actions.hidden = true;
    const actionLink = document.createElement("a");
    actionLink.className = "gocire-tooltip__action";
    actionLink.textContent = "Open link";
    actions.appendChild(actionLink);
    tooltip.appendChild(actions);

    document.body.appendChild(tooltip);
    tooltip.addEventListener("mouseenter", cancelHide);
    tooltip.addEventListener("mouseleave", scheduleHide);
    tooltip.addEventListener("focusin", cancelHide);
    tooltip.addEventListener("focusout", scheduleHide);
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

  const closestToken = (target) => {
    return target instanceof Element ? target.closest(hoverSelector) : null;
  };

  const isTouchPointer = (event) => {
    return event.pointerType === "touch" || event.pointerType === "pen";
  };

  const tokenHref = (token) => {
    return token instanceof HTMLAnchorElement ? token.getAttribute("href") : "";
  };

  const isFocusableElement = (element) => {
    return element instanceof HTMLElement && !element.hidden && element.getAttribute("aria-hidden") !== "true";
  };

  const tooltipFocusableElements = () => {
    if (!tooltip || tooltip.hidden) {
      return [];
    }

    return Array.from(tooltip.querySelectorAll(focusableSelector)).filter(isFocusableElement);
  };

  const pageFocusableElements = () => {
    return Array.from(document.querySelectorAll(focusableSelector)).filter((element) => {
      return isFocusableElement(element) && (!tooltip || !tooltip.contains(element));
    });
  };

  const focusElement = (element) => {
    if (!(element instanceof HTMLElement)) {
      return false;
    }

    try {
      element.focus({ preventScroll: true });
    } catch {
      element.focus();
    }
    return document.activeElement === element;
  };

  const focusFirstTooltipItem = () => {
    const focusable = tooltipFocusableElements();
    if (focusable.length === 0) {
      return false;
    }

    cancelHide();
    return focusElement(focusable[0]);
  };

  const focusPageElementAdjacentToToken = (token, direction) => {
    if (!(token instanceof HTMLElement)) {
      return false;
    }

    const focusable = pageFocusableElements();
    const tokenIndex = focusable.indexOf(token);
    if (tokenIndex === -1) {
      return false;
    }

    return focusElement(focusable[tokenIndex + direction]);
  };

  const setTooltipAction = (floating, token) => {
    const actions = floating.querySelector(".gocire-tooltip__actions");
    const actionLink = floating.querySelector(".gocire-tooltip__action");
    if (!actions || !(actionLink instanceof HTMLAnchorElement)) {
      return;
    }

    const href = tokenHref(token);
    if (!href) {
      actions.hidden = true;
      actionLink.removeAttribute("href");
      return;
    }

    actionLink.href = href;
    actionLink.textContent = "Open link";
    actions.hidden = false;
  };

  const setTooltipContent = (floating, token) => {
    const content = floating.querySelector(".gocire-tooltip__content");
    if (!content) {
      return false;
    }

    const html = decodeBase64(token.getAttribute("data-hover-html"));
    if (html) {
      content.innerHTML = html;
      setTooltipAction(floating, token);
      return true;
    }

    const text = decodeBase64(token.getAttribute("data-hover"));
    if (text) {
      content.textContent = text;
      setTooltipAction(floating, token);
      return true;
    }

    content.textContent = "";
    setTooltipAction(floating, token);
    return false;
  };

  const updatePosition = async (token) => {
    const floating = ensureTooltip();
    const { x, y } = await computePosition(token, floating, {
      placement: "top-start",
      strategy: "fixed",
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

  const cancelHide = () => {
    if (hideTimer) {
      window.clearTimeout(hideTimer);
      hideTimer = undefined;
    }
  };

  const clearTokenState = (token) => {
    if (!token) {
      return;
    }

    token.removeAttribute("aria-describedby");
    token.removeAttribute("aria-controls");
    token.removeAttribute("aria-expanded");
  };

  const isTooltipActive = () => {
    if (!activeToken || !tooltip || tooltip.hidden) {
      return false;
    }
    if (mode === "touchPinned" || mode === "keyboardPinned") {
      return true;
    }

    const activeElement = document.activeElement;
    if (activeElement instanceof Node && (activeToken.contains(activeElement) || tooltip.contains(activeElement))) {
      return true;
    }

    return activeToken.matches(":hover") || tooltip.matches(":hover");
  };

  const hideTooltip = (token, options = {}) => {
    if (token && activeToken && token !== activeToken) {
      return;
    }
    if (!options.force && isTooltipActive()) {
      return;
    }

    cancelHide();
    stopPositionUpdates();
    clearTokenState(activeToken);
    activeToken = undefined;
    pointerCandidate = undefined;
    mode = "closed";

    if (tooltip) {
      tooltip.hidden = true;
    }
  };

  const scheduleHide = () => {
    cancelHide();
    hideTimer = window.setTimeout(() => {
      hideTooltip(activeToken);
    }, hideDelayMs);
  };

  const showTooltip = (token, nextMode = "hover") => {
    const floating = ensureTooltip();
    if (!setTooltipContent(floating, token)) {
      hideTooltip(undefined, { force: true });
      return;
    }

    cancelHide();
    floating.hidden = false;
    if (activeToken && activeToken !== token) {
      clearTokenState(activeToken);
    }
    activeToken = token;
    mode = nextMode;
    token.setAttribute("aria-describedby", floating.id);
    token.setAttribute("aria-controls", floating.id);
    token.setAttribute("aria-expanded", "true");

    stopPositionUpdates();
    cleanupPosition = autoUpdate(token, floating, () => {
      updatePosition(token);
    });
    updatePosition(token);
  };

  const suppressNextClick = (token) => {
    suppressClickToken = token;
    if (suppressClickTimer) {
      window.clearTimeout(suppressClickTimer);
    }
    suppressClickTimer = window.setTimeout(() => {
      if (suppressClickToken === token) {
        suppressClickToken = undefined;
      }
      suppressClickTimer = undefined;
    }, 800);
  };

  const handleTouchTap = (token) => {
    suppressNextClick(token);
    if (activeToken === token && mode === "touchPinned") {
      hideTooltip(token, { force: true });
      return;
    }

    showTooltip(token, "touchPinned");
  };

  const handleKeyboardActivation = (token, event) => {
    if (token instanceof HTMLAnchorElement || (event.key !== "Enter" && event.key !== " ")) {
      return;
    }

    event.preventDefault();
    event.stopPropagation();
    if (activeToken === token && mode === "keyboardPinned") {
      hideTooltip(token, { force: true });
      return;
    }

    showTooltip(token, "keyboardPinned");
    focusFirstTooltipItem();
  };

  const handleTooltipTab = (event) => {
    if (event.key !== "Tab" || !tooltip || tooltip.hidden || !activeToken) {
      return;
    }

    const focusable = tooltipFocusableElements();
    if (focusable.length === 0) {
      return;
    }

    const activeElement = document.activeElement;
    if (activeElement === activeToken && !event.shiftKey) {
      event.preventDefault();
      focusFirstTooltipItem();
      return;
    }

    if (!(activeElement instanceof Node) || !tooltip.contains(activeElement)) {
      return;
    }

    const focusIndex = focusable.indexOf(activeElement);
    if (event.shiftKey && focusIndex === 0) {
      event.preventDefault();
      focusElement(activeToken);
      return;
    }

    if (!event.shiftKey && focusIndex === focusable.length - 1) {
      const tokenToLeave = activeToken;
      event.preventDefault();
      hideTooltip(tokenToLeave, { force: true });
      focusPageElementAdjacentToToken(tokenToLeave, 1);
    }
  };

  for (const token of tokens) {
    if (!token.hasAttribute("tabindex")) {
      token.setAttribute("tabindex", "0");
    }
    if (!(token instanceof HTMLAnchorElement) && !token.hasAttribute("role")) {
      token.setAttribute("role", "button");
    }

    token.addEventListener("mouseenter", () => {
      if (mode !== "touchPinned") {
        showTooltip(token, "hover");
      }
    });
    token.addEventListener("mouseleave", scheduleHide);
    token.addEventListener("focus", () => {
      if (mode !== "touchPinned" || activeToken !== token) {
        showTooltip(token, "focus");
      }
    });
    token.addEventListener("blur", scheduleHide);
    token.addEventListener("keydown", (event) => handleKeyboardActivation(token, event));
  }

  document.addEventListener("keydown", (event) => {
    if (event.key === "Escape") {
      const shouldRestoreFocus = tooltip && document.activeElement instanceof Node && tooltip.contains(document.activeElement);
      const tokenToRestore = activeToken;
      hideTooltip(activeToken, { force: true });
      if (shouldRestoreFocus && tokenToRestore instanceof HTMLElement) {
        tokenToRestore.focus({ preventScroll: true });
      }
      return;
    }
    handleTooltipTab(event);
  });

  document.addEventListener(
    "pointerdown",
    (event) => {
      const target = event.target instanceof Node ? event.target : null;
      const hoveredToken = closestToken(event.target);

      if (tooltip && target && tooltip.contains(target)) {
        cancelHide();
        return;
      }

      if (isTouchPointer(event) && hoveredToken) {
        pointerCandidate = {
          pointerId: event.pointerId,
          token: hoveredToken,
          x: event.clientX,
          y: event.clientY,
        };
        return;
      }

      if (!hoveredToken && (!tooltip || !target || !tooltip.contains(target))) {
        hideTooltip(activeToken, { force: true });
      }
    },
    true,
  );

  document.addEventListener(
    "pointermove",
    (event) => {
      if (!pointerCandidate || pointerCandidate.pointerId !== event.pointerId) {
        return;
      }

      const dx = event.clientX - pointerCandidate.x;
      const dy = event.clientY - pointerCandidate.y;
      if (Math.hypot(dx, dy) > tapMoveThreshold) {
        pointerCandidate = undefined;
      }
    },
    true,
  );

  document.addEventListener(
    "pointerup",
    (event) => {
      if (!pointerCandidate || pointerCandidate.pointerId !== event.pointerId) {
        return;
      }

      const token = pointerCandidate.token;
      pointerCandidate = undefined;
      handleTouchTap(token);
      event.preventDefault();
      event.stopPropagation();
    },
    true,
  );

  document.addEventListener(
    "pointercancel",
    (event) => {
      if (pointerCandidate && pointerCandidate.pointerId === event.pointerId) {
        pointerCandidate = undefined;
      }
    },
    true,
  );

  document.addEventListener(
    "click",
    (event) => {
      const clickedToken = closestToken(event.target);
      if (clickedToken && suppressClickToken === clickedToken) {
        suppressClickToken = undefined;
        event.preventDefault();
        event.stopPropagation();
      }
    },
    true,
  );
}
`
}
