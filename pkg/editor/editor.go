// Package editor implements the corec edit command: a block-based,
// distraction-free WYSIWYG editor served from a single embedded HTTP server.
package editor

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"

	"github.com/valueforvalue/coreui/pkg/compiler"
	"github.com/valueforvalue/coreui/pkg/registry"
	"github.com/valueforvalue/coreui/pkg/renderers"
)

//go:embed assets/index.html
var editorHTML string

// AttributeInfo is the JSON-serialisable form of a registry attribute spec.
type AttributeInfo struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	Required bool     `json:"required,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

// ComponentInfo is the JSON-serialisable form of a registry component spec.
type ComponentInfo struct {
	Name        string          `json:"name"`
	HasChildren bool            `json:"hasChildren,omitempty"`
	Attributes  []AttributeInfo `json:"attributes"`
}

// Editor serves the corec edit WYSIWYG editor.
type Editor struct {
	filePath string
	mux      *http.ServeMux
}

// New creates an Editor bound to the given .cui file path.
func New(filePath string) *Editor {
	e := &Editor{filePath: filePath}
	mux := http.NewServeMux()
	mux.HandleFunc("/", e.handleIndex)
	mux.HandleFunc("/api/meta", e.handleMeta)
	mux.HandleFunc("/api/source", e.handleSource)
	mux.HandleFunc("/api/registry", e.handleRegistry)
	mux.HandleFunc("/api/compile", e.handleCompile)
	mux.HandleFunc("/api/preview", e.handlePreview)
	mux.HandleFunc("/api/upload", e.handleUpload)
	e.mux = mux
	return e
}

// Run starts the HTTP server, opens the browser, and blocks until ctx is
// cancelled or the server otherwise stops.
func (e *Editor) Run(ctx context.Context) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("editor: listen: %w", err)
	}

	url := "http://" + listener.Addr().String()
	server := &http.Server{Handler: e.mux}

	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()

	fmt.Printf("corec edit: serving at %s  (Ctrl+C to stop)\n", url)
	go openBrowser(url)

	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleIndex serves the editor SPA.
func (e *Editor) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, editorHTML)
}

// handleMeta returns server-side metadata (file path, registry version).
func (e *Editor) handleMeta(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"file":            e.filePath,
		"registryVersion": registry.Version,
	})
}

// handleSource serves GET (read) and PUT (write) for the .cui source file.
func (e *Editor) handleSource(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		data, err := os.ReadFile(e.filePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(data)

	case http.MethodPut:
		var req struct{ Source string }
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := os.WriteFile(e.filePath, []byte(req.Source), 0o644); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleRegistry returns all registered components with their attribute specs,
// so the frontend can generate registry-first UI controls.
func (e *Editor) handleRegistry(w http.ResponseWriter, r *http.Request) {
	components := registry.AllComponents()
	result := make([]ComponentInfo, 0, len(components))
	for _, spec := range components {
		result = append(result, ComponentInfo{
			Name:        spec.Name,
			HasChildren: spec.HasChildren,
			Attributes:  specToAttributeInfo(spec),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// handleCompile accepts a JSON body {"Source":"..."} and returns the compiled
// blueprint JSON, or a 422 with {"error":"..."} on parse/validation failure.
func (e *Editor) handleCompile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Source string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, err := compiler.CompileSource("", req.Source, compiler.Options{})
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	_, _ = w.Write(data)
}

// handlePreview accepts a JSON body {"Source":"..."}, compiles it, and returns
// a self-contained standalone HTML document ready to be used as iframe srcdoc.
func (e *Editor) handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct{ Source string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data, compileErr := compiler.CompileSource("", req.Source, compiler.Options{})
	if compileErr != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": compileErr.Error()})
		return
	}
	html, err := renderers.BuildStandaloneHTML(data, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, html)
}

// handleUpload accepts a raw image body (jpg, png, or webp), gzip-compresses
// it, base64-encodes the result, and returns the compressed_src value as JSON.
func (e *Editor) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	const maxSize = 20 << 20 // 20 MiB
	r.Body = http.MaxBytesReader(w, r.Body, maxSize)

	imgBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read image body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(imgBytes) == 0 {
		http.Error(w, "empty image body", http.StatusBadRequest)
		return
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(imgBytes); err != nil {
		http.Error(w, "compression error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := gz.Close(); err != nil {
		http.Error(w, "compression close error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"compressed_src": encoded})
}

// specToAttributeInfo converts a registry ComponentSpec into a sorted slice of
// AttributeInfo values suitable for JSON serialisation.
func specToAttributeInfo(spec registry.ComponentSpec) []AttributeInfo {
	names := make([]string, 0, len(spec.Attributes))
	for name := range spec.Attributes {
		names = append(names, name)
	}
	sort.Strings(names)

	attrs := make([]AttributeInfo, 0, len(names))
	for _, name := range names {
		a := spec.Attributes[name]
		info := AttributeInfo{
			Name:     name,
			Type:     string(a.Type),
			Required: a.Required,
		}
		if len(a.Enum) > 0 {
			vals := make([]string, 0, len(a.Enum))
			for v := range a.Enum {
				vals = append(vals, v)
			}
			sort.Strings(vals)
			info.Enum = vals
		}
		attrs = append(attrs, info)
	}
	return attrs
}

// openBrowser attempts to open url in the default system browser.
func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd, args = "cmd", []string{"/c", "start", url}
	case "darwin":
		cmd, args = "open", []string{url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	_ = exec.Command(cmd, args...).Start()
}
