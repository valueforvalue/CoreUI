package flow_test

import (
	"testing"

	"github.com/valueforvalue/coreui/pkg/flow"
)

func TestParseStateBlock(t *testing.T) {
	src := `State {
    var count = 0
    var name = "Alice"
    var active = true
    list items
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc.State == nil {
		t.Fatal("expected State block, got nil")
	}
	if len(doc.State.Vars) != 4 {
		t.Fatalf("expected 4 var decls, got %d", len(doc.State.Vars))
	}

	check := func(i int, kind flow.VarKind, name string, tokKind flow.ExprTokenKind, val string) {
		t.Helper()
		v := doc.State.Vars[i]
		if v.Kind != kind {
			t.Errorf("var[%d] kind: want %s, got %s", i, kind, v.Kind)
		}
		if v.Name != name {
			t.Errorf("var[%d] name: want %s, got %s", i, name, v.Name)
		}
		if tokKind == "" {
			return // list/map have no init expression to check
		}
		if len(v.Init) == 0 {
			t.Errorf("var[%d] init expr is empty", i)
			return
		}
		if v.Init[0].Kind != tokKind {
			t.Errorf("var[%d] init kind: want %s, got %s", i, tokKind, v.Init[0].Kind)
		}
		if v.Init[0].Val != val {
			t.Errorf("var[%d] init val: want %q, got %q", i, val, v.Init[0].Val)
		}
	}

	check(0, flow.VarKindVar, "count", flow.TokInt, "0")
	check(1, flow.VarKindVar, "name", flow.TokStr, "Alice")
	check(2, flow.VarKindVar, "active", flow.TokBool, "true")
	check(3, flow.VarKindList, "items", "", "")
}

func TestParseOnBlock(t *testing.T) {
	src := `On(id="btn", event="click") {
    add count 1
    toggle isDark
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.OnBlocks) != 1 {
		t.Fatalf("expected 1 On block, got %d", len(doc.OnBlocks))
	}
	on := doc.OnBlocks[0]
	if on.TargetID != "btn" {
		t.Errorf("targetID: want %q, got %q", "btn", on.TargetID)
	}
	if on.Event != "click" {
		t.Errorf("event: want %q, got %q", "click", on.Event)
	}
	if len(on.Statements) != 2 {
		t.Fatalf("expected 2 statements, got %d", len(on.Statements))
	}

	add := on.Statements[0]
	if add.Kind != flow.StmtAdd {
		t.Errorf("stmt[0] kind: want add, got %s", add.Kind)
	}
	if add.VarName != "count" {
		t.Errorf("stmt[0] varName: want count, got %s", add.VarName)
	}
	if len(add.Amount) == 0 || add.Amount[0].Val != "1" {
		t.Errorf("stmt[0] amount: want 1, got %v", add.Amount)
	}

	tog := on.Statements[1]
	if tog.Kind != flow.StmtToggle {
		t.Errorf("stmt[1] kind: want toggle, got %s", tog.Kind)
	}
	if tog.VarName != "isDark" {
		t.Errorf("stmt[1] varName: want isDark, got %s", tog.VarName)
	}
}

func TestParseSetStatement(t *testing.T) {
	src := `On(id="x", event="click") {
    set message = "hello"
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stmt := doc.OnBlocks[0].Statements[0]
	if stmt.Kind != flow.StmtSet {
		t.Errorf("kind: want set, got %s", stmt.Kind)
	}
	if stmt.VarName != "message" {
		t.Errorf("varName: want message, got %s", stmt.VarName)
	}
	if len(stmt.Value) == 0 || stmt.Value[0].Val != "hello" {
		t.Errorf("value: want hello, got %v", stmt.Value)
	}
}

func TestParseIfElse(t *testing.T) {
	src := `On(id="check", event="click") {
    if count > 0 {
        set status = "positive"
    } else {
        set status = "zero"
    }
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stmt := doc.OnBlocks[0].Statements[0]
	if stmt.Kind != flow.StmtIf {
		t.Fatalf("kind: want if, got %s", stmt.Kind)
	}
	if stmt.Condition.Op != ">" {
		t.Errorf("cond op: want >, got %s", stmt.Condition.Op)
	}
	if len(stmt.Then) != 1 || stmt.Then[0].Kind != flow.StmtSet {
		t.Errorf("then branch: want 1 set stmt")
	}
	if len(stmt.Else) != 1 || stmt.Else[0].Kind != flow.StmtSet {
		t.Errorf("else branch: want 1 set stmt")
	}
}

func TestParseComputeBlock(t *testing.T) {
	src := `Compute(target="greeting") {
    name + "!"
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(doc.Computes) != 1 {
		t.Fatalf("expected 1 Compute block, got %d", len(doc.Computes))
	}
	c := doc.Computes[0]
	if c.Target != "greeting" {
		t.Errorf("target: want greeting, got %s", c.Target)
	}
	if len(c.Expr) != 3 {
		t.Fatalf("expected 3 tokens in expr, got %d: %v", len(c.Expr), c.Expr)
	}
	if c.Expr[0].Kind != flow.TokIdent || c.Expr[0].Val != "name" {
		t.Errorf("expr[0]: want ident:name, got %v", c.Expr[0])
	}
	if c.Expr[1].Kind != flow.TokOp || c.Expr[1].Val != "+" {
		t.Errorf("expr[1]: want op:+, got %v", c.Expr[1])
	}
	if c.Expr[2].Kind != flow.TokStr || c.Expr[2].Val != "!" {
		t.Errorf("expr[2]: want str:!, got %v", c.Expr[2])
	}
}

func TestParseCallService(t *testing.T) {
	src := `On(id="save_btn", event="click") {
    call_service "api/save" (name="Alice", value="data")
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stmt := doc.OnBlocks[0].Statements[0]
	if stmt.Kind != flow.StmtService {
		t.Fatalf("kind: want call_service, got %s", stmt.Kind)
	}
	if stmt.Service != "api/save" {
		t.Errorf("service: want api/save, got %s", stmt.Service)
	}
	if len(stmt.Params) != 2 {
		t.Errorf("params: want 2, got %d", len(stmt.Params))
	}
}

func TestParseErrors(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"unknown block", `Foo { }`},
		{"duplicate State", "State { var x = 0 }\nState { var y = 1 }"},
		{"On missing id", `On(event="click") { }`},
		{"bad var keyword", `State { set x = 0 }`},
		{"unterminated On", `On(id="btn", event="click") { add x 1`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := flow.ParseFlow(tc.src)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestExprIdentNames(t *testing.T) {
	expr := flow.Expr{
		{Kind: flow.TokIdent, Val: "firstName"},
		{Kind: flow.TokOp, Val: "+"},
		{Kind: flow.TokStr, Val: " "},
		{Kind: flow.TokOp, Val: "+"},
		{Kind: flow.TokIdent, Val: "lastName"},
	}
	names := expr.IdentNames()
	if len(names) != 2 {
		t.Fatalf("want 2 names, got %d: %v", len(names), names)
	}
}
