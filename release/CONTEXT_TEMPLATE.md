# CoreUI Context Stream

**Registry Version:** 1.3.0
**Schema Compatibility:** 1.0
**Last Updated:** 2026-04-26

## Iron-Clad Architecture Principles

This document serves as the high-level mandate for the CoreUI ecosystem. Any modification that violates these principles is a hard failure.

1. The Registry-First Mandate
Source of Truth: The pkg/registry is the absolute authority. No component, attribute, or property is valid unless it is explicitly defined in the Go registry.

No Implicit Logic: Renderers are forbidden from "guessing" what an attribute does. If it isn't in the JSON blueprint, the renderer must not invent it.

2. The Multi-Platform Mirror
Parity Guarantee: The GOTH (Server) and JS (Client) renderers must produce functionally and visually identical results for the same JSON input.

Contractual Compliance: Any feature added to the GOTH renderer must have a corresponding implementation in the JS renderer within the same update cycle.

3. The Shadow DOM Isolation
The Safe Room: The JavaScript renderer must always utilize the Shadow DOM. Bypassing the shadow root to interact with the global DOM is strictly prohibited to prevent CSS leakage and "muddle."

4. The Action Protocol (Tiered Communication)
ui: Namespace: Reserved for core, contractual primitives (navigate, notify, etc.). These must be strictly validated.

app: Namespace: Reserved for user-defined intent. The compiler validates the syntax, while the application logic handles the intent.

5. Unit Integrity
Type Safety: Units (px, %, *) are distinct from raw strings. The compiler must never "coerce" a string into a unit or vice-versa. If it looks like a unit, it must be validated as a unit.

## Component Catalog


### Box

- **HasChildren:** true

| Attribute | Type | Requirement |
| --- | --- | --- |
| background | String | Optional |
| border | Int | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| padding | Unit | Optional |
| style | String | Optional |
| variant | String | Optional |



### Color

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| key | String | Required |
| value | String | Required |



### DataTable

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| selectable | Bool | Optional |
| source | String | Optional |
| style | String | Optional |



### Graph

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| color | Theme Token | Optional |
| data | JSON Array or app:reference | Required |
| height | Unit | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| labels | []string | Optional |
| style | String | Optional |
| type | String | Required |



### Grid

- **HasChildren:** true

| Attribute | Type | Requirement |
| --- | --- | --- |
| cols | Unit Array | Optional |
| gap | Unit | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| rows | Unit Array | Optional |
| style | String | Optional |



### Image

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| alt | String | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| src | String | Required |
| style | String | Optional |
| width | Unit | Optional |



### Input

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| bind | String | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| label | String | Optional |
| style | String | Optional |
| type | String | Optional |



### Stack

- **HasChildren:** true

| Attribute | Type | Requirement |
| --- | --- | --- |
| align | String | Optional |
| dir | String | Optional |
| gap | Unit | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| style | String | Optional |



### Text

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| size | Unit | Optional |
| style | String | Optional |
| value | String | Optional |
| weight | String | Optional |



### Theme

- **HasChildren:** true

| Attribute | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| style | String | Optional |



### Trigger

- **HasChildren:** false

| Attribute | Type | Requirement |
| --- | --- | --- |
| action | Action | Optional |
| hidden | Bool | Optional |
| id | String | Required |
| label | String | Optional |
| style | String | Optional |
| variant | String | Optional |



### View

- **HasChildren:** true

| Attribute | Type | Requirement |
| --- | --- | --- |
| hidden | Bool | Optional |
| id | String | Required |
| style | String | Optional |
| theme | String | Optional |
| title | String | Optional |



## BI & Visualization Guidance

The `Graph` component is the BI and visualization primitive for CoreUI.

- `labels` is a string array for axis or legend text.
- `data` is either a literal numeric JSON array or a quoted `app:` reference string when your simulation data is resolved at runtime.
- Keep `labels` and literal `data` arrays aligned by index: the first label describes the first value, the second label describes the second value, and so on.
- When emitting runtime references, prefer stable names such as `app:simulation.temperature_series` or `app:ops.queue_depth` so downstream agents can map data producers to Graph nodes deterministically.

Example:

~~~cui
Graph(
    id="temperature_graph",
    type="area",
    color="primary",
    height=240px,
    labels=["T+0", "T+5", "T+10", "T+15"],
    data="app:simulation.temperature_series"
)
~~~

## Theme Tokens

These are the standard starter tokens used by CoreUI onboarding templates:

- `primary`
- `surface`
- `panel`
- `background`
- `text`
- `radius`
- `shadow`
- `speed`


### Semantic token values

- `radius`: `none`, `sm`, `md`, `lg`, `full`
- `shadow`: `none`, `soft`, `deep`
- `speed`: `instant`, `smooth`, `lazy`


## Factory Themes


### Industrial

| Token | Value |
| --- | --- |
| background | surface |
| panel | #ffffff |
| primary | #2563eb |
| radius | none |
| shadow | none |
| speed | instant |
| surface | #dbe4f0 |
| text | #111827 |



### Modern

| Token | Value |
| --- | --- |
| background | #f8fafc |
| panel | #ffffff |
| primary | #6366f1 |
| radius | md |
| shadow | soft |
| speed | smooth |
| surface | #ffffff |
| text | #0f172a |



### Cyber

| Token | Value |
| --- | --- |
| background | #000000 |
| panel | #000000 |
| primary | #00ff00 |
| radius | none |
| shadow | none |
| speed | instant |
| surface | #000000 |
| text | #00ff00 |




## Action Protocol

`ui:` is reserved for registry-validated CoreUI primitives. `app:` is open for application intent, but must still follow valid CoreUI action syntax and emit the same `{ namespace, call, params }` structure.

### Built-in `ui:` actions

- `ui:navigate(target="id")`
- `ui:toggle(id="target_id")`
- `ui:close()`
- `ui:notify(msg="Done", type="success")`


### User-defined `app:` actions

Use `app:` for application-specific intent. Preserve the same structured payload shape and route execution in your own application layer.

## Wiring Snippets

### Go (GOTH)

~~~go
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
~~~

### JavaScript (Standalone)

~~~js
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
~~~
