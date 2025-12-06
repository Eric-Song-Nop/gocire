package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Eric-Song-Nop/gocire/internal"
)

func main() {
	srcPath := flag.String("src", "", "source file path")
	indexPath := flag.String("index", "./index.scip", "SCIP Index File Path")
	outPath := flag.String("output", "", "Output file path (optional). Defaults to source file path with appropriate extension")
	lang := flag.String("lang", "", "Language for syntax highlighting (optional)")
	format := flag.String("format", "mdx", "Output format: markdown or mdx")
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

	var tokens []internal.TokenInfo

	if scipAnalyzer != nil {
		tokens = append(tokens, scipAnalyzer.Analyze(absSrcPath)...)
	}

	// Syntax Highlighting Analysis
	if *lang != "" {
		highlightAnalyzer := internal.NewHighlightAnalyzer(*lang)
		highlightTokens, err := highlightAnalyzer.Analyze(absSrcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Highlight analysis failed: %v\n", err)
			os.Exit(1)
		}
		tokens = append(tokens, highlightTokens...)
	}

	// Sort and merge tokens first
	internal.SortTokens(tokens)
	tokens, err = internal.MergeSplitTokens(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: MergeSplitTokens failed: %v\n", err)
		os.Exit(1)
	}

	var output string
	if *format == "mdx" {
		generator, err := internal.NewMDXGenerator(absSrcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Load source file failed for MDX generator: %v\n", err)
			os.Exit(1)
		}
		output = generator.GenerateMDX(tokens)
	} else {
		generator, err := internal.NewMarkdownGenerator(absSrcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Load source file failed for generator: %v\n", err)
			os.Exit(1)
		}
		output = generator.GenerateMarkdown(tokens)
	}

	finalOutPath := *outPath
	if finalOutPath == "" {
		if *format == "mdx" {
			finalOutPath = absSrcPath + ".mdx"
		} else {
			finalOutPath = absSrcPath + ".md"
		}
	}

	err = os.WriteFile(finalOutPath, []byte(output), 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("%s generated at: %s\n", *format, finalOutPath)
}
