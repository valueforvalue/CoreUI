package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"

	"coreui/pkg/registry"
)

type attributeDoc struct {
	Name        string
	Type        string
	Requirement string
}

type componentDoc struct {
	Name        string
	HasChildren bool
	Attributes  []attributeDoc
}

type documentData struct {
	RegistryVersion     string
	SchemaCompatibility string
	LastUpdated         string
	Components          []componentDoc
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
{{ range .Attributes }}
| {{ .Name }} | {{ .Type }} | {{ .Requirement }} |
{{ end }}

{{ end }}
`

func main() {
	components := registry.AllComponents()
	data := documentData{
		RegistryVersion:     registry.Version,
		SchemaCompatibility: registry.SchemaCompatibility,
		LastUpdated:         registry.LastUpdated,
		Components:          make([]componentDoc, 0, len(components)),
	}

	for _, component := range components {
		doc := componentDoc{
			Name:        component.Name,
			HasChildren: component.HasChildren,
			Attributes:  make([]attributeDoc, 0, len(component.Attributes)),
		}

		names := make([]string, 0, len(component.Attributes))
		for name := range component.Attributes {
			names = append(names, name)
		}
		sort.Strings(names)

		for _, name := range names {
			attribute := component.Attributes[name]
			requirement := "Optional"
			if attribute.Required {
				requirement = "Required"
			}

			doc.Attributes = append(doc.Attributes, attributeDoc{
				Name:        name,
				Type:        formatType(attribute.Type),
				Requirement: requirement,
			})
		}

		data.Components = append(data.Components, doc)
	}

	tmpl, err := template.New("components").Parse(componentsTemplate)
	if err != nil {
		fail(err)
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, data); err != nil {
		fail(err)
	}

	if err := os.WriteFile("COMPONENTS.md", buffer.Bytes(), 0o644); err != nil {
		fail(err)
	}

	fmt.Println("COMPONENTS.md")
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

func fail(err error) {
	fmt.Fprintln(os.Stderr, err.Error())
	os.Exit(1)
}
