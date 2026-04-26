package diag

import "fmt"

type Error struct {
	Line    int
	Col     int
	Message string
}

func New(line, col int, message string) *Error {
	return &Error{
		Line:    line,
		Col:     col,
		Message: message,
	}
}

func Newf(line, col int, format string, args ...any) *Error {
	return &Error{
		Line:    line,
		Col:     col,
		Message: fmt.Sprintf(format, args...),
	}
}

func (e *Error) Error() string {
	return fmt.Sprintf("[Error] Line %d, Col %d: %s", e.Line, e.Col, e.Message)
}
