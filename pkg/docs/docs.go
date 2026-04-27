package docs

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/valueforvalue/coreui/pkg/registry"
)

type AttributeDoc struct {
	Name        string
	Type        string
	Requirement string
}

type ComponentDoc struct {
	Name          string
	HasChildren   bool
	Intent        string
	Attributes    []AttributeDoc
	BestPractices string
}

type CatalogData struct {
	RegistryVersion     string
	SchemaCompatibility string
	LastUpdated         string
	Components          []ComponentDoc
}

type ContextData struct {
	Catalog             CatalogData
	Architecture        string
	ThemeTokens         []string
	SemanticTokens      []SemanticTokenDoc
	FactoryThemes       []FactoryThemeDoc
	BuiltInActions      []string
	GoWiringSnippet     string
	JSWiringSnippet     string
	ActionProtocolIntro string
	GraphGuidance       string
}

type SemanticTokenDoc struct {
	Name   string
	Values []string
}

type FactoryThemeDoc struct {
	Name   string
	Tokens []ThemeTokenDoc
}

type ThemeTokenDoc struct {
	Key   string
	Value string
}

const componentsTemplate = `# CoreUI Components Reference

**Registry Version:** {{ .RegistryVersion }}
**Schema Compatibility:** {{ .SchemaCompatibility }}
**Last Updated:** {{ .LastUpdated }}

## Global Actions

CoreUI action values use the form ` + "`namespace:function(key=\"value\")`" + `.

- ` + "`ui:`" + ` is **strictly validated** against the built-in UI action registry.
- ` + "`app:`" + ` is **user-defined/application-specific** and passes through as long as it follows valid action syntax.

{{ range .Components }}
## {{ .Name }}

**HasChildren:** {{ if .HasChildren }}true{{ else }}false{{ end }}

| Prop | Type | Requirement |
| --- | --- | --- |
{{ range .Attributes }}| {{ .Name }} | {{ .Type }} | {{ .Requirement }} |
{{ end }}

{{ if .BestPractices -}}
### Best Practices

{{ .BestPractices }}

{{ end -}}

{{ end }}
## Plugin Development

CoreUI supports external component definitions loaded from ` + "`./components/*.json`" + ` files.
Running ` + "`corec init <project>`" + ` creates an example plugin at ` + "`./components/plugin_example.json`" + `.

### Schema Requirements

Each plugin file is a JSON object with a top-level ` + "`\"components\"`" + ` array.  Every
entry in that array must include at minimum a ` + "`\"name\"`" + ` string and an
` + "`\"attributes\"`" + ` object.

Supported ` + "`\"type\"`" + ` values for attributes:

| Type token | Description |
| --- | --- |
| ` + "`string`" + ` | Quoted text value |
| ` + "`bool`" + ` | ` + "`true`" + ` / ` + "`false`" + ` literal |
| ` + "`int`" + ` | Integer literal |
| ` + "`unit`" + ` | Dimensional value (e.g. ` + "`20px`" + `, ` + "`50%`" + `, ` + "`1*`" + `) |
| ` + "`action`" + ` | Action expression (e.g. ` + "`app:doSomething(key=\"v\")`" + `) |
| ` + "`unit_array`" + ` | Array of unit values |
| ` + "`string_array`" + ` | Array of strings |

Optional fields per attribute:

- ` + "`\"required\": true`" + ` — the parser will reject the component if the attribute is absent.
- ` + "`\"enum\": [\"a\", \"b\"]`" + ` — restrict the attribute to one of the listed string values.
- ` + "`\"doc_type\": \"Human label\"`" + ` — overrides the type label shown in ` + "`COMPONENTS.md`" + `.

### Registry Mapping

Plugin components are merged into ` + "`AllComponents()`" + ` and are therefore visible in:

- The **Inspector** panel of ` + "`corec edit`" + ` (attribute editor and component palette).
- The ` + "`/api/registry`" + ` endpoint consumed by the editor frontend.
- The ` + "`corec context`" + ` AI-onboarding output.

The ` + "`has_children`" + ` boolean maps directly to whether the component accepts a ` + "`{ }`" + ` child block in ` + "`.cui`" + ` source.

### Implementation Note

Plugin files define **structure and validation rules** only.  The JS renderer
(` + "`pkg/renderers/renderer.js`" + `) will render unknown component types as a plain
error-boundary box unless you extend the ` + "`renderNode`" + ` ` + "`switch`" + ` statement with a
matching ` + "`case`" + `.  The GOTH server renderer requires the same treatment.  Plugins
are therefore most useful when paired with a custom renderer build.
`

