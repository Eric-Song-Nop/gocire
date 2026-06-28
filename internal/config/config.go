package config

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultConfigFile = ".gocire.yml"

var defaultInclude = []string{
	"**/*.go",
	"**/*.rs",
	"**/*.ts",
	"**/*.tsx",
	"**/*.js",
	"**/*.jsx",
	"**/*.py",
	"**/*.cpp",
	"**/*.cxx",
	"**/*.cc",
	"**/*.hpp",
	"**/*.c",
	"**/*.h",
	"**/*.hs",
	"**/*.java",
	"**/*.rb",
	"**/*.cs",
	"**/*.php",
	"**/*.dart",
}

var defaultExclude = []string{
	".git/**",
	"node_modules/**",
	"vendor/**",
	"dist/**",
	"build/**",
	".gocire/**",
}

type ProjectConfig struct {
	Site    SiteConfig     `yaml:"site"`
	Project ProjectSection `yaml:"project"`
	Content ContentConfig  `yaml:"content"`
	Source  SourceConfig   `yaml:"source"`
	Output  OutputConfig   `yaml:"output"`
}

type SiteConfig struct {
	Title string `yaml:"title"`
}

type ProjectSection struct {
	Root string `yaml:"root"`
}

type ContentConfig struct {
	Docs  string `yaml:"docs"`
	Blogs string `yaml:"blogs"`
}

type SourceConfig struct {
	RoutePrefix string   `yaml:"routePrefix"`
	Include     []string `yaml:"include"`
	Exclude     []string `yaml:"exclude"`
}

type OutputConfig struct {
	Dir string `yaml:"dir"`
}

type rawProjectConfig struct {
	Site    *rawSiteConfig     `yaml:"site"`
	Project *rawProjectSection `yaml:"project"`
	Content *rawContentConfig  `yaml:"content"`
	Source  *rawSourceConfig   `yaml:"source"`
	Output  *rawOutputConfig   `yaml:"output"`
}

type rawSiteConfig struct {
	Title *string `yaml:"title"`
}

type rawProjectSection struct {
	Root *string `yaml:"root"`
}

type rawContentConfig struct {
	Docs  *string `yaml:"docs"`
	Blogs *string `yaml:"blogs"`
}

type rawSourceConfig struct {
	RoutePrefix      *string   `yaml:"routePrefix"`
	RoutePrefixSnake *string   `yaml:"route_prefix"`
	Include          *[]string `yaml:"include"`
	Exclude          *[]string `yaml:"exclude"`
}

type rawOutputConfig struct {
	Dir *string `yaml:"dir"`
}

func DefaultConfig() *ProjectConfig {
	return &ProjectConfig{
		Project: ProjectSection{
			Root: ".",
		},
		Content: ContentConfig{
			Docs:  "docs",
			Blogs: "blogs",
		},
		Source: SourceConfig{
			RoutePrefix: "/_source",
			Include:     cloneStrings(defaultInclude),
			Exclude:     cloneStrings(defaultExclude),
		},
		Output: OutputConfig{
			Dir: ".gocire/site",
		},
	}
}

func Load(configPath string) (*ProjectConfig, error) {
	usedDefaultPath := configPath == ""
	if usedDefaultPath {
		configPath = defaultConfigFile
	}

	absConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("resolve config path %q: %w", configPath, err)
	}

	cfg := DefaultConfig()
	configDir := filepath.Dir(absConfigPath)

	data, err := os.ReadFile(absConfigPath)
	if err != nil {
		if os.IsNotExist(err) && usedDefaultPath {
			if err := cfg.Normalize(configDir); err != nil {
				return nil, err
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config %q: %w", configPath, err)
	}

	var raw rawProjectConfig
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}

	applyRawConfig(cfg, &raw)

	if err := cfg.Normalize(configDir); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *ProjectConfig) Normalize(baseDir string) error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return fmt.Errorf("resolve config base dir %q: %w", baseDir, err)
	}

	if c.Project.Root, err = normalizePath(absBaseDir, c.Project.Root, "project.root"); err != nil {
		return err
	}
	if c.Content.Docs, err = normalizePath(absBaseDir, c.Content.Docs, "content.docs"); err != nil {
		return err
	}
	if c.Content.Blogs, err = normalizePath(absBaseDir, c.Content.Blogs, "content.blogs"); err != nil {
		return err
	}
	if c.Output.Dir, err = normalizePath(absBaseDir, c.Output.Dir, "output.dir"); err != nil {
		return err
	}

	c.Source.RoutePrefix = normalizeRoutePrefix(c.Source.RoutePrefix)

	return c.Validate()
}

func (c *ProjectConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("config is nil")
	}

	if err := validatePath("project.root", c.Project.Root); err != nil {
		return err
	}
	if err := validatePath("content.docs", c.Content.Docs); err != nil {
		return err
	}
	if err := validatePath("content.blogs", c.Content.Blogs); err != nil {
		return err
	}
	if err := validatePath("output.dir", c.Output.Dir); err != nil {
		return err
	}

	if c.Source.RoutePrefix == "" {
		return fmt.Errorf("source.routePrefix is required")
	}
	if !strings.HasPrefix(c.Source.RoutePrefix, "/") {
		return fmt.Errorf("source.routePrefix must start with /")
	}
	if c.Source.RoutePrefix != "/" && strings.HasSuffix(c.Source.RoutePrefix, "/") {
		return fmt.Errorf("source.routePrefix must not end with /")
	}

	return nil
}

func applyRawConfig(cfg *ProjectConfig, raw *rawProjectConfig) {
	if raw.Site != nil && raw.Site.Title != nil {
		cfg.Site.Title = *raw.Site.Title
	}
	if raw.Project != nil && raw.Project.Root != nil {
		cfg.Project.Root = *raw.Project.Root
	}
	if raw.Content != nil {
		if raw.Content.Docs != nil {
			cfg.Content.Docs = *raw.Content.Docs
		}
		if raw.Content.Blogs != nil {
			cfg.Content.Blogs = *raw.Content.Blogs
		}
	}
	if raw.Source != nil {
		if raw.Source.RoutePrefix != nil {
			cfg.Source.RoutePrefix = *raw.Source.RoutePrefix
		}
		if raw.Source.RoutePrefixSnake != nil {
			cfg.Source.RoutePrefix = *raw.Source.RoutePrefixSnake
		}
		if raw.Source.Include != nil {
			cfg.Source.Include = cloneStrings(*raw.Source.Include)
		}
		if raw.Source.Exclude != nil {
			cfg.Source.Exclude = cloneStrings(*raw.Source.Exclude)
		}
	}
	if raw.Output != nil && raw.Output.Dir != nil {
		cfg.Output.Dir = *raw.Output.Dir
	}
}

func normalizePath(baseDir, value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if err := validatePath(field, value); err != nil {
		return "", err
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(baseDir, value)
	}
	return filepath.Clean(value), nil
}

func validatePath(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s contains a null byte", field)
	}
	return nil
}

func normalizeRoutePrefix(routePrefix string) string {
	routePrefix = strings.TrimSpace(routePrefix)
	if routePrefix == "" {
		return "/"
	}
	return path.Clean("/" + strings.TrimLeft(routePrefix, "/"))
}

func cloneStrings(values []string) []string {
	return append([]string(nil), values...)
}
