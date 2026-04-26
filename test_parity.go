package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"coreui/renderers/goth"
)

func runParityServer(addr string) error {
	node, theme, err := loadCompiledNode("comprehensive.json")
	if err != nil {
		return err
	}

	rendererPath := filepath.Join("renderers", "js", "renderer.js")
	rendererSource, err := os.ReadFile(rendererPath)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprint(w, `<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>CoreUI Renderer Parity</title>
  <style>
    * { background: red !important; }
    body { margin: 0; font-family: Arial, Helvetica, sans-serif; }
    .page { display: grid; grid-template-columns: 1fr 1fr; gap: 24px; padding: 24px; }
    .pane { border: 2px solid #111827; padding: 16px; min-height: 100vh; }
    .pane h2 { margin-top: 0; }
  </style>
</head>
<body>
  <div class="page">
    <section class="pane" id="goth-pane">
      <h2>Left Pane (GOTH)</h2>`)
		if err := goth.RenderWithTheme(node, theme).Render(r.Context(), w); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, `
    </section>
    <section class="pane" id="js-pane">
      <h2>Right Pane (JS)</h2>
      <div id="js-target"></div>
    </section>
  </div>
  <script type="module">
    import CoreUI from "/renderers/js/renderer.js";

    const response = await fetch("/comprehensive.json");
    const json = await response.json();
    const app = new CoreUI(json);
    app.onAction((action) => {
      console.log("JS ActionRequest", action);
      window.__coreuiJsActions = window.__coreuiJsActions || [];
      window.__coreuiJsActions.push(action);
    });
    app.render(document.getElementById("js-target"));
    window.__coreuiApp = app;
  </script>
</body>
</html>`)
	})

	mux.HandleFunc("/comprehensive.json", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "comprehensive.json")
	})

	mux.HandleFunc("/renderers/js/renderer.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		_, _ = w.Write(rendererSource)
	})

	mux.Handle("/coreui/action", goth.HandleAction(goth.ActionExecutorFunc(func(ctx context.Context, action goth.ActionRequest, w http.ResponseWriter, r *http.Request) error {
		log.Printf("Parity ActionRequest: namespace=%s call=%s params=%v", action.Namespace, action.Call, action.Params)
		w.Header().Set("Content-Type", "application/json")
		return json.NewEncoder(w).Encode(action)
	})))

	log.Printf("Parity harness listening on http://127.0.0.1%s", addr)
	return http.ListenAndServe(addr, mux)
}
