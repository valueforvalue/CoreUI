# CoreUI: Purpose and Philosophy

## The Mission
To provide a deterministic, structural UI language that bridges the gap between human intent and AI execution. CoreUI is designed to be "hallucination-proof" by restricting the UI to rigid, predictable primitives.

## The Problem
Standard web technologies (HTML/CSS) are too verbose and flexible. When an AI generates complex UIs, it often loses spatial awareness, leading to overlapping elements, broken layouts, and "CSS drift." 

## The Solution
CoreUI uses a strict, declarative DSL that compiles into a universal JSON schema. By decoupling **Structure** (the .cui file) from **Implementation** (the platform-specific renderer), we ensure that the UI remains a stable blueprint regardless of the target environment.

## Target Personas
1. **The Logic-First Developer:** Focuses on backend systems (Go, Python); needs a UI that "just works" without writing CSS.
2. **The AI Coding Agent:** The primary generator of CoreUI code. It needs a high-precision, low-token syntax to minimize structural errors.
3. **The Rapid Prototyper:** Mocks up complex, data-heavy systems (e.g., service tools, simulations) with iron-clad referencing via mandatory IDs.

## Guiding Principles
* **Whole-to-Part Design:** Layout containers (Stacks, Grids) dictate the boundaries of children.
* **Stateless Communication:** The UI only emits intent via the Action Protocol.
* **Zero-Dependency:** The core parser must remain a lightweight, portable Go binary using only the standard library.