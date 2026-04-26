package goth

import (
	"fmt"
	"sort"
	"strings"
	"unicode"

	"coreui/pkg/ast"
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
		if sanitizeThemeKey(key) != "" && sanitizeCSSValue(theme[key]) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return templ.Raw("")
	}

	var builder strings.Builder
	builder.WriteString("<style>:root{")
	for _, key := range keys {
		builder.WriteString(fmt.Sprintf("--coreui-%s:%s;", sanitizeThemeKey(key), sanitizeCSSValue(theme[key])))
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

func isColorProperty(property string) bool {
	switch property {
	case "background", "background-color", "color", "border-color", "fill", "stroke":
		return true
	default:
		return false
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
