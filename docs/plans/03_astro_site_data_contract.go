package plans

// # Astro site data contract
//
// The Astro backend should use more Astro features, but only at the site shell
// and static endpoint layer. The source of truth for code semantics remains
// Go: source scanning, LSP and SCIP analysis, definition hrefs, hover data, and
// source anchors are generated before Astro sees the page.
//
// Astro should consume a stable generated contract instead of rediscovering the
// site shape from generated files. That keeps the semantic graph in Go while
// giving Astro enough structured data to build navigation, indexes, feeds, and
// metadata endpoints.
func AstroConsumesSiteDataContract() bool {
	return true
}

// The first stack branch should introduce `src/generated/site-data.ts`.
//
// The contract should be generated from Go using structured JSON encoding and
// then exported as TypeScript:
//
//   - `site`: title, description, optional canonical URL, and trailing slash
//     policy,
//   - `pages`: route parameter, href, generated module path, kind, title,
//     source path, language, date, tags, and author,
//   - `navigation`: the existing docs and blog navigation model.
//
// Route parameters and hrefs must remain distinct. `routeParam` feeds Astro's
// catch-all route, while `href` is the user-facing URL with the trailing slash
// policy applied.
func GenerateSiteDataFirst() string {
	return "site-data"
}

// The existing `navigation.ts` import path should remain as a compatibility
// shim while templates move toward the broader site data contract.
//
// ```ts
// export { navigation } from "./site-data";
// ```
//
// This keeps user template overrides and existing generated imports working
// while future templates import richer data from `site-data`.
func PreserveNavigationCompatibility() bool {
	return true
}

// The catch-all route should remain the routing primitive. Go already writes
// generated pages under `src/generated/pages`, and Astro's `[...gocire].astro`
// can use `getStaticPaths` plus `import.meta.glob` to load them.
//
// Do not switch to one physical `src/pages` route per source file. Source pages
// use `/_source`, nested paths, and extensions such as `.go.html`; those paths
// are easier and safer to enumerate from Go than to infer from Astro file
// system routing.
func KeepCatchAllGeneratedRoutes() bool {
	return true
}

// The first data-contract branch should also make the current URL policy
// explicit in generated Astro configuration:
//
//   - `trailingSlash: "always"`,
//   - `build.format: "directory"`,
//   - optional `site` only when the project config provides a canonical URL.
//
// The canonical site URL should not be guessed. It is required for high-quality
// RSS and sitemap output, but local docsite generation must still work without
// it.
func ExplicitAstroURLPolicy() string {
	return "directory-with-trailing-slashes"
}

// Once site data exists, the next stack branch can add Astro static endpoints
// that consume it:
//
//   - `sitemap.xml.ts` for all pages,
//   - `rss.xml.ts` for blog posts only,
//   - `search-index.json.ts` for metadata-only search.
//
// The first search index should stay small and deterministic. It should expose
// title, href, kind, language, source path, date, tags, and author. Full-text
// indexing can come later if the product gets an actual search UI.
func AddStaticEndpointsAfterSiteData() []string {
	return []string{
		"sitemap.xml",
		"rss.xml",
		"search-index.json",
	}
}

// Blog information architecture should start with a real `/blog/` landing page.
// The current shell can link directly to the latest blog post, but a stable
// blog index is a better public route and a better target for top navigation.
//
// Pagination, archives, and tag pages should come later when there is enough
// content and a stable tag taxonomy.
func AddBlogLandingBeforePagination() bool {
	return true
}

// Source discovery can come after blog and endpoint work. A language index can
// group source pages by detected language, but it should be paginated or capped
// for large repositories and should not replace semantic cross-file links.
func AddSourceLanguageIndexLater() string {
	return "phase-after-blog-and-endpoints"
}

// Platform features that change the content model or runtime model should stay
// out of the initial stack:
//
//   - do not move semantic data into Astro Content Collections,
//   - do not add ClientRouter or View Transitions until tooltip and theme
//     scripts are lifecycle-safe,
//   - do not introduce i18n routing without a locale-aware content model,
//   - do not add SSR, Actions, Server Islands, or Astro DB to the static
//     docsite.
func AvoidRuntimeAndContentModelExpansions() bool {
	return true
}
