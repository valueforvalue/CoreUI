package goth

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"coreui/pkg/ast"
	"github.com/a-h/templ"
)

// ActionRequest is the HTMX action payload emitted by Trigger components.
type ActionRequest struct {
	Namespace string         `json:"namespace"`
	Call      string         `json:"call"`
	Params    map[string]any `json:"params,omitempty"`
}

// ActionExecutor handles decoded CoreUI actions.
type ActionExecutor interface {
	ExecuteAction(ctx context.Context, action ActionRequest, w http.ResponseWriter, r *http.Request) error
}

// ActionExecutorFunc adapts a function to the ActionExecutor interface.
type ActionExecutorFunc func(ctx context.Context, action ActionRequest, w http.ResponseWriter, r *http.Request) error

// ExecuteAction calls f(ctx, action, w, r).
func (f ActionExecutorFunc) ExecuteAction(ctx context.Context, action ActionRequest, w http.ResponseWriter, r *http.Request) error {
	return f(ctx, action, w, r)
}

// HandleAction returns an http.HandlerFunc that decodes CoreUI HTMX action
// submissions and forwards them to executor.
//
// Typical usage with net/http:
//
//	mux := http.NewServeMux()
//	mux.Handle("/coreui/action", goth.HandleAction(goth.ActionExecutorFunc(func(ctx context.Context, action goth.ActionRequest, w http.ResponseWriter, r *http.Request) error {
//		_, err := w.Write([]byte("ok"))
//		return err
//	})))
func HandleAction(executor ActionExecutor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if executor == nil {
			http.Error(w, "coreui action executor is nil", http.StatusInternalServerError)
			return
		}

		action, err := ParseActionRequest(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		tracker := &statusTrackingResponseWriter{ResponseWriter: w}
		if err := executor.ExecuteAction(r.Context(), action, tracker, r); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if !tracker.wroteHeader && !tracker.wroteBody {
			w.WriteHeader(http.StatusNoContent)
		}
	}
}

// ParseActionRequest decodes an ActionRequest from an HTMX or JSON request.
//
// It accepts three formats:
//  1. an "action" form field containing the full JSON payload
//  2. form fields named namespace, call, and params
//  3. a raw JSON request body matching ActionRequest
func ParseActionRequest(r *http.Request) (ActionRequest, error) {
	if r == nil {
		return ActionRequest{}, errors.New("request is nil")
	}

	if err := r.ParseForm(); err != nil {
		return ActionRequest{}, err
	}

	if encoded := strings.TrimSpace(r.Form.Get("action")); encoded != "" {
		return decodeActionJSON(encoded)
	}

	namespace := strings.TrimSpace(r.Form.Get("namespace"))
	call := strings.TrimSpace(r.Form.Get("call"))
	paramsPayload := strings.TrimSpace(r.Form.Get("params"))
	if namespace != "" || call != "" || paramsPayload != "" {
		params := map[string]any{}
		if paramsPayload != "" {
			if err := json.Unmarshal([]byte(paramsPayload), &params); err != nil {
				return ActionRequest{}, errors.New("invalid params payload")
			}
		}
		mergeStructuredParams(params, r)

		action := ActionRequest{
			Namespace: namespace,
			Call:      call,
			Params:    params,
		}
		if err := action.validate(); err != nil {
			return ActionRequest{}, err
		}
		return action, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return ActionRequest{}, err
	}
	if strings.TrimSpace(string(body)) == "" {
		return ActionRequest{}, errors.New("missing action payload")
	}
	return decodeActionJSON(string(body))
}

func actionRequestFromAction(action ast.Action) ActionRequest {
	params := make(map[string]any, len(action.Params))
	for key, value := range action.Params {
		params[key] = valueToAny(value)
	}
	return ActionRequest{
		Namespace: action.Namespace,
		Call:      action.Call,
		Params:    params,
	}
}

func encodeActionRequest(action ActionRequest) string {
	text, err := templ.JSONString(action)
	if err != nil {
		return `{}`
	}
	return text
}

func decodeActionJSON(payload string) (ActionRequest, error) {
	var action ActionRequest
	if err := json.Unmarshal([]byte(payload), &action); err != nil {
		return ActionRequest{}, errors.New("invalid action payload")
	}
	if action.Params == nil {
		action.Params = map[string]any{}
	}
	if err := action.validate(); err != nil {
		return ActionRequest{}, err
	}
	return action, nil
}

func mergeStructuredParams(params map[string]any, r *http.Request) {
	for key, values := range r.Form {
		if len(values) == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(key, "params."):
			params[strings.TrimPrefix(key, "params.")] = values[0]
		case strings.HasPrefix(key, "params[") && strings.HasSuffix(key, "]"):
			params[strings.TrimSuffix(strings.TrimPrefix(key, "params["), "]")] = values[0]
		}
	}
}

func (a ActionRequest) validate() error {
	if strings.TrimSpace(a.Namespace) == "" {
		return errors.New("missing action namespace")
	}
	if strings.TrimSpace(a.Call) == "" {
		return errors.New("missing action call")
	}
	if a.Params == nil {
		a.Params = map[string]any{}
	}
	return nil
}

func valueToAny(value ast.Value) any {
	switch value.Kind {
	case ast.StringKind, ast.UnitKind, ast.BoolKind, ast.IntKind, ast.NumberKind:
		return value.Data
	case ast.ArrayKind:
		items, _ := value.Data.([]ast.Value)
		out := make([]any, 0, len(items))
		for _, item := range items {
			out = append(out, valueToAny(item))
		}
		return out
	case ast.ActionKind:
		action, _ := value.Data.(ast.Action)
		return actionRequestFromAction(action)
	default:
		return value.Data
	}
}

type statusTrackingResponseWriter struct {
	http.ResponseWriter
	wroteHeader bool
	wroteBody   bool
}

func (w *statusTrackingResponseWriter) WriteHeader(statusCode int) {
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusTrackingResponseWriter) Write(data []byte) (int, error) {
	w.wroteBody = true
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(data)
}
