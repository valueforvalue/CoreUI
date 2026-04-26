# CoreUI

CoreUI is a deterministic UI DSL and compiler for building structural user interfaces without hand-writing HTML and CSS. You describe a screen in a small `.cui` language, compile it into a stable JSON blueprint, and render that blueprint through either a static Go renderer (**GOTH**) or a browser-side JavaScript renderer.

The project is built to be predictable for both humans and AI agents:

- **Registry-first**: every valid component and attribute is defined in `pkg\registry`
- **Deterministic output**: the compiler always emits the same root JSON contract: `tree`, `index`, and `metadata`
- **Renderer parity**: GOTH and JS are expected to mirror each other for the same input
- **Strict units and IDs**: units are typed, and every component must have a globally unique `id`

## Why CoreUI exists

Traditional HTML/CSS gives AIs and rapid prototypers too much freedom. That flexibility turns into layout drift, invalid structure, and inconsistent behavior. CoreUI narrows the surface area to a fixed set of layout and interaction primitives so the UI remains stable across tools and platforms.

CoreUI is aimed at:

1. **Logic-first developers** who want a UI layer without building a front-end stack from scratch
2. **AI coding agents** that need a low-ambiguity syntax and a strict output contract
3. **Rapid prototypers** building dashboards, operator consoles, simulation panels, and data-heavy interfaces

## What you get

CoreUI ships with:

- `corec`: the compiler CLI
- `coredoc`: the component reference generator
- a strict parser, lexer, registry, and JSON generator
- a **GOTH** renderer for static/server rendering
- a **JS** renderer for Shadow DOM browser rendering
- release artifacts with generated docs and onboarding context

## Current feature set

### Core components

- `View`
- `Stack`
- `Grid`
- `Box`
- `Text`
- `Input`
- `Image`
- `Trigger`
- `DataTable`
- `Graph`
- `Theme`
- `Color`

### BI and visualization

`Graph` is the visualization primitive introduced in `v1.3.0`.

- `type`: `line`, `bar`, `area`, or `pie`
- `data`: a numeric array or a **quoted** app reference string such as `data="app:metrics.cpu"`
- `color`: a theme token
- `height`: a typed unit such as `240px` or `40%`
- `labels`: a string array aligned by index with `data`

Important contract details:

- `app:` references are **quoted strings**, not special parser syntax
- invalid units are rejected at compile time
- GOTH emits raw SVG so charts remain visible without JavaScript
- the JS renderer uses lightweight SVG logic inside the Shadow DOM

## Repository layout

| Path | Purpose |
| --- | --- |
| `cmd\corec` | Compiler CLI entrypoint |
| `cmd\coredoc` | Generates `COMPONENTS.md` from the registry |
| `pkg\lexer` | Tokenization and unit lexing |
| `pkg\parser` | Recursive-descent parsing, validation, action parsing, ID tracking |
| `pkg\registry` | Source of truth for components, attributes, types, enums, and themes |
| `pkg\renderers` | Embedded browser renderer assets and standalone HTML helpers |
| `pkg\generator` | Converts AST into deterministic JSON |
| `pkg\compiler` | Wires parsing and generation for tests and CLI use |
| `renderers\goth` | Static/server renderer |
| `testdata` | Golden compiler fixtures |
| `tests` | Sentinel and end-to-end checks |
| `release` | Packaged docs, prompt files, and distribution artifacts |

## Requirements

- **Go 1.26.1** or newer
- Optional: `bash` for `scripts\build_release.sh`
- Optional: `gh` if you want to publish GitHub releases from the CLI

The compiler core stays lightweight and standard-library focused. The GOTH renderer uses `github.com/a-h/templ`.

## Quick Install

```powershell
go get github.com/valueforvalue/coreui
```

## Getting started

### 1. Clone and build

```powershell
git clone https://github.com/valueforvalue/coreui.git
Set-Location CoreUI
go test ./...
go build ./cmd/corec
go build ./cmd/coredoc
```

### 2. Create a starter project

```powershell
go run .\cmd\corec init hello_world
```

That writes `hello_world.cui` with starter themes and a sample view, including a `Graph(type="line")`.

### 3. Compile to JSON

```powershell
go run .\cmd\corec -o hello_world.json hello_world.cui
```

### 4. Compile to a standalone HTML file

```powershell
go run .\cmd\corec -s -o hello_world.html hello_world.cui
```

Open `hello_world.html` in a browser to see the JS renderer run against the compiled JSON embedded in the page.

## Your first `.cui` file

```cui
Theme(id="Modern") {
    Color(key="radius", value="md"),
    Color(key="shadow", value="soft"),
    Color(key="speed", value="smooth"),
    Color(key="surface", value="#ffffff"),
    Color(key="panel", value="#ffffff"),
    Color(key="background", value="#f8fafc"),
    Color(key="text", value="#0f172a"),
    Color(key="primary", value="#6366f1")
}

View(id="root", title="Operations", theme="Modern") {
    Stack(id="layout", dir="v", gap=16px) {
        Text(id="title", value="Operations Dashboard", size=24px, weight="bold", style="color: primary")
        Graph(
            id="cpu_graph",
            type="line",
            color="primary",
            height=240px,
            labels=["08:00", "10:00", "12:00", "14:00"],
            data=[18, 24, 21, 29]
        )
        Trigger(id="refresh", label="Refresh", variant="primary", action="app:refresh(target=\"cpu_graph\")")
    }
}
```

