# gocire

`gocire` is a CLI tool designed to generate MDX or Markdown documentation from your source code, specifically optimized for use with [Docusaurus](https://docusaurus.io/). It leverages [SCIP](https://github.com/sourcegraph/scip) for precise code navigation and Tree-sitter for syntax highlighting and comment extraction.

## Features

- **MDX & Markdown Generation:** Generates documentation that can be directly used in Docusaurus projects.
- **Code Navigation:** Uses SCIP data to link identifiers to their definitions.
- **Syntax Highlighting:** Supports syntax highlighting for multiple languages.
- **Comment Extraction:** Extracts comments and interleaves them with the code.
- **Custom Code Wrappers:** Allows customization of the HTML/JSX wrapping the code blocks (e.g., for collapsible details).

## Installation

```bash
go install github.com/Eric-Song-Nop/gocire/cmd/gocire@latest
```

## Usage

The basic usage requires specifying the source file and the language.

```bash
gocire -src <path/to/source/file> -lang <language>
```

### Arguments

| Flag                  | Description                                                                                | Default                                   | Required                   |
| :-------------------- | :----------------------------------------------------------------------------------------- | :---------------------------------------- | :------------------------- |
| `-src`                | Path to the source code file to analyze.                                                   | -                                         | **Yes**                    |
| `-lang`               | Programming language of the source file (see [Supported Languages](#supported-languages)). | -                                         | **Yes** (for highlighting) |
| `-index`              | Path to the SCIP index file.                                                               | `./index.scip`                            | No                         |
| `-output`             | Path for the generated output file.                                                        | `<src_path>.mdx` or `<src_path>.md`       | No                         |
| `-format`             | Output format: `mdx` or `markdown`.                                                        | `mdx`                                     | No                         |
| `-code-wrapper-start` | Custom opening HTML/JSX for code blocks.                                                   | `<details><summary>...</summary><pre...>` | No                         |
| `-code-wrapper-end`   | Custom closing HTML/JSX for code blocks.                                                   | `</code></pre></details>`                 | No                         |

### Supported Languages

`gocire` supports the following languages. You can use the language ID or any of its aliases for the `-lang` flag.

| Language   | Aliases              |
| :--------- | :------------------- |
| C          | `c`                  |
| C++        | `cpp`, `c++`         |
| C#         | `csharp`, `c#`, `cs` |
| Dart       | `dart`               |
| Go         | `go`, `golang`       |
| Haskell    | `haskell`            |
| Java       | `java`               |
| JavaScript | `javascript`, `js`   |
| PHP        | `php`                |
| Python     | `python`, `py`       |
| Ruby       | `ruby`               |
| Rust       | `rust`               |
| TypeScript | `typescript`, `ts`   |

## Styling Code Blocks in Docusaurus

`gocire` generates code blocks with the CSS class `.cire`: `<pre className="cire"><code>`. To apply a consistent theme to these blocks within your Docusaurus site, you can include the provided `gruvbox.css` example.

1. **Locate your Docusaurus `custom.css`:**
   In a typical Docusaurus project, this file is located at `src/css/custom.css`. If it doesn't exist, create it.

2. **Copy the styles:**
   Copy the entire content of `examples/gruvbox.css` into your Docusaurus project's `src/css/custom.css` file.

   The `gruvbox.css` file provides a Gruvbox theme (light and dark modes) for the `.cire` code blocks, ensuring they blend seamlessly with your Docusaurus site's aesthetics. It includes styles for syntax highlighting and ensures responsiveness.

---

## Docusaurus Integration Example

To generate an MDX file for a Go source file to be used in Docusaurus:

1. **Generate SCIP Index (Optional but Recommended):**
   For precise navigation, generate a SCIP index for your project using a SCIP indexer (e.g., `scip-go`).

   ```bash
   scip-go
   ```

2. **Run `gocire`:**

   ```bash
   gocire -src internal/MyComponent.go -lang go -index index.scip -output docs/MyComponent.mdx
   ```

3. **Use in Docusaurus:**
   Place the generated `docs/MyComponent.mdx` file in your Docusaurus `docs` directory. It will be rendered as a page with your source code, comments, and navigation links.

   The default output wraps the code in a `<details><pre><code>` tag, making it collapsible. You can customize this using the `-code-wrapper-start` and `-code-wrapper-end` flags if you have custom React components in Docusaurus, but make sure code is still in `<pre><code>` tags.

   **Example with custom component:**

   ```bash
   gocire -src internal/MyComponent.go -lang go \
     -code-wrapper-start "<MyCodeBlock>" \
     -code-wrapper-end "</MyCodeBlock>"
   ```

   To make hover available in Docusaurus, install:\

   ```bash
   pnpm i @rc-component/tooltip
   ```

   Then import in `MDXComponents.ts`:

   ```ts
   import Tooltip from "@rc-component/tooltip";
   import MDXComponents from "@theme-original/MDXComponents";

   export default {
     ...MDXComponents,
     Tooltip,
   };
   ```

   To support math, make sure you followed Docusaurus' instructions to [install KaTeX](https://docusaurus.io/docs/markdown-features/math-equations).
