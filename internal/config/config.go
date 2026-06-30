package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

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
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	URL         string `yaml:"url"`
	TemplateDir string `yaml:"templateDir"`
}

type ProjectSection struct {
	Root string `yaml:"root"`
}

type ContentConfig struct {
	Docs     string                     `yaml:"docs"`
	Blogs    string                     `yaml:"blogs"`
	Metadata map[string]ContentMetadata `yaml:"metadata"`
}

type ContentMetadata struct {
	Title  string   `yaml:"title"`
	Date   string   `yaml:"date"`
	Tags   []string `yaml:"tags"`
	Author string   `yaml:"author"`
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
	Title            *string `yaml:"title"`
	Description      *string `yaml:"description"`
	URL              *string `yaml:"url"`
	TemplateDir      *string `yaml:"templateDir"`
	TemplateDirSnake *string `yaml:"template_dir"`
}

type rawProjectSection struct {
	Root *string `yaml:"root"`
}

type rawContentConfig struct {
	Docs     *string                       `yaml:"docs"`
	Blogs    *string                       `yaml:"blogs"`
	Metadata map[string]rawContentMetadata `yaml:"metadata"`
}

type rawContentMetadata struct {
	Title  *string   `yaml:"title"`
	Date   *string   `yaml:"date"`
	Tags   *[]string `yaml:"tags"`
	Author *string   `yaml:"author"`
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
			Docs:     "docs",
			Blogs:    "blogs",
			Metadata: map[string]ContentMetadata{},
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

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
	}
	if err := validateContentMetadataFields(&root); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", configPath, err)
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

func validateContentMetadataFields(root *yaml.Node) error {
	doc := documentNodeContent(root)
	if doc == nil || doc.Kind != yaml.MappingNode {
		return nil
	}

	content := mappingValue(doc, "content")
	if content == nil || content.Kind != yaml.MappingNode {
		return nil
	}

	metadata := mappingValue(content, "metadata")
	if metadata == nil || metadata.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(metadata.Content); i += 2 {
		pageKey := metadata.Content[i]
		pageMetadata := metadata.Content[i+1]
		if pageMetadata.Kind != yaml.MappingNode {
			continue
		}
		for j := 0; j+1 < len(pageMetadata.Content); j += 2 {
			field := pageMetadata.Content[j]
			if !isAllowedContentMetadataField(field.Value) {
				return fmt.Errorf("unknown metadata field %q in content.metadata[%q]", field.Value, pageKey.Value)
			}
		}
	}
	return nil
}

