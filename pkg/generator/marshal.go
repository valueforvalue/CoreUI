package generator

import (
	"fmt"
	"sort"
	"strings"

	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/valueforvalue/coreui/pkg/registry"
)

// MarshalDSL converts a generator.Output back to a .cui DSL source string.
//
// The attribute values are serialised using ast.Value.ToDSLString() to ensure
// the output is byte-for-byte compatible with the .cui parser.  Theme metadata
// is not reconstructed (the generator.Output does not preserve individual Theme
// component definitions); any View.theme reference is preserved as-is.
func MarshalDSL(output Output) (string, error) {
	if output.Tree == nil {
		return "", fmt.Errorf("marshal: output tree is nil")
	}
	var b strings.Builder
	if err := marshalNode(&b, output.Tree, 0); err != nil {
		return "", err
	}
	return b.String(), nil
}

func marshalNode(b *strings.Builder, node *Node, depth int) error {
	if node == nil {
		return nil
	}

	spec, ok := registry.GetComponent(node.Type)
	if !ok {
		return fmt.Errorf("marshal: unknown component %q", node.Type)
	}

	indent := strings.Repeat("    ", depth)
	b.WriteString(indent)
	b.WriteString(node.Type)
	b.WriteByte('(')

	// Build ordered attribute assignments: id first, then sorted remainder.
	type attrAssign struct {
		key string
		dsl string
	}

	assigns := make([]attrAssign, 0, len(node.Attributes)+1)

	// Always emit id.
	assigns = append(assigns, attrAssign{key: "id", dsl: `"` + marshalEscapeString(node.ID) + `"`})

	// Remaining attributes in sorted order.
	keys := make([]string, 0, len(node.Attributes))
	for k := range node.Attributes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, key := range keys {
		attrSpec, exists := spec.Attributes[key]
		if !exists {
			continue // skip unrecognised attributes
		}
		dsl, err := marshalAttrValue(node.Attributes[key], attrSpec.Type)
		if err != nil {
			return fmt.Errorf("marshal: attribute %q on %s: %w", key, node.Type, err)
		}
		assigns = append(assigns, attrAssign{key: key, dsl: dsl})
	}

	parts := make([]string, 0, len(assigns))
	for _, a := range assigns {
		parts = append(parts, a.key+"="+a.dsl)
	}
	b.WriteString(strings.Join(parts, ", "))
	b.WriteByte(')')

	if len(node.Children) == 0 {
		return nil
	}

	if !spec.HasChildren {
		return nil
	}

	b.WriteString(" {\n")
	for i, child := range node.Children {
		if err := marshalNode(b, child, depth+1); err != nil {
			return err
		}
		if i < len(node.Children)-1 {
			b.WriteByte(',')
		}
		b.WriteByte('\n')
	}
	b.WriteString(indent)
	b.WriteByte('}')

	return nil
}

// marshalAttrValue formats a raw JSON attribute value (as stored in
// generator.Node.Attributes) using the registry type information and the
// ast.Value.ToDSLString() method.
func marshalAttrValue(raw any, vtype registry.ValueType) (string, error) {
	v, err := reconstructValue(raw, vtype)
	if err != nil {
		return "", err
	}

	dsl := v.ToDSLString()

	// Wrap string values in double quotes for the .cui attribute assignment.
	if v.Kind == ast.StringKind {
		return `"` + marshalEscapeString(dsl) + `"`, nil
	}
	return dsl, nil
}

