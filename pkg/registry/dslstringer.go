package registry

// DSLStringer is implemented by any value type that can serialise itself back
// to a CoreUI DSL attribute-value token.  The returned string is compatible
// with the .cui parser: for scalar types (bool, int, unit, action) it can be
// placed directly after the `=` sign; for string values the caller must wrap
// the result in double quotes before writing to a .cui source file.
//
// Implementations are provided on ast.Value and ast.Action in pkg/ast.
type DSLStringer interface {
	ToDSLString() string
}
