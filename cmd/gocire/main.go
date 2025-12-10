// ---
// title: Main entry for cli
// ---
//
// The main file entry point for the CLI.
// {/* truncate */}
package main

import (
	"fmt"
	"os"
)

func main() {
	cfg, err := ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	pipeline, err := NewPipeline(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := pipeline.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
