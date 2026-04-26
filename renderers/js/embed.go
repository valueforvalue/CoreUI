package jsrenderer

import _ "embed"

// Source contains the standalone Vanilla JS renderer source.
//
//go:embed renderer.js
var Source string