const contextTemplate = `# CoreUI Context Stream

**Registry Version:** {{ .Catalog.RegistryVersion }}
**Schema Compatibility:** {{ .Catalog.SchemaCompatibility }}
**Last Updated:** {{ .Catalog.LastUpdated }}

## Iron-Clad Architecture Principles

{{ .Architecture }}

## Component Catalog

{{ range .Catalog.Components }}
### {{ .Name }}

- **HasChildren:** {{ if .HasChildren }}true{{ else }}false{{ end }}
- **Intent:** {{ if .Intent }}` + "`" + `{{ .Intent }}` + "`" + `{{ else }}(none){{ end }}

| Attribute | Type | Requirement |
| --- | --- | --- |
{{ range .Attributes }}| {{ .Name }} | {{ .Type }} | {{ .Requirement }} |
{{ end }}

{{ end }}
## BI & Visualization Guidance

{{ .GraphGuidance }}

## Theme Tokens

These are the standard starter tokens used by CoreUI onboarding templates:

{{ range .ThemeTokens -}}
- ` + "`{{ . }}`" + `
{{ end }}

### Semantic token values

{{ range .SemanticTokens -}}
- ` + "`{{ .Name }}`" + `: {{ range $index, $value := .Values }}{{ if $index }}, {{ end }}` + "`{{$value}}`" + `{{ end }}
{{ end }}

## Factory Themes

{{ range .FactoryThemes }}
### {{ .Name }}

| Token | Value |
| --- | --- |
{{ range .Tokens }}| {{ .Key }} | {{ .Value }} |
{{ end }}

{{ end }}

## Action Protocol

{{ .ActionProtocolIntro }}

### Built-in ` + "`ui:`" + ` actions

{{ range .BuiltInActions -}}
- ` + "`{{ . }}`" + `
{{ end }}

### User-defined ` + "`app:`" + ` actions

Use ` + "`app:`" + ` for application-specific intent. Preserve the same structured payload shape and route execution in your own application layer.

## Agentic Workflows (v1.6.0)

### Intent Field

Every component in the compiled JSON blueprint includes an ` + "`intent`" + ` field derived from the registry. Use this field to verify that the UI layout matches the user's functional goal. Example intents:

| Intent | Meaning |
| --- | --- |
| ` + "`action-trigger`" + ` | Component initiates an action (e.g. Trigger/Button) |
| ` + "`data-label`" + ` | Component displays a static or bound text label |
| ` + "`data-entry`" + ` | Component accepts user input |
| ` + "`data-display`" + ` | Component presents read-only data or media |
| ` + "`layout-container`" + ` | Component organises child components spatially |
| ` + "`layout-root`" + ` | The top-level view container |
| ` + "`theme-definition`" + ` | Component defines visual theme tokens |
| ` + "`theme-token`" + ` | A single key/value theme token |

Plugin components may declare a custom intent in their JSON plugin file via ` + "`\"intent\": \"my-intent\"`" + `.

### Structured Error Output (--json-errors)

When running ` + "`corec --json-errors input.cui`" + `, compilation errors are written to ` + "`stderr`" + ` as a JSON object instead of human-readable text. This enables AI agents to ingest the error and generate a patch autonomously.

**Schema:**
` + "```json" + `
{
  "status": "error",
  "errors": [
    {
      "line": 12,
      "column": 5,
      "error_code": "INVALID_ATTRIBUTE_TYPE",
      "message": "attribute \"gap\" expects unit",
      "expected": "unit (e.g. 20px, 50%, 1*)",
      "context_snippet": "    Stack(id=\"layout\", gap=bad_value)"
    }
  ]
}
` + "```" + `

**Error codes:** ` + "`UNKNOWN_ATTRIBUTE`" + `, ` + "`UNKNOWN_COMPONENT`" + `, ` + "`DUPLICATE_ID`" + `, ` + "`DUPLICATE_ATTRIBUTE`" + `, ` + "`MISSING_REQUIRED_ATTRIBUTE`" + `, ` + "`INVALID_ATTRIBUTE_TYPE`" + `, ` + "`INVALID_ENUM_VALUE`" + `, ` + "`INVALID_CHILDREN`" + `, ` + "`SYNTAX_ERROR`" + `, ` + "`COMPILE_ERROR`" + `.

### Knowledge Injection (corec explain)

Run ` + "`corec explain`" + ` to generate a single high-density Markdown reference document from the live registry. Pipe this output into any local LLM to give it full CoreUI context while offline:

` + "```sh" + `
corec explain | ollama run llama3 "Generate a login form using CoreUI DSL"
` + "```" + `

The document includes: EBNF grammar, full component/attribute/intent registry table, action wiring code, and three golden ` + "`.cui`" + ` examples.

## Asset Pipeline (v1.5.0)

CoreUI supports compressed image assets for portable single-file manuals.

- **Drag-and-drop** ` + "`.jpg`" + `, ` + "`.png`" + `, or ` + "`.webp`" + ` files onto the ` + "`corec edit`" + ` canvas.
- The server compresses the image with ` + "`gzip`" + ` and returns a Base64 string via ` + "`POST /api/upload`" + `.
- The snippet ` + "`Image(id=\"…\", compressed_src=\"<base64>\")`" + ` is inserted into your ` + "`.cui`" + ` source.
- The JS renderer inflates the data on the fly using the native ` + "`DecompressionStream(\"gzip\")`" + ` API and displays the image as a ` + "`blob:`" + ` URL.
- Target: 40–60% file-size reduction vs. raw Base64 embedding.

## Plugin Development (v1.5.0)

External component definitions are loaded from ` + "`./components/*.json`" + ` at startup.

### Quick-start

1. Run ` + "`corec init <project>`" + ` — this creates ` + "`./components/plugin_example.json`" + ` with a sample ` + "`Rating`" + ` component.
2. Edit the JSON to define your own components following the schema below.
3. Restart ` + "`corec`" + ` or ` + "`corec edit`" + ` — plugins are merged automatically.

### Plugin JSON schema

` + "```json" + `
{
  "components": [
    {
      "name": "MyWidget",
      "has_children": false,
      "intent": "data-display",
      "attributes": {
        "id":    { "type": "string", "required": true },
        "value": { "type": "int" },
        "mode":  { "type": "string", "enum": ["read", "write"] }
      }
    }
  ]
}
` + "```" + `

Supported type tokens: ` + "`string`" + `, ` + "`bool`" + `, ` + "`int`" + `, ` + "`unit`" + `, ` + "`action`" + `, ` + "`unit_array`" + `, ` + "`string_array`" + `.
The optional ` + "`\"intent\"`" + ` field annotates the component's semantic role in the UI, exported in JSON blueprints.

### Renderer note

Plugin files define structure and validation.  To render custom visuals you must
extend the ` + "`renderNode`" + ` switch in ` + "`pkg/renderers/renderer.js`" + ` and the GOTH renderer.

### Plugin Lifecycle

1. **Define** – Add a ` + "`*.json`" + ` file to ` + "`./components/`" + `.
2. **Startup** – Registry merges plugin definitions; Core-First collision guard rejects any plugin that shadows a core component.
3. **Parse** – The ` + "`.cui`" + ` parser validates plugin attributes against the merged registry.
4. **Render (server)** – GOTH renderer emits ` + "`data-coreui-plugin`" + ` and ` + "`data-cui-{attr}`" + ` HTML attributes via ` + "`ast.Value.ToDSLString()`" + `.
5. **Render (client)** – JS renderer sets ` + "`element.dataset.cui_{attr}`" + ` entries during hydration.
6. **Inspect** – Plugin components appear in ` + "`corec edit`" + ` Inspector and ` + "`GET /api/registry`" + `.

## Diagnostics (v1.5.0)

Run ` + "`corec doctor`" + ` to execute a self-healing diagnostic suite:

- **Registry Health** — checks for naming collisions between core and plugin components.
- **Marshalling Round-Trip** — verifies ` + "`ast.Value.ToDSLString()`" + ` produces correct output for all primitive types.
- **Asset Health** — verifies the renderer JS and CSS are correctly loaded in memory.
- **Permissions** — checks write access for the current directory and ` + "`./history`" + `.
- **Port Availability** — verifies that the OS can bind a local TCP port for ` + "`corec edit`" + `.

Each check reports ` + "`[PASS]`" + ` or ` + "`[FAIL]`" + ` with a specific remediation step on failure.

## Attribute Marshalling (v1.5.0)

Every attribute value implements ` + "`registry.DSLStringer`" + ` via ` + "`ast.Value.ToDSLString()`" + `.

### DSLStringer Interface

` + "```go" + `
type DSLStringer interface {
    ToDSLString() string
}
` + "```" + `

### ToDSLString() Output

| Kind | .cui source | ToDSLString() output |
| --- | --- | --- |
| string | ` + "`label=\"hello\"`" + ` | ` + "`hello`" + ` |
| bool | ` + "`hidden=true`" + ` | ` + "`true`" + ` |
| int | ` + "`value=42`" + ` | ` + "`42`" + ` |
| unit | ` + "`gap=20px`" + ` | ` + "`20px`" + ` |
| action (no params) | ` + "`action=app:save`" + ` | ` + "`app:save`" + ` |
| action (with params) | ` + "`action=ui:notify(msg=\"Done\", type=\"success\")`" + ` | ` + "`ui:notify(msg=\"Done\", type=\"success\")`" + ` |
| array | ` + "`labels=[\"a\",\"b\"]`" + ` | ` + "`[\"a\", \"b\"]`" + ` |

String values do **not** include outer quotes; the caller wraps them for ` + "`.cui`" + ` source output.

### POST /api/save

` + "`POST /api/save`" + ` in ` + "`corec edit`" + ` accepts a JSON-encoded ` + "`generator.Output`" + ` body, marshals it back to ` + "`.cui`" + ` source via ` + "`generator.MarshalDSL()`" + ` (which calls ` + "`ToDSLString()`" + ` for each attribute), validates by re-compiling, then writes the file.

## Wiring Snippets

### Go (GOTH)

~~~go
{{ .GoWiringSnippet }}
~~~

### JavaScript (Standalone)

~~~js
{{ .JSWiringSnippet }}
~~~
`

