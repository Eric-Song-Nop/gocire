package docs

// # Navigation rail alignment
//
// This page is intentionally small and heading-dense. It gives the generated
// Astro site a stable documentation page with multiple table-of-contents
// markers, so the right-side navigation rail can be checked with more than one
// tick.
//
// ## Top-level section
//
// The first marker should align to the same right edge as every other marker in
// the rail. When its active state grows wider, the extra width should extend to
// the left.
//
// ### Nested section
//
// A nested marker is narrower by design, but its right edge should still match
// the rail edge. This catches centered alignment regressions that are hard to
// see on pages with only one heading.
//
// #### Deep section
//
// The deepest visible marker gives the rail another width variant. All marker
// widths should share the same right anchor.
//
// ## Final section
//
// A second H2 keeps the rail from being a single cluster and makes the fixture
// useful for scrolling and active-marker checks.
func NavigationRailAlignmentFixture() bool {
	return true
}
