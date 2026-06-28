package project

import (
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/languages"
)

type PageKind string

const (
	PageKindDocs   PageKind = "docs"
	PageKindBlog   PageKind = "blog"
	PageKindSource PageKind = "source"
)

type SourceFile struct {
	AbsPath  string
	RelPath  string
	Language string
	Kind     PageKind
}

type scanConfig struct {
	root    string
	docs    []string
	blogs   []string
	include []string
	exclude []string
}

func Scan(cfg config.ProjectConfig) ([]SourceFile, error) {
	scanCfg, err := readScanConfig(cfg)
	if err != nil {
		return nil, err
	}

	root := scanCfg.root
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve project root: %w", err)
	}

	var files []SourceFile
	err = filepath.WalkDir(absRoot, func(absPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := relativeSlashPath(absRoot, absPath)
		if err != nil {
			return err
		}
		if relPath == "" {
			return nil
		}

		excluded, err := matchesAny(scanCfg.exclude, relPath)
		if err != nil {
			return err
		}
		if excluded {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if entry.IsDir() {
			return nil
		}

		included, err := isIncluded(scanCfg.include, relPath)
		if err != nil {
			return err
		}
		if !included {
			return nil
		}

		language, err := languages.DetectLanguage(absPath)
		if err != nil {
			return nil
		}

		kind, err := classify(relPath, scanCfg.docs, scanCfg.blogs)
		if err != nil {
			return err
		}

		files = append(files, SourceFile{
			AbsPath:  absPath,
			RelPath:  relPath,
			Language: language,
			Kind:     kind,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return files, nil
}

func readScanConfig(cfg config.ProjectConfig) (scanConfig, error) {
	root := cfg.Project.Root
	if strings.TrimSpace(root) == "" {
		return scanConfig{}, fmt.Errorf("project config missing Project.Root")
	}

	return scanConfig{
		root:    root,
		docs:    contentGlobPatterns(root, []string{cfg.Content.Docs}, "docs"),
		blogs:   contentGlobPatterns(root, []string{cfg.Content.Blogs}, "blogs"),
		include: normalizePatterns(cfg.Source.Include),
		exclude: normalizePatterns(cfg.Source.Exclude),
	}, nil
}

func contentGlobPatterns(root string, configured []string, fallback string) []string {
	if len(configured) == 0 {
		configured = []string{fallback}
	}

	absRoot := root
	if abs, err := filepath.Abs(root); err == nil {
		absRoot = abs
	}

	patterns := make([]string, 0, len(configured))
	for _, pattern := range configured {
		pattern = relativeContentPattern(absRoot, pattern)
		pattern = normalizePath(pattern)
		if pattern == "" {
			pattern = "**"
		}
		if !containsGlobMeta(pattern) {
			pattern = path.Join(pattern, "**")
		}
		patterns = append(patterns, pattern)
	}
	return normalizePatterns(patterns)
}

func relativeContentPattern(root, pattern string) string {
	if !filepath.IsAbs(pattern) {
		return pattern
	}

	relPath, err := filepath.Rel(root, pattern)
	if err != nil {
		return pattern
	}
	return relPath
}

func containsGlobMeta(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func classify(relPath string, docs []string, blogs []string) (PageKind, error) {
	isDocs, err := matchesAny(docs, relPath)
	if err != nil {
		return "", err
	}
	if isDocs {
		return PageKindDocs, nil
	}

	isBlog, err := matchesAny(blogs, relPath)
	if err != nil {
		return "", err
	}
	if isBlog {
		return PageKindBlog, nil
	}

	return PageKindSource, nil
}

func isIncluded(include []string, relPath string) (bool, error) {
	if len(include) == 0 {
		return true, nil
	}
	return matchesAny(include, relPath)
}

func matchesAny(patterns []string, relPath string) (bool, error) {
	for _, pattern := range patterns {
		matched, err := matchGlob(pattern, relPath)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func matchGlob(pattern, relPath string) (bool, error) {
	pattern = normalizePath(pattern)
	relPath = normalizePath(relPath)
	if pattern == "" || relPath == "" {
		return false, nil
	}

	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(relPath, "/")
	return matchGlobParts(patternParts, pathParts)
}

func matchGlobParts(patternParts, pathParts []string) (bool, error) {
	if len(patternParts) == 0 {
		return len(pathParts) == 0, nil
	}

	patternPart := patternParts[0]
	if patternPart == "**" {
		matched, err := matchGlobParts(patternParts[1:], pathParts)
		if err != nil || matched {
			return matched, err
		}
		for i := range pathParts {
			matched, err := matchGlobParts(patternParts[1:], pathParts[i+1:])
			if err != nil || matched {
				return matched, err
			}
		}
		return false, nil
	}

	if len(pathParts) == 0 {
		return false, nil
	}

	matched, err := path.Match(patternPart, pathParts[0])
	if err != nil {
		return false, fmt.Errorf("invalid glob pattern %q: %w", strings.Join(patternParts, "/"), err)
	}
	if !matched {
		return false, nil
	}
	return matchGlobParts(patternParts[1:], pathParts[1:])
}

func relativeSlashPath(root, filePath string) (string, error) {
	relPath, err := filepath.Rel(root, filePath)
	if err != nil {
		return "", fmt.Errorf("relative path for %q: %w", filePath, err)
	}
	return normalizePath(relPath), nil
}

func normalizePatterns(patterns []string) []string {
	normalized := make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		pattern = normalizePath(pattern)
		if pattern != "" {
			normalized = append(normalized, pattern)
		}
	}
	return normalized
}

func normalizePath(filePath string) string {
	filePath = strings.ReplaceAll(filepath.ToSlash(filePath), "\\", "/")
	filePath = strings.TrimPrefix(filePath, "./")
	filePath = path.Clean(filePath)
	if filePath == "." {
		return ""
	}
	return filePath
}
