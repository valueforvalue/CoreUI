package registry

import (
	"os"
	"path/filepath"
	"testing"
)

// writePlugin creates a temporary JSON plugin file in dir with the given content.
func writePlugin(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writePlugin: %v", err)
	}
}

func TestLoadPluginsFromDir_ValidPlugin(t *testing.T) {
	dir := t.TempDir()
	writePlugin(t, dir, "rating.json", `{
		"components": [
			{
				"name": "Rating",
				"has_children": false,
				"attributes": {
					"id":    { "type": "string", "required": true },
					"value": { "type": "int" },
					"max":   { "type": "int" }
				}
			}
		]
	}`)

	// Snapshot state before loading so we can restore it.
	before := len(componentSpecs)
	collisions := LoadPluginsFromDir(dir)
	after := len(componentSpecs)

	if len(collisions) != 0 {
		t.Fatalf("expected no collisions, got %v", collisions)
	}
	if after != before+1 {
		t.Fatalf("expected one new component, componentSpecs grew by %d", after-before)
	}
	if _, ok := componentSpecs["Rating"]; !ok {
		t.Fatal("expected Rating to be registered")
	}
	if !IsPluginComponent("Rating") {
		t.Fatal("IsPluginComponent should return true for Rating")
	}

	// Cleanup: remove the test component so other tests are not affected.
	delete(componentSpecs, "Rating")
}

func TestLoadPluginsFromDir_CollisionIsRejected(t *testing.T) {
	dir := t.TempDir()
	// "Box" is a core component — the plugin must be rejected.
	writePlugin(t, dir, "collision.json", `{
		"components": [
			{
				"name": "Box",
				"has_children": true,
				"attributes": {
					"id": { "type": "string", "required": true }
				}
			}
		]
	}`)

	coreBefore := componentSpecs["Box"]
	collisions := LoadPluginsFromDir(dir)

	if len(collisions) != 1 || collisions[0] != "Box" {
		t.Fatalf("expected collision on Box, got %v", collisions)
	}
	// Core-First: the core component must be unchanged.
	if componentSpecs["Box"].Name != coreBefore.Name {
		t.Fatal("core Box component was overwritten by plugin — Core-First rule violated")
	}
	if IsPluginComponent("Box") {
		t.Fatal("IsPluginComponent should return false for core component Box")
	}
}

func TestLoadPluginsFromDir_MissingDirIsIgnored(t *testing.T) {
	collisions := LoadPluginsFromDir("/nonexistent/path/that/does/not/exist")
	if len(collisions) != 0 {
		t.Fatalf("expected no collisions for missing dir, got %v", collisions)
	}
}

func TestIsPluginComponent_CoreComponentReturnsFalse(t *testing.T) {
	for _, name := range []string{"View", "Stack", "Grid", "Box", "Text", "Trigger"} {
		if IsPluginComponent(name) {
			t.Errorf("IsPluginComponent(%q) should be false for core component", name)
		}
	}
}

func TestIsPluginComponent_UnknownComponentReturnsFalse(t *testing.T) {
	if IsPluginComponent("DoesNotExist") {
		t.Fatal("IsPluginComponent should return false for unknown component")
	}
}