const goWiringSnippet = `mux.Handle("/coreui/action", goth.HandleAction(goth.ActionExecutorFunc(func(ctx context.Context, action goth.ActionRequest, w http.ResponseWriter, r *http.Request) error {
	switch action.Namespace {
	case "ui":
		return handleBuiltInUIAction(action, w)
	case "app":
		return routeAppAction(action.Call, action.Params, w)
	default:
		http.Error(w, "unsupported action namespace", http.StatusBadRequest)
		return nil
	}
})))`

const jsWiringSnippet = `const ui = new window.CoreUI(jsonData);
ui.onAction((data) => {
  if (data.namespace !== "app") {
    return;
  }

  switch (data.call) {
    case "notify":
      window.dispatchEvent(new CustomEvent("coreui:notify", { detail: data.params }));
      break;
    default:
      console.warn("Unhandled CoreUI app action", data);
  }
});

ui.render(document.getElementById("coreui-root"));`

const actionProtocolIntro = "`ui:` is reserved for registry-validated CoreUI primitives. `app:` is open for application intent, but must still follow valid CoreUI action syntax and emit the same `{ namespace, call, params }` structure."

const graphBestPractices = `Use ` + "`labels`" + ` and ` + "`data`" + ` as parallel arrays with the same length so each label maps to the value at the same index.

- Prefer a quoted app reference such as ` + "`data=\"app:simulation.pressure_series\"`" + ` when live data comes from your application layer.
- Use ` + "`type=\"line\"`" + ` or ` + "`type=\"area\"`" + ` for time series, ` + "`type=\"bar\"`" + ` for ranked comparisons, and ` + "`type=\"pie\"`" + ` only for small part-to-whole slices.
- Keep ` + "`height`" + ` unit-backed (for example ` + "`220px`" + ` or ` + "`40%`" + `) so the compiler can reject invalid units before rendering.

~~~cui
Graph(
    id="throughput_graph",
    type="line",
    color="primary",
    height=240px,
    labels=["08:00", "10:00", "12:00", "14:00"],
    data=[18, 24, 21, 29]
)
~~~`

