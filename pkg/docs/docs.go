package docs

import (
	"bytes"
	"strings"
	"text/template"

	"coreui/pkg/registry"
)

type AttributeDoc struct {
	Name        string
	Type        string
	Requirement string
}

type ComponentDoc struct {
	Name        string
	HasChildren bool
	Attributes  []AttributeDoc
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

{{ end }}
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

| Attribute | Type | Requirement |
| --- | --- | --- |
{{ range .Attributes }}| {{ .Name }} | {{ .Type }} | {{ .Requirement }} |
{{ end }}

{{ end }}
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
				Type:        formatType(attribute.Type),
				Requirement: requirement,
			})
		}

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
	})
}

func GoWiringSnippet() string {
	return goWiringSnippet
}

func JSWiringSnippet() string {
	return jsWiringSnippet
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
