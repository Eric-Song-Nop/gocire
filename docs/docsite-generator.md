# Docsite Generator
The plan is to implement a static documentation site generator like [docusaurus](https://docusaurus.io/) based on `gocire`. We directly generate the site from source code and its comments.

## File and Routes Structure
Codes in folder:
- `docs` will be generated as documents docusaurus documents.
- `blogs` will be generated as blogs as docusaurus blog.
- all other places will be generated as source code exploration pages, which looks just like ide in browser, no need to turn comments into text, but should keep lsp infos. They should not appear as first class entry appear anywhere, we can only enter by jumping from other file.
- `.gocire` or `docs/.gocire` can be project config file.

## Content Model

Files under `docs` and `blogs` are still normal source files. We do not introduce
a separate documentation file format. The generator renders these source files
differently based on where they are placed:

- files under `docs` become documentation pages.
- files under `blogs` become blog posts.
- files outside `docs` and `blogs` become source exploration pages.

For `docs` and `blogs`, comments are turned into prose and nearby code is turned
into semantic code blocks. The result should read like documentation or a blog
post, not like a raw source file.

The default rendering rule follows the current `gocire` behavior: standalone
comments become prose, and the source code between those comments is rendered as
semantic code blocks in the original source order. Inline comments remain part
of the code.

For source exploration pages, comments stay inside the source code. These pages
exist to show complete code context and to preserve LSP information such as
semantic tokens, hover, inlay hints, definitions, and references.

## Docs and Blogs

Documentation pages and blog posts have no fundamental difference in how source
content is interpreted. Both are generated from normal source files and comments.

The difference is organization:

- docs are long-lived structured documents, usually organized by sidebars and
  ordered sections.
- blogs are posts, usually organized by dates, tags, authors, and archives.

Page metadata should be inferred from file paths, file names, and optional
frontmatter comments. Frontmatter can provide explicit values such as title,
slug, tags, date, authors, draft status, and ordering. When frontmatter is not
present, the generator should use reasonable defaults from the file location and
name.

## Source Exploration Pages

Source exploration pages are generated for code outside `docs` and `blogs`, but
they are not first-class navigation entries. They should not appear as a main
section of the generated site.

Users reach source exploration pages through semantic navigation from docs,
blogs, or other source pages. The main expected entry is jump-to-definition.

Jump-to-definition should always go to the source definition. It should not be
rewritten to a documentation explanation page. Documentation navigation and
language navigation are separate concepts.

The default source exploration page should act as a stable source location, not
as a primary browsing experience. It should show the complete source file, file
path context, line anchors, syntax highlighting, hover information, inlay hints,
definition navigation, references, and a lightweight outline when available.

It should not expose a top-level source browser by default. A full repository
file tree can be a configurable feature later, but it should not define the
default product experience.

## Docusaurus Reference

Docusaurus is a reference for product structure and user expectations, not a hard
implementation requirement. We can borrow ideas such as docs, blogs, sidebars,
navbar, footer, themes, and generated routes when they fit.

The default theme can be Docusaurus-like.

## Output Model

Markdown, MDX, and the future docsite output should be treated like different
compiler backends. They should share the same intermediate representation of the
source content and language information, then render that shared representation
into different output formats.

The docsite output is an additional bundled site output, not a replacement for
the existing Markdown or MDX outputs.

## Frontend Runtime

The first docsite backend should use Astro to build the static site. `gocire`
should focus on source analysis and semantic page data, while Astro handles page
layout, routing, assets, and the documentation/blog/source page presentation.

The generated site should not depend on React, Vue, or another client-side app
framework by default. Most core features should remain static HTML and CSS:

- cross-file jump-to-definition uses normal links.
- line anchors use normal HTML ids.
- syntax highlighting uses generated classes and CSS.
- docs, blogs, source pages, sidebars, and navigation are static pages.

The only planned client-side dependency for the first version is Floating UI for
hover tooltips. Tooltip behavior benefits from JavaScript because placement,
edge flipping, scrolling, focus, and mobile interactions are difficult to handle
well with CSS alone.

Other JavaScript features such as search, command palettes, complex filtering,
or persisted user preferences can be considered later. They are not part of the
initial frontend runtime decision.

## Configuration

The generator needs project configuration. The long-term model should support
both declarative configuration and code-based extension, but the first version
should not require much configuration.

The first version should prefer conventions and defaults:

- `docs` is the default documentation directory.
- `blogs` is the default blog directory.
- other supported source files become source exploration pages.
- navigation, sidebars, routes, theme, language features, and LSP settings can
  use defaults at first.

A root-level `.gocire.yml` file can be introduced as the first declarative
configuration entry point when defaults are not enough.

Future configuration areas:

- site title and metadata,
- theme,
- docs and blog routes,
- sidebar ordering,
- blog metadata,
- source include and exclude patterns,
- LSP settings,
- feature flags for hover, inlay hints, definitions, references, and other
  language features.

Code-based configuration can be considered later for advanced customization.

## Generation Behavior

The generator should have default skip rules for files and directories that
should not become generated pages. The API should be designed so these rules can
later be configured through `.gocire.yml`.

When a file cannot be analyzed or some language information is unavailable, the
generator should prefer warnings and continue generating the rest of the site.

Search is not part of the initial plan.

## Discussion Focus

- The current renderer is designed around one source file. The docsite should
  think about the whole site and the relationships between generated pages.
- Cross-file LSP information is not fully implemented yet. The docsite needs to
  support language information and jump-to-definition across generated pages.
- Source exploration pages should keep comments inside the code while still
  preserving language information.
- Configuration needs more discussion, especially theme options, feature flags,
  and when a config file is not enough.
