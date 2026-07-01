package plans

// # Astro RSS and sitemap static endpoints
//
// PR2 should be stacked directly on top of the `site-data.ts` contract branch.
// Its job is to prove that Astro can consume the generated site model for
// static metadata endpoints without changing the source-code semantic pipeline.
// Go remains authoritative for source discovery, route generation, LSP and SCIP
// links, token anchors, hover data, page metadata, and generated source pages.
func AstroStaticEndpointsBuildOnSiteData() string {
	return "codex/astro-blog-endpoints"
}

// The user-facing scope is intentionally small:
//
//   - add `src/pages/rss.xml.ts` as a static Astro endpoint,
//   - add `src/pages/sitemap.xml.ts` as a static Astro endpoint,
//   - have RSS read `siteData.pages` and include only pages whose `kind` is
//     `"blog"`,
//   - have the sitemap read `siteData.pages` and include every generated page,
//   - build absolute URLs from the configured canonical `site.url` plus each
//     page `href`, preserving the configured base path and trailing slash.
func AddStaticMetadataEndpoints() []string {
	return []string{
		"rss.xml",
		"sitemap.xml",
	}
}

// A canonical `site.url` is required for valid absolute RSS and sitemap links.
// If the project config does not provide one, each endpoint should fail with a
// clear build-time error that names the missing config field and endpoint.
func RequireCanonicalSiteURLForEndpoints() []string {
	return []string{
		"site.url is required to generate rss.xml",
		"site.url is required to generate sitemap.xml",
	}
}

// The implementation should stay at the Astro shell and endpoint layer:
//
//   - register new embedded template assets in `astroSiteAssetTemplates`,
//   - create `src/pages/rss.xml.ts`,
//   - create `src/pages/sitemap.xml.ts`,
//   - add small local helpers only if needed for URL joining and XML escaping.
//
// Avoid moving this logic into Go-generated strings unless a value is already
// part of the generated site data contract.
func AddStaticAstroEndpointAssetsOnly() []string {
	return []string{
		"src/pages/rss.xml.ts",
		"src/pages/sitemap.xml.ts",
	}
}

// PR2 should explicitly avoid product and routing expansion:
//
//   - no `/blog/` landing page,
//   - no `src/pages/blog/index.astro`,
//   - no Blog navigation changes,
//   - no SiteLayout brand or home-link changes,
//   - no pagination,
//   - no tag pages,
//   - no source index,
//   - no search index,
//   - no Astro Content Collections,
//   - no ClientRouter or View Transitions,
//   - no changes to token links, hover cards, prefetch, or semantic jump
//     behavior.
func KeepPR2Narrow() bool {
	return true
}

// The first validation layer should be Go tests around the generated Astro
// template contract. Those tests should check that the expected endpoint files
// are written and that the endpoints import `site-data` rather than rediscovering
// pages from generated modules.
func ValidateTemplateContractWithGoTests() []string {
	return []string{
		"TestAstroTemplateOutputFilesMatchExpectedAssetContract",
		"TestWriteAstroSiteAssetsWritesExpectedFiles",
	}
}

// The second validation layer should be an Astro smoke build with fixture
// generated data. The smoke fixture should cover:
//
//   - configured `site.url` with a base path, such as
//     `https://example.com/docs`,
//   - at least two blog pages,
//   - at least one non-blog docs or source page,
//   - RSS output containing only blog URLs,
//   - sitemap output containing all generated page URLs,
//   - generated endpoint URLs preserving the `site.url` base path.
//
// A missing-URL fixture or focused unit test should also assert that each
// endpoint error is clear.
func ValidateAstroSmokeBuild() bool {
	return true
}

// If the existing project export fixture can support it without heavy setup,
// PR2 should also assert that a real Astro project export writes these files:
//
//   - `src/pages/rss.xml.ts`,
//   - `src/pages/sitemap.xml.ts`.
//
// This fixture is useful because it verifies the Go asset registry and template
// overlay path, while the Astro smoke build verifies runtime imports and build
// behavior.
func ValidateProjectExportWritesEndpointFiles() []string {
	return []string{
		"rss.xml",
		"sitemap.xml",
	}
}

// Acceptance criteria:
//
//   - `go test ./cmd/gocire` passes,
//   - the Astro smoke build passes,
//   - `rss.xml` includes only blog pages,
//   - `sitemap.xml` includes all generated pages,
//   - endpoint URLs preserve configured `site.url` base paths,
//   - builds fail clearly when feed or sitemap generation lacks `site.url`,
//   - no `/blog/` landing, Blog nav, SiteLayout brand/home, semantic page
//     generation, or link-resolution behavior changes.
func PR2AcceptanceCriteria() []string {
	return []string{
		"go-test",
		"astro-smoke-build",
		"rss-blog-only",
		"sitemap-all-pages",
		"site-url-base-path-preserved",
		"site-url-required",
		"blog-landing-unchanged",
		"semantic-jump-unchanged",
	}
}
