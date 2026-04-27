// Package flow provides the parser and AST for the CoreFlow (.flow) DSL.
// CoreFlow is a minimal, block-based language for defining reactive state
// and event-driven logic that wires to a CoreUI (.cui) layout.
package flow

// FlowDocument is the top-level AST node for a parsed .flow file.
type FlowDocument struct {
	State    *StateBlock
	OnBlocks []*OnBlock
	Computes []*ComputeBlock
}

// StateBlock declares reactive state variables.
type StateBlock struct {
	Line int
	Col  int
	Vars []*VarDecl
}

// VarKind indicates the declared kind of a state variable.
type VarKind string

const (
	VarKindVar  VarKind = "var"
	VarKindList VarKind = "list"
	VarKindMap  VarKind = "map"
)

// VarDecl is a single variable declaration inside a State block.
type VarDecl struct {
	Line int
	Col  int
	Kind VarKind
	Name string
	Init Expr
}

// OnBlock attaches a set of statements to a UI element's event.
type OnBlock struct {
	Line       int
	Col        int
	TargetID   string
	Event      string
	Statements []Statement
}

// ComputeBlock defines a derived state variable that updates automatically
// when any referenced state variable changes.
type ComputeBlock struct {
	Line   int
	Col    int
	Target string
	Expr   Expr
}

// StatementKind identifies the category of a logic statement.
type StatementKind string

const (
	StmtSet     StatementKind = "set"
	StmtAdd     StatementKind = "add"
	StmtToggle  StatementKind = "toggle"
	StmtIf      StatementKind = "if"
	StmtService StatementKind = "call_service"
)

// Statement is a single logic operation inside an On or if/else block.
type Statement struct {
	Line int
	Col  int
	Kind StatementKind

	// StmtSet: variable to assign and the value expression.
	VarName string
	Value   Expr

	// StmtAdd: variable to increment and the amount expression.
	Amount Expr

	// StmtIf: condition expression and branches.
	Condition Condition
	Then      []Statement
	Else      []Statement

	// StmtService: service endpoint and parameter map.
	Service string
	Params  map[string]Expr
}

// Condition is a comparison used in if statements.
type Condition struct {
	Left  Expr
	Op    string // ==  !=  >  <  >=  <=
	Right Expr
}

// ExprTokenKind identifies the category of a single expression token.
type ExprTokenKind string

const (
	TokIdent ExprTokenKind = "ident"
	TokStr   ExprTokenKind = "str"
	TokInt   ExprTokenKind = "int"
	TokFloat ExprTokenKind = "float"
	TokBool  ExprTokenKind = "bool"
	TokOp    ExprTokenKind = "op"
)

// ExprToken is one element of a flat expression sequence.
type ExprToken struct {
	Kind ExprTokenKind
	Val  string
}

// Expr is a flat, ordered sequence of expression tokens.
// The JS generator converts this directly to a JavaScript expression string.
type Expr []ExprToken

// IsEmpty reports whether the expression has no tokens.
func (e Expr) IsEmpty() bool { return len(e) == 0 }

// IdentNames returns the list of identifier token values in the expression.
// This is used by the flowgen to discover state variable dependencies.
func (e Expr) IdentNames() []string {
	seen := map[string]bool{}
	var names []string
	for _, tok := range e {
		if tok.Kind == TokIdent && !seen[tok.Val] {
			seen[tok.Val] = true
			names = append(names, tok.Val)
		}
	}
	return names
}
