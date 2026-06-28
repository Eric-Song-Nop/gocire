package main

import (
	"flag"
	"io"
	"os"
	"strings"
	"testing"
)

func TestParseConfigProjectModeDoesNotRequireSrc(t *testing.T) {
	withCommandLine(t, []string{"gocire", "-project"})

	cfg, err := ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if !cfg.ProjectMode() {
		t.Fatal("ProjectMode returned false")
	}
	if cfg.AbsSrcPath != "" {
		t.Fatalf("AbsSrcPath = %q, want empty in project mode", cfg.AbsSrcPath)
	}
	if cfg.Jobs < 1 {
		t.Fatalf("Jobs = %d, want positive", cfg.Jobs)
	}
}

func TestParseConfigAllowsProjectAstroFormat(t *testing.T) {
	withCommandLine(t, []string{"gocire", "-project", "-format", "astro"})

	cfg, err := ParseConfig()
	if err != nil {
		t.Fatalf("ParseConfig returned error: %v", err)
	}
	if cfg.Format != "astro" {
		t.Fatalf("Format = %q, want astro", cfg.Format)
	}
	if !cfg.ProjectMode() {
		t.Fatal("ProjectMode returned false")
	}
}

func TestParseConfigRejectsSingleFileAstroFormat(t *testing.T) {
	withCommandLine(t, []string{"gocire", "-src", "main.go", "-format", "astro"})

	_, err := ParseConfig()
	if err == nil {
		t.Fatal("ParseConfig returned nil error")
	}
	if !strings.Contains(err.Error(), "only supported with -project") {
		t.Fatalf("error = %q, want project-only message", err.Error())
	}
}

func withCommandLine(t *testing.T, args []string) {
	t.Helper()

	oldArgs := os.Args
	oldCommandLine := flag.CommandLine

	flag.CommandLine = flag.NewFlagSet(args[0], flag.ContinueOnError)
	flag.CommandLine.SetOutput(io.Discard)
	os.Args = args

	t.Cleanup(func() {
		os.Args = oldArgs
		flag.CommandLine = oldCommandLine
	})
}
