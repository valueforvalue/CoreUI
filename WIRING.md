# CoreUI Wiring Reference

## Go (GOTH)

Use the GOTH action handler to decode the structured CoreUI action payload and route it into your backend logic.

```go
package main

import (
    "context"
    "encoding/json"
    "net/http"

    "coreui/renderers/goth"
)

func main() {
    mux := http.NewServeMux()
    mux.Handle("/coreui/action", goth.HandleAction(goth.ActionExecutorFunc(func(ctx context.Context, action goth.ActionRequest, w http.ResponseWriter, r *http.Request) error {
        switch action.Namespace {
        case "ui":
            return handleBuiltInUIAction(action, w)
        case "app":
            return routeAppAction(action.Call, action.Params, w)
        default:
            http.Error(w, "unsupported action namespace", http.StatusBadRequest)
            return nil
        }
    })))

    _ = http.ListenAndServe(":8080", mux)
}

func handleBuiltInUIAction(action goth.ActionRequest, w http.ResponseWriter) error {
    w.Header().Set("Content-Type", "application/json")
    return json.NewEncoder(w).Encode(map[string]any{
        "namespace": action.Namespace,
        "call":      action.Call,
        "params":    action.Params,
    })
}

func routeAppAction(call string, params map[string]any, w http.ResponseWriter) error {
    w.Header().Set("Content-Type", "application/json")
    return json.NewEncoder(w).Encode(map[string]any{
        "handled": true,
        "call":    call,
        "params":  params,
    })
}
```

## JavaScript (Standalone)

Use `onAction` to bridge CoreUI `app:` actions into your existing browser-side logic.

```js
const ui = new window.CoreUI(jsonData);

ui.onAction((data) => {
  if (data.namespace !== "app") {
    return;
  }

  switch (data.call) {
    case "notify":
      window.dispatchEvent(
        new CustomEvent("coreui:notify", { detail: data.params })
      );
      break;
    default:
      console.warn("Unhandled CoreUI app action", data);
  }
});

ui.render(document.getElementById("coreui-root"));
```
