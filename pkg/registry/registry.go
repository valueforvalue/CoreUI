package registry

import (
	"fmt"
	"sort"
	"strings"

	"coreui/pkg/ast"
)

const (
	Version             = "1.3.0"
	SchemaCompatibility = "1.0"
	LastUpdated         = "2026-04-26"
)

var standardThemeTokens = []string{"primary", "surface", "panel", "background", "text", "radius", "shadow", "speed"}

var semanticTokens = map[string][]string{
	"radius": {"none", "sm", "md", "lg", "full"},
	"shadow": {"none", "soft", "deep"},
	"speed":  {"instant", "smooth", "lazy"},
}

var builtInActions = []string{
	`ui:navigate(target="id")`,
	`ui:toggle(id="target_id")`,
	`ui:close()`,
	`ui:notify(msg="Done", type="success")`,
}

type FactoryTheme struct {
	Name   string
	Tokens map[string]string
}

var factoryThemes = []FactoryTheme{
	{
		Name: "Industrial",
		Tokens: map[string]string{
			"radius":     "none",
			"shadow":     "none",
			"speed":      "instant",
			"surface":    "#dbe4f0",
			"panel":      "#ffffff",
			"background": "surface",
			"text":       "#111827",
			"primary":    "#2563eb",
		},
	},
	{
		Name: "Modern",
		Tokens: map[string]string{
			"radius":     "md",
			"shadow":     "soft",
			"speed":      "smooth",
			"surface":    "#ffffff",
			"panel":      "#ffffff",
			"background": "#f8fafc",
			"text":       "#0f172a",
			"primary":    "#6366f1",
		},
	},
	{
		Name: "Cyber",
		Tokens: map[string]string{
			"radius":     "none",
			"shadow":     "none",
			"speed":      "instant",
			"surface":    "#000000",
			"panel":      "#000000",
			"background": "#000000",
			"text":       "#00ff00",
			"primary":    "#00ff00",
		},
	},
}

type ValueType string

const (
	StringType                 ValueType = "string"
	BoolType                   ValueType = "bool"
	IntType                    ValueType = "int"
	NumberType                 ValueType = "number"
	UnitType                   ValueType = "unit"
	UnitArrayType              ValueType = "unit_array"
	StringArrayType            ValueType = "string_array"
	NumberArrayOrReferenceType ValueType = "number_array_or_reference"
	ActionType                 ValueType = "action"
)

type AttributeSpec struct {
	Type     ValueType
	Required bool
	Enum     map[string]struct{}
	DocType  string
}

type ComponentSpec struct {
	Name        string
	HasChildren bool
	Attributes  map[string]AttributeSpec
}

var commonAttributes = map[string]AttributeSpec{
	"id":     {Type: StringType, Required: true},
	"hidden": {Type: BoolType},
	"style":  {Type: StringType},
}

