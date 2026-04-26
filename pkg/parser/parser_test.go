package parser_test

import (
	"strings"
	"testing"

	"coreui/pkg/parser"
)

func TestParserRejectsDuplicateIDs(t *testing.T) {
	source := `
View(id="root") {
  Text(id="dup", value="First")
  Text(id="dup", value="Second")
}
`

	_, err := parser.New(source).Parse()
	if err == nil {
		t.Fatal("expected duplicate ID error")
	}
	if !strings.Contains(err.Error(), "Duplicate/Missing ID") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParserRejectsUnknownAttribute(t *testing.T) {
	source := `Text(id="t1", value="Hello", bogus="x")`

	_, err := parser.New(source).Parse()
	if err == nil {
		t.Fatal("expected unknown attribute error")
	}
	if !strings.Contains(err.Error(), `unknown attribute "bogus"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParserRejectsTypeMismatch(t *testing.T) {
	source := `Stack(id="stack1", dir="v", gap="16px")`

	_, err := parser.New(source).Parse()
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
	if !strings.Contains(err.Error(), `expects unit`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParserRejectsMalformedAction(t *testing.T) {
	source := `Trigger(id="trigger1", label="Open", action=ui:navigate(target), variant="primary")`

	_, err := parser.New(source).Parse()
	if err == nil {
		t.Fatal("expected malformed action error")
	}
	if !strings.Contains(err.Error(), "invalid action parameter") {
		t.Fatalf("unexpected error: %v", err)
	}
}
