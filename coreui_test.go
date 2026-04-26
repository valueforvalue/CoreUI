package coreui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRendererRejectsInvalidJSON(t *testing.T) {
	if _, err := NewRenderer([]byte("{")); err == nil {
		t.Fatal("expected invalid JSON to be rejected")
	}
}

func TestNewRendererRejectsBlueprintWithoutTree(t *testing.T) {
	if _, err := NewRenderer([]byte(`{"index":{},"metadata":{}}`)); err == nil {
		t.Fatal("expected blueprint without tree to be rejected")
	}
}

func TestStandaloneHTMLIncludesRendererAndData(t *testing.T) {
	renderer, err := NewRenderer([]byte(`{"tree":{"type":"View","id":"root"},"index":{"root":{"path":"/tree","type":"View"}},"metadata":{"compiled_at":"2026-01-01T00:00:00Z","version":"dev","registry_version":"1.3.0"}}`))
	if err != nil {
		t.Fatalf("NewRenderer failed: %v", err)
	}

	html, err := renderer.StandaloneHTML(map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("StandaloneHTML failed: %v", err)
	}

	for _, want := range []string{
		`window.CoreUI = CoreUI;`,
		`window.CoreUIData = {"message":"hello"};`,
		`const jsonData =`,
		`document.addEventListener("DOMContentLoaded"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected standalone HTML to contain %q", want)
		}
	}
}

func TestRenderIncludesRendererAndData(t *testing.T) {
	html, err := Render([]byte(`{"tree":{"type":"View","id":"root"},"index":{"root":{"path":"/tree","type":"View"}},"metadata":{"compiled_at":"2026-01-01T00:00:00Z","version":"dev","registry_version":"1.3.0"}}`), map[string]any{"message": "hello"})
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	for _, want := range []string{
		`window.CoreUI = CoreUI;`,
		`window.CoreUIData = {"message":"hello"};`,
		`const jsonData =`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered HTML to contain %q", want)
		}
	}
}

func TestCompileCompilesFile(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "simple.cui")
	source := `View(id="root", title="Sample") {}`
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	blueprint, err := Compile(sourcePath, CompileOptions{Version: "test"})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	for _, want := range []string{`"tree"`, `"id": "root"`, `"version": "test"`} {
		if !strings.Contains(string(blueprint), want) {
			t.Fatalf("expected compiled blueprint to contain %q", want)
		}
	}
}
