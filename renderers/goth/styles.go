package goth

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/a-h/templ"
)

// UnitContext controls how fractional CoreUI units are translated to CSS.
type UnitContext int

const (
	// UnitContextLiteral keeps literal CSS units such as px, %, and auto.
	UnitContextLiteral UnitContext = iota
	// UnitContextGridTrack converts fractional units to CSS fr tracks.
	UnitContextGridTrack
	// UnitContextFlex converts fractional units to numeric flex values.
	UnitContextFlex
)

type styleDecl struct {
	property string
	value    string
}

// UnitToCSS converts a CoreUI unit into a CSS-compatible value.
func UnitToCSS(unit string, context UnitContext) string {
	unit = strings.TrimSpace(unit)
	if unit == "" {
		return ""
	}

	switch {
	case unit == "auto":
		return unit
	case strings.HasSuffix(unit, "px"), strings.HasSuffix(unit, "%"):
		if sanitizeCSSValue(unit) == "" {
			return ""
		}
		return unit
	case strings.HasSuffix(unit, "*"):
		weight := strings.TrimSpace(strings.TrimSuffix(unit, "*"))
		if weight == "" {
			weight = "1"
		}
		if !isNumericToken(weight) {
			return ""
		}

		switch context {
		case UnitContextGridTrack:
			return weight + "fr"
		case UnitContextFlex:
			return weight
		default:
			return ""
		}
	default:
		return ""
	}
}

func baseStyle(node *ast.Node, theme map[string]string, decls ...styleDecl) string {
	merged := make([]string, 0, len(decls)+1)

	if boolAttribute(node, "hidden") {
		merged = append(merged, "display: none")
	}
	for _, decl := range decls {
		if sanitized := sanitizeDeclaration(decl.property, decl.value, theme); sanitized != "" {
			merged = append(merged, sanitized)
		}
	}
	merged = append(merged, sanitizeInlineStyle(stringAttribute(node, "style"), theme)...)

	return strings.Join(merged, "; ")
}

func styleDeclFor(property, value string) styleDecl {
	return styleDecl{property: property, value: value}
}

func sanitizeInlineStyle(raw string, theme map[string]string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		index := strings.Index(part, ":")
		if index <= 0 {
			continue
		}
		property := strings.TrimSpace(part[:index])
		value := strings.TrimSpace(part[index+1:])
		if sanitized := sanitizeDeclaration(property, value, theme); sanitized != "" {
			out = append(out, sanitized)
		}
	}
	return out
}

func sanitizeDeclaration(property, value string, theme map[string]string) string {
	property = sanitizeCSSProperty(property)
	value = sanitizeStyleValue(property, value, theme)
	if property == "" || value == "" {
		return ""
	}
	return property + ": " + value
}

// RenderTheme renders CSS custom properties for a compiled CoreUI theme.
func RenderTheme(theme map[string]string) templ.Component {
	if len(theme) == 0 {
		return templ.Raw("")
	}

	keys := make([]string, 0, len(theme))
	for key := range theme {
		if sanitizeThemeKey(key) != "" && resolveThemeDefinitionValue(key, theme[key], theme) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return templ.Raw("")
	}

	var builder strings.Builder
	builder.WriteString("<style>:root{")
	if radius := resolveThemeDefinitionValue("radius", theme["radius"], theme); radius != "" {
		builder.WriteString(fmt.Sprintf("--cui-radius:%s;", radius))
	}
	if shadow := resolveThemeDefinitionValue("shadow", theme["shadow"], theme); shadow != "" {
		builder.WriteString(fmt.Sprintf("--cui-shadow:%s;", shadow))
	}
	if speed := resolveThemeDefinitionValue("speed", theme["speed"], theme); speed != "" {
		builder.WriteString(fmt.Sprintf("--cui-speed:%s;", speed))
	}
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("--coreui-%s:%s;", sanitizeThemeKey(key), resolveThemeDefinitionValue(key, theme[key], theme)))
	}
	builder.WriteString("}</style>")
	return templ.Raw(builder.String())
}

func sanitizeCSSProperty(property string) string {
	property = strings.TrimSpace(strings.ToLower(property))
	if property == "" {
		return ""
	}
	for _, r := range property {
		if (r < 'a' || r > 'z') && r != '-' {
			return ""
		}
	}
	return property
}

func sanitizeCSSValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, r := range value {
		if strings.ContainsRune("\"';{}<>[]\\`\n\r", r) {
			return ""
		}
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" #%.(),-_/+", r)) {
			return ""
		}
	}
	return value
}

func sanitizeStyleValue(property, value string, theme map[string]string) string {
	if isColorProperty(property) {
		if token := resolveThemeToken(value, theme); token != "" {
			return token
		}
	}
	return sanitizeCSSValue(value)
}

func sanitizeCSSToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_') {
			return ""
		}
	}
	return value
}

func sanitizeHTMLToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	for _, r := range value {
		if !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == ':') {
			return ""
		}
	}
	return value
}

func sanitizeThemeKey(value string) string {
	return sanitizeCSSToken(value)
}

func resolveThemeToken(value string, theme map[string]string) string {
	key := sanitizeThemeKey(value)
	if key == "" {
		return ""
	}
	if _, ok := theme[key]; ok {
		return "var(--coreui-" + key + ")"
	}
	return ""
}

func resolveThemeDefinitionValue(key, value string, theme map[string]string) string {
	if semantic := semanticTokenCSSValue(key, value); semantic != "" {
		return semantic
	}
	if token := resolveThemeToken(value, theme); token != "" {
		return token
	}
	return sanitizeCSSValue(value)
}

func isColorProperty(property string) bool {
	switch property {
	case "background", "background-color", "color", "border-color", "fill", "stroke":
		return true
	default:
		return false
	}
}

func semanticTokenCSSValue(key, value string) string {
	switch key {
	case "radius":
		switch value {
		case "none":
			return "0"
		case "sm":
			return "4px"
		case "md":
			return "8px"
		case "lg":
			return "12px"
		case "full":
			return "9999px"
		}
	case "shadow":
		switch value {
		case "none":
			return "none"
		case "soft":
			return "0 10px 30px rgba(15, 23, 42, 0.12)"
		case "deep":
			return "0 18px 45px rgba(15, 23, 42, 0.22)"
		}
	case "speed":
		switch value {
		case "instant":
			return "all 0s linear"
		case "smooth":
			return "all 180ms ease"
		case "lazy":
			return "all 320ms ease"
		}
	}
	return ""
}

func semanticStyleDecls(theme map[string]string, interactive bool, elevated bool) []styleDecl {
	decls := make([]styleDecl, 0, 3)
	if _, ok := theme["radius"]; ok {
		decls = append(decls, styleDeclFor("border-radius", "var(--cui-radius)"))
	}
	if elevated {
		if _, ok := theme["shadow"]; ok {
			decls = append(decls, styleDeclFor("box-shadow", "var(--cui-shadow)"))
		}
	}
	if interactive {
		if _, ok := theme["speed"]; ok {
			decls = append(decls, styleDeclFor("transition", "var(--cui-speed)"))
		}
	}
	return decls
}

func variantStyleDecls(variant string, theme map[string]string) []styleDecl {
	primary := resolveThemeToken("primary", theme)
	text := resolveThemeToken("text", theme)
	if text == "" {
		text = "inherit"
	}

	switch sanitizeCSSToken(variant) {
	case "primary":
		if primary == "" {
			return nil
		}
		return []styleDecl{
			styleDeclFor("background", primary),
			styleDeclFor("border-width", "1px"),
			styleDeclFor("border-style", "solid"),
			styleDeclFor("border-color", primary),
			styleDeclFor("color", text),
		}
	case "secondary", "outline":
		if primary == "" {
			return nil
		}
		return []styleDecl{
			styleDeclFor("background", "transparent"),
			styleDeclFor("border-width", "1px"),
			styleDeclFor("border-style", "solid"),
			styleDeclFor("border-color", primary),
			styleDeclFor("color", primary),
		}
	case "ghost":
		if primary == "" {
			return nil
		}
		return []styleDecl{
			styleDeclFor("background", "transparent"),
			styleDeclFor("border-width", "1px"),
			styleDeclFor("border-style", "solid"),
			styleDeclFor("border-color", "transparent"),
			styleDeclFor("color", primary),
		}
	default:
		return nil
	}
}

func isNumericToken(value string) bool {
	if value == "" {
		return false
	}
	dotSeen := false
	for _, r := range value {
		if r == '.' {
			if dotSeen {
				return false
			}
			dotSeen = true
			continue
		}
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
