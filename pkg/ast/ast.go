package ast

import (
	"sort"
	"strconv"
	"strings"
)

type Position struct {
	Line int
	Col  int
}

type ValueKind string

const (
	StringKind ValueKind = "string"
	NumberKind ValueKind = "number"
	IntKind    ValueKind = "int"
	BoolKind   ValueKind = "bool"
	UnitKind   ValueKind = "unit"
	ArrayKind  ValueKind = "array"
	ActionKind ValueKind = "action"
)

type Value struct {
	Kind ValueKind
	Data any
}

type Action struct {
	Namespace string
	Call      string
	Params    map[string]Value
}

type Document struct {
	Tree  *Node
	Theme map[string]string
}

type Node struct {
	Type       string
	Attributes map[string]Value
	Children   []*Node
	Position   Position
}

// ToDSLString returns the DSL attribute-value representation of the value.
// The returned string is byte-for-byte compatible with what the .cui parser
// accepts as an attribute value token.  Callers that need to reconstruct a
// full attribute assignment must wrap string values in double quotes, e.g.:
//
//	key="value"   ← for StringKind
//	key=20px      ← for UnitKind
//	key=true      ← for BoolKind
func (v Value) ToDSLString() string {
	switch v.Kind {
	case StringKind:
		s, _ := v.Data.(string)
		return s
	case BoolKind:
		b, _ := v.Data.(bool)
		if b {
			return "true"
		}
		return "false"
	case IntKind:
		n, _ := v.Data.(int64)
		return strconv.FormatInt(n, 10)
	case NumberKind:
		f, _ := v.Data.(float64)
		return strconv.FormatFloat(f, 'f', -1, 64)
	case UnitKind:
		s, _ := v.Data.(string)
		return s
	case ActionKind:
		a, _ := v.Data.(Action)
		return a.ToDSLString()
	case ArrayKind:
		items, _ := v.Data.([]Value)
		parts := make([]string, 0, len(items))
		for _, item := range items {
			// Array items must be self-contained DSL tokens; string items are quoted.
			if item.Kind == StringKind {
				s, _ := item.Data.(string)
				parts = append(parts, `"`+escapeActionString(s)+`"`)
			} else {
				parts = append(parts, item.ToDSLString())
			}
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return ""
	}
}

// ToDSLString returns the action in its DSL form, e.g. "app:save" or
// "app:notify(msg=\"Done\", type=\"success\")".  Parameter keys are emitted in
// sorted order so that output is deterministic.  String parameter values are
// wrapped in double quotes with backslash escaping.
func (a Action) ToDSLString() string {
	base := a.Namespace + ":" + a.Call
	if len(a.Params) == 0 {
		return base
	}

	keys := make([]string, 0, len(a.Params))
	for k := range a.Params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		v := a.Params[k]
		var valStr string
		if v.Kind == StringKind {
			s, _ := v.Data.(string)
			valStr = `"` + escapeActionString(s) + `"`
		} else {
			valStr = v.ToDSLString()
		}
		parts = append(parts, k+"="+valStr)
	}
	return base + "(" + strings.Join(parts, ", ") + ")"
}

// escapeActionString escapes backslashes and double quotes within an action
// string parameter value so that the result can be safely wrapped in `"..."`.
func escapeActionString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}

func (n *Node) ID() string {
	if n == nil {
		return ""
	}

	value, ok := n.Attributes["id"]
	if !ok || value.Kind != StringKind {
		return ""
	}

	id, _ := value.Data.(string)
	return id
}
