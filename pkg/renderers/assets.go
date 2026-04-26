package renderers

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"html/template"
	"strings"
)

// EmbeddedAssets exposes the browser renderer assets bundled into the module.
type EmbeddedAssets struct {
	// RendererJS is the processed browser renderer module with embedded base styles.
	RendererJS string
	// BaseCSS is the default stylesheet injected into the renderer shadow root.
	BaseCSS string
}

var (
	//go:embed renderer.js
	rendererJSTemplate string

	//go:embed base.css
	baseCSS string

	// Assets contains the processed browser renderer assets bundled into the binary.
	Assets = EmbeddedAssets{
		RendererJS: buildRendererJS(),
		BaseCSS:    baseCSS,
	}
)

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
{{- if .HasData }}
    window.CoreUIData = {{ .DataJSON }};
{{- end }}
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
	DataJSON   template.JS
	HasData    bool
}

// GetRendererJS returns the browser renderer module with embedded base styles.
func GetRendererJS() string {
	return Assets.RendererJS
}

// GetBaseCSS returns the default CSS injected into the CoreUI shadow root.
func GetBaseCSS() string {
	return Assets.BaseCSS
}

// BuildStandaloneHTML renders a self-contained HTML document for a compiled blueprint.
//
// When data is non-nil, it is JSON-encoded and exposed to the page as
// window.CoreUIData before the renderer bootstraps.
func BuildStandaloneHTML(blueprintJSON []byte, data interface{}) (string, error) {
	if !json.Valid(blueprintJSON) {
		return "", errors.New("coreui blueprint is not valid JSON")
	}

	pageData := standalonePageData{
		RendererJS: template.JS(escapeInlineScript(GetRendererJS() + "\nwindow.CoreUI = CoreUI;\n")),
		JSONData:   template.JS(escapeJSONForScript(string(blueprintJSON))),
	}

	if data != nil {
		dataJSON, err := json.Marshal(data)
		if err != nil {
			return "", err
		}
		pageData.HasData = true
		pageData.DataJSON = template.JS(escapeJSONForScript(string(dataJSON)))
	}

	tmpl, err := template.New("standalone").Parse(standaloneTemplate)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, pageData); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func buildRendererJS() string {
	baseCSSJSON, err := json.Marshal(baseCSS)
	if err != nil {
		panic(err)
	}

	return strings.Replace(rendererJSTemplate, "\"__COREUI_BASE_CSS__\"", string(baseCSSJSON), 1)
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
