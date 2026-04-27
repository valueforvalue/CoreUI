package flow

import "fmt"

// TokenType identifies the kind of a lexer token.
type TokenType string

const (
	ttEOF     TokenType = "EOF"
	ttIllegal TokenType = "ILLEGAL"
	ttIdent   TokenType = "IDENT"
	ttString  TokenType = "STRING"
	ttInt     TokenType = "INT"
	ttFloat   TokenType = "FLOAT"
	ttTrue    TokenType = "TRUE"
	ttFalse   TokenType = "FALSE"
	ttLBrace  TokenType = "{"
	ttRBrace  TokenType = "}"
	ttLParen  TokenType = "("
	ttRParen  TokenType = ")"
	ttComma   TokenType = ","
	ttEq      TokenType = "="
	ttEqEq    TokenType = "=="
	ttNeq     TokenType = "!="
	ttGt      TokenType = ">"
	ttLt      TokenType = "<"
	ttGte     TokenType = ">="
	ttLte     TokenType = "<="
	ttPlus    TokenType = "+"
	ttMinus   TokenType = "-"
	ttStar    TokenType = "*"
	ttSlash   TokenType = "/"
)

// token is a single lexeme produced by the flow lexer.
type token struct {
	Type    TokenType
	Literal string
	Line    int
	Col     int
}

// flowLexer tokenises a CoreFlow source string.
type flowLexer struct {
	src  []rune
	pos  int
	line int
	col  int
}

func newLexer(src string) *flowLexer {
	return &flowLexer{src: []rune(src), pos: 0, line: 1, col: 1}
}

func (l *flowLexer) peek() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *flowLexer) peekAt(offset int) rune {
	idx := l.pos + offset
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *flowLexer) advance() rune {
	if l.pos >= len(l.src) {
		return 0
	}
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *flowLexer) skipWhitespaceAndComments() {
	for l.pos < len(l.src) {
		ch := l.peek()
		switch {
		case ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n':
			l.advance()
		case ch == '#':
			for l.pos < len(l.src) && l.peek() != '\n' {
				l.advance()
			}
		case ch == '/' && l.peekAt(1) == '/':
			for l.pos < len(l.src) && l.peek() != '\n' {
				l.advance()
			}
		default:
			return
		}
	}
}

// Next returns the next token from the source.
func (l *flowLexer) Next() token {
	l.skipWhitespaceAndComments()

	if l.pos >= len(l.src) {
		return token{Type: ttEOF, Line: l.line, Col: l.col}
	}

	startLine, startCol := l.line, l.col
	ch := l.peek()

	switch {
	case ch == '{':
		l.advance()
		return token{Type: ttLBrace, Literal: "{", Line: startLine, Col: startCol}
	case ch == '}':
		l.advance()
		return token{Type: ttRBrace, Literal: "}", Line: startLine, Col: startCol}
	case ch == '(':
		l.advance()
		return token{Type: ttLParen, Literal: "(", Line: startLine, Col: startCol}
	case ch == ')':
		l.advance()
		return token{Type: ttRParen, Literal: ")", Line: startLine, Col: startCol}
	case ch == ',':
		l.advance()
		return token{Type: ttComma, Literal: ",", Line: startLine, Col: startCol}
	case ch == '+':
		l.advance()
		return token{Type: ttPlus, Literal: "+", Line: startLine, Col: startCol}
	case ch == '*':
		l.advance()
		return token{Type: ttStar, Literal: "*", Line: startLine, Col: startCol}
	case ch == '/':
		l.advance()
		return token{Type: ttSlash, Literal: "/", Line: startLine, Col: startCol}
	case ch == '-':
		// Negative number literal: "-" followed by a digit.
		if isDigit(l.peekAt(1)) {
			return l.readNumber(startLine, startCol)
		}
		l.advance()
		return token{Type: ttMinus, Literal: "-", Line: startLine, Col: startCol}
	case ch == '=':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return token{Type: ttEqEq, Literal: "==", Line: startLine, Col: startCol}
		}
		return token{Type: ttEq, Literal: "=", Line: startLine, Col: startCol}
	case ch == '!':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return token{Type: ttNeq, Literal: "!=", Line: startLine, Col: startCol}
		}
		return token{Type: ttIllegal, Literal: "unexpected '!'", Line: startLine, Col: startCol}
	case ch == '>':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return token{Type: ttGte, Literal: ">=", Line: startLine, Col: startCol}
		}
		return token{Type: ttGt, Literal: ">", Line: startLine, Col: startCol}
	case ch == '<':
		l.advance()
		if l.peek() == '=' {
			l.advance()
			return token{Type: ttLte, Literal: "<=", Line: startLine, Col: startCol}
		}
		return token{Type: ttLt, Literal: "<", Line: startLine, Col: startCol}
	case ch == '"':
		return l.readString(startLine, startCol)
	case isDigit(ch):
		return l.readNumber(startLine, startCol)
	case isLetter(ch) || ch == '_':
		return l.readIdent(startLine, startCol)
	default:
		l.advance()
		return token{Type: ttIllegal, Literal: fmt.Sprintf("unexpected character %q", ch), Line: startLine, Col: startCol}
	}
}

func (l *flowLexer) readString(line, col int) token {
	l.advance() // consume opening "
	var buf []rune
	for l.pos < len(l.src) {
		ch := l.peek()
		if ch == '"' {
			l.advance()
			return token{Type: ttString, Literal: string(buf), Line: line, Col: col}
		}
		if ch == '\\' {
			l.advance()
			esc := l.advance()
			switch esc {
			case 'n':
				buf = append(buf, '\n')
			case 't':
				buf = append(buf, '\t')
			case '"':
				buf = append(buf, '"')
			case '\\':
				buf = append(buf, '\\')
			default:
				buf = append(buf, '\\', esc)
			}
			continue
		}
		buf = append(buf, ch)
		l.advance()
	}
	return token{Type: ttIllegal, Literal: "unterminated string", Line: line, Col: col}
}

func (l *flowLexer) readNumber(line, col int) token {
	var buf []rune
	// Allow leading minus for negative literals.
	if l.peek() == '-' {
		buf = append(buf, l.advance())
	}
	for isDigit(l.peek()) {
		buf = append(buf, l.advance())
	}
	if l.peek() == '.' && isDigit(l.peekAt(1)) {
		buf = append(buf, l.advance()) // consume '.'
		for isDigit(l.peek()) {
			buf = append(buf, l.advance())
		}
		return token{Type: ttFloat, Literal: string(buf), Line: line, Col: col}
	}
	return token{Type: ttInt, Literal: string(buf), Line: line, Col: col}
}

func (l *flowLexer) readIdent(line, col int) token {
	var buf []rune
	for isLetter(l.peek()) || isDigit(l.peek()) || l.peek() == '_' {
		buf = append(buf, l.advance())
	}
	lit := string(buf)
	switch lit {
	case "true":
		return token{Type: ttTrue, Literal: lit, Line: line, Col: col}
	case "false":
		return token{Type: ttFalse, Literal: lit, Line: line, Col: col}
	}
	return token{Type: ttIdent, Literal: lit, Line: line, Col: col}
}

func isDigit(ch rune) bool { return ch >= '0' && ch <= '9' }
func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}
