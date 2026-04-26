package lexer

import "strings"

type TokenType string

const (
	Illegal    TokenType = "ILLEGAL"
	EOF        TokenType = "EOF"
	Identifier TokenType = "IDENTIFIER"
	Number     TokenType = "NUMBER"
	String     TokenType = "STRING"
	Boolean    TokenType = "BOOLEAN"
	Unit       TokenType = "UNIT"
	LParen     TokenType = "("
	RParen     TokenType = ")"
	LBrace     TokenType = "{"
	RBrace     TokenType = "}"
	LBracket   TokenType = "["
	RBracket   TokenType = "]"
	Comma      TokenType = ","
	Equal      TokenType = "="
	Colon      TokenType = ":"
)

type Token struct {
	Type      TokenType
	Literal   string
	Line      int
	Col       int
	Offset    int
	EndOffset int
}

type Lexer struct {
	src  string
	pos  int
	line int
	col  int
}

func New(src string) *Lexer {
	return &Lexer{
		src:  src,
		line: 1,
		col:  1,
	}
}

func (l *Lexer) NextToken() Token {
	l.skipWhitespace()

	startPos := l.pos
	startLine := l.line
	startCol := l.col

	if l.pos >= len(l.src) {
		return Token{Type: EOF, Line: startLine, Col: startCol, Offset: startPos, EndOffset: startPos}
	}

	ch := l.current()
	switch ch {
	case '(':
		l.advance()
		return Token{Type: LParen, Literal: "(", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case ')':
		l.advance()
		return Token{Type: RParen, Literal: ")", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case '{':
		l.advance()
		return Token{Type: LBrace, Literal: "{", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case '}':
		l.advance()
		return Token{Type: RBrace, Literal: "}", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case '[':
		l.advance()
		return Token{Type: LBracket, Literal: "[", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case ']':
		l.advance()
		return Token{Type: RBracket, Literal: "]", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case ',':
		l.advance()
		return Token{Type: Comma, Literal: ",", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case '=':
		l.advance()
		return Token{Type: Equal, Literal: "=", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case ':':
		l.advance()
		return Token{Type: Colon, Literal: ":", Line: startLine, Col: startCol, Offset: startPos, EndOffset: l.pos}
	case '"':
		return l.readString()
	default:
		if isLetter(ch) {
			return l.readIdentifier()
		}
		if isDigit(ch) {
			return l.readNumberOrUnit()
		}
	}

	l.advance()
	return Token{
		Type:      Illegal,
		Literal:   "unexpected character",
		Line:      startLine,
		Col:       startCol,
		Offset:    startPos,
		EndOffset: l.pos,
	}
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.src) {
		ch := l.current()
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			return
		}
		l.advance()
	}
}

func (l *Lexer) readIdentifier() Token {
	startPos := l.pos
	startLine := l.line
	startCol := l.col

	for l.pos < len(l.src) && isIdentifierPart(l.current()) {
		l.advance()
	}

	literal := l.src[startPos:l.pos]
	tokenType := Identifier
	if literal == "true" || literal == "false" {
		tokenType = Boolean
	}

	return Token{
		Type:      tokenType,
		Literal:   literal,
		Line:      startLine,
		Col:       startCol,
		Offset:    startPos,
		EndOffset: l.pos,
	}
}

func (l *Lexer) readNumberOrUnit() Token {
	startPos := l.pos
	startLine := l.line
	startCol := l.col

	for l.pos < len(l.src) && isDigit(l.current()) {
		l.advance()
	}

	if l.pos < len(l.src) && l.current() == '.' {
		l.advance()
		for l.pos < len(l.src) && isDigit(l.current()) {
			l.advance()
		}
	}

	tokenType := Number
	switch {
	case strings.HasPrefix(l.src[l.pos:], "px"):
		l.advance()
		l.advance()
		tokenType = Unit
	case strings.HasPrefix(l.src[l.pos:], "%"):
		l.advance()
		tokenType = Unit
	case strings.HasPrefix(l.src[l.pos:], "*"):
		l.advance()
		tokenType = Unit
	case strings.HasPrefix(l.src[l.pos:], "auto"):
		l.advance()
		l.advance()
		l.advance()
		l.advance()
		tokenType = Unit
	}

	return Token{
		Type:      tokenType,
		Literal:   l.src[startPos:l.pos],
		Line:      startLine,
		Col:       startCol,
		Offset:    startPos,
		EndOffset: l.pos,
	}
}

func (l *Lexer) readString() Token {
	startPos := l.pos
	startLine := l.line
	startCol := l.col

	l.advance()
	var builder strings.Builder

	for l.pos < len(l.src) {
		ch := l.current()
		if ch == '"' {
			l.advance()
			return Token{
				Type:      String,
				Literal:   builder.String(),
				Line:      startLine,
				Col:       startCol,
				Offset:    startPos,
				EndOffset: l.pos,
			}
		}

		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.src) {
				break
			}
			switch l.current() {
			case '"', '\\':
				builder.WriteByte(l.current())
			case 'n':
				builder.WriteByte('\n')
			case 't':
				builder.WriteByte('\t')
			default:
				return Token{
					Type:      Illegal,
					Literal:   "invalid escape sequence",
					Line:      startLine,
					Col:       startCol,
					Offset:    startPos,
					EndOffset: l.pos,
				}
			}
			l.advance()
			continue
		}

		builder.WriteByte(ch)
		l.advance()
	}

	return Token{
		Type:      Illegal,
		Literal:   "unterminated string literal",
		Line:      startLine,
		Col:       startCol,
		Offset:    startPos,
		EndOffset: l.pos,
	}
}

func (l *Lexer) current() byte {
	return l.src[l.pos]
}

func (l *Lexer) advance() {
	if l.pos >= len(l.src) {
		return
	}

	if l.src[l.pos] == '\n' {
		l.line++
		l.col = 1
		l.pos++
		return
	}

	l.pos++
	l.col++
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentifierPart(ch byte) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_'
}