// reconstructValue converts a raw any value (from JSON) and the registry type
// spec back into a typed ast.Value so that ToDSLString() can be called.
func reconstructValue(raw any, vtype registry.ValueType) (ast.Value, error) {
	switch vtype {
	case registry.StringType:
		s, ok := raw.(string)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected string, got %T", raw)
		}
		return ast.Value{Kind: ast.StringKind, Data: s}, nil

	case registry.BoolType:
		b, ok := raw.(bool)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected bool, got %T", raw)
		}
		return ast.Value{Kind: ast.BoolKind, Data: b}, nil

	case registry.IntType:
		switch n := raw.(type) {
		case int64:
			return ast.Value{Kind: ast.IntKind, Data: n}, nil
		case float64:
			return ast.Value{Kind: ast.IntKind, Data: int64(n)}, nil
		default:
			return ast.Value{}, fmt.Errorf("expected int, got %T", raw)
		}

	case registry.NumberType:
		switch n := raw.(type) {
		case float64:
			return ast.Value{Kind: ast.NumberKind, Data: n}, nil
		case int64:
			return ast.Value{Kind: ast.NumberKind, Data: float64(n)}, nil
		default:
			return ast.Value{}, fmt.Errorf("expected number, got %T", raw)
		}

	case registry.UnitType:
		s, ok := raw.(string)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected unit string, got %T", raw)
		}
		return ast.Value{Kind: ast.UnitKind, Data: s}, nil

	case registry.UnitArrayType:
		return reconstructArray(raw, ast.UnitKind)

	case registry.StringArrayType:
		return reconstructArray(raw, ast.StringKind)

	case registry.NumberArrayOrReferenceType:
		// Could be a string reference (app:…) or a numeric array.
		if s, ok := raw.(string); ok {
			return ast.Value{Kind: ast.StringKind, Data: s}, nil
		}
		return reconstructNumericArray(raw)

	case registry.ActionType:
		m, ok := raw.(map[string]any)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected action object, got %T", raw)
		}
		ns, _ := m["namespace"].(string)
		call, _ := m["call"].(string)
		rawParams, _ := m["params"].(map[string]any)
		params := make(map[string]ast.Value, len(rawParams))
		for k, pv := range rawParams {
			pval, err := inferActionParamValue(pv)
			if err != nil {
				return ast.Value{}, fmt.Errorf("action param %q: %w", k, err)
			}
			params[k] = pval
		}
		return ast.Value{Kind: ast.ActionKind, Data: ast.Action{
			Namespace: ns,
			Call:      call,
			Params:    params,
		}}, nil

	default:
		s, ok := raw.(string)
		if !ok {
			return ast.Value{}, fmt.Errorf("unsupported type %q for value %T", vtype, raw)
		}
		return ast.Value{Kind: ast.StringKind, Data: s}, nil
	}
}

// inferActionParamValue infers the ast.Value kind from the native Go type of an
// action parameter value decoded from JSON.  Action params support string, bool,
// int, float64, and unit (a string suffixed with px/% etc.).
func inferActionParamValue(raw any) (ast.Value, error) {
	switch v := raw.(type) {
	case string:
		return ast.Value{Kind: ast.StringKind, Data: v}, nil
	case bool:
		return ast.Value{Kind: ast.BoolKind, Data: v}, nil
	case float64:
		if v == float64(int64(v)) {
			return ast.Value{Kind: ast.IntKind, Data: int64(v)}, nil
		}
		return ast.Value{Kind: ast.NumberKind, Data: v}, nil
	case int64:
		return ast.Value{Kind: ast.IntKind, Data: v}, nil
	default:
		return ast.Value{}, fmt.Errorf("unsupported action param type %T", raw)
	}
}

func reconstructArray(raw any, itemKind ast.ValueKind) (ast.Value, error) {
	items, ok := raw.([]any)
	if !ok {
		return ast.Value{}, fmt.Errorf("expected array, got %T", raw)
	}
	values := make([]ast.Value, 0, len(items))
	for _, item := range items {
		s, ok := item.(string)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected array item string, got %T", item)
		}
		values = append(values, ast.Value{Kind: itemKind, Data: s})
	}
	return ast.Value{Kind: ast.ArrayKind, Data: values}, nil
}

func reconstructNumericArray(raw any) (ast.Value, error) {
	items, ok := raw.([]any)
	if !ok {
		return ast.Value{}, fmt.Errorf("expected numeric array, got %T", raw)
	}
	values := make([]ast.Value, 0, len(items))
	for _, item := range items {
		switch n := item.(type) {
		case float64:
			values = append(values, ast.Value{Kind: ast.NumberKind, Data: n})
		case int64:
			values = append(values, ast.Value{Kind: ast.NumberKind, Data: float64(n)})
		default:
			return ast.Value{}, fmt.Errorf("expected numeric array item, got %T", item)
		}
	}
	return ast.Value{Kind: ast.ArrayKind, Data: values}, nil
}

// marshalEscapeString escapes backslashes and double quotes for use inside
// a double-quoted DSL string literal.
func marshalEscapeString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
