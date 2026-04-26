package tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestSentinelGuardrails(t *testing.T) {
	root := repoRoot(t)

	t.Run("golden kitchen sink json", func(t *testing.T) {
		tempDir := t.TempDir()
		actualPath := filepath.Join(tempDir, "kitchen_sink.json")
		fixturePath := filepath.Join("tests", "fixtures", "kitchen_sink.cui")

		runGo(t, root, "run", "./cmd/corec", "-o", actualPath, fixturePath)

		expected := readJSONFixture(t, filepath.Join(root, "tests", "golden", "kitchen_sink.json"))
		actual := readJSONFixture(t, actualPath)
		normalImageSrc := imageSourceForID(t, actual, "hero_image")
		if normalImageSrc != "coreui-logo.svg" {
			t.Fatalf("expected normal mode image src to remain a short file path, got %q", normalImageSrc)
		}

		normalizeMetadataTimestamp(expected)
		normalizeMetadataTimestamp(actual)

		if !reflect.DeepEqual(expected, actual) {
			expectedJSON := marshalIndented(t, expected)
			actualJSON := marshalIndented(t, actual)
			t.Fatalf("kitchen_sink output drifted from tests\\golden\\kitchen_sink.json\nexpected:\n%s\nactual:\n%s", expectedJSON, actualJSON)
		}

		if got := nodeTypeForID(t, actual, "traffic_graph"); got != "Graph" {
			t.Fatalf("expected kitchen sink graph node, got %q", got)
		}
	})

	t.Run("graph fixture json", func(t *testing.T) {
		tempDir := t.TempDir()
		actualPath := filepath.Join(tempDir, "Graph_fixture.json")
		fixturePath := filepath.Join("testdata", "Graph_fixture.cui")

		runGo(t, root, "run", "./cmd/corec", "-o", actualPath, fixturePath)

		expected := readJSONFixture(t, filepath.Join(root, "testdata", "Graph_fixture.json"))
		actual := readJSONFixture(t, actualPath)

		normalizeMetadataTimestamp(expected)
		normalizeMetadataTimestamp(actual)
		normalizeMetadataVersion(expected)
		normalizeMetadataVersion(actual)

		if !reflect.DeepEqual(expected, actual) {
			expectedJSON := marshalIndented(t, expected)
			actualJSON := marshalIndented(t, actual)
			t.Fatalf("Graph fixture output drifted from testdata\\Graph_fixture.json\nexpected:\n%s\nactual:\n%s", expectedJSON, actualJSON)
		}
	})

	t.Run("standalone html essentials", func(t *testing.T) {
		tempDir := t.TempDir()
		htmlPath := filepath.Join(tempDir, "kitchen_sink.html")
		fixturePath := filepath.Join("tests", "fixtures", "kitchen_sink.cui")

		runGo(t, root, "run", "./cmd/corec", "-standalone", "-o", htmlPath, fixturePath)

		htmlBytes, err := os.ReadFile(htmlPath)
		if err != nil {
			t.Fatalf("read standalone output: %v", err)
		}

		html := string(htmlBytes)
		requiredSnippets := []string{
			`<div id="coreui-root"></div>`,
			`class CoreUI`,
			`case "Graph":`,
			`const jsonData =`,
			`document.addEventListener("DOMContentLoaded"`,
		}
		for _, snippet := range requiredSnippets {
			if !strings.Contains(html, snippet) {
				t.Fatalf("standalone output missing required snippet: %s", snippet)
			}
		}

		match := regexp.MustCompile(`(?s)const jsonData = (.*?);\s*document\.addEventListener\("DOMContentLoaded"`).FindStringSubmatch(html)
		if len(match) != 2 {
			t.Fatal("standalone output is missing an extractable jsonData payload")
		}
		if !json.Valid([]byte(match[1])) {
			t.Fatal("standalone output contains invalid JSON in const jsonData")
		}

		var embedded any
		if err := json.Unmarshal([]byte(match[1]), &embedded); err != nil {
			t.Fatalf("decode standalone jsonData: %v", err)
		}
		embeddedImageSrc := imageSourceForID(t, embedded, "hero_image")
		if !strings.HasPrefix(embeddedImageSrc, "data:image/svg+xml;base64,") {
			t.Fatalf("expected standalone image src to be an embedded data URL, got %q", embeddedImageSrc)
		}
		if len(embeddedImageSrc) <= len("coreui-logo.svg") {
			t.Fatalf("expected standalone image src to be longer than the normal file path, got %d bytes", len(embeddedImageSrc))
		}
		if got := nodeTypeForID(t, embedded, "traffic_graph"); got != "Graph" {
			t.Fatalf("expected standalone payload graph node, got %q", got)
		}
	})

	t.Run("components docs in sync", func(t *testing.T) {
		componentsPath := filepath.Join(root, "COMPONENTS.md")
		original, err := os.ReadFile(componentsPath)
		if err != nil {
			t.Fatalf("read COMPONENTS.md: %v", err)
		}

		t.Cleanup(func() {
			if writeErr := os.WriteFile(componentsPath, original, 0o644); writeErr != nil {
				t.Fatalf("restore COMPONENTS.md: %v", writeErr)
			}
		})

		runGo(t, root, "run", "./cmd/coredoc")

		regenerated, err := os.ReadFile(componentsPath)
		if err != nil {
			t.Fatalf("read regenerated COMPONENTS.md: %v", err)
		}

		if !bytes.Equal(original, regenerated) {
			t.Fatal("COMPONENTS.md is out of date; run `go run ./cmd/coredoc` and commit the result")
		}
	})
}

