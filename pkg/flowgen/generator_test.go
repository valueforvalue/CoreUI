package flowgen_test

import (
	"strings"
	"testing"

	"github.com/valueforvalue/coreui/pkg/flow"
	"github.com/valueforvalue/coreui/pkg/flowgen"
)

const counterFlow = `State {
    var count = 0
}

On(id="inc_btn", event="click") {
    add count 1
}

On(id="dec_btn", event="click") {
    add count -1
}

On(id="reset_btn", event="click") {
    set count = 0
}
`

func TestGenerateBasicCounter(t *testing.T) {
	doc, err := flow.ParseFlow(counterFlow)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	uiIDs := map[string]bool{
		"inc_btn":   true,
		"dec_btn":   true,
		"reset_btn": true,
	}

	js, err := flowgen.Generate(doc, uiIDs, nil, flowgen.Options{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	cases := []string{
		"_state = {",
		"count: 0,",
		"_set(",
		"_get(",
		"_sub(",
		`getById("inc_btn")`,
		`addEventListener("click"`,
		"window.CoreFlowState",
		"CoreFlow State Engine",
	}
	for _, want := range cases {
		if !strings.Contains(js, want) {
			t.Errorf("generated JS missing %q", want)
		}
	}
}

func TestGenerateWiringGap(t *testing.T) {
	doc, err := flow.ParseFlow(`On(id="ghost_btn", event="click") { toggle x }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	uiIDs := map[string]bool{"other_btn": true}
	_, err = flowgen.Generate(doc, uiIDs, nil, flowgen.Options{})
	if err == nil {
		t.Fatal("expected wiring gap error, got nil")
	}

	gap, ok := err.(*flowgen.WiringGap)
	if !ok {
		t.Fatalf("expected *flowgen.WiringGap, got %T: %v", err, err)
	}
	if gap.FlowRef != "ghost_btn" {
		t.Errorf("FlowRef: want ghost_btn, got %s", gap.FlowRef)
	}
	if !strings.Contains(gap.Error(), "WIRING_GAP") {
		t.Errorf("error message missing WIRING_GAP: %s", gap.Error())
	}
}

func TestGenerateWithBindings(t *testing.T) {
	doc, err := flow.ParseFlow(`State { var count = 0 }`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	bindings := []flowgen.Binding{
		{ElementID: "counter_text", AttrName: "value", VarName: "count"},
	}

	js, err := flowgen.Generate(doc, map[string]bool{}, bindings, flowgen.Options{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !strings.Contains(js, `getById("counter_text")`) {
		t.Error("generated JS missing getById(\"counter_text\")")
	}
	if !strings.Contains(js, "textContent = String(v)") {
		t.Error("generated JS missing textContent assignment")
	}
}

func TestGenerateComputeBlock(t *testing.T) {
	src := `State {
    var firstName = "Alice"
    var lastName = "Smith"
}

Compute(target="fullName") {
    firstName + " " + lastName
}
`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	js, err := flowgen.Generate(doc, map[string]bool{}, nil, flowgen.Options{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !strings.Contains(js, "_compute_fullName") {
		t.Error("missing compute function")
	}
	if !strings.Contains(js, `_sub("firstName"`) {
		t.Error("missing firstName subscription")
	}
	if !strings.Contains(js, `_sub("lastName"`) {
		t.Error("missing lastName subscription")
	}
}

func TestGenerateIfElse(t *testing.T) {
	src := `State { var count = 0 }
On(id="check_btn", event="click") {
    if count > 0 {
        set message = "positive"
    } else {
        set message = "zero"
    }
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	uiIDs := map[string]bool{"check_btn": true}
	js, err := flowgen.Generate(doc, uiIDs, nil, flowgen.Options{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !strings.Contains(js, "if (_state.count > 0)") {
		t.Errorf("missing if condition in generated JS; got:\n%s", js)
	}
	if !strings.Contains(js, "} else {") {
		t.Error("missing else branch in generated JS")
	}
}

func TestCollectBindings(t *testing.T) {
	tree := map[string]any{
		"id":   "root",
		"type": "View",
		"children": []any{
			map[string]any{
				"id":   "counter_text",
				"type": "Text",
				"attributes": map[string]any{
					"value": "flow:count",
				},
			},
			map[string]any{
				"id":   "plain_text",
				"type": "Text",
				"attributes": map[string]any{
					"value": "Hello",
				},
			},
		},
	}

	bindings := flowgen.CollectBindings(tree)
	if len(bindings) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(bindings))
	}
	b := bindings[0]
	if b.ElementID != "counter_text" {
		t.Errorf("ElementID: want counter_text, got %s", b.ElementID)
	}
	if b.VarName != "count" {
		t.Errorf("VarName: want count, got %s", b.VarName)
	}
}

func TestGenerateCallService(t *testing.T) {
	src := `On(id="save_btn", event="click") {
    call_service "api/save" (key="value")
}`
	doc, err := flow.ParseFlow(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	uiIDs := map[string]bool{"save_btn": true}
	js, err := flowgen.Generate(doc, uiIDs, nil, flowgen.Options{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if !strings.Contains(js, "coreflow:service") {
		t.Error("missing coreflow:service dispatch")
	}
	if !strings.Contains(js, `"api/save"`) {
		t.Error("missing service endpoint in generated JS")
	}
}
