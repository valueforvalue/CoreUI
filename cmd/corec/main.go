package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/valueforvalue/coreui/pkg/compiler"
	"github.com/valueforvalue/coreui/pkg/diag"
	"github.com/valueforvalue/coreui/pkg/docs"
	"github.com/valueforvalue/coreui/pkg/editor"
	"github.com/valueforvalue/coreui/pkg/registry"
	"github.com/valueforvalue/coreui/pkg/renderers"
)

const version = "dev"

const initTemplate = `Theme(id="Industrial") {
    Color(key="radius", value="none"),
    Color(key="shadow", value="none"),
    Color(key="speed", value="instant"),
    Color(key="surface", value="#dbe4f0"),
    Color(key="panel", value="#ffffff"),
    Color(key="background", value="surface"),
    Color(key="text", value="#111827"),
    Color(key="primary", value="#2563eb")
}

Theme(id="Modern") {
    Color(key="radius", value="md"),
    Color(key="shadow", value="soft"),
    Color(key="speed", value="smooth"),
    Color(key="surface", value="#ffffff"),
    Color(key="panel", value="#ffffff"),
    Color(key="background", value="#f8fafc"),
    Color(key="text", value="#0f172a"),
    Color(key="primary", value="#6366f1")
}

Theme(id="Cyber") {
    Color(key="radius", value="none"),
    Color(key="shadow", value="none"),
    Color(key="speed", value="instant"),
    Color(key="surface", value="#000000"),
    Color(key="panel", value="#000000"),
    Color(key="background", value="#000000"),
    Color(key="text", value="#00ff00"),
    Color(key="primary", value="#00ff00")
}

View(id="root", title="New CoreUI Project", theme="Modern") {
    Stack(id="main_stack", dir="v", gap=20px) {
        Text(id="header_text", value="Welcome to CoreUI", size=28px, weight="bold", style="color: primary")
        Graph(id="trend_graph", type="line", color="primary", height=240px, labels=["08:00", "10:00", "12:00", "14:00", "16:00"], data=[18, 24, 21, 29, 34])
        Box(id="panel_box", padding=20px, background="background", variant="outline") {
            Text(id="panel_text", value="Use this panel to sketch your first screen.", style="color: text")
        }
        Image(id="hero_image", src="placeholder.png", width=100px, alt="Placeholder image")
        Trigger(id="notify_button", label="Click Me", variant="primary", action="app:notify(msg=\"Hello from CoreUI!\")")
    }
}
`

