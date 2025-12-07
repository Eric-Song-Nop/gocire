// ---
// title: Main entry for cli
// ---
//
// The main file entry point for the CLI.
// {/* truncate */}
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Eric-Song-Nop/gocire/internal"
)

func main() {
	srcPath := flag.String("src", "", "source file path")
	indexPath := flag.String("index", "./index.scip", "SCIP Index File Path")
	outPath := flag.String("output", "", "Output file path (optional). Defaults to source file path with appropriate extension")
	lang := flag.String("lang", "", "Language for syntax highlighting (optional)")
	format := flag.String("format", "mdx", "Output format: markdown or mdx")
	codeWrapperStart := flag.String("code-wrapper-start", "<details open=\"true\">\n<summary>Expand to view code</summary>\n<pre className=\"cire\"><code>", "Custom opening HTML/JSX for code blocks")
	codeWrapperEnd := flag.String("code-wrapper-end", "</code></pre>\n</details>", "Custom closing HTML/JSX for code blocks")
	flag.Parse()

	if *srcPath == "" {
		fmt.Fprintf(os.Stderr, "Error: source file path is required\n")
		flag.Usage()
		os.Exit(1)
	}

	absSrcPath, err := filepath.Abs(*srcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve source path: %v\n", err)
		os.Exit(1)
	}

	// Read source file content once
	sourceContent, err := os.ReadFile(absSrcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to read source file: %v\n", err)
		os.Exit(1)
	}
	// Prepare source lines for generators
	sourceLines := strings.Split(string(sourceContent), "\n")

	// SCIP Analysis (optional if index file doesn't exist)
	var scipAnalyzer *internal.SCIPAnalyzer
	absIndexPath, err := filepath.Abs(*indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to resolve index path: %v. SCIP analysis will be skipped.\n", err)
	} else {
		scipAnalyzer, err = internal.NewSCIPAnalyzer(absIndexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Load SCIP index file failed: %v. SCIP analysis will be skipped.\n", err)
		}
	}

	// Validate format
	if *format != "markdown" && *format != "mdx" {
		fmt.Fprintf(os.Stderr, "Error: format must be 'markdown' or 'mdx'\n")
		flag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Source path: %s\n", absSrcPath)
	if scipAnalyzer != nil {
		fmt.Printf("Index path: %s\n", absIndexPath)
	}

	// Concurrency control
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs []error

	// Result containers
	var scipTokens []internal.TokenInfo
	var highlightTokens []internal.TokenInfo
	var commentTokens []internal.CommentInfo

	// Helper to collect errors
	addError := func(err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, err)
	}

	// 1. Run SCIP analysis
	if scipAnalyzer != nil {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					addError(fmt.Errorf("SCIP analysis panic: %v", r))
				}
			}()
			scipTokens = scipAnalyzer.Analyze(absSrcPath)
		})
	}

	// 2. Run syntax highlighting analysis
	if *lang != "" {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					addError(fmt.Errorf("Highlight analysis panic: %v", r))
				}
			}()
			highlightAnalyzer := internal.NewHighlightAnalyzer(*lang)
			tokens, err := highlightAnalyzer.Analyze(sourceContent)
			if err != nil {
				addError(err)
				return
			}
			highlightTokens = tokens
		})
	}

	// 3. Run comment analysis
	if *lang != "" {
		wg.Go(func() {
			defer func() {
				if r := recover(); r != nil {
					addError(fmt.Errorf("Comment analysis panic: %v", r))
				}
			}()
			commentAnalyzer := internal.NewCommentAnalyzer(*lang)
			comments, err := commentAnalyzer.Analyze(sourceContent)
			if err != nil {
				addError(err)
				return
			}
			commentTokens = comments
		})
	}

	// Wait for all analyses to complete
	wg.Wait()

	// Check if any errors occurred
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "Error: Analysis failed: %v\n", e)
		}
		os.Exit(1)
	}

	// Merge results
	var tokens []internal.TokenInfo
	tokens = append(tokens, scipTokens...)
	tokens = append(tokens, highlightTokens...)

	// Sort and merge tokens
	internal.SortBySpan(tokens)
	internal.SortBySpan(commentTokens)

	tokens, err = internal.MergeSplitTokens(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: MergeSplitTokens failed: %v\n", err)
		os.Exit(1)
	}

	var output string
	if *format == "mdx" {
		generator := internal.NewMDXGenerator(sourceLines)
		if *codeWrapperStart != "" {
			generator.CodeWrapperStart = *codeWrapperStart
		}
		if *codeWrapperEnd != "" {
			generator.CodeWrapperEnd = *codeWrapperEnd
		}
		output = generator.GenerateMDX(tokens, commentTokens)
	} else {
		generator := internal.NewMarkdownGenerator(sourceLines)
		if *codeWrapperStart != "" {
			generator.CodeWrapperStart = *codeWrapperStart
		}
		if *codeWrapperEnd != "" {
			generator.CodeWrapperEnd = *codeWrapperEnd
		}
		output = generator.GenerateMarkdown(tokens)
	}

	finalOutPath := *outPath
	if finalOutPath == "" {
		dir := filepath.Dir(absSrcPath)
		base := filepath.Base(absSrcPath)
		datePrefix := time.Now().Format("2006-01-02")

		if *format == "mdx" {
			finalOutPath = filepath.Join(dir, fmt.Sprintf("%s-%s.mdx", datePrefix, base))
		} else {
			finalOutPath = filepath.Join(dir, fmt.Sprintf("%s-%s.md", datePrefix, base))
		}
	}

	err = os.WriteFile(finalOutPath, []byte(output), 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s generated at: %s\n", *format, finalOutPath)
}
