package main

import (
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"coreui/pkg/compiler"
	"coreui/pkg/docs"
	"coreui/pkg/registry"
	jsrenderer "coreui/renderers/js"
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
        Box(id="panel_box", padding=20px, background="background", variant="outline") {
            Text(id="panel_text", value="Use this panel to sketch your first screen.", style="color: text")
        }
        Image(id="hero_image", src="placeholder.png", width=100px, alt="Placeholder image")
        Trigger(id="notify_button", label="Click Me", variant="primary", action="app:notify(msg=\"Hello from CoreUI!\")")
    }
}
`

const standaloneTemplate = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>CoreUI Standalone</title>
  <style>
    :root {
      color-scheme: light dark;
    }
    html, body {
      margin: 0;
      min-height: 100%;
    }
    body {
      background: #f5f7fb;
      color: #111827;
      font-family: Arial, Helvetica, sans-serif;
    }
    @media (prefers-color-scheme: dark) {
      body {
        background: #0f172a;
        color: #e5e7eb;
      }
    }
    #coreui-root {
      min-height: 100vh;
    }
  </style>
</head>
<body>
  <div id="coreui-root"></div>
  <script type="module">{{ .RendererJS }}</script>
  <script type="module">
    const jsonData = {{ .JSONData }};
    document.addEventListener("DOMContentLoaded", () => {
      new window.CoreUI(jsonData).render(document.getElementById("coreui-root"));
    });
  </script>
</body>
</html>
`

type standalonePageData struct {
	RendererJS template.JS
	JSONData   template.JS
}

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
		fmt.Fprintln(os.Stderr, "usage: corec init <project-name> | corec context | corec [-standalone] [-o output.{json|html}] input.cui")
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
	tmpl, err := template.New("standalone").Parse(standaloneTemplate)
	if err != nil {
		return nil, err
	}

	data := standalonePageData{
		RendererJS: template.JS(escapeInlineScript(jsrenderer.Source + "\nwindow.CoreUI = CoreUI;\n")),
		JSONData:   template.JS(escapeJSONForScript(string(jsonData))),
	}

	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return nil, err
	}

	return []byte(builder.String()), nil
}

func escapeInlineScript(value string) string {
	return strings.ReplaceAll(value, "</", "<\\/")
}

func escapeJSONForScript(value string) string {
	replacer := strings.NewReplacer(
		"<", "\\u003c",
		">", "\\u003e",
		"&", "\\u0026",
		"\u2028", "\\u2028",
		"\u2029", "\\u2029",
	)
	return replacer.Replace(value)
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
