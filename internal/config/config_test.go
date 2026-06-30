package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadMissingDefaultConfigUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get changed cwd: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	assertPath(t, cfg.Project.Root, wd)
	assertPath(t, cfg.Content.Docs, filepath.Join(wd, "docs"))
	assertPath(t, cfg.Content.Blogs, filepath.Join(wd, "blogs"))
	if len(cfg.Content.Metadata) != 0 {
		t.Fatalf("content metadata = %#v, want empty", cfg.Content.Metadata)
	}
	assertPath(t, cfg.Output.Dir, filepath.Join(wd, ".gocire", "site"))

	if cfg.Source.RoutePrefix != "/_source" {
		t.Fatalf("route prefix = %q, want %q", cfg.Source.RoutePrefix, "/_source")
	}
	for _, pattern := range []string{"**/*.go", "**/*.rs", "**/*.ts", "**/*.tsx", "**/*.js", "**/*.jsx", "**/*.py"} {
		if !contains(cfg.Source.Include, pattern) {
			t.Fatalf("include missing %q from %#v", pattern, cfg.Source.Include)
		}
	}
	for _, pattern := range []string{".git/**", "node_modules/**", "vendor/**", "dist/**", "build/**", ".gocire/**"} {
		if !contains(cfg.Source.Exclude, pattern) {
			t.Fatalf("exclude missing %q from %#v", pattern, cfg.Source.Exclude)
		}
	}
}

func TestLoadEmptyYAMLUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, "")

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	assertPath(t, cfg.Project.Root, dir)
	assertPath(t, cfg.Content.Docs, filepath.Join(dir, "docs"))
	assertPath(t, cfg.Content.Blogs, filepath.Join(dir, "blogs"))
	if len(cfg.Content.Metadata) != 0 {
		t.Fatalf("content metadata = %#v, want empty", cfg.Content.Metadata)
	}
}

func TestLoadYAMLOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
site:
  title: Example Site
project:
  root: repo
content:
  docs: content/docs
  blogs: posts
source:
  routePrefix: source/
  include:
    - src/**/*.go
  exclude:
    - tmp/**
output:
  dir: public
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Site.Title != "Example Site" {
		t.Fatalf("site title = %q, want %q", cfg.Site.Title, "Example Site")
	}
	assertPath(t, cfg.Project.Root, filepath.Join(dir, "repo"))
	assertPath(t, cfg.Content.Docs, filepath.Join(dir, "content", "docs"))
	assertPath(t, cfg.Content.Blogs, filepath.Join(dir, "posts"))
	assertPath(t, cfg.Output.Dir, filepath.Join(dir, "public"))
	if cfg.Source.RoutePrefix != "/source" {
		t.Fatalf("route prefix = %q, want %q", cfg.Source.RoutePrefix, "/source")
	}
	if !reflect.DeepEqual(cfg.Source.Include, []string{"src/**/*.go"}) {
		t.Fatalf("include = %#v, want %#v", cfg.Source.Include, []string{"src/**/*.go"})
	}
	if !reflect.DeepEqual(cfg.Source.Exclude, []string{"tmp/**"}) {
		t.Fatalf("exclude = %#v, want %#v", cfg.Source.Exclude, []string{"tmp/**"})
	}
}

func TestLoadPartialYAMLKeepsDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
content:
  docs: guides
source:
  include:
    - src/**/*.go
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	assertPath(t, cfg.Project.Root, dir)
	assertPath(t, cfg.Content.Docs, filepath.Join(dir, "guides"))
	assertPath(t, cfg.Content.Blogs, filepath.Join(dir, "blogs"))
	assertPath(t, cfg.Output.Dir, filepath.Join(dir, ".gocire", "site"))
	if cfg.Source.RoutePrefix != "/_source" {
		t.Fatalf("route prefix = %q, want default %q", cfg.Source.RoutePrefix, "/_source")
	}
	if !reflect.DeepEqual(cfg.Source.Include, []string{"src/**/*.go"}) {
		t.Fatalf("include = %#v, want YAML override %#v", cfg.Source.Include, []string{"src/**/*.go"})
	}
	for _, pattern := range []string{".git/**", "node_modules/**", "vendor/**", "dist/**", "build/**", ".gocire/**"} {
		if !contains(cfg.Source.Exclude, pattern) {
			t.Fatalf("exclude missing default %q from %#v", pattern, cfg.Source.Exclude)
		}
	}
}

func TestLoadContentMetadata(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
content:
  metadata:
    docs/intro.md:
      title: "  Intro  "
      date: "2024-02-29"
      tags:
        - " guide "
        - reference
      author: " Ada "
    blogs/post.md:
      title: Blog Post
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	want := ContentMetadata{
		Title:  "Intro",
		Date:   "2024-02-29",
		Tags:   []string{"guide", "reference"},
		Author: "Ada",
	}
	if !reflect.DeepEqual(cfg.Content.Metadata["docs/intro.md"], want) {
		t.Fatalf("metadata = %#v, want %#v", cfg.Content.Metadata["docs/intro.md"], want)
	}
	if cfg.Content.Metadata["blogs/post.md"].Title != "Blog Post" {
		t.Fatalf("blog metadata = %#v, want title Blog Post", cfg.Content.Metadata["blogs/post.md"])
	}
}

