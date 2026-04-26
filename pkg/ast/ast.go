package ast

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
