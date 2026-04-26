package generator

import (
	"time"

	"coreui/pkg/ast"
)

type IndexEntry struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

type Metadata struct {
	CompiledAt string `json:"compiled_at"`
	Version    string `json:"version"`
}

type Node struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
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
			CompiledAt: compiledAt.UTC().Format(time.RFC3339),
			Version:    version,
		},
	}
}

func buildNode(node *ast.Node, path string, index map[string]IndexEntry) *Node {
	jsonNode := &Node{
		Type: node.Type,
		ID:   node.ID(),
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
