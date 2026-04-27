package generator

import (
	"time"

	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/valueforvalue/coreui/pkg/flow"
	"github.com/valueforvalue/coreui/pkg/registry"
)

type IndexEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type Metadata struct {
	CompiledAt      string         `json:"compiled_at"`
	Version         string         `json:"version"`
	RegistryVersion string         `json:"registry_version"`
	FlowState       map[string]any `json:"flow_state,omitempty"`
}

type Node struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	Intent     string         `json:"intent,omitempty"`
	Attributes map[string]any `json:"attributes,omitempty"`
	Children   []*Node        `json:"children,omitempty"`
}

type Output struct {
	Tree     *Node                 `json:"tree"`
	Index    map[string]IndexEntry `json:"index"`
	Theme    map[string]string     `json:"theme,omitempty"`
	Metadata Metadata              `json:"metadata"`
}

func Build(document *ast.Document, compiledAt time.Time, version string) Output {
	index := map[string]IndexEntry{}
	var tree *Node
	if document != nil && document.Tree != nil {
		tree = buildNode(document.Tree, "/tree", index)
	}

	return Output{
		Tree:  tree,
		Index: index,
		Theme: cloneTheme(document),
		Metadata: Metadata{
			CompiledAt:      compiledAt.UTC().Format(time.RFC3339),
			Version:         version,
			RegistryVersion: registry.Version,
		},
	}
}

// BuildWithFlow is like Build but also embeds the CoreFlow initial state values
// into metadata.flow_state so the GOTH server renderer can seed the server-side
// view to match the client state engine's starting values.
func BuildWithFlow(document *ast.Document, flowDoc *flow.FlowDocument, compiledAt time.Time, version string) Output {
	output := Build(document, compiledAt, version)
	if flowDoc != nil && flowDoc.State != nil {
		output.Metadata.FlowState = extractFlowState(flowDoc.State)
	}
	return output
}

// extractFlowState converts a StateBlock into a flat map of initial values
// for inclusion in the JSON metadata.
func extractFlowState(state *flow.StateBlock) map[string]any {
	result := make(map[string]any, len(state.Vars))
	for _, v := range state.Vars {
		switch v.Kind {
		case flow.VarKindList:
			result[v.Name] = []any{}
		case flow.VarKindMap:
			result[v.Name] = map[string]any{}
		default:
			result[v.Name] = flowExprToValue(v.Init)
		}
	}
	return result
}

// flowExprToValue converts a simple flow Expr to a Go value suitable for JSON.
func flowExprToValue(expr flow.Expr) any {
	if len(expr) == 0 {
		return nil
	}
	// Single-token expressions are the common case for var declarations.
	if len(expr) == 1 {
		tok := expr[0]
		switch tok.Kind {
		case flow.TokBool:
			return tok.Val == "true"
		case flow.TokInt:
			var n int64
			for _, ch := range tok.Val {
				if ch == '-' {
					continue
				}
				n = n*10 + int64(ch-'0')
			}
			if len(tok.Val) > 0 && tok.Val[0] == '-' {
				n = -n
			}
			return n
		case flow.TokFloat:
			// Return as string; caller can parse if needed.
			return tok.Val
		case flow.TokStr:
			return tok.Val
		}
	}
	// Multi-token — return the raw expression string for reference.
	parts := make([]string, 0, len(expr))
	for _, tok := range expr {
		parts = append(parts, tok.Val)
	}
	return parts
}

func buildNode(node *ast.Node, path string, index map[string]IndexEntry) *Node {
	jsonNode := &Node{
		Type: node.Type,
		ID:   node.ID(),
	}

	if spec, ok := registry.GetComponent(node.Type); ok && spec.Intent != "" {
		jsonNode.Intent = spec.Intent
	}

	if attributes := convertAttributes(node.Attributes); len(attributes) > 0 {
		jsonNode.Attributes = attributes
	}

	index[node.ID()] = IndexEntry{
		Path: path,
		Type: node.Type,
	}

	if len(node.Children) > 0 {
		jsonNode.Children = make([]*Node, 0, len(node.Children))
		for i, child := range node.Children {
			childPath := path + "/children/" + itoa(i)
			jsonNode.Children = append(jsonNode.Children, buildNode(child, childPath, index))
		}
	}

	return jsonNode
}

func convertAttributes(attributes map[string]ast.Value) map[string]any {
	if len(attributes) == 0 {
		return nil
	}

	converted := map[string]any{}
	for key, value := range attributes {
		if key == "id" {
			continue
		}
		converted[key] = convertValue(value)
	}

	if len(converted) == 0 {
		return nil
	}

	return converted
}

func convertValue(value ast.Value) any {
	switch value.Kind {
	case ast.StringKind, ast.UnitKind:
		return value.Data
	case ast.BoolKind:
		return value.Data
	case ast.IntKind:
		return value.Data
	case ast.NumberKind:
		return value.Data
	case ast.ArrayKind:
		items, _ := value.Data.([]ast.Value)
		converted := make([]any, 0, len(items))
		for _, item := range items {
			converted = append(converted, convertValue(item))
		}
		return converted
	case ast.ActionKind:
		action, _ := value.Data.(ast.Action)
		params := map[string]any{}
		for key, item := range action.Params {
			params[key] = convertValue(item)
		}
		return map[string]any{
			"namespace": action.Namespace,
			"call":      action.Call,
			"params":    params,
		}
	default:
		return value.Data
	}
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}

	buf := [20]byte{}
	pos := len(buf)
	for value > 0 {
		pos--
		buf[pos] = byte('0' + (value % 10))
		value /= 10
	}

	return string(buf[pos:])
}

func cloneTheme(document *ast.Document) map[string]string {
	if document == nil || len(document.Theme) == 0 {
		return nil
	}
	theme := make(map[string]string, len(document.Theme))
	for key, value := range document.Theme {
		theme[key] = value
	}
	return theme
}
