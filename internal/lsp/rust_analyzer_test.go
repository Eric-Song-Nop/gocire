package lsp

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRustAnalyzer(t *testing.T) {
	// Check if rust-analyzer is available
	cmdPath, err := exec.LookPath("rust-analyzer")
	if err != nil {
		t.Skip("rust-analyzer not found in PATH")
	}

	// Create a temporary directory for the Rust project
	tmpDir, err := os.MkdirTemp("", "gocire-rust-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create src directory
	srcDir := filepath.Join(tmpDir, "src")
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a simple main.rs
	mainRsContent := `fn main() {
    let message = "Hello, world!";
    println!("{}", message);
}`
	mainRsPath := filepath.Join(srcDir, "main.rs")
	if err := os.WriteFile(mainRsPath, []byte(mainRsContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create Cargo.toml
	cargoTomlContent := `[package]
name = "hello_world"
version = "0.1.0"
edition = "2021"

[dependencies]
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoTomlContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Initialize Client
	client, err := NewClient(ctx, cmdPath, []string{})
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown()

	// Initialize LSP
	if err := client.Initialize(tmpDir); err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	// Notify DidOpen
	if err := client.DidOpen(mainRsPath, "rust", mainRsContent); err != nil {
		t.Fatalf("DidOpen failed: %v", err)
	}

	// Test Hover with retry
	var hover *Hover
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for Hover response")
		case <-ticker.C:
			hover, err = client.Hover(mainRsPath, 2, 5)
			if err == nil && hover != nil && (strings.Contains(hover.Contents.Value, "println") || strings.Contains(hover.Contents.Value, "macro")) {
				goto Done
			}
		}
	}
Done:

	t.Logf("Hover content kind: %s", hover.Contents.Kind)
	t.Logf("Hover content value: %s", hover.Contents.Value)
}