func repoRoot(t *testing.T) string {
	t.Helper()

	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func runGo(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go %s failed: %v\n%s", strings.Join(args, " "), err, string(output))
	}
}

func readJSONFixture(t *testing.T, path string) any {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read JSON fixture %s: %v", path, err)
	}

	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode JSON fixture %s: %v", path, err)
	}
	return decoded
}

func normalizeMetadataTimestamp(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}

	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return
	}

	if _, ok := metadata["compiled_at"].(string); ok {
		metadata["compiled_at"] = "<normalized>"
	}
}

func marshalIndented(t *testing.T, value any) string {
	t.Helper()

	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatalf("marshal JSON: %v", err)
	}
	return string(data)
}

func normalizeMetadataVersion(value any) {
	object, ok := value.(map[string]any)
	if !ok {
		return
	}

	metadata, ok := object["metadata"].(map[string]any)
	if !ok {
		return
	}

	if _, ok := metadata["version"].(string); ok {
		metadata["version"] = "<normalized>"
	}
}

func imageSourceForID(t *testing.T, value any, id string) string {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object while searching for %s", id)
	}

	tree, ok := object["tree"]
	if !ok {
		t.Fatalf("missing tree while searching for %s", id)
	}

	node := findNodeByID(tree, id)
	if node == nil {
		t.Fatalf("missing node with id %s", id)
	}

	attributes, ok := node["attributes"].(map[string]any)
	if !ok {
		t.Fatalf("node %s is missing attributes", id)
	}

	src, ok := attributes["src"].(string)
	if !ok {
		t.Fatalf("node %s is missing a string src attribute", id)
	}
	return src
}

func nodeTypeForID(t *testing.T, value any, id string) string {
	t.Helper()

	object, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected JSON object while searching for %s", id)
	}
	tree, ok := object["tree"]
	if !ok {
		t.Fatalf("missing tree while searching for %s", id)
	}

	node := findNodeByID(tree, id)
	if node == nil {
		t.Fatalf("missing node with id %s", id)
	}

	nodeType, ok := node["type"].(string)
	if !ok {
		t.Fatalf("node %s is missing a string type", id)
	}
	return nodeType
}

func findNodeByID(value any, id string) map[string]any {
	node, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	if nodeID, _ := node["id"].(string); nodeID == id {
		return node
	}

	children, ok := node["children"].([]any)
	if !ok {
		return nil
	}
	for _, child := range children {
		if match := findNodeByID(child, id); match != nil {
			return match
		}
	}
	return nil
}
