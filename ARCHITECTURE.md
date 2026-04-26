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