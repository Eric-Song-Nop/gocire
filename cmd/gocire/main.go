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

	"github.com/Eric-Song-Nop/gocire/internal/languages"
)

func main() {
	cfg, err := ParseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Auto-detect language if not provided
	if cfg.Lang == "" && cfg.AbsSrcPath != "" {
		info, err := os.Stat(cfg.AbsSrcPath)
		if err == nil && !info.IsDir() {
			lang, err := languages.DetectLanguage(cfg.AbsSrcPath)
			if err == nil {
				fmt.Printf("Auto-detected language: %s\n", lang)
				cfg.Lang = lang
			}
		}
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