const componentsReadme = `# CoreUI Plugin Components

This directory contains JSON plugin definition files loaded automatically by CoreUI.

## Quick Start

Each ` + "`" + `.json` + "`" + ` file must follow the plugin schema:

` + "```json" + `
{
  "components": [
    {
      "name": "MyWidget",
      "has_children": false,
      "attributes": {
        "id":    { "type": "string", "required": true },
        "value": { "type": "int" },
        "mode":  { "type": "string", "enum": ["read", "write"] }
      }
    }
  ]
}
` + "```" + `

## Supported Attribute Types

| Type          | DSL Example         |
|---------------|---------------------|
| ` + "`string`" + `      | ` + "`label=\"hello\"`" + `  |
| ` + "`bool`" + `        | ` + "`hidden=true`" + `      |
| ` + "`int`" + `         | ` + "`value=5`" + `          |
| ` + "`unit`" + `        | ` + "`gap=20px`" + `         |
| ` + "`action`" + `      | ` + "`on_change=app:fn()`" + ` |
| ` + "`unit_array`" + `  | ` + "`cols=[1*, 2*]`" + `     |
| ` + "`string_array`" + `| ` + "`labels=[\"a\",\"b\"]`" + ` |

## Plugin Lifecycle

1. Place ` + "`" + `*.json` + "`" + ` files in this directory.
2. Start ` + "`corec edit`" + ` or ` + "`go build ./cmd/corec`" + ` — plugins are merged at startup.
3. Use the component name in ` + "`.cui`" + ` source files.
4. Extend the ` + "`renderNode`" + ` switch in ` + "`pkg/renderers/renderer.js`" + ` to render custom visuals.

## Attribute Marshalling (DSLStringer)

Every ` + "`ast.Value`" + ` implements ` + "`registry.DSLStringer`" + ` via its ` + "`ToDSLString()`" + ` method.
The method returns the canonical DSL token for the value:

- **string** → raw value (caller wraps in ` + "`\"...\"`" + ` for source output)
- **bool** → ` + "`true`" + ` / ` + "`false`" + `
- **int** → decimal integer string
- **unit** → e.g. ` + "`20px`" + `, ` + "`50%`" + `, ` + "`1*`" + `
- **action** → e.g. ` + "`app:save`" + `, ` + "`ui:notify(msg=\"Done\", type=\"success\")`" + `
- **array** → e.g. ` + "`[\"a\", \"b\"]`" + `, ` + "`[1*, 2*]`" + `

The GOTH server renderer populates ` + "`data-cui-{attr}`" + ` HTML attributes on plugin
component elements by calling ` + "`ToDSLString()`" + ` on each attribute value.
The JS renderer inflates these into the element's dataset during client-side hydration.
`

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init" {
		if err := runInit(os.Args[2:], os.Stdout); err != nil {
			log.SetFlags(0)
			log.Fatalf("%s", err.Error())
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "doctor" {
		if err := runDoctor(os.Stdout); err != nil {
			log.SetFlags(0)
			log.Fatalf("%s", err.Error())
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "context" {
		if err := runContext(os.Stdout); err != nil {
			log.SetFlags(0)
			log.Fatalf("%s", err.Error())
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "explain" {
		if err := runExplain(os.Stdout); err != nil {
			log.SetFlags(0)
			log.Fatalf("%s", err.Error())
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "edit" {
		if err := runEdit(os.Args[2:]); err != nil {
			log.SetFlags(0)
			log.Fatalf("%s", err.Error())
		}
		return
	}

	var outputPath string
	var showVersion bool
	var standalone bool
	var jsonErrors bool

	flag.StringVar(&outputPath, "o", "", "output JSON path")
	flag.BoolVar(&standalone, "standalone", false, "write standalone HTML output")
	flag.BoolVar(&standalone, "s", false, "write standalone HTML output")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.BoolVar(&jsonErrors, "json-errors", false, "emit structured JSON error object to stderr on compile failure")
	flag.Parse()

	if showVersion {
		fmt.Printf("corec %s (registry %s, schema %s)\n", version, registry.Version, registry.SchemaCompatibility)
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: corec init <project-name> | corec doctor | corec context | corec explain | corec edit <file.cui> | corec [-standalone] [-json-errors] [-o output.{json|html}] input.cui")
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	if outputPath == "" {
		outputPath = defaultOutputPath(inputPath, standalone)
	}

	data, err := compiler.CompileFile(inputPath, compiler.Options{Version: version, Standalone: standalone})
	if err != nil {
		if jsonErrors {
			source, _ := os.ReadFile(inputPath)
			writeJSONErrors(os.Stderr, err, string(source))
		} else {
			fmt.Fprintln(os.Stderr, err.Error())
		}
		os.Exit(1)
	}

	outputData := data
	if standalone {
		outputData, err = buildStandaloneHTML(data)
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	if err := os.WriteFile(outputPath, outputData, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Println(outputPath)
}

func runInit(args []string, stdout io.Writer) error {
	if len(args) != 1 {
		return errors.New("usage: corec init <project-name>")
	}
	if strings.TrimSpace(args[0]) == "" {
		return errors.New("usage: corec init <project-name>")
	}

	fileName := initFileName(args[0])
	if _, err := os.Stat(fileName); err == nil {
		return fmt.Errorf("File %s already exists. Aborting to prevent overwrite.", fileName)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := os.WriteFile(fileName, []byte(initTemplate), 0o644); err != nil {
		return err
	}

	// Create the ./components directory and write the plugin example file.
	if err := os.MkdirAll("components", 0o755); err == nil {
		examplePath := filepath.Join("components", registry.PluginExampleName)
		if _, statErr := os.Stat(examplePath); errors.Is(statErr, os.ErrNotExist) {
			_ = os.WriteFile(examplePath, []byte(registry.PluginExampleContent), 0o644)
		}
		readmePath := filepath.Join("components", "README.md")
		if _, statErr := os.Stat(readmePath); errors.Is(statErr, os.ErrNotExist) {
			_ = os.WriteFile(readmePath, []byte(componentsReadme), 0o644)
		}
	}

	fmt.Fprintf(stdout, "[Success] Initialized '%s'\n\nQuick Start:\n1. Edit '%s' to design your UI.\n2. Run 'corec -s %s' to bundle it.\n3. Open '%s' in any browser.\n", fileName, fileName, fileName, defaultOutputPath(fileName, true))
	return nil
}

func runContext(stdout io.Writer) error {
	architecturePath, err := findProjectFile("ARCHITECTURE.md")
	if err != nil {
		return err
	}

	architecture, err := os.ReadFile(architecturePath)
	if err != nil {
		return err
	}

	content, err := docs.RenderContext(string(architecture))
	if err != nil {
		return err
	}

	_, err = io.WriteString(stdout, content)
	return err
}

// runDoctor performs a suite of environmental health checks and prints a
// PASS/FAIL report with remediation steps for any failures.
func runDoctor(stdout io.Writer) error {
	type check struct {
		name string
		run  func() (passed bool, detail string)
	}

	checks := []check{
		{
			name: "Registry Health (no core/plugin name collisions)",
			run: func() (bool, string) {
				collisions := registry.RegistryCollisions()
				if len(collisions) == 0 {
					return true, "No naming collisions detected."
				}
				return false, fmt.Sprintf(
					"Plugin components shadow core names: %s\n  Remediation: rename the plugin components in ./components/*.json.",
					strings.Join(collisions, ", "),
				)
			},
		},
		{
			name: "Marshalling Round-Trip (DSLStringer correctness)",
			run: func() (bool, string) {
				type rtCase struct {
					value    ast.Value
					expected string
				}
				cases := []rtCase{
					{ast.Value{Kind: ast.StringKind, Data: "hello"}, "hello"},
					{ast.Value{Kind: ast.BoolKind, Data: true}, "true"},
					{ast.Value{Kind: ast.BoolKind, Data: false}, "false"},
					{ast.Value{Kind: ast.IntKind, Data: int64(42)}, "42"},
					{ast.Value{Kind: ast.UnitKind, Data: "20px"}, "20px"},
					{ast.Value{Kind: ast.UnitKind, Data: "50%"}, "50%"},
					{ast.Value{Kind: ast.ActionKind, Data: ast.Action{Namespace: "app", Call: "save", Params: map[string]ast.Value{}}}, "app:save"},
					{ast.Value{Kind: ast.ActionKind, Data: ast.Action{
						Namespace: "ui",
						Call:      "notify",
						Params: map[string]ast.Value{
							"msg":  {Kind: ast.StringKind, Data: "Done"},
							"type": {Kind: ast.StringKind, Data: "success"},
						},
					}}, `ui:notify(msg="Done", type="success")`},
				}
				for _, c := range cases {
					got := c.value.ToDSLString()
					if got != c.expected {
						return false, fmt.Sprintf(
							"DSLStringer round-trip FAILED: ToDSLString() returned %q, want %q\n  Remediation: ensure pkg/ast ToDSLString() is up-to-date and `go build ./...` succeeds.",
							got, c.expected,
						)
					}
				}
				return true, fmt.Sprintf("All %d DSLStringer round-trip cases passed.", len(cases))
			},
		},
		{
			name: "Asset Health (renderer assets loaded in memory)",
			run: func() (bool, string) {
				js := renderers.GetRendererJS()
				css := renderers.GetBaseCSS()
				if len(js) == 0 {
					return false, "renderer.js is empty or failed to embed.\n  Remediation: rebuild the binary with `go build ./cmd/corec`."
				}
				if len(css) == 0 {
					return false, "base.css is empty or failed to embed.\n  Remediation: rebuild the binary with `go build ./cmd/corec`."
				}
				return true, fmt.Sprintf("renderer.js (%d bytes), base.css (%d bytes) loaded.", len(js), len(css))
			},
		},
		{
			name: "Write Permissions (current directory)",
			run: func() (bool, string) {
				probe := ".corec_doctor_probe"
				if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
					return false, fmt.Sprintf("Cannot write to current directory: %v\n  Remediation: check directory permissions with `ls -ld .`.", err)
				}
				_ = os.Remove(probe)
				return true, "Write access confirmed for current directory."
			},
		},
		{
			name: "Write Permissions (./history directory)",
			run: func() (bool, string) {
				if err := os.MkdirAll("history", 0o755); err != nil {
					return false, fmt.Sprintf("Cannot create ./history: %v\n  Remediation: check directory permissions.", err)
				}
				probe := filepath.Join("history", ".corec_doctor_probe")
				if err := os.WriteFile(probe, []byte("ok"), 0o600); err != nil {
					return false, fmt.Sprintf("Cannot write to ./history: %v\n  Remediation: check directory permissions.", err)
				}
				_ = os.Remove(probe)
				return true, "Write access confirmed for ./history."
			},
		},
		{
			name: "Port Availability (edit server default range 49152–65535)",
			run: func() (bool, string) {
				ln, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					return false, fmt.Sprintf("Cannot bind a local TCP port: %v\n  Remediation: check firewall or per-process socket limits.", err)
				}
				_ = ln.Close()
				return true, fmt.Sprintf("Successfully bound ephemeral port %s.", ln.Addr().String())
			},
		},
	}

	passes := 0
	for _, c := range checks {
		passed, detail := c.run()
		marker := "PASS"
		if !passed {
			marker = "FAIL"
		} else {
			passes++
		}
		fmt.Fprintf(stdout, "[%s] %s\n      %s\n\n", marker, c.name, detail)
	}

	fmt.Fprintf(stdout, "─────────────────────────────────────────\n")
	fmt.Fprintf(stdout, "Results: %d/%d checks passed.\n", passes, len(checks))
	return nil
}

func initFileName(projectName string) string {
	projectName = strings.TrimSpace(projectName)
	if strings.HasSuffix(projectName, ".cui") {
		return projectName
	}
	return projectName + ".cui"
}

func defaultOutputPath(inputPath string, standalone bool) string {
	suffix := ".json"
	if standalone {
		suffix = ".html"
	}

	ext := filepath.Ext(inputPath)
	if ext == "" {
		return inputPath + suffix
	}
	return strings.TrimSuffix(inputPath, ext) + suffix
}

func buildStandaloneHTML(jsonData []byte) ([]byte, error) {
	html, err := renderers.BuildStandaloneHTML(jsonData, nil)
	if err != nil {
		return nil, err
	}
	return []byte(html), nil
}

func findProjectFile(name string) (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := workingDir
	for {
		candidate := filepath.Join(dir, name)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not locate %s", name)
}

func runEdit(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: corec edit <file.cui>")
	}

	filePath := args[0]
	if _, err := os.Stat(filePath); err != nil {
		return fmt.Errorf("cannot open %s: %w", filePath, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	return editor.New(filePath).Run(ctx)
}

// jsonErrorEntry is a single structured error entry for --json-errors output.
type jsonErrorEntry struct {
	Line           int    `json:"line"`
	Column         int    `json:"column"`
	ErrorCode      string `json:"error_code"`
	Message        string `json:"message"`
	Expected       string `json:"expected,omitempty"`
	ContextSnippet string `json:"context_snippet,omitempty"`
}

// jsonErrorResponse is the top-level --json-errors response envelope.
type jsonErrorResponse struct {
	Status string           `json:"status"`
	Errors []jsonErrorEntry `json:"errors"`
}

// writeJSONErrors writes a machine-readable JSON error object to w. It extracts
// structured diagnostic information from err (which may be a *diag.Error) and
// includes the offending source line as context_snippet when available.
func writeJSONErrors(w io.Writer, err error, source string) {
	lines := strings.Split(source, "\n")
	entries := collectErrorEntries(err, lines)
	resp := jsonErrorResponse{
		Status: "error",
		Errors: entries,
	}
	data, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Fprintln(w, string(data))
}

func collectErrorEntries(err error, sourceLines []string) []jsonErrorEntry {
	if err == nil {
		return nil
	}
	var de *diag.Error
	if errors.As(err, &de) {
		return []jsonErrorEntry{diagToEntry(de, sourceLines)}
	}
	return []jsonErrorEntry{{
		Line:      0,
		Column:    0,
		ErrorCode: "COMPILE_ERROR",
		Message:   err.Error(),
	}}
}

func diagToEntry(de *diag.Error, sourceLines []string) jsonErrorEntry {
	entry := jsonErrorEntry{
		Line:      de.Line,
		Column:    de.Col,
		ErrorCode: classifyError(de.Message),
		Message:   de.Message,
	}

	entry.Expected = inferExpected(de.Message)

	if de.Line > 0 && de.Line <= len(sourceLines) {
		entry.ContextSnippet = sourceLines[de.Line-1]
	}

	return entry
}

// classifyError maps a human-readable error message to a machine-readable error code.
func classifyError(message string) string {
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "unknown attribute"):
		return "UNKNOWN_ATTRIBUTE"
	case strings.Contains(msg, "unknown component"):
		return "UNKNOWN_COMPONENT"
	case strings.Contains(msg, "duplicate") && strings.Contains(msg, "id"):
		return "DUPLICATE_ID"
	case strings.Contains(msg, "duplicate attribute"):
		return "DUPLICATE_ATTRIBUTE"
	case strings.Contains(msg, "missing required"):
		return "MISSING_REQUIRED_ATTRIBUTE"
	case strings.Contains(msg, "expects string"):
		return "INVALID_ATTRIBUTE_TYPE"
	case strings.Contains(msg, "expects bool"):
		return "INVALID_ATTRIBUTE_TYPE"
	case strings.Contains(msg, "expects int"):
		return "INVALID_ATTRIBUTE_TYPE"
	case strings.Contains(msg, "expects unit"):
		return "INVALID_ATTRIBUTE_TYPE"
	case strings.Contains(msg, "expects action"):
		return "INVALID_ATTRIBUTE_TYPE"
	case strings.Contains(msg, "expects array"):
		return "INVALID_ATTRIBUTE_TYPE"
	case strings.Contains(msg, "does not allow value"):
		return "INVALID_ENUM_VALUE"
	case strings.Contains(msg, "does not accept children"):
		return "INVALID_CHILDREN"
	case strings.Contains(msg, "unterminated"):
		return "SYNTAX_ERROR"
	case strings.Contains(msg, "expected"):
		return "SYNTAX_ERROR"
	default:
		return "COMPILE_ERROR"
	}
}

// inferExpected extracts a concise expected-value hint from common error messages.
func inferExpected(message string) string {
	msg := strings.ToLower(message)
	switch {
	case strings.Contains(msg, "expects string"):
		return "string"
	case strings.Contains(msg, "expects bool"):
		return "bool"
	case strings.Contains(msg, "expects int"):
		return "int"
	case strings.Contains(msg, "expects unit"):
		return "unit (e.g. 20px, 50%, 1*)"
	case strings.Contains(msg, "expects action"):
		return "action (e.g. app:call() or ui:navigate(target=\"id\"))"
	case strings.Contains(msg, "expects array"):
		return "array"
	case strings.Contains(msg, "expected '('"):
		return "'('"
	case strings.Contains(msg, "expected ')'"):
		return "')'"
	case strings.Contains(msg, "expected '='"):
		return "'='"
	case strings.Contains(msg, "expected ']'"):
		return "']'"
	case strings.Contains(msg, "component type"):
		return "ComponentType (e.g. View, Stack, Text)"
	case strings.Contains(msg, "attribute name"):
		return "attribute name (lowercase identifier)"
	case strings.Contains(msg, "attribute value"):
		return "attribute value (string, bool, int, unit, action, or array)"
	default:
		return ""
	}
}

func runExplain(stdout io.Writer) error {
	content, err := docs.RenderExplain()
	if err != nil {
		return err
	}
	_, err = io.WriteString(stdout, content)
	return err
}