var componentSpecs = map[string]ComponentSpec{
	"View": {
		Name:        "View",
		HasChildren: true,
		Attributes: mergeCommon(map[string]AttributeSpec{
			"title": {Type: StringType},
			"theme": {Type: StringType},
		}),
	},
	"Stack": {
		Name:        "Stack",
		HasChildren: true,
		Attributes: mergeCommon(map[string]AttributeSpec{
			"dir":   {Type: StringType, Enum: enumSet("h", "v")},
			"gap":   {Type: UnitType},
			"align": {Type: StringType},
		}),
	},
	"Grid": {
		Name:        "Grid",
		HasChildren: true,
		Attributes: mergeCommon(map[string]AttributeSpec{
			"cols": {Type: UnitArrayType},
			"rows": {Type: UnitArrayType},
			"gap":  {Type: UnitType},
		}),
	},
	"Box": {
		Name:        "Box",
		HasChildren: true,
		Attributes: mergeCommon(map[string]AttributeSpec{
			"padding":    {Type: UnitType},
			"border":     {Type: IntType},
			"background": {Type: StringType},
			"variant":    {Type: StringType, Enum: enumSet("primary", "secondary", "outline", "ghost")},
		}),
	},
	"Text": {
		Name: "Text",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"value":  {Type: StringType},
			"size":   {Type: UnitType},
			"weight": {Type: StringType},
		}),
	},
	"Input": {
		Name: "Input",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"type":  {Type: StringType},
			"label": {Type: StringType},
			"bind":  {Type: StringType},
		}),
	},
	"Image": {
		Name: "Image",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"src":   {Type: StringType, Required: true},
			"width": {Type: UnitType},
			"alt":   {Type: StringType},
		}),
	},
	"Trigger": {
		Name: "Trigger",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"label":   {Type: StringType},
			"action":  {Type: ActionType},
			"variant": {Type: StringType, Enum: enumSet("primary", "secondary", "outline", "ghost")},
		}),
	},
	"DataTable": {
		Name: "DataTable",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"source":     {Type: StringType},
			"selectable": {Type: BoolType},
		}),
	},
	"Graph": {
		Name: "Graph",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"type":   {Type: StringType, Required: true, Enum: enumSet("line", "bar", "area", "pie")},
			"data":   {Type: NumberArrayOrReferenceType, Required: true, DocType: "JSON Array or app:reference"},
			"color":  {Type: StringType, DocType: "Theme Token"},
			"height": {Type: UnitType},
			"labels": {Type: StringArrayType, DocType: "[]string"},
		}),
	},
	"Theme": {
		Name:        "Theme",
		HasChildren: true,
		Attributes:  mergeCommon(map[string]AttributeSpec{}),
	},
	"Color": {
		Name: "Color",
		Attributes: map[string]AttributeSpec{
			"key":   {Type: StringType, Required: true},
			"value": {Type: StringType, Required: true},
		},
	},
}

func mergeCommon(attributes map[string]AttributeSpec) map[string]AttributeSpec {
	merged := make(map[string]AttributeSpec, len(commonAttributes)+len(attributes))
	for key, spec := range commonAttributes {
		merged[key] = spec
	}
	for key, spec := range attributes {
		merged[key] = spec
	}
	return merged
}

