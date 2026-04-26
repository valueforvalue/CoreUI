package main

import (
	"context"
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

	"github.com/valueforvalue/coreui/pkg/compiler"
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

	flag.StringVar(&outputPath, "o", "", "output JSON path")
	flag.BoolVar(&standalone, "standalone", false, "write standalone HTML output")
	flag.BoolVar(&standalone, "s", false, "write standalone HTML output")
	flag.BoolVar(&showVersion, "version", false, "print version")
	flag.Parse()

	if showVersion {
		fmt.Printf("corec %s (registry %s, schema %s)\n", version, registry.Version, registry.SchemaCompatibility)
		return
	}

	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: corec init <project-name> | corec doctor | corec context | corec edit <file.cui> | corec [-standalone] [-o output.{json|html}] input.cui")
		os.Exit(1)
	}

	inputPath := flag.Arg(0)
	if outputPath == "" {
		outputPath = defaultOutputPath(inputPath, standalone)
	}

	data, err := compiler.CompileFile(inputPath, compiler.Options{Version: version, Standalone: standalone})
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
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