const graphGuidance = `The ` + "`Graph`" + ` component is the BI and visualization primitive for CoreUI.

- ` + "`labels`" + ` is a string array for axis or legend text.
- ` + "`data`" + ` is either a literal numeric JSON array or a quoted ` + "`app:`" + ` reference string when your simulation data is resolved at runtime.
- Keep ` + "`labels`" + ` and literal ` + "`data`" + ` arrays aligned by index: the first label describes the first value, the second label describes the second value, and so on.
- When emitting runtime references, prefer stable names such as ` + "`app:simulation.temperature_series`" + ` or ` + "`app:ops.queue_depth`" + ` so downstream agents can map data producers to Graph nodes deterministically.

Example:

~~~cui
Graph(
    id="temperature_graph",
    type="area",
    color="primary",
    height=240px,
    labels=["T+0", "T+5", "T+10", "T+15"],
    data="app:simulation.temperature_series"
)
~~~`

func BuildCatalog() CatalogData {
	components := registry.AllComponents()
	data := CatalogData{
		RegistryVersion:     registry.Version,
		SchemaCompatibility: registry.SchemaCompatibility,
		LastUpdated:         registry.LastUpdated,
		Components:          make([]ComponentDoc, 0, len(components)),
	}

	for _, component := range components {
		doc := ComponentDoc{
			Name:        component.Name,
			HasChildren: component.HasChildren,
			Intent:      component.Intent,
			Attributes:  make([]AttributeDoc, 0, len(component.Attributes)),
		}

		names := make([]string, 0, len(component.Attributes))
		for name := range component.Attributes {
			names = append(names, name)
		}
		sortStrings(names)

		for _, name := range names {
			attribute := component.Attributes[name]
			requirement := "Optional"
			if attribute.Required {
				requirement = "Required"
			}

			doc.Attributes = append(doc.Attributes, AttributeDoc{
				Name:        name,
				Type:        docType(attribute),
				Requirement: requirement,
			})
		}

		doc.BestPractices = componentBestPractices(component.Name)
		data.Components = append(data.Components, doc)
	}

	return data
}

