package compiler

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/valueforvalue/coreui/pkg/generator"
	"github.com/valueforvalue/coreui/pkg/parser"
)

type Options struct {
	Timestamp  time.Time
	Version    string
	Standalone bool
}

func CompileSource(name, source string, options Options) ([]byte, error) {
	_ = name

	if options.Timestamp.IsZero() {
		options.Timestamp = time.Now().UTC()
	}
	if options.Version == "" {
		options.Version = "dev"
	}

	document, err := parser.New(source).ParseDocument()
	if err != nil {
		return nil, err
	}

	if options.Standalone {
		embedImageSources(document, assetBaseDir(name))
	}

	output := generator.Build(document, options.Timestamp, options.Version)
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}

	return append(data, '\n'), nil
}

func CompileFile(path string, options Options) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return CompileSource(path, string(data), options)
}

func embedImageSources(document *ast.Document, baseDir string) {
	if document == nil || document.Tree == nil {
		return
	}
	rewriteImageSources(document.Tree, baseDir)
}

func rewriteImageSources(node *ast.Node, baseDir string) {
	if node == nil {
		return
	}

	if node.Type == "Image" {
		if value, ok := node.Attributes["src"]; ok && value.Kind == ast.StringKind {
			if source, ok := value.Data.(string); ok {
				if embedded, ok := imageDataURL(source, baseDir); ok {
					node.Attributes["src"] = ast.Value{Kind: ast.StringKind, Data: embedded}
				}
			}
		}
	}

	for _, child := range node.Children {
		rewriteImageSources(child, baseDir)
	}
}

func imageDataURL(source, baseDir string) (string, bool) {
	source = strings.TrimSpace(source)
	if source == "" || strings.HasPrefix(source, "data:") || strings.Contains(source, "://") {
		return "", false
	}

	path := filepath.FromSlash(source)
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}

	mimeType := mimeTypeFor(path, data)
	if mimeType == "" {
		return "", false
	}

	return "data:" + mimeType + ";base64," + base64.StdEncoding.EncodeToString(data), true
}

func mimeTypeFor(path string, data []byte) string {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case ".svg":
		return "image/svg+xml"
	default:
		mimeType := http.DetectContentType(data)
		if mimeType == "application/octet-stream" {
			return ""
		}
		return mimeType
	}
}

func assetBaseDir(name string) string {
	if name == "" {
		return "."
	}
	return filepath.Dir(name)
}
