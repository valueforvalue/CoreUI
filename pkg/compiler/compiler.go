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
	"github.com/valueforvalue/coreui/pkg/flow"
	"github.com/valueforvalue/coreui/pkg/flowgen"
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

// FlowResult holds the outputs produced by CompileWithFlow.
type FlowResult struct {
	// BlueprintJSON is the compiled .cui JSON blueprint.
	BlueprintJSON []byte
	// FlowJS is the generated CoreFlow state engine JavaScript.
	FlowJS string
	// Bindings are the reactive text/input bindings detected in the blueprint.
	Bindings []flowgen.Binding
}

// CompileWithFlow compiles a .cui source together with a .flow source.
// It validates that all On(id=...) references in the .flow file correspond to
// component IDs in the .cui blueprint (Wiring Gap validation).
//
// When options.Standalone is true, the returned FlowResult.FlowJS is ready to
// be injected into the standalone HTML via renderers.BuildStandaloneHTMLWithFlow.
func CompileWithFlow(name, cuiSource, flowSource string, options Options) (*FlowResult, error) {
	if options.Timestamp.IsZero() {
		options.Timestamp = time.Now().UTC()
	}
	if options.Version == "" {
		options.Version = "dev"
	}

	// Parse and compile the .cui document.
	document, err := parser.New(cuiSource).ParseDocument()
	if err != nil {
		return nil, err
	}

	if options.Standalone {
		embedImageSources(document, assetBaseDir(name))
	}

	// Parse the .flow document first so we can pass it to BuildWithFlow.
	flowDoc, err := flow.ParseFlow(flowSource)
	if err != nil {
		return nil, err
	}

	// Build the JSON output, embedding initial flow state for GOTH parity.
	output := generator.BuildWithFlow(document, flowDoc, options.Timestamp, options.Version)
	blueprintJSON, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return nil, err
	}
	blueprintJSON = append(blueprintJSON, '\n')

	// Collect UI IDs from the compiled blueprint.
	uiIDs := collectUIIDs(output.Index)

	// Collect reactive bindings by traversing the generator node tree directly,
	// avoiding a JSON round-trip.
	bindings := collectBindingsFromNode(output.Tree)

	// Generate the JS state engine (also validates wiring).
	flowJS, err := flowgen.Generate(flowDoc, uiIDs, bindings, flowgen.Options{
		RendererVar: "_cuiRenderer",
	})
	if err != nil {
		return nil, err
	}

	return &FlowResult{
		BlueprintJSON: blueprintJSON,
		FlowJS:        flowJS,
		Bindings:      bindings,
	}, nil
}

// CompileFileWithFlow reads both a .cui file and a .flow file and compiles them.
func CompileFileWithFlow(cuiPath, flowPath string, options Options) (*FlowResult, error) {
	cuiData, err := os.ReadFile(cuiPath)
	if err != nil {
		return nil, err
	}
	flowData, err := os.ReadFile(flowPath)
	if err != nil {
		return nil, err
	}
	return CompileWithFlow(cuiPath, string(cuiData), string(flowData), options)
}

// collectUIIDs extracts the set of all component IDs from a generator index.
func collectUIIDs(index map[string]generator.IndexEntry) map[string]bool {
	ids := make(map[string]bool, len(index))
	for id := range index {
		ids[id] = true
	}
	return ids
}

// collectBindingsFromNode traverses a generator.Node tree and returns all
// flow: attribute bindings without a JSON serialisation round-trip.
func collectBindingsFromNode(node *generator.Node) []flowgen.Binding {
	if node == nil {
		return nil
	}
	var bindings []flowgen.Binding
	walkNode(node, &bindings)
	return bindings
}

func walkNode(node *generator.Node, out *[]flowgen.Binding) {
	if node.ID != "" && node.Attributes != nil {
		for attrName, attrVal := range node.Attributes {
			s, ok := attrVal.(string)
			if ok && len(s) > 5 && s[:5] == "flow:" {
				varName := s[5:]
				if varName != "" {
					*out = append(*out, flowgen.Binding{
						ElementID: node.ID,
						AttrName:  attrName,
						VarName:   varName,
					})
				}
			}
		}
	}
	for _, child := range node.Children {
		walkNode(child, out)
	}
}
