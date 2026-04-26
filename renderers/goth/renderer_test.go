package goth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"coreui/pkg/ast"
)

func TestUnitToCSS(t *testing.T) {
	if got := UnitToCSS("10px", UnitContextLiteral); got != "10px" {
		t.Fatalf("expected px literal, got %q", got)
	}
	if got := UnitToCSS("50%", UnitContextLiteral); got != "50%" {
		t.Fatalf("expected percent literal, got %q", got)
	}
	if got := UnitToCSS("1*", UnitContextGridTrack); got != "1fr" {
		t.Fatalf("expected grid fractional unit, got %q", got)
	}
	if got := UnitToCSS("2*", UnitContextFlex); got != "2" {
		t.Fatalf("expected flex fractional unit, got %q", got)
	}
}

func TestRenderTriggerIncludesHTMXActionPayload(t *testing.T) {
	component := Render(ast.Node{
		Type: "Trigger",
		Attributes: map[string]ast.Value{
			"id":    {Kind: ast.StringKind, Data: "btn_logs"},
			"label": {Kind: ast.StringKind, Data: "View Logs"},
			"action": {Kind: ast.ActionKind, Data: ast.Action{
				Namespace: "sys",
				Call:      "fetch_logs",
				Params: map[string]ast.Value{
					"limit": {Kind: ast.IntKind, Data: int64(50)},
					"level": {Kind: ast.StringKind, Data: "debug"},
				},
			}},
		},
	})

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`id="btn_logs"`,
		`hx-post="/coreui/action"`,
		`hx-target="#main-content"`,
		`namespace`,
		`fetch_logs`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered html to contain %q, got %s", want, html)
		}
	}
}

func TestParseActionRequestFromForm(t *testing.T) {
	values := url.Values{}
	values.Set("namespace", "ui")
	values.Set("call", "navigate")
	values.Set("params", `{"target":"settings"}`)

	request := httptest.NewRequest(http.MethodPost, "/coreui/action", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	action, err := ParseActionRequest(request)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if action.Namespace != "ui" || action.Call != "navigate" {
		t.Fatalf("unexpected action: %+v", action)
	}
	if action.Params["target"] != "settings" {
		t.Fatalf("expected target param, got %+v", action.Params)
	}
}

func TestHandleActionInvokesExecutor(t *testing.T) {
	values := url.Values{}
	values.Set("action", `{"namespace":"ui","call":"close","params":{}}`)

	request := httptest.NewRequest(http.MethodPost, "/coreui/action", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	handler := HandleAction(ActionExecutorFunc(func(ctx context.Context, action ActionRequest, w http.ResponseWriter, r *http.Request) error {
		if action.Namespace != "ui" || action.Call != "close" {
			t.Fatalf("unexpected action: %+v", action)
		}
		_, err := w.Write([]byte("ok"))
		return err
	}))

	handler(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200 status, got %d", recorder.Code)
	}
	if body := recorder.Body.String(); body != "ok" {
		t.Fatalf("unexpected body: %q", body)
	}
}
