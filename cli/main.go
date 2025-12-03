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

	absIndexPath, err := filepath.Abs(*indexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to resolve index path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Source path: %s\n", absSrcPath)
	fmt.Printf("Index path: %s\n", absIndexPath)

	analyzer, err := internal.NewSCIPAnalyer(absIndexPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Load scip index file failed: %v\n", err)
		os.Exit(1)
	}

	generator, err := internal.NewMarkdownGenerator(absSrcPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Load source file failed for generator: %v\n", err)
		os.Exit(1)
	}

	tokens := analyzer.Analyze(absSrcPath)

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

	err = os.WriteFile(finalOutPath, []byte(markdown), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Markdown generated at: %s\n", finalOutPath)
}
