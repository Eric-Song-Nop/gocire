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
	outPath := flag.String("output", "", "Output file path (optional). Defaults to source file path with .md extension")
	lang := flag.String("lang", "", "Language for syntax highlighting (optional)")
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
	var scipAnalyzer *internal.SCIPAnalyer
	absIndexPath, err := filepath.Abs(*indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to resolve index path: %v. SCIP analysis will be skipped.\n", err)
	} else {
		scipAnalyzer, err = internal.NewSCIPAnalyer(absIndexPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Load SCIP index file failed: %v. SCIP analysis will be skipped.\n", err)
		}
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

	generator, err := internal.NewMarkdownGenerator(absSrcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Load source file failed for generator: %v\n", err)
		os.Exit(1)
	}

	internal.SortTokens(tokens)
	tokens, err = internal.MergeSplitTokens(tokens)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: MergeSplitTokens failed: %v\n", err)
		os.Exit(1)
	}

	markdown := generator.GenerateMarkdown(tokens)

	finalOutPath := *outPath
	if finalOutPath == "" {
		finalOutPath = absSrcPath + ".md"
	}

	err = os.WriteFile(finalOutPath, []byte(markdown), 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Markdown generated at: %s\n", finalOutPath)
}
