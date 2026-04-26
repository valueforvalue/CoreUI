package goth

import (
	"strings"

	"github.com/a-h/templ"
	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/valueforvalue/coreui/pkg/registry"
)

// Render converts a CoreUI AST node into a templ component tree.
//
// To serve the result with net/http, call Render on the root node and render
// the returned component inside your HTTP handler or page template.
func Render(node ast.Node) templ.Component {
	return RenderWithTheme(node, nil)
}

// RenderWithTheme converts a CoreUI AST node into a templ component tree and
// injects CSS custom properties for the provided theme tokens.
func RenderWithTheme(node ast.Node, theme map[string]string) templ.Component {
	return renderNode(&node, theme)
}

func renderNode(node *ast.Node, theme map[string]string) templ.Component {
	if node == nil {
		return templ.Raw("")
	}

	children := renderChildren(node.Children, theme)
	id := node.ID()

	switch node.Type {
	case "View":
		return viewComponent(id, stringAttribute(node, "title"), baseStyle(node, theme), RenderTheme(theme), children)
	case "Stack":
		direction := "column"
		if stringAttribute(node, "dir") == "h" {
			direction = "row"
		}

		decls := []styleDecl{
			styleDeclFor("display", "flex"),
			styleDeclFor("flex-direction", direction),
		}
		if gap := UnitToCSS(unitAttribute(node, "gap"), UnitContextLiteral); gap != "" {
			decls = append(decls, styleDeclFor("gap", gap))
		}
		if align := sanitizeCSSToken(stringAttribute(node, "align")); align != "" {
			decls = append(decls, styleDeclFor("align-items", align))
		}

		return stackComponent(id, baseStyle(node, theme, decls...), children)
	case "Grid":
		decls := []styleDecl{
			styleDeclFor("display", "grid"),
		}
		if cols := unitArrayAttribute(node, "cols", UnitContextGridTrack); cols != "" {
			decls = append(decls, styleDeclFor("grid-template-columns", cols))
		}
		if rows := unitArrayAttribute(node, "rows", UnitContextGridTrack); rows != "" {
			decls = append(decls, styleDeclFor("grid-template-rows", rows))
		}
		if gap := UnitToCSS(unitAttribute(node, "gap"), UnitContextLiteral); gap != "" {
			decls = append(decls, styleDeclFor("gap", gap))
		}

		return gridComponent(id, baseStyle(node, theme, decls...), children)
	case "Box":
		decls := make([]styleDecl, 0, 8)
		decls = append(decls, semanticStyleDecls(theme, false, true)...)
		decls = append(decls, variantStyleDecls(stringAttribute(node, "variant"), theme)...)
		if padding := UnitToCSS(unitAttribute(node, "padding"), UnitContextLiteral); padding != "" {
			decls = append(decls, styleDeclFor("padding", padding))
		}
		if background := stringAttribute(node, "background"); background != "" {
			decls = append(decls, styleDeclFor("background", background))
		}
		if border, ok := intAttribute(node, "border"); ok && border > 0 {
			decls = append(decls,
				styleDeclFor("border-width", formatInt(border)+"px"),
				styleDeclFor("border-style", "solid"),
			)
		}

		return boxComponent(id, baseStyle(node, theme, decls...), children)
	case "Text":
		decls := make([]styleDecl, 0, 2)
		if size := UnitToCSS(unitAttribute(node, "size"), UnitContextLiteral); size != "" {
			decls = append(decls, styleDeclFor("font-size", size))
		}
		if weight := sanitizeCSSToken(stringAttribute(node, "weight")); weight != "" {
			decls = append(decls, styleDeclFor("font-weight", weight))
		}

		return textComponent(id, stringAttribute(node, "value"), baseStyle(node, theme, decls...))
	case "Input":
		inputType := sanitizeHTMLToken(stringAttribute(node, "type"))
		if inputType == "" {
			inputType = "text"
		}
		decls := semanticStyleDecls(theme, true, false)
		return inputComponent(
			id,
			stringAttribute(node, "label"),
			stringAttribute(node, "bind"),
			inputType,
			baseStyle(node, theme, decls...),
		)
	case "Image":
		decls := make([]styleDecl, 0, 1)
		if width := UnitToCSS(unitAttribute(node, "width"), UnitContextLiteral); width != "" {
			decls = append(decls, styleDeclFor("width", width))
		}
		return imageComponent(
			id,
			stringAttribute(node, "src"),
			stringAttribute(node, "alt"),
			baseStyle(node, theme, decls...),
		)
	case "Trigger":
		action, _ := actionAttribute(node, "action")
		payload := encodeActionRequest(actionRequestFromAction(action))
		decls := make([]styleDecl, 0, 8)
		decls = append(decls, semanticStyleDecls(theme, true, true)...)
		decls = append(decls, variantStyleDecls(stringAttribute(node, "variant"), theme)...)
		return triggerComponent(
			id,
			stringAttribute(node, "label"),
			sanitizeHTMLToken(stringAttribute(node, "variant")),
			payload,
			baseStyle(node, theme, decls...),
		)
	case "DataTable":
		return dataTableComponent(
			id,
			stringAttribute(node, "source"),
			boolAttribute(node, "selectable"),
			baseStyle(node, theme),
			children,
		)
	case "Graph":
		values, reference := graphDataAttribute(node, "data")
		decls := []styleDecl{
			styleDeclFor("display", "flex"),
			styleDeclFor("flex-direction", "column"),
			styleDeclFor("gap", "12px"),
			styleDeclFor("width", "100%"),
		}
		height := UnitToCSS(unitAttribute(node, "height"), UnitContextLiteral)
		if height == "" {
			height = "240px"
		}
		return graphComponent(
			id,
			baseStyle(node, theme, decls...),
			stringAttribute(node, "type"),
			values,
			stringArrayAttribute(node, "labels"),
			reference,
			graphColorValue(stringAttribute(node, "color"), theme),
			graphRadiusValue(theme),
			height,
		)
	default:
		if registry.IsPluginComponent(node.Type) {
			return pluginComponent(id, node.Type, baseStyle(node, theme), children)
		}
		return unknownComponent(id, node.Type, baseStyle(node, theme), children)
	}
}

