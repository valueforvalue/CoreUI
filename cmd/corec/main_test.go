package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"coreui/pkg/compiler"
)

func TestRunInitCreatesTemplateAndInstructions(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	var output bytes.Buffer
	if err := runInit([]string{"hello_world"}, &output); err != nil {
		t.Fatalf("runInit failed: %v", err)
	}

	content, err := os.ReadFile("hello_world.cui")
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	for _, want := range []string{
		`Theme(id="standard")`,
		`View(id="root", title="New CoreUI Project")`,
		`Stack(id="main_stack", dir="v", gap=20px)`,
		`Box(id="panel_box", padding=20px, background="panel")`,
		`Image(id="hero_image", src="placeholder.png", width=100px, alt="Placeholder image")`,
		`Trigger(id="notify_button", label="Click Me", action="app:notify(msg=\"Hello from CoreUI!\")")`,
		`[Success] Initialized 'hello_world.cui'`,
		`Run 'corec -s hello_world.cui' to bundle it.`,
		`Open 'hello_world.html' in any browser.`,
	} {
		if !strings.Contains(string(content)+output.String(), want) {
			t.Fatalf("expected init output to contain %q", want)
		}
	}
}

func TestRunInitRefusesOverwrite(t *testing.T) {
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	if err := os.WriteFile("hello_world.cui", []byte("existing"), 0o644); err != nil {
		t.Fatalf("seed existing file: %v", err)
	}

	err = runInit([]string{"hello_world"}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected overwrite protection error")
	}
	if got, want := err.Error(), "File hello_world.cui already exists. Aborting to prevent overwrite."; got != want {
		t.Fatalf("unexpected error message: got %q want %q", got, want)
	}
}

func TestInitTemplateCompilesToStandaloneHTML(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "hello_world.cui")
	if err := os.WriteFile(filePath, []byte(initTemplate), 0o644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	jsonData, err := compiler.CompileFile(filePath, compiler.Options{Version: version, Standalone: true})
	if err != nil {
		t.Fatalf("compile standalone template: %v", err)
	}

	html, err := buildStandaloneHTML(jsonData)
	if err != nil {
		t.Fatalf("build standalone html: %v", err)
	}

	for _, want := range []string{
		`<div id="coreui-root"></div>`,
		`class CoreUI`,
		`const jsonData =`,
		`document.addEventListener("DOMContentLoaded"`,
	} {
		if !strings.Contains(string(html), want) {
			t.Fatalf("expected standalone html to contain %q", want)
		}
	}
}
