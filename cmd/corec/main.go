package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
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
		fmt.Fprintln(os.Stderr, "usage: corec init <project-name> | corec context | corec edit <file.cui> | corec [-standalone] [-o output.{json|html}] input.cui")
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