func RenderComponentsReference() (string, error) {
	return executeTemplate(componentsTemplate, BuildCatalog())
}

func RenderContext(architecture string) (string, error) {
	return executeTemplate(contextTemplate, ContextData{
		Catalog:             BuildCatalog(),
		Architecture:        strings.TrimSpace(architecture),
		ThemeTokens:         registry.StandardThemeTokens(),
		SemanticTokens:      buildSemanticTokens(),
		FactoryThemes:       buildFactoryThemes(),
		BuiltInActions:      registry.BuiltInActions(),
		GoWiringSnippet:     goWiringSnippet,
		JSWiringSnippet:     jsWiringSnippet,
		ActionProtocolIntro: actionProtocolIntro,
		GraphGuidance:       graphGuidance,
	})
}

func GoWiringSnippet() string {
	return goWiringSnippet
}

func JSWiringSnippet() string {
	return jsWiringSnippet
}

// ExplainData is the template data for the corec explain command.
type ExplainData struct {
	RegistryVersion string
	LastUpdated     string
	Components      []ExplainComponentDoc
	BuiltInActions  []string
	GoWiringSnippet string
	JSWiringSnippet string
}

// ExplainComponentDoc holds a compact row for the explain registry table.
type ExplainComponentDoc struct {
	Name        string
	Intent      string
	HasChildren bool
	Attributes  []ExplainAttrDoc
}

// ExplainAttrDoc holds one attribute row for the explain table.
type ExplainAttrDoc struct {
	Name     string
	Type     string
	Required bool
	Enum     string
}

