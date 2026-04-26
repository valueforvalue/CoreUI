package goth

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/valueforvalue/coreui/pkg/ast"
	"github.com/valueforvalue/coreui/pkg/registry"
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

func TestRenderWithThemeIncludesCSSVariablesAndTokenizedStyles(t *testing.T) {
	component := RenderWithTheme(ast.Node{
		Type: "View",
		Attributes: map[string]ast.Value{
			"id":    {Kind: ast.StringKind, Data: "root"},
			"title": {Kind: ast.StringKind, Data: "Themed"},
		},
		Children: []*ast.Node{
			{
				Type: "Box",
				Attributes: map[string]ast.Value{
					"id":         {Kind: ast.StringKind, Data: "panel"},
					"background": {Kind: ast.StringKind, Data: "surface"},
					"style":      {Kind: ast.StringKind, Data: "color: primary"},
				},
			},
		},
	}, map[string]string{
		"primary": "#0984e3",
		"surface": "#1e1e1e",
	})

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`--coreui-primary:#0984e3;`,
		`--coreui-surface:#1e1e1e;`,
		`background: var(--coreui-surface)`,
		`color: var(--coreui-primary)`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected themed html to contain %q, got %s", want, html)
		}
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

func TestRenderImageIncludesImgTagAndWidth(t *testing.T) {
	component := Render(ast.Node{
		Type: "Image",
		Attributes: map[string]ast.Value{
			"id":    {Kind: ast.StringKind, Data: "hero_image"},
			"src":   {Kind: ast.StringKind, Data: "testdata/coreui-logo.svg"},
			"alt":   {Kind: ast.StringKind, Data: "CoreUI logo"},
			"width": {Kind: ast.UnitKind, Data: "200px"},
		},
	})

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`<img`,
		`id="hero_image"`,
		`src="testdata/coreui-logo.svg"`,
		`alt="CoreUI logo"`,
		`width: 200px`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered html to contain %q, got %s", want, html)
		}
	}
}

func TestRenderGraphIncludesSVGAndLegend(t *testing.T) {
	component := RenderWithTheme(ast.Node{
		Type: "Graph",
		Attributes: map[string]ast.Value{
			"id":     {Kind: ast.StringKind, Data: "trend_graph"},
			"type":   {Kind: ast.StringKind, Data: "line"},
			"color":  {Kind: ast.StringKind, Data: "primary"},
			"height": {Kind: ast.UnitKind, Data: "220px"},
			"labels": {Kind: ast.ArrayKind, Data: []ast.Value{
				{Kind: ast.StringKind, Data: "08:00"},
				{Kind: ast.StringKind, Data: "10:00"},
			}},
			"data": {Kind: ast.ArrayKind, Data: []ast.Value{
				{Kind: ast.IntKind, Data: int64(18)},
				{Kind: ast.NumberKind, Data: 24.5},
			}},
		},
	}, map[string]string{
		"primary": "#0984e3",
		"radius":  "md",
	})

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	html := buf.String()
	for _, want := range []string{
		`data-coreui-type="Graph"`,
		`<svg`,
		`stroke="var(--coreui-primary)"`,
		`08:00: 18`,
		`10:00: 24.5`,
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

func TestRenderPluginComponentUsesDataCoreuiPlugin(t *testing.T) {
	// Register a synthetic plugin component so the renderer can detect it.
	// This test uses an internal (same-package) call to the registry helper.
	// We load a temporary plugin using LoadPluginsFromDir on a pre-seeded temp dir.
	dir := t.TempDir()
	pluginJSON := `{
		"components": [
			{
				"name": "TestPlugin",
				"has_children": true,
				"attributes": {
					"id": { "type": "string", "required": true }
				}
			}
		]
	}`
	if err := os.WriteFile(filepath.Join(dir, "testplugin.json"), []byte(pluginJSON), 0o644); err != nil {
		t.Fatalf("write plugin: %v", err)
	}
	registry.LoadPluginsFromDir(dir)
	t.Cleanup(func() { registry.UnregisterPluginComponent("TestPlugin") })

	component := Render(ast.Node{
		Type: "TestPlugin",
		Attributes: map[string]ast.Value{
			"id": {Kind: ast.StringKind, Data: "plug1"},
		},
	})

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	html := buf.String()
	if !strings.Contains(html, `data-coreui-plugin="TestPlugin"`) {
		t.Fatalf("expected data-coreui-plugin attribute, got %s", html)
	}
	if strings.Contains(html, `data-coreui-type="TestPlugin"`) {
		t.Fatalf("plugin component must not use data-coreui-type, got %s", html)
	}
}
