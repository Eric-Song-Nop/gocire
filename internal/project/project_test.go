package project

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/Eric-Song-Nop/gocire/internal/config"
)

func TestScanIncludesGoFilesAndExcludesGeneratedTrees(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "main.go")
	writeTestFile(t, root, "pkg/util.go")
	writeTestFile(t, root, "vendor/pkg/skip.go")
	writeTestFile(t, root, "node_modules/pkg/skip.go")
	writeTestFile(t, root, ".git/hooks/skip.go")
	writeTestFile(t, root, "README.md")

	files, err := Scan(testProjectConfig(t, root,
		[]string{"**/*.go"},
		[]string{"vendor/**", "node_modules/**", ".git/**"},
		nil,
		nil,
	))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	relPaths := sourceRelPaths(files)
	want := []string{"main.go", "pkg/util.go"}
	if !slices.Equal(relPaths, want) {
		t.Fatalf("rel paths = %#v, want %#v", relPaths, want)
	}

	for _, file := range files {
		if file.Language != "go" {
			t.Fatalf("file %s language = %q, want go", file.RelPath, file.Language)
		}
	}
}

func TestScanSkipsAllDefaultGeneratedAndDependencyTrees(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "pkg/keep.go")
	writeTestFile(t, root, ".git/hooks/skip.go")
	writeTestFile(t, root, "node_modules/pkg/skip.go")
	writeTestFile(t, root, "vendor/pkg/skip.go")
	writeTestFile(t, root, "dist/assets/skip.go")
	writeTestFile(t, root, "build/generated/skip.go")
	writeTestFile(t, root, ".gocire/cache/skip.go")

	files, err := Scan(testProjectConfig(t, root,
		[]string{"**/*.go"},
		[]string{".git/**", "node_modules/**", "vendor/**", "dist/**", "build/**", ".gocire/**"},
		nil,
		nil,
	))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	relPaths := sourceRelPaths(files)
	want := []string{"pkg/keep.go"}
	if !slices.Equal(relPaths, want) {
		t.Fatalf("rel paths = %#v, want %#v", relPaths, want)
	}
}

func TestScanClassifiesDocsBlogsAndSource(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/guide.go")
	writeTestFile(t, root, "blogs/post.go")
	writeTestFile(t, root, "pkg/service.go")

	files, err := Scan(testProjectConfig(t, root,
		[]string{"**/*.go"},
		nil,
		[]string{"docs"},
		[]string{"blogs"},
	))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}

	kinds := map[string]PageKind{}
	for _, file := range files {
		kinds[file.RelPath] = file.Kind
	}

	want := map[string]PageKind{
		"docs/guide.go":  PageKindDocs,
		"blogs/post.go":  PageKindBlog,
		"pkg/service.go": PageKindSource,
	}
	if !reflect.DeepEqual(kinds, want) {
		t.Fatalf("kinds = %#v, want %#v", kinds, want)
	}
}

func TestGlobMatchingNormalizesBackslashes(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		relPath string
	}{
		{
			name:    "double star extension",
			pattern: `**\*.go`,
			relPath: `pkg\service.go`,
		},
		{
			name:    "directory prefix",
			pattern: `docs\**`,
			relPath: `docs\guide.go`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := matchGlob(tt.pattern, tt.relPath)
			if err != nil {
				t.Fatalf("matchGlob returned error: %v", err)
			}
			if !matched {
				t.Fatalf("matchGlob(%q, %q) = false, want true", tt.pattern, tt.relPath)
			}
		})
	}
}

func TestScanReturnsSlashRelativePaths(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, root, "docs/nested/example.go")

	files, err := Scan(testProjectConfig(t, root,
		[]string{"**/*.go"},
		nil,
		[]string{"docs/**"},
		nil,
	))
	if err != nil {
		t.Fatalf("Scan returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(files))
	}
	if files[0].RelPath != "docs/nested/example.go" {
		t.Fatalf("RelPath = %q, want slash-normalized docs/nested/example.go", files[0].RelPath)
	}
	if strings.Contains(files[0].RelPath, `\`) {
		t.Fatalf("RelPath contains backslash: %q", files[0].RelPath)
	}
}

func sourceRelPaths(files []SourceFile) []string {
	paths := make([]string, len(files))
	for i, file := range files {
		paths[i] = file.RelPath
	}
	slices.Sort(paths)
	return paths
}

func writeTestFile(t *testing.T, root, relPath string) {
	t.Helper()

	absPath := filepath.Join(append([]string{root}, strings.Split(relPath, "/")...)...)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q): %v", filepath.Dir(absPath), err)
	}
	if err := os.WriteFile(absPath, []byte("package test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q): %v", absPath, err)
	}
}

func testProjectConfig(t *testing.T, root string, include, exclude, docs, blogs []string) config.ProjectConfig {
	t.Helper()

	cfg := config.ProjectConfig{
		Project: config.ProjectSection{Root: root},
		Source: config.SourceConfig{
			Include: include,
			Exclude: exclude,
		},
	}
	if len(docs) > 0 {
		cfg.Content.Docs = docs[0]
	}
	if len(blogs) > 0 {
		cfg.Content.Blogs = blogs[0]
	}
	return cfg
}
