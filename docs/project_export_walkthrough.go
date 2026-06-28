package docs

import (
	"fmt"

	cire "github.com/Eric-Song-Nop/gocire/internal"
	projectconfig "github.com/Eric-Song-Nop/gocire/internal/config"
	"github.com/Eric-Song-Nop/gocire/internal/project"
	"github.com/sourcegraph/scip/bindings/go/scip"
)

// # Project export starts with a site-shaped scan
//
// A docsite export begins by loading `.gocire.yml`, scanning the repository, and
// classifying each source file as docs, blog, or source. This page is regular Go
// code, but the standalone comments become prose in the generated docs page.
//
// The identifiers in the code below intentionally point at real gocire APIs.
// In the generated Astro site, jump-to-definition should take readers to the
// source exploration page for each definition.
func BuildProjectExportWalkthrough(configPath string) (*ProjectExportWalkthrough, error) {
	cfg, err := projectconfig.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("load project config: %w", err)
	}

	files, err := project.Scan(*cfg)
	if err != nil {
		return nil, fmt.Errorf("scan project files: %w", err)
	}

	manifest, err := buildSourceRoutes(*cfg, files)
	if err != nil {
		return nil, fmt.Errorf("build source routes: %w", err)
	}

	return &ProjectExportWalkthrough{
		Config:   cfg,
		Files:    files,
		Manifest: manifest,
	}, nil
}

// The route manifest is deliberately independent from LSP. A file can link to a
// definition in another file as long as both paths share this stable route
// mapping. That keeps individual files independently generatable.
func buildSourceRoutes(cfg projectconfig.ProjectConfig, files []project.SourceFile) (cire.SourceRouteManifest, error) {
	sourcePaths := make([]string, 0, len(files))
	for _, file := range files {
		sourcePaths = append(sourcePaths, file.AbsPath)
	}

	return cire.NewSourceRouteManifestWithPrefix(cfg.Project.Root, cfg.Source.RoutePrefix, sourcePaths)
}

// Position anchors are the other half of cross-file navigation. LSP returns
// zero-based positions, while generated anchors use one-based line and column
// labels such as `#L12C4`.
func ExampleDefinitionAnchor(line, column int32) string {
	return cire.LineColumnAnchor(scip.Position{
		Line:      line,
		Character: column,
	})
}

type ProjectExportWalkthrough struct {
	Config   *projectconfig.ProjectConfig
	Files    []project.SourceFile
	Manifest cire.SourceRouteManifest
}
