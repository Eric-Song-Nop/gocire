package plans

// # Astro asset templates
//
// The Astro backend should keep shipping a complete default site shell, but the
// shell should not live as large JavaScript, CSS, and Astro raw strings inside
// Go source. The current approach keeps the generated site self-contained, but
// it makes frontend code hard to read, format, lint, review, and test.
//
// The preferred direction is to move static site assets into real files and
// embed those files into the Go binary. Go should continue to own project
// analysis, route generation, navigation generation, and generated source
// pages. Astro, CSS, and browser runtime files should live in their native
// formats.
func AstroAssetsShouldBeRealFiles() bool {
	return true
}

// Phase 1 is a narrow infrastructure migration:
//
//   - create an embeddable template directory under the Go package that writes
//     Astro site assets,
//   - move static `package.json`, `astro.config.mjs`, `.astro`, `.css`, and `.js`
//     assets into that directory,
//   - use `go:embed` to include the template directory in the `gocire` binary,
//   - copy embedded files to the output site during Astro project preparation,
//   - template only small dynamic values such as the site title.
//
// The output contract should remain the same: the generated Astro project still
// contains the same public files, repeated exports remain stable, and users do
// not need Node.js to run `go build` for the CLI.
func EmbedAstroTemplateFiles() string {
	return "phase-1"
}

// The first migration should not move dynamic generated data into the template
// system. These remain Go responsibilities:
//
// - generated source pages,
// - route index generation,
// - navigation data,
// - source route manifests,
// - LSP and syntax-highlight analysis.
//
// Keeping this boundary small avoids turning a readability fix into a rewrite
// of the project export backend.
func KeepGeneratedDataInGo() bool {
	return true
}

// Phase 2 can make the embedded template a first-class Astro scaffold. That
// means the template directory can be opened and checked as an Astro project
// with fixture generated data. This gives maintainers native feedback from the
// frontend toolchain without changing the runtime behavior of `gocire`.
//
// Useful checks include:
//
//   - Astro build against fixture generated pages,
//   - CSS and JavaScript formatting,
//   - accessibility-focused browser smoke tests for hover cards and theme
//     toggling,
//   - a Go test that verifies embedded files are written to the expected output
//     paths.
func PromoteTemplateToScaffold() string {
	return "phase-2"
}

// User-controlled template overrides should use a deterministic overlay:
//
// - built-in embedded files are always available,
// - a configured theme directory may replace files by the same relative path,
// - missing files fall back to the built-in version,
// - unknown override files should warn instead of silently doing nothing,
// - overrides replace whole files rather than merging file contents.
//
// Whole-file overlay keeps the compatibility model explicit. It also makes
// tests straightforward: verify default behavior, one-file override behavior,
// fallback behavior, invalid path rejection, and repeated export stability.
func AddWholeFileTemplateOverlay() string {
	return "phase-3"
}

// An npm package such as `@gocire/astro-theme` is a later productization step,
// not the first migration. It would improve independent theme release cycles,
// but it introduces version compatibility between the Go CLI, generated page
// props, CSS classes, runtime data attributes, and the npm package.
//
// Before publishing a theme package, the local embedded template API should be
// stable and documented.
func PublishThemePackageLater() bool {
	return true
}

// A frontend build chain can be useful for maintainers, especially if tooltip
// and theme runtimes grow into TypeScript modules. It should not become a
// runtime requirement for `gocire` users or for ordinary Go builds.
//
// If introduced, generated frontend assets should be committed or otherwise
// checked for freshness in CI, and the embedded output should stay readable
// enough for users inspecting generated sites.
func KeepFrontendBuildOptional() bool {
	return true
}
