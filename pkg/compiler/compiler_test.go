package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/valueforvalue/coreui/pkg/compiler"
)

func TestGoldenFiles(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")
	entries, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}

	options := compiler.Options{
		Timestamp: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		Version:   "test",
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".cui" {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".cui")
		t.Run(name, func(t *testing.T) {
			inputPath := filepath.Join(testdataDir, entry.Name())
			expectedPath := filepath.Join(testdataDir, name+".json")

			input, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("read input: %v", err)
			}
			expected, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("read golden file: %v", err)
			}

			actual, err := compiler.CompileSource(inputPath, string(input), options)
			if err != nil {
				t.Fatalf("compile source: %v", err)
			}

			expectedText := strings.ReplaceAll(string(expected), "\r\n", "\n")
			actualText := strings.ReplaceAll(string(actual), "\r\n", "\n")

			if actualText != expectedText {
				t.Fatalf("golden mismatch\nexpected:\n%s\nactual:\n%s", expectedText, actualText)
			}
		})
	}
}
