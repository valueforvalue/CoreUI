# Copilot Instructions for CoreUI

- Refer to ARCHITECTURE.md before making any code changes.

## Build, test, and formatting commands
- Run the full test suite with `go test ./...`
- Run parser-only negative tests with `go test ./pkg/parser -run TestParserRejectsDuplicateIDs -count=1`
- Run the golden-file integration test with `go test ./pkg/compiler -run TestGoldenFiles/dashboard -count=1`
- Format the repository with `gofmt -w cmd pkg`
- Compile a `.cui` file from the CLI with `go run ./cmd/corec -o <output.json> <input.cui>`

## High-level architecture
- `cmd/corec` is the CLI entrypoint. It reads one `.cui` file, compiles it, and writes JSON output beside the source file unless `-o` is provided.
- `pkg/lexer` performs tokenization and contains the custom unit handling. Numbers immediately followed by `px`, `%`, `*`, or `auto` must become a single unit token.
- `pkg/parser` owns recursive-descent parsing, global ID tracking, registry-backed validation, and the dedicated `action` sub-parser. `action` is parsed as an opaque attribute value first, then decomposed into `namespace`, `call`, and `params`.
- `pkg/registry` is the source of truth for allowed components, attributes, required fields, and type validation.
- `pkg/generator` converts the typed AST into deterministic JSON with three root keys: `tree`, `index`, and `metadata`. The `index` maps each component `id` to its JSON path and component type.
- `pkg/compiler` wires parsing and generation together for both tests and CLI usage.

## Key conventions
- Stay zero-dependency: use only the Go standard library.
- Treat `Master_Prompt.txt` as the top-priority implementation source when it conflicts with broader prose docs.
- Every component must have a globally unique `id`; parser failures use the exact error format `[Error] Line <X>, Col <Y>: <Message>`.
- Validate every attribute through `pkg/registry` before accepting it into the AST. Unknown attributes and type mismatches are fatal parse errors.
- Keep `action` structured in JSON output under `attributes.action`; do not add a separate top-level `actions` object unless the contract changes.
- Golden-file coverage lives in `/testdata` and uses a fixed timestamp/version through `pkg/compiler.Options` so expected JSON stays stable.