## CLI reference

### `corec`

Compile a `.cui` file:

```powershell
corec [-standalone] [-o output.{json|html}] input.cui
```

Commands:

- `corec init <project-name>`: write a starter `.cui` file
- `corec context`: print the AI onboarding context stream
- `corec -o out.json input.cui`: compile to JSON
- `corec -s input.cui`: compile to standalone HTML
- `corec -version`: print compiler and registry version information

### `coredoc`

Generate the component reference from the registry:

```powershell
go run .\cmd\coredoc
```

This rewrites `COMPONENTS.md`.

## The CoreUI language

### Key rules

1. Every component must have a globally unique `id`
2. Strings use double quotes
3. Units are typed and not coerced from raw strings
4. Arrays preserve typed values
5. `Theme` is top-level metadata and is not emitted into the UI tree
6. `action` is parsed as structured data and emitted inline under `attributes.action`

### Units

Supported unit forms:

- `px`
- `%`
- `*`
- `auto`

Examples:

- `gap=16px`
- `cols=[1*, 300px]`
- `height=40%`

If you write an invalid unit such as `500xyz`, the compiler rejects it.

### Action protocol

Actions use this shape:

```text
namespace:function(key="value")
```

Namespaces:

- `ui:` for built-in, registry-validated UI primitives
- `app:` for application-owned intent that still follows valid CoreUI action syntax

Examples:

```cui
Trigger(id="open", label="Open", action="ui:navigate(target=\"settings\")")
Trigger(id="notify", label="Notify", action="app:notify(msg=\"Done\", channel=\"ops\")")
```

## Compiler output contract

The compiler emits:

```json
{
  "tree": {},
  "index": {},
  "theme": {},
  "metadata": {}
}
```

- `tree`: the nested component tree
- `index`: maps each `id` to a JSON path and component type
- `theme`: flattened theme tokens
- `metadata`: compile timestamp and version data

This contract is stable enough to use as a handoff format between the compiler and either renderer.

## Rendering model

### JavaScript renderer

- runs inside a Shadow DOM root
- consumes compiled JSON
- handles `Trigger` callbacks via `onAction`
- renders `Graph` with SVG and lightweight client-side logic
- uses theme tokens like `speed` and `radius` for graph transitions and bar corners

### GOTH renderer

- renders static/server HTML from the AST
- uses raw SVG output for `Graph`
- keeps graph content visible even with JavaScript disabled
- is expected to stay in visual and behavioral parity with the JS renderer

## Themes and tokens

CoreUI ships with starter themes such as `Industrial`, `Modern`, and `Cyber`.

Standard tokens:

- `primary`
- `surface`
- `panel`
- `background`
- `text`
- `radius`
- `shadow`
- `speed`

Semantic token values:

- `radius`: `none`, `sm`, `md`, `lg`, `full`
- `shadow`: `none`, `soft`, `deep`
- `speed`: `instant`, `smooth`, `lazy`

## Docs and onboarding files

These files matter when you are learning or extending the project:

- `ARCHITECTURE.md`: non-negotiable design rules
- `PURPOSE.md`: mission and product framing
- `SPEC.md`: grammar, contract, and language rules
- `COMPONENTS.md`: generated component reference
- `WIRING.md`: Go and JavaScript integration examples
- `release\CONTEXT_TEMPLATE.md`: AI onboarding context generated from the project state

## Development workflow

### Run the full test suite

```powershell
go test ./...
```

### Run parser-only negative coverage

```powershell
go test ./pkg/parser -run TestParserRejectsDuplicateIDs -count=1
```

### Run the dashboard golden integration test

```powershell
go test ./pkg/compiler -run TestGoldenFiles/dashboard -count=1
```

### Format the repository

```powershell
gofmt -w cmd pkg
```

### Regenerate docs

```powershell
go run ./cmd/coredoc
go run ./cmd/corec context > release/CONTEXT_TEMPLATE.md
```

## Release packaging

Build the cross-platform release bundle:

```powershell
bash scripts/build_release.sh
```

That script:

1. runs tests and sentinel checks
2. regenerates `COMPONENTS.md`
3. builds `corec` and `coredoc` for Windows, Linux, and macOS
4. regenerates `release\CONTEXT_TEMPLATE.md`
5. creates `release\coreui-v<version>.zip`

## Extending CoreUI

When adding a component or attribute:

1. Define it in `pkg\registry`
2. Validate its types there instead of guessing in renderers
3. Make sure the parser and generator preserve the typed shape
4. implement both renderers in the same update cycle
5. update docs, fixtures, and sentinel coverage

The architecture rules are strict on purpose: if something is not in the registry, it is not part of the language.

## Example files

- `testdata\dashboard.cui`: dashboard-shaped sample
- `tests\fixtures\kitchen_sink.cui`: broader integration sample
- `testdata\Graph_fixture.cui`: focused graph fixture

## Status

Current registry version: **1.3.0**

CoreUI is set up as a compiler + renderer contract, not a general-purpose browser framework. If you need rigid structure, deterministic JSON, and an AI-friendly UI description language, this repository is the starting point.
