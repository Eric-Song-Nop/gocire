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

## Docusaurus Reference

Docusaurus is a reference for product structure and user expectations, not a hard
implementation requirement. We can borrow ideas such as docs, blogs, sidebars,
navbar, footer, themes, and generated routes when they fit.

## Configuration

The generator needs project configuration. A config file may be enough for many
projects, but advanced projects may need code-based configuration.

Possible configuration areas:

- site title and metadata,
- theme,
- docs and blog routes,
- sidebar ordering,
- blog metadata,
- source include and exclude patterns,
- LSP settings,
- feature flags for hover, inlay hints, definitions, references, and other
  language features.

## Discussion Focus

- The current renderer is designed around one source file. The docsite should
  think about the whole site and the relationships between generated pages.
- Jump-to-definition should work across generated pages, not only inside one
  page.
- Existing Markdown and MDX output should remain part of the product direction.
  New site output should not make the older outputs second-class.
- We may need a clearer shared content model so different outputs can preserve
  the same source comments, code blocks, and language information. The exact
  shape is undecided.
- Source exploration pages should keep comments inside the code while still
  preserving language information.
- Configuration needs more discussion, especially theme options, feature flags,
  and when a config file is not enough.
