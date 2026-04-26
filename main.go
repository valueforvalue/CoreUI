package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strings"

	"coreui/pkg/ast"
	"coreui/pkg/generator"
	"coreui/pkg/registry"
	"coreui/renderers/goth"
)

func main() {
	if os.Getenv("COREUI_MODE") == "parity" {
		if err := runParityServer(":8080"); err != nil {
			log.Fatal(err)
		}
		return
	}

	node, theme, err := loadCompiledNode("kitchen_sink.json")
	if err != nil {
		log.Fatalf("load compiled node: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, "<!doctype html><html><head><meta charset=\"utf-8\"><title>CoreUI Kitchen Sink</title></head><body><main id=\"main-content\">")
		if err := goth.RenderWithTheme(node, theme).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, "</main></body></html>")
	})

	mux.Handle("/coreui/action", goth.HandleAction(goth.ActionExecutorFunc(func(ctx context.Context, action goth.ActionRequest, w http.ResponseWriter, r *http.Request) error {
		log.Printf("ActionRequest: namespace=%s call=%s params=%v", action.Namespace, action.Call, action.Params)
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(action)
	})))

	addr := ":8080"
	log.Printf("Kitchen sink harness listening on http://127.0.0.1%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func loadCompiledNode(path string) (ast.Node, map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ast.Node{}, nil, err
	}

	var output generator.Output
	if err := json.Unmarshal(data, &output); err != nil {
		return ast.Node{}, nil, err
	}
	if output.Tree == nil {
		return ast.Node{}, nil, fmt.Errorf("compiled tree missing")
	}

	node, err := inflateNode(output.Tree)
	if err != nil {
		return ast.Node{}, nil, err
	}
	return *node, output.Theme, nil
}

func inflateNode(source *generator.Node) (*ast.Node, error) {
	spec, ok := registry.GetComponent(source.Type)
	if !ok {
		return nil, fmt.Errorf("unknown component %q in compiled JSON", source.Type)
	}

	attributes := map[string]ast.Value{
		"id": {Kind: ast.StringKind, Data: source.ID},
	}
	for key, raw := range source.Attributes {
		attribute, ok := spec.Attributes[key]
		if !ok {
			return nil, fmt.Errorf("unknown attribute %q for %s in compiled JSON", key, source.Type)
		}

		value, err := inflateValue(attribute.Type, raw)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", source.Type, key, err)
		}
		attributes[key] = value
	}

	children := make([]*ast.Node, 0, len(source.Children))
	for _, child := range source.Children {
		inflated, err := inflateNode(child)
		if err != nil {
			return nil, err
		}
		children = append(children, inflated)
	}

	return &ast.Node{
		Type:       source.Type,
		Attributes: attributes,
		Children:   children,
	}, nil
}

func inflateValue(valueType registry.ValueType, raw any) (ast.Value, error) {
	switch valueType {
	case registry.StringType:
		text, ok := raw.(string)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected string")
		}
		return ast.Value{Kind: ast.StringKind, Data: text}, nil
	case registry.BoolType:
		flag, ok := raw.(bool)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected bool")
		}
		return ast.Value{Kind: ast.BoolKind, Data: flag}, nil
	case registry.IntType:
		number, ok := raw.(float64)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected int")
		}
		return ast.Value{Kind: ast.IntKind, Data: int64(math.Round(number))}, nil
	case registry.NumberType:
		number, ok := raw.(float64)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected number")
		}
		return ast.Value{Kind: ast.NumberKind, Data: number}, nil
	case registry.UnitType:
		text, ok := raw.(string)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected unit")
		}
		return ast.Value{Kind: ast.UnitKind, Data: text}, nil
	case registry.UnitArrayType:
		items, ok := raw.([]any)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected unit array")
		}

		values := make([]ast.Value, 0, len(items))
		for _, item := range items {
			text, ok := item.(string)
			if !ok {
				return ast.Value{}, fmt.Errorf("expected unit array item")
			}
			values = append(values, ast.Value{Kind: ast.UnitKind, Data: text})
		}
		return ast.Value{Kind: ast.ArrayKind, Data: values}, nil
	case registry.ActionType:
		payload, ok := raw.(map[string]any)
		if !ok {
			return ast.Value{}, fmt.Errorf("expected action object")
		}
		action, err := inflateAction(payload)
		if err != nil {
			return ast.Value{}, err
		}
		return ast.Value{Kind: ast.ActionKind, Data: action}, nil
	default:
		return ast.Value{}, fmt.Errorf("unsupported value type %q", valueType)
	}
}

func inflateAction(payload map[string]any) (ast.Action, error) {
	namespace, _ := payload["namespace"].(string)
	call, _ := payload["call"].(string)
	params := map[string]ast.Value{}

	rawParams, _ := payload["params"].(map[string]any)
	for key, raw := range rawParams {
		params[key] = inferDynamicValue(raw)
	}

	if namespace == "" || call == "" {
		return ast.Action{}, fmt.Errorf("action missing namespace or call")
	}
	return ast.Action{
		Namespace: namespace,
		Call:      call,
		Params:    params,
	}, nil
}

func inferDynamicValue(raw any) ast.Value {
	switch value := raw.(type) {
	case string:
		if looksLikeUnit(value) {
			return ast.Value{Kind: ast.UnitKind, Data: value}
		}
		return ast.Value{Kind: ast.StringKind, Data: value}
	case bool:
		return ast.Value{Kind: ast.BoolKind, Data: value}
	case float64:
		if math.Mod(value, 1) == 0 {
			return ast.Value{Kind: ast.IntKind, Data: int64(value)}
		}
		return ast.Value{Kind: ast.NumberKind, Data: value}
	case []any:
		items := make([]ast.Value, 0, len(value))
		for _, item := range value {
			items = append(items, inferDynamicValue(item))
		}
		return ast.Value{Kind: ast.ArrayKind, Data: items}
	case map[string]any:
		action, err := inflateAction(value)
		if err == nil {
			return ast.Value{Kind: ast.ActionKind, Data: action}
		}
	}
	return ast.Value{Kind: ast.StringKind, Data: fmt.Sprint(raw)}
}

func looksLikeUnit(value string) bool {
	return value == "auto" || strings.HasSuffix(value, "px") || strings.HasSuffix(value, "%") || strings.HasSuffix(value, "*")
}
