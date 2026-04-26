// Package coreui exposes the public SDK for rendering compiled CoreUI blueprints.
package coreui

import (
	"encoding/json"
	"errors"

	"github.com/valueforvalue/coreui/pkg/compiler"
	"github.com/valueforvalue/coreui/pkg/renderers"
)

// CompileOptions configures blueprint compilation.
type CompileOptions = compiler.Options

// Renderer wraps a compiled CoreUI blueprint and renders it into distributable HTML.
type Renderer struct {
	blueprintJSON []byte
}

type blueprintEnvelope struct {
	Tree json.RawMessage `json:"tree"`
}

// NewRenderer validates compiled CoreUI blueprint JSON and prepares it for rendering.
//
// The blueprintJSON input should be the JSON emitted by the CoreUI compiler.
func NewRenderer(blueprintJSON []byte) (*Renderer, error) {
	if !json.Valid(blueprintJSON) {
		return nil, errors.New("coreui blueprint is not valid JSON")
	}

	var envelope blueprintEnvelope
	if err := json.Unmarshal(blueprintJSON, &envelope); err != nil {
		return nil, err
	}
	if len(envelope.Tree) == 0 {
		return nil, errors.New("coreui blueprint is missing tree")
	}

	cloned := append([]byte(nil), blueprintJSON...)
	return &Renderer{blueprintJSON: cloned}, nil
}

// Compile compiles a .cui file into CoreUI blueprint JSON.
func Compile(path string, options CompileOptions) ([]byte, error) {
	return compiler.CompileFile(path, options)
}

// Render renders compiled CoreUI blueprint JSON into a standalone HTML document.
//
// When data is non-nil, it is serialized to JSON and exposed on window.CoreUIData
// before the embedded browser renderer starts.
func Render(blueprintJSON []byte, data interface{}) (string, error) {
	renderer, err := NewRenderer(blueprintJSON)
	if err != nil {
		return "", err
	}
	return renderer.StandaloneHTML(data)
}

// StandaloneHTML renders the renderer state into a self-contained HTML document.
//
// When data is non-nil, it is serialized to JSON and exposed on window.CoreUIData
// before the embedded browser renderer starts.
func (r *Renderer) StandaloneHTML(data interface{}) (string, error) {
	if r == nil {
		return "", errors.New("coreui renderer is nil")
	}

	return renderers.BuildStandaloneHTML(r.blueprintJSON, data)
}
