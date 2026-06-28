# Cross-File Jump To Definition

This plan focuses only on cross-file jump-to-definition for generated source
pages. It does not cover homepage design, search, references, or broader docsite
navigation.

## Goal

Generated files should be able to link to definitions in other generated files.

For example, if LSP reports that a token in `main.go` is defined in
`internal/store.go`, the generated output for `main.go` should link to the
generated source page for `internal/store.go`.

```text
main.go token
-> LSP definition: internal/store.go line 12 character 4
-> generated link: /_source/internal/store.go.html#L13C5
```

LSP positions are zero-based. Generated anchors should use one-based line and
column numbers.

## Main Constraint

Files should remain independently generatable.

Generating file A should not require file B to have already been analyzed or
rendered. File A only needs two stable mapping rules:

- how a source file path maps to a generated source page URL.
- how a source position maps to an anchor inside that generated page.

This keeps cross-file navigation cheap and avoids a large global semantic index
as a requirement for the first version.

## Route Mapping

Every source file that may be used as a jump target should have a source page.
Jump-to-definition should always target source pages, not narrative docs or blog
pages.

Example mappings:

```text
docs/intro.go
-> document page: /docs/intro
-> source page: /_source/docs/intro.go.html

blogs/2026-06-28-lsp.go
-> blog page: /blog/2026/06/28/lsp
-> source page: /_source/blogs/2026-06-28-lsp.go.html

internal/cache/store.go
-> source page: /_source/internal/cache/store.go.html
```

The source page route should be derived from the repository-relative path. This
route mapping can be built by scanning files and applying include/exclude rules.
It does not need LSP.

## Anchor Mapping

Source pages should generate stable anchors from source positions.

At minimum, every rendered line should have a line anchor:

```text
line 12 -> #L13
```

When a token starts at a known column, the page can also generate a line-column
anchor:

```text
line 12 character 4 -> #L13C5
```

The first version can jump to line anchors only if that is simpler:

```text
/_source/internal/cache/store.go.html#L13
```

Line-column anchors are more precise, but line anchors are more robust because
they do not depend on target-file token analysis succeeding.

## Link Resolution

The LSP analyzer should preserve the real definition location returned by LSP.
The generator should not have to infer this from a symbol string.

Resolution rule:

```text
LSP definition URI + range
-> normalize URI to repository-relative path
-> find source page route from the route mapping
-> append position anchor
-> write href on the current token
```

Example:

```text
LSP definition:
  file:///repo/internal/cache/store.go
  line: 12
  character: 4

Route mapping:
  internal/cache/store.go -> /_source/internal/cache/store.go.html

Generated href:
  /_source/internal/cache/store.go.html#L13C5
```

If the target file is the current file, the link can be page-local:

```text
#L13C5
```

If the target file is another generated file, the link should be relative or
absolute according to the output backend's routing rules.

## Missing Targets

If LSP returns a definition target that cannot be mapped to a generated source
page, generation should continue with a warning.

Common cases:

- the definition is outside the repository.
- the definition is in a dependency or standard library.
- the definition file is skipped by include/exclude rules.
- the target language or file type is unsupported.

The first version can leave the token unlinked in these cases.

## Current Implementation Gap

The current `LSPAnalyzer` already calls `textDocument/definition`, but it does
not preserve the returned target location as a first-class field. It turns the
location into a symbol-like string and only marks references when the target is
inside the current file.

The current generators also render references as page-local links:

```html
<a href="#symbol">...</a>
```

Cross-file jump-to-definition requires keeping the LSP definition location until
a later link-resolution step can turn it into a generated page URL.

## Proposed Shape

The implementation should keep the existing analyzer and generator structure,
but add a small link-resolution layer.

Conceptually:

```text
File analysis
-> tokens with hover/highlight/definition location
-> link resolver uses route mapping
-> tokens with href/anchor
-> generator renders links
```

The generator should consume already resolved link fields. It should not know
whether a link is same-file or cross-file.

## First Version

The first implementation should aim for the smallest working cross-file loop:

1. Build a source route mapping from repository-relative paths.
2. Preserve LSP definition locations on tokens.
3. Resolve each definition location to a source page URL plus line anchor.
4. Render token links using resolved hrefs.
5. Ensure source pages generate matching line anchors.
6. Warn when a definition target cannot be mapped.

Column-accurate anchors, external dependency links, references lists, and search
can be added later.
