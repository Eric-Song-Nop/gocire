package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/Eric-Song-Nop/gocire/internal"
)

type testTokenAnalyzer struct{}

func (a testTokenAnalyzer) Analyze(ctx context.Context, content []byte) ([]internal.TokenInfo, error) {
	return nil, nil
}

func TestNewPipelinePassesExplicitLSPRequest(t *testing.T) {
	root := t.TempDir()
	sourcePath := filepath.Join(root, "main.go")
	writeProjectTestFile(t, sourcePath, "package main\n")

	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "pipeline")

	var gotCtx context.Context
	var gotReq PipelineLSPRequest
	pipeline, err := NewPipelineWithOptions(&Config{
		SrcPath:    sourcePath,
		AbsSrcPath: sourcePath,
		Lang:       "go",
		UseLSP:     true,
		LSPRoot:    root,
		Format:     "markdown",
	}, PipelineOptions{
		Context: ctx,
		LSPAnalyzerFactory: func(ctx context.Context, req PipelineLSPRequest) (TokenAnalyzer, error) {
			gotCtx = ctx
			gotReq = req
			return testTokenAnalyzer{}, nil
		},
	})
	if err != nil {
		t.Fatalf("NewPipelineWithOptions returned error: %v", err)
	}
	if pipeline == nil {
		t.Fatal("NewPipelineWithOptions returned nil pipeline")
	}

	if gotCtx != ctx {
		t.Fatal("LSPAnalyzerFactory did not receive PipelineOptions.Context")
	}
	if gotReq.SourcePath != sourcePath {
		t.Fatalf("SourcePath = %q, want %q", gotReq.SourcePath, sourcePath)
	}
	if gotReq.Language != "go" {
		t.Fatalf("Language = %q, want go", gotReq.Language)
	}
	if gotReq.WorkspaceRoot != root {
		t.Fatalf("WorkspaceRoot = %q, want %q", gotReq.WorkspaceRoot, root)
	}
}
