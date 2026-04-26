# AI Behavioral Instructions

## Implementation Rules
1. **Standard Library Only:** Use only Go's standard library. No external modules.
2. **Registry-First Validation:** All properties must be checked against `pkg/registry`.
3. **Mandatory Identity:** Reject any component missing a unique global `id`.
4. **Action Protocol:** Sub-parse `action` strings into structured objects in JSON output.
5. **Hard Boundary Rule:** Do not generate raw HTML, JSX, or CSS from CoreUI source.

## Testing Mandate
- Include negative tests for duplicate IDs, unknown properties, malformed action strings, and type mismatches.
- Keep `.cui` to `.json` golden files under `/testdata`.
- Validate built-in `ui` actions.

## Error Reporting
- Emit errors in the format `[Error] Line <X>, Col <Y>: <Message>`.