func TestLoadIgnoresUnknownFieldsOutsideContentMetadata(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
unknownTopLevel: true
site:
  title: Example Site
  theme: ignored
content:
  docs: guides
  slug: ignored-outside-metadata
  custom:
    slug: ignored
source:
  routePrefix: code/
  experimental: ignored
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Site.Title != "Example Site" {
		t.Fatalf("site title = %q, want %q", cfg.Site.Title, "Example Site")
	}
	assertPath(t, cfg.Content.Docs, filepath.Join(dir, "guides"))
	if cfg.Source.RoutePrefix != "/code" {
		t.Fatalf("route prefix = %q, want %q", cfg.Source.RoutePrefix, "/code")
	}
}

func TestLoadContentMetadataNormalizesKeys(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
content:
  metadata:
    ./docs//guide.md:
      title: Guide
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if _, ok := cfg.Content.Metadata["docs/guide.md"]; !ok {
		t.Fatalf("metadata keys = %#v, want normalized docs/guide.md", cfg.Content.Metadata)
	}
	if _, ok := cfg.Content.Metadata["./docs//guide.md"]; ok {
		t.Fatalf("metadata kept unnormalized key: %#v", cfg.Content.Metadata)
	}
}

func TestLoadContentMetadataRejectsInvalidDate(t *testing.T) {
	tests := []struct {
		name string
		date string
	}{
		{name: "not real", date: "2024-02-30"},
		{name: "not strict format", date: " 2024-02-29 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".gocire.yml")
			writeFile(t, configPath, `
content:
  metadata:
    docs/intro.md:
      date: "`+tt.date+`"
`)

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Load returned nil error for invalid metadata date")
			}
			if !strings.Contains(err.Error(), ".date") {
				t.Fatalf("error = %q, want date context", err.Error())
			}
		})
	}
}

func TestLoadContentMetadataRejectsInvalidKeys(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{name: "empty", key: ""},
		{name: "absolute", key: "/docs/intro.md"},
		{name: "parent", key: "docs/../intro.md"},
		{name: "nul", key: `docs\u0000intro.md`},
		{name: "backslash", key: `docs\\intro.md`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".gocire.yml")
			writeFile(t, configPath, `
content:
  metadata:
    "`+tt.key+`":
      title: Intro
`)

			_, err := Load(configPath)
			if err == nil {
				t.Fatal("Load returned nil error for invalid metadata key")
			}
			if !strings.Contains(err.Error(), "content.metadata key") {
				t.Fatalf("error = %q, want metadata key context", err.Error())
			}
		})
	}
}

func TestLoadContentMetadataRejectsEmptyTag(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
content:
  metadata:
    docs/intro.md:
      tags:
        - reference
        - " "
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load returned nil error for empty metadata tag")
	}
	if !strings.Contains(err.Error(), ".tags[1]") {
		t.Fatalf("error = %q, want tag index context", err.Error())
	}
}

func TestLoadContentMetadataRejectsSlug(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, `
content:
  metadata:
    docs/intro.md:
      title: Intro
      slug: intro
`)

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load returned nil error for unsupported metadata slug")
	}
	if !strings.Contains(err.Error(), "slug") {
		t.Fatalf("error = %q, want slug context", err.Error())
	}
}

func TestLoadRelativePathsUseConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "nested", "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "gocire.yml")
	writeFile(t, configPath, `
project:
  root: ../repo
content:
  docs: ../docs
  blogs: ../writing/blogs
output:
  dir: ../out/site
`)

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	assertPath(t, cfg.Project.Root, filepath.Join(configDir, "..", "repo"))
	assertPath(t, cfg.Content.Docs, filepath.Join(configDir, "..", "docs"))
	assertPath(t, cfg.Content.Blogs, filepath.Join(configDir, "..", "writing", "blogs"))
	assertPath(t, cfg.Output.Dir, filepath.Join(configDir, "..", "out", "site"))
}

func TestLoadNormalizesSourceRoutePrefix(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want string
	}{
		{
			name: "adds leading slash",
			yaml: `
source:
  routePrefix: _source
`,
			want: "/_source",
		},
		{
			name: "trims trailing slash",
			yaml: `
source:
  routePrefix: /_source/
`,
			want: "/_source",
		},
		{
			name: "accepts snake case",
			yaml: `
source:
  route_prefix: code/
`,
			want: "/code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			configPath := filepath.Join(dir, ".gocire.yml")
			writeFile(t, configPath, tt.yaml)

			cfg, err := Load(configPath)
			if err != nil {
				t.Fatalf("Load returned error: %v", err)
			}
			if cfg.Source.RoutePrefix != tt.want {
				t.Fatalf("route prefix = %q, want %q", cfg.Source.RoutePrefix, tt.want)
			}
		})
	}
}

func TestLoadInvalidYAMLReturnsError(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ".gocire.yml")
	writeFile(t, configPath, "site: [")

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load returned nil error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("error = %q, want parse config context", err.Error())
	}
}

func writeFile(t *testing.T, path string, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertPath(t *testing.T, got, want string) {
	t.Helper()
	wantAbs, err := filepath.Abs(want)
	if err != nil {
		t.Fatalf("resolve want path %q: %v", want, err)
	}
	wantAbs = filepath.Clean(wantAbs)
	if got != wantAbs {
		t.Fatalf("path = %q, want %q", got, wantAbs)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