const explainTemplate = `# CoreUI v{{ .RegistryVersion }} — Expert Reference ({{ .LastUpdated }})

## 1. Grammar (EBNF)

` + "```ebnf" + `
document     = { theme-block } root-component ;
theme-block  = "Theme" "(" attrs ")" "{" { color-decl "," } "}" ;
color-decl   = "Color" "(" "key" "=" string "," "value" "=" string ")" ;
root-component = component ;
component    = ident "(" [ attr-list ] ")" [ "{" { component [","] } "}" ] ;
attr-list    = attr { "," attr } ;
attr         = ident "=" value ;
value        = string | bool | int | unit | action | array ;
string       = '"' { char } '"' ;
bool         = "true" | "false" ;
int          = digit { digit } ;
unit         = int ( "px" | "%" | "*" ) | "auto" ;
action       = namespace ":" call [ "(" [ param-list ] ")" ] ;
namespace    = "ui" | "app" ;
param-list   = param { "," param } ;
param        = ident "=" value ;
array        = "[" [ value { "," value } ] "]" ;
` + "```" + `

**Constraints:**
- Every component must have a globally unique ` + "`id`" + ` attribute.
- Unknown attributes and type mismatches are fatal parse errors.
- ` + "`ui:`" + ` actions are registry-validated; ` + "`app:`" + ` actions pass through if syntactically valid.
- Theme blocks must precede the root component.

## 2. Component Registry

| Component | Intent | HasChildren | Attributes (type, req?) |
|-----------|--------|-------------|------------------------|
{{ range .Components -}}
| **{{ .Name }}** | {{ .Intent }} | {{ if .HasChildren }}✓{{ else }}✗{{ end }} | {{ range $i, $a := .Attributes }}{{ if $i }}, {{ end }}` + "`" + `{{ $a.Name }}` + "`" + ` {{ $a.Type }}{{ if $a.Required }}*{{ end }}{{ if $a.Enum }} [{{ $a.Enum }}]{{ end }}{{ end }} |
{{ end }}
_* = required. Enum values shown in brackets._

## 3. Action Wiring

Actions use the form ` + "`namespace:call(key=\"value\")`" + `.

### Built-in ui: actions
{{ range .BuiltInActions -}}
- ` + "`{{ . }}`" + `
{{ end }}
### Go backend wiring

` + "```go" + `
{{ .GoWiringSnippet }}
` + "```" + `

### JavaScript wiring

` + "```js" + `
{{ .JSWiringSnippet }}
` + "```" + `

## 4. Golden Examples

### 4.1 Dashboard
` + "```cui" + `
Theme(id="dark") {
  Color(key="primary", value="#0984e3"),
  Color(key="surface", value="#1e1e1e"),
  Color(key="panel", value="#2d3436")
}
View(id="app", title="Ops Console", theme="dark") {
  Stack(id="layout", dir="v", gap=16px) {
    Text(id="header", value="Operations", size=24px, weight="bold")
    Grid(id="content", cols=[1*, 2*], rows=[1*, 1*], gap=8px) {
      Box(id="filters", padding=12px, border=1, background="surface") {
        Input(id="query", type="text", label="Search", bind="query")
        Trigger(id="run_btn", label="Run", action=app:search(query="query"), variant="primary")
      }
      DataTable(id="table", source="incidents", selectable=true)
    }
  }
}
` + "```" + `

### 4.2 Form
` + "```cui" + `
View(id="form_view", title="New Incident") {
  Stack(id="form_stack", dir="v", gap=16px) {
    Text(id="form_title", value="Report Incident", size=24px, weight="bold")
    Input(id="title_input", type="text", label="Title", bind="title")
    Input(id="severity_input", type="text", label="Severity", bind="severity")
    Input(id="desc_input", type="text", label="Description", bind="description")
    Stack(id="actions_row", dir="h", gap=8px) {
      Trigger(id="submit_btn", label="Submit", action=app:submit(), variant="primary")
      Trigger(id="cancel_btn", label="Cancel", action=ui:close(), variant="secondary")
    }
  }
}
` + "```" + `

### 4.3 Technical Diagram
` + "```cui" + `
Theme(id="Modern") {
  Color(key="primary", value="#6366f1"),
  Color(key="surface", value="#ffffff"),
  Color(key="background", value="#f8fafc")
}
View(id="diagram_view", title="System Metrics", theme="Modern") {
  Stack(id="diagram_stack", dir="v", gap=20px) {
    Text(id="diagram_title", value="Throughput Analysis", size=22px, weight="bold")
    Graph(id="throughput_graph", type="line", color="primary", height=240px,
          labels=["08:00","10:00","12:00","14:00"],
          data="app:metrics.throughput_series")
    Graph(id="error_graph", type="bar", color="primary", height=180px,
          labels=["HTTP 2xx","HTTP 4xx","HTTP 5xx"],
          data=[842, 37, 12])
  }
}
` + "```" + `
`

