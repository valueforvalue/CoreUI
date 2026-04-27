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

6. Agentic Workflows (v1.6.0)
Intent Layer: Every component in the registry carries an `intent` field (e.g. `action-trigger`, `data-label`, `layout-container`) that is exported into the compiled JSON blueprint. AI agents must use this field to verify that a layout matches the user's functional goal before rendering or patching. Invalid intent placements (e.g. a `data-label` inside a navigation-global zone) must be flagged by the agent before submission.

Structured Error Reporting: The `corec` CLI accepts a `--json-errors` flag. When a compilation fails and this flag is present, the compiler suppresses human-readable terminal output and instead writes a single JSON object to `stderr` with the following schema:

```json
{
  "status": "error",
  "errors": [
    {
      "line": 12,
      "column": 5,
      "error_code": "INVALID_ATTRIBUTE_TYPE",
      "message": "attribute \"gap\" expects unit",
      "expected": "unit (e.g. 20px, 50%, 1*)",
      "context_snippet": "    Stack(id=\"layout\", gap=bad_value)"
    }
  ]
}
```

AI agents must ingest this JSON and generate a patch for the `.cui` source file autonomously, without human intervention. The `errors` array may contain multiple entries to enable batch correction.

Knowledge Injection: The `corec explain` command generates a single high-density Markdown document from the live registry and parser specs. It is designed to be piped into a local LLM to provide 100% CoreUI context while completely offline. The output includes: an EBNF grammar summary, a flat component/attribute/intent registry table, action wiring examples, and three golden `.cui` code examples.