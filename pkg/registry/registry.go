package registry

import (
	"fmt"

	"coreui/pkg/ast"
)

type ValueType string

const (
	StringType    ValueType = "string"
	BoolType      ValueType = "bool"
	IntType       ValueType = "int"
	NumberType    ValueType = "number"
	UnitType      ValueType = "unit"
	UnitArrayType ValueType = "unit_array"
	ActionType    ValueType = "action"
)

type AttributeSpec struct {
	Type     ValueType
	Required bool
	Enum     map[string]struct{}
}

type ComponentSpec struct {
	Name       string
	Attributes map[string]AttributeSpec
}

var commonAttributes = map[string]AttributeSpec{
	"id":     {Type: StringType, Required: true},
	"hidden": {Type: BoolType},
	"style":  {Type: StringType},
}

var componentSpecs = map[string]ComponentSpec{
	"View": {
		Name: "View",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"title": {Type: StringType},
			"theme": {Type: StringType},
		}),
	},
	"Stack": {
		Name: "Stack",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"dir":   {Type: StringType, Enum: enumSet("h", "v")},
			"gap":   {Type: UnitType},
			"align": {Type: StringType},
		}),
	},
	"Grid": {
		Name: "Grid",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"cols": {Type: UnitArrayType},
			"rows": {Type: UnitArrayType},
			"gap":  {Type: UnitType},
		}),
	},
	"Box": {
		Name: "Box",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"padding":    {Type: UnitType},
			"border":     {Type: IntType},
			"background": {Type: StringType},
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
	"Trigger": {
		Name: "Trigger",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"label":   {Type: StringType},
			"action":  {Type: ActionType},
			"variant": {Type: StringType},
		}),
	},
	"DataTable": {
		Name: "DataTable",
		Attributes: mergeCommon(map[string]AttributeSpec{
			"source":     {Type: StringType},
			"selectable": {Type: BoolType},
		}),
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

func IsAttributeAllowed(component, attr string) bool {
	spec, ok := componentSpecs[component]
	if !ok {
		return false
	}

	_, ok = spec.Attributes[attr]
	return ok
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