func documentNodeContent(root *yaml.Node) *yaml.Node {
	if root == nil {
		return nil
	}
	if root.Kind != yaml.DocumentNode {
		return root
	}
	if len(root.Content) == 0 {
		return nil
	}
	return root.Content[0]
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Kind == yaml.ScalarNode && node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func isAllowedContentMetadataField(field string) bool {
	switch field {
	case "title", "date", "tags", "author":
		return true
	default:
		return false
	}
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
	if c.Site.TemplateDir, err = normalizeOptionalPath(absBaseDir, c.Site.TemplateDir, "site.templateDir"); err != nil {
		return err
	}
	c.Site.Description = strings.TrimSpace(c.Site.Description)
	if c.Site.URL, err = normalizeSiteURL(c.Site.URL); err != nil {
		return err
	}
	if c.Content.Docs, err = normalizePath(absBaseDir, c.Content.Docs, "content.docs"); err != nil {
		return err
	}
	if c.Content.Blogs, err = normalizePath(absBaseDir, c.Content.Blogs, "content.blogs"); err != nil {
		return err
	}
	if c.Content.Metadata, err = normalizeContentMetadata(c.Content.Metadata); err != nil {
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
	if err := validateOptionalPath("site.templateDir", c.Site.TemplateDir); err != nil {
		return err
	}
	if err := validateSiteURL(c.Site.URL); err != nil {
		return err
	}
	if err := validatePath("content.docs", c.Content.Docs); err != nil {
		return err
	}
	if err := validatePath("content.blogs", c.Content.Blogs); err != nil {
		return err
	}
	if err := validateContentMetadata(c.Content.Metadata); err != nil {
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
	if raw.Site != nil {
		if raw.Site.Title != nil {
			cfg.Site.Title = *raw.Site.Title
		}
		if raw.Site.Description != nil {
			cfg.Site.Description = *raw.Site.Description
		}
		if raw.Site.URL != nil {
			cfg.Site.URL = *raw.Site.URL
		}
		if raw.Site.TemplateDir != nil {
			cfg.Site.TemplateDir = *raw.Site.TemplateDir
		}
		if raw.Site.TemplateDirSnake != nil {
			cfg.Site.TemplateDir = *raw.Site.TemplateDirSnake
		}
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
		if raw.Content.Metadata != nil {
			cfg.Content.Metadata = make(map[string]ContentMetadata, len(raw.Content.Metadata))
			for key, metadata := range raw.Content.Metadata {
				cfg.Content.Metadata[key] = contentMetadataFromRaw(metadata)
			}
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

func contentMetadataFromRaw(raw rawContentMetadata) ContentMetadata {
	var metadata ContentMetadata
	if raw.Title != nil {
		metadata.Title = *raw.Title
	}
	if raw.Date != nil {
		metadata.Date = *raw.Date
	}
	if raw.Tags != nil {
		metadata.Tags = cloneStrings(*raw.Tags)
	}
	if raw.Author != nil {
		metadata.Author = *raw.Author
	}
	return metadata
}

func normalizeContentMetadata(metadata map[string]ContentMetadata) (map[string]ContentMetadata, error) {
	if len(metadata) == 0 {
		return map[string]ContentMetadata{}, nil
	}

	normalized := make(map[string]ContentMetadata, len(metadata))
	for key, value := range metadata {
		normalizedKey, err := normalizeMetadataKey(key)
		if err != nil {
			return nil, err
		}
		if _, exists := normalized[normalizedKey]; exists {
			return nil, fmt.Errorf("content.metadata key %q normalizes to duplicate %q", key, normalizedKey)
		}
		normalizedValue, err := normalizeMetadataValue(normalizedKey, value)
		if err != nil {
			return nil, err
		}
		normalized[normalizedKey] = normalizedValue
	}
	return normalized, nil
}

func normalizeMetadataKey(key string) (string, error) {
	key = strings.TrimSpace(key)
	if err := validateMetadataKey(key); err != nil {
		return "", err
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(key))), nil
}

func normalizeMetadataValue(key string, metadata ContentMetadata) (ContentMetadata, error) {
	metadata.Title = strings.TrimSpace(metadata.Title)
	metadata.Author = strings.TrimSpace(metadata.Author)
	if len(metadata.Tags) > 0 {
		tags := make([]string, len(metadata.Tags))
		for i, tag := range metadata.Tags {
			tags[i] = strings.TrimSpace(tag)
			if tags[i] == "" {
				return ContentMetadata{}, fmt.Errorf("content.metadata[%q].tags[%d] is required", key, i)
			}
		}
		metadata.Tags = tags
	}
	return metadata, nil
}

func validateContentMetadata(metadata map[string]ContentMetadata) error {
	for key, value := range metadata {
		if err := validateMetadataKey(strings.TrimSpace(key)); err != nil {
			return err
		}
		if err := validateMetadataValue(key, value); err != nil {
			return err
		}
	}
	return nil
}

func validateMetadataKey(key string) error {
	if key == "" {
		return fmt.Errorf("content.metadata key is required")
	}
	if strings.ContainsRune(key, 0) {
		return fmt.Errorf("content.metadata key %q contains a null byte", key)
	}
	if strings.Contains(key, `\`) {
		return fmt.Errorf("content.metadata key %q must use slash separators", key)
	}
	if path.IsAbs(key) || filepath.IsAbs(filepath.FromSlash(key)) || isWindowsAbsPath(key) {
		return fmt.Errorf("content.metadata key %q must be project-relative", key)
	}
	for _, component := range strings.Split(key, "/") {
		if component == ".." {
			return fmt.Errorf("content.metadata key %q must not contain parent directory components", key)
		}
	}
	if path.Clean(key) == "." {
		return fmt.Errorf("content.metadata key %q must reference a file", key)
	}
	return nil
}

func validateMetadataValue(key string, metadata ContentMetadata) error {
	date := metadata.Date
	if date != "" {
		parsed, err := time.Parse("2006-01-02", date)
		if err != nil || parsed.Format("2006-01-02") != date {
			return fmt.Errorf("content.metadata[%q].date must be a real YYYY-MM-DD date", key)
		}
	}
	for i, tag := range metadata.Tags {
		if strings.TrimSpace(tag) == "" {
			return fmt.Errorf("content.metadata[%q].tags[%d] is required", key, i)
		}
	}
	return nil
}

func isWindowsAbsPath(value string) bool {
	if len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return strings.HasPrefix(value, `\\`)
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

func normalizeOptionalPath(baseDir, value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if err := validateOptionalPath(field, value); err != nil {
		return "", err
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(baseDir, value)
	}
	return filepath.Clean(value), nil
}

func normalizeSiteURL(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if err := validateSiteURL(value); err != nil {
		return "", err
	}
	return value, nil
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

func validateOptionalPath(field, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s contains a null byte", field)
	}
	return nil
}

func validateSiteURL(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("site.url must be an absolute http or https URL with a host: %w", err)
	}
	scheme := strings.ToLower(parsed.Scheme)
	if !parsed.IsAbs() || (scheme != "http" && scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("site.url must be an absolute http or https URL with a host")
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