func renderChildren(children []*ast.Node, theme map[string]string) []templ.Component {
	if len(children) == 0 {
		return nil
	}

	components := make([]templ.Component, 0, len(children))
	for _, child := range children {
		components = append(components, renderNode(child, theme))
	}
	return components
}

func stringAttribute(node *ast.Node, key string) string {
	if node == nil {
		return ""
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.StringKind {
		return ""
	}
	text, _ := value.Data.(string)
	return text
}

func unitAttribute(node *ast.Node, key string) string {
	if node == nil {
		return ""
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.UnitKind {
		return ""
	}
	text, _ := value.Data.(string)
	return text
}

func unitArrayAttribute(node *ast.Node, key string, context UnitContext) string {
	if node == nil {
		return ""
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.ArrayKind {
		return ""
	}
	items, _ := value.Data.([]ast.Value)
	if len(items) == 0 {
		return ""
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.Kind != ast.UnitKind {
			continue
		}
		text, _ := item.Data.(string)
		if css := UnitToCSS(text, context); css != "" {
			parts = append(parts, css)
		}
	}

	return strings.Join(parts, " ")
}

func boolAttribute(node *ast.Node, key string) bool {
	if node == nil {
		return false
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.BoolKind {
		return false
	}
	flag, _ := value.Data.(bool)
	return flag
}

func intAttribute(node *ast.Node, key string) (int64, bool) {
	if node == nil {
		return 0, false
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.IntKind {
		return 0, false
	}
	number, ok := value.Data.(int64)
	return number, ok
}

func formatInt(value int64) string {
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

func actionAttribute(node *ast.Node, key string) (ast.Action, bool) {
	if node == nil {
		return ast.Action{}, false
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.ActionKind {
		return ast.Action{}, false
	}
	action, ok := value.Data.(ast.Action)
	return action, ok
}

func stringArrayAttribute(node *ast.Node, key string) []string {
	if node == nil {
		return nil
	}
	value, ok := node.Attributes[key]
	if !ok || value.Kind != ast.ArrayKind {
		return nil
	}
	items, _ := value.Data.([]ast.Value)
	if len(items) == 0 {
		return nil
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		if item.Kind != ast.StringKind {
			continue
		}
		text, _ := item.Data.(string)
		out = append(out, text)
	}
	return out
}

func graphDataAttribute(node *ast.Node, key string) ([]float64, string) {
	if node == nil {
		return nil, ""
	}
	value, ok := node.Attributes[key]
	if !ok {
		return nil, ""
	}

	switch value.Kind {
	case ast.StringKind:
		text, _ := value.Data.(string)
		return nil, text
	case ast.ArrayKind:
		items, _ := value.Data.([]ast.Value)
		out := make([]float64, 0, len(items))
		for _, item := range items {
			switch item.Kind {
			case ast.IntKind:
				number, _ := item.Data.(int64)
				out = append(out, float64(number))
			case ast.NumberKind:
				number, _ := item.Data.(float64)
				out = append(out, number)
			}
		}
		return out, ""
	default:
		return nil, ""
	}
}

func graphColorValue(value string, theme map[string]string) string {
	if token := resolveThemeToken(value, theme); token != "" {
		return token
	}
	if token := resolveThemeToken("primary", theme); token != "" {
		return token
	}
	if sanitized := sanitizeCSSValue(value); sanitized != "" {
		return sanitized
	}
	return "#6366f1"
}

func graphRadiusValue(theme map[string]string) string {
	if radius := semanticTokenCSSValue("radius", theme["radius"]); radius != "" {
		return radius
	}
	return "8px"
}
