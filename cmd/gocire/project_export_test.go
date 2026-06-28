package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Eric-Song-Nop/gocire/internal"
	"github.com/Eric-Song-Nop/gocire/internal/project"
)

func TestProjectOutputPathUsesManifestRoute(t *testing.T) {
	root := t.TempDir()
	srcPath := filepath.Join(root, "pkg", "main.go")
	writeProjectTestFile(t, srcPath, "package main\n")

	manifest, err := internal.NewSourceRouteManifestWithPrefix(root, "/code", []string{srcPath})
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "site")
	got, err := ProjectOutputPath(outDir, manifest, project.SourceFile{
		AbsPath: srcPath,
		RelPath: "pkg/main.go",
	}, "mdx")
	if err != nil {
		t.Fatalf("ProjectOutputPath returned error: %v", err)
	}

	want := filepath.Join(outDir, "code", "pkg", "main.go.mdx")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
	if got == srcPath {
		t.Fatal("output path overwrote source path")
	}
}

func TestProjectOutputPathFallsBackToRelPath(t *testing.T) {
	manifestRoot := t.TempDir()
	sourceRoot := t.TempDir()
	srcPath := filepath.Join(sourceRoot, "lib", "app.go")

	manifest, err := internal.NewSourceRouteManifestWithPrefix(manifestRoot, "/code", nil)
	if err != nil {
		t.Fatalf("NewSourceRouteManifestWithPrefix returned error: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "site")
	got, err := ProjectOutputPath(outDir, manifest, project.SourceFile{
		AbsPath: srcPath,
		RelPath: "lib/app.go",
	}, "markdown")
	if err != nil {
		t.Fatalf("ProjectOutputPath returned error: %v", err)
	}

	want := filepath.Join(outDir, "lib", "app.go.md")
	if got != want {
		t.Fatalf("output path = %q, want %q", got, want)
	}
}

func TestOutputPathRejectsEscapingRelPath(t *testing.T) {
	_, err := outputPathForRelPath(t.TempDir(), "../outside.go", "mdx")
	if err == nil {
		t.Fatal("outputPathForRelPath returned nil error for escaping path")
	}
}

func TestNewProjectExportPlanScansAndBuildsManifest(t *testing.T) {
	root := t.TempDir()
	writeProjectTestFile(t, filepath.Join(root, "repo", "pkg", "app.go"), "package app\n")
	writeProjectTestFile(t, filepath.Join(root, "repo", "README.md"), "# docs\n")

	configPath := filepath.Join(root, ".gocire.yml")
	writeProjectTestFile(t, configPath, `
project:
  root: repo
source:
  routePrefix: code
  include:
    - "**/*.go"
output:
  dir: public
`)

	plan, err := NewProjectExportPlan(&Config{
		Project:    true,
		ConfigPath: configPath,
		Format:     "mdx",
	})
	if err != nil {
		t.Fatalf("NewProjectExportPlan returned error: %v", err)
	}

	if len(plan.Files) != 1 {
		t.Fatalf("len(plan.Files) = %d, want 1", len(plan.Files))
	}
	if plan.Files[0].RelPath != "pkg/app.go" {
		t.Fatalf("RelPath = %q, want pkg/app.go", plan.Files[0].RelPath)
	}
	if plan.Config.Output.Dir != filepath.Join(root, "public") {
		t.Fatalf("output dir = %q, want %q", plan.Config.Output.Dir, filepath.Join(root, "public"))
	}

	route, ok := plan.Manifest.RouteForRelPath("pkg/app.go")
	if !ok {
		t.Fatal("RouteForRelPath returned ok=false")
	}
	if route != "/code/pkg/app.go.html" {
		t.Fatalf("route = %q, want /code/pkg/app.go.html", route)
	}
}

func TestProjectExportRunnerWritesProjectFiles(t *testing.T) {
	root := t.TempDir()
	writeProjectTestFile(t, filepath.Join(root, "repo", "main.go"), "package main\n\nfunc main() {}\n")
	writeProjectTestFile(t, filepath.Join(root, "repo", "pkg", "util.go"), "package pkg\n\nfunc Util() {}\n")

	configPath := filepath.Join(root, ".gocire.yml")
	writeProjectTestFile(t, configPath, `
project:
  root: repo
source:
  include:
    - "**/*.go"
output:
  dir: site
`)

	runner, err := NewProjectExportRunner(&Config{
		Project:    true,
		ConfigPath: configPath,
		Jobs:       2,
		Format:     "markdown",
	})
	if err != nil {
		t.Fatalf("NewProjectExportRunner returned error: %v", err)
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, relPath := range []string{
		filepath.Join("_source", "main.go.md"),
		filepath.Join("_source", "pkg", "util.go.md"),
	} {
		outPath := filepath.Join(root, "site", relPath)
		if _, err := os.Stat(outPath); err != nil {
			t.Fatalf("expected output file %q: %v", outPath, err)
		}
	}
}

func writeProjectTestFile(t *testing.T, path string, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", path, err)
	}
}