// RenderExplain generates a high-density Markdown reference document from the
// live registry and parser specs, optimised for LLM token efficiency.
func RenderExplain() (string, error) {
	components := registry.AllComponents()
	data := ExplainData{
		RegistryVersion: registry.Version,
		LastUpdated:     registry.LastUpdated,
		Components:      make([]ExplainComponentDoc, 0, len(components)),
		BuiltInActions:  registry.BuiltInActions(),
		GoWiringSnippet: goWiringSnippet,
		JSWiringSnippet: jsWiringSnippet,
	}

	for _, comp := range components {
		doc := ExplainComponentDoc{
			Name:        comp.Name,
			Intent:      comp.Intent,
			HasChildren: comp.HasChildren,
		}

		names := make([]string, 0, len(comp.Attributes))
		for name := range comp.Attributes {
			names = append(names, name)
		}
		sortStrings(names)

		for _, name := range names {
			attr := comp.Attributes[name]
			enumVals := make([]string, 0, len(attr.Enum))
			for v := range attr.Enum {
				enumVals = append(enumVals, v)
			}
			sortStrings(enumVals)
			enumStr := strings.Join(enumVals, "|")

			doc.Attributes = append(doc.Attributes, ExplainAttrDoc{
				Name:     name,
				Type:     string(attr.Type),
				Required: attr.Required,
				Enum:     enumStr,
			})
		}

		data.Components = append(data.Components, doc)
	}

	return executeTemplate(explainTemplate, data)
}

func executeTemplate(tmplText string, data any) (string, error) {
	tmpl, err := template.New("docs").Parse(tmplText)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		return "", err
	}

	return buffer.String(), nil
}

func formatType(valueType registry.ValueType) string {
	parts := strings.Split(string(valueType), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func docType(attribute registry.AttributeSpec) string {
	if attribute.DocType != "" {
		return attribute.DocType
	}
	return formatType(attribute.Type)
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func buildSemanticTokens() []SemanticTokenDoc {
	semantic := registry.SemanticTokens()
	names := make([]string, 0, len(semantic))
	for name := range semantic {
		names = append(names, name)
	}
	sortStrings(names)

	out := make([]SemanticTokenDoc, 0, len(names))
	for _, name := range names {
		out = append(out, SemanticTokenDoc{
			Name:   name,
			Values: append([]string(nil), semantic[name]...),
		})
	}
	return out
}

func buildFactoryThemes() []FactoryThemeDoc {
	themes := registry.FactoryThemes()
	out := make([]FactoryThemeDoc, 0, len(themes))
	for _, theme := range themes {
		keys := make([]string, 0, len(theme.Tokens))
		for key := range theme.Tokens {
			keys = append(keys, key)
		}
		sortStrings(keys)

		doc := FactoryThemeDoc{
			Name:   theme.Name,
			Tokens: make([]ThemeTokenDoc, 0, len(keys)),
		}
		for _, key := range keys {
			doc.Tokens = append(doc.Tokens, ThemeTokenDoc{
				Key:   key,
				Value: theme.Tokens[key],
			})
		}
		out = append(out, doc)
	}
	return out
}

func componentBestPractices(name string) string {
	switch name {
	case "Graph":
		return graphBestPractices
	default:
		return ""
	}
}
