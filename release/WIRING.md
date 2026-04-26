# CoreUI Release Wiring Reference

Use this alongside `CONTEXT_TEMPLATE.md` when integrating CoreUI into an existing application.

## Go (GOTH)

```go
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
```

## JavaScript (Standalone)

```js
const ui = new window.CoreUI(jsonData);

ui.onAction((data) => {
  if (data.namespace !== "app") {
    return;
  }

  switch (data.call) {
    case "notify":
      window.dispatchEvent(new CustomEvent("coreui:notify", { detail: data.params }));
      break;
    default:
      console.warn("Unhandled CoreUI app action", data);
  }
});

ui.render(document.getElementById("coreui-root"));
```