func enumSet(values ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func GetComponent(name string) (ComponentSpec, bool) {
	spec, ok := componentSpecs[name]
	return spec, ok
}

func AllComponents() []ComponentSpec {
	names := make([]string, 0, len(componentSpecs))
	for name := range componentSpecs {
		names = append(names, name)
	}
	sort.Strings(names)

	components := make([]ComponentSpec, 0, len(names))
	for _, name := range names {
		spec := componentSpecs[name]
		spec.Attributes = cloneAttributes(spec.Attributes)
		components = append(components, spec)
	}

	return components
}

func IsAttributeAllowed(component, attr string) bool {
	spec, ok := componentSpecs[component]
	if !ok {
		return false
	}

	_, ok = spec.Attributes[attr]
	return ok
}

func ValidAttributes(component string) []string {
	spec, ok := componentSpecs[component]
	if !ok {
		return nil
	}

	attributes := make([]string, 0, len(spec.Attributes))
	for key := range spec.Attributes {
		attributes = append(attributes, key)
	}
	sort.Strings(attributes)
	return attributes
}

func RequiredAttributes(component string) []string {
	spec, ok := componentSpecs[component]
	if !ok {
		return nil
	}

	required := make([]string, 0, len(spec.Attributes))
	for key, attribute := range spec.Attributes {
		if attribute.Required {
			required = append(required, key)
		}
	}

	return required
}

func RequiresID(component string) bool {
	spec, ok := componentSpecs[component]
	if !ok {
		return false
	}
	attribute, ok := spec.Attributes["id"]
	return ok && attribute.Required
}

func ValidateValue(component, attr string, value ast.Value) error {
	spec, ok := componentSpecs[component]
	if !ok {
		return fmt.Errorf("unknown component %q", component)
	}

	attribute, ok := spec.Attributes[attr]
	if !ok {
		return fmt.Errorf("unknown attribute %q for %s", attr, component)
	}

	switch attribute.Type {
	case StringType:
		if value.Kind != ast.StringKind {
			return fmt.Errorf("attribute %q expects string", attr)
		}
	case BoolType:
		if value.Kind != ast.BoolKind {
			return fmt.Errorf("attribute %q expects bool", attr)
		}
	case IntType:
		if value.Kind != ast.IntKind {
			return fmt.Errorf("attribute %q expects int", attr)
		}
	case NumberType:
		if value.Kind != ast.NumberKind && value.Kind != ast.IntKind {
			return fmt.Errorf("attribute %q expects number", attr)
		}
	case UnitType:
		if value.Kind != ast.UnitKind {
			return fmt.Errorf("attribute %q expects unit", attr)
		}
	case UnitArrayType:
		if value.Kind != ast.ArrayKind {
			return fmt.Errorf("attribute %q expects array", attr)
		}
		values, ok := value.Data.([]ast.Value)
		if !ok {
			return fmt.Errorf("attribute %q expects unit array", attr)
		}
		for _, item := range values {
			if item.Kind != ast.UnitKind {
				return fmt.Errorf("attribute %q expects array of units", attr)
			}
		}
	case StringArrayType:
		if value.Kind != ast.ArrayKind {
			return fmt.Errorf("attribute %q expects array", attr)
		}
		values, ok := value.Data.([]ast.Value)
		if !ok {
			return fmt.Errorf("attribute %q expects string array", attr)
		}
		for _, item := range values {
			if item.Kind != ast.StringKind {
				return fmt.Errorf("attribute %q expects array of strings", attr)
			}
		}
	case NumberArrayOrReferenceType:
		switch value.Kind {
		case ast.ArrayKind:
			values, ok := value.Data.([]ast.Value)
			if !ok {
				return fmt.Errorf("attribute %q expects numeric array or app:reference", attr)
			}
			for _, item := range values {
				if item.Kind != ast.IntKind && item.Kind != ast.NumberKind {
					return fmt.Errorf("attribute %q expects array of numbers", attr)
				}
			}
		case ast.StringKind:
			raw, _ := value.Data.(string)
			if !strings.HasPrefix(strings.TrimSpace(raw), "app:") {
				return fmt.Errorf("attribute %q expects JSON array or app:reference", attr)
			}
		default:
			return fmt.Errorf("attribute %q expects JSON array or app:reference", attr)
		}
	case ActionType:
		if value.Kind != ast.ActionKind {
			return fmt.Errorf("attribute %q expects action", attr)
		}
	default:
		return fmt.Errorf("unsupported attribute type %q", attribute.Type)
	}

	if len(attribute.Enum) > 0 {
		raw, ok := value.Data.(string)
		if !ok {
			return fmt.Errorf("attribute %q expects one of the allowed values", attr)
		}
		if _, exists := attribute.Enum[raw]; !exists {
			return fmt.Errorf("attribute %q does not allow value %q", attr, raw)
		}
	}

	return nil
}

func cloneAttributes(attributes map[string]AttributeSpec) map[string]AttributeSpec {
	cloned := make(map[string]AttributeSpec, len(attributes))
	for key, spec := range attributes {
		cloned[key] = spec
	}
	return cloned
}

func StandardThemeTokens() []string {
	return append([]string(nil), standardThemeTokens...)
}

func BuiltInActions() []string {
	return append([]string(nil), builtInActions...)
}

func SemanticTokens() map[string][]string {
	cloned := make(map[string][]string, len(semanticTokens))
	for key, values := range semanticTokens {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func FactoryThemes() []FactoryTheme {
	cloned := make([]FactoryTheme, 0, len(factoryThemes))
	for _, theme := range factoryThemes {
		tokens := make(map[string]string, len(theme.Tokens))
		for key, value := range theme.Tokens {
			tokens[key] = value
		}
		cloned = append(cloned, FactoryTheme{
			Name:   theme.Name,
			Tokens: tokens,
		})
	}
	return cloned
}
