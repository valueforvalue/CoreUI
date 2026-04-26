package parser

import (
	"fmt"
	"strconv"
	"strings"

	"coreui/pkg/ast"
)

type actionParser struct {
	input string
	pos   int
}

func parseAction(raw string) (ast.Action, error) {
	parser := &actionParser{input: strings.TrimSpace(raw)}
	return parser.parse()
}

func (p *actionParser) parse() (ast.Action, error) {
	namespace, err := p.readIdentifier()
	if err != nil {
		return ast.Action{}, err
	}

	if !p.consume(':') {
		return ast.Action{}, fmt.Errorf("action must include namespace and call")
	}

	call, err := p.readIdentifier()
	if err != nil {
		return ast.Action{}, err
	}

	params := map[string]ast.Value{}
	p.skipSpaces()

	if p.consume('(') {
		p.skipSpaces()
		if !p.consume(')') {
			for {
				key, err := p.readIdentifier()
				if err != nil {
					return ast.Action{}, err
				}

				if !p.consume('=') {
					return ast.Action{}, fmt.Errorf("invalid action parameter %q", key)
				}

				value, err := p.readValue()
				if err != nil {
					return ast.Action{}, err
				}

				params[key] = value
				p.skipSpaces()

				if p.consume(')') {
					break
				}
				if !p.consume(',') {
					return ast.Action{}, fmt.Errorf("expected ',' or ')' in action parameters")
				}
				p.skipSpaces()
			}
		}
	}

	p.skipSpaces()
	if !p.done() {
		return ast.Action{}, fmt.Errorf("unexpected trailing content in action")
	}

	action := ast.Action{
		Namespace: namespace,
		Call:      call,
		Params:    params,
	}

	if err := validateBuiltInAction(action); err != nil {
		return ast.Action{}, err
	}

	return action, nil
}

func validateBuiltInAction(action ast.Action) error {
	if action.Namespace != "ui" {
		return nil
	}

	switch action.Call {
	case "navigate":
		if len(action.Params) != 1 {
			return fmt.Errorf("ui:navigate requires exactly one parameter")
		}
		if _, ok := action.Params["target"]; !ok {
			return fmt.Errorf("ui:navigate requires target parameter")
		}
	case "toggle":
		if len(action.Params) != 1 {
			return fmt.Errorf("ui:toggle requires exactly one parameter")
		}
		if _, ok := action.Params["id"]; !ok {
			return fmt.Errorf("ui:toggle requires id parameter")
		}
	case "close":
		if len(action.Params) != 0 {
			return fmt.Errorf("ui:close does not accept parameters")
		}
	default:
		return fmt.Errorf("unsupported built-in action %q", action.Call)
	}

	return nil
}

func (p *actionParser) readValue() (ast.Value, error) {
	p.skipSpaces()
	if p.done() {
		return ast.Value{}, fmt.Errorf("missing action parameter value")
	}

	switch ch := p.input[p.pos]; {
	case ch == '"':
		text, err := p.readQuotedString()
		if err != nil {
			return ast.Value{}, err
		}
		return ast.Value{Kind: ast.StringKind, Data: text}, nil
	case isActionNumberStart(ch):
		return p.readNumericLikeValue()
	case isActionIdentifierStart(ch):
		identifier, err := p.readIdentifier()
		if err != nil {
			return ast.Value{}, err
		}
		switch identifier {
		case "true":
			return ast.Value{Kind: ast.BoolKind, Data: true}, nil
		case "false":
			return ast.Value{Kind: ast.BoolKind, Data: false}, nil
		default:
			return ast.Value{Kind: ast.StringKind, Data: identifier}, nil
		}
	default:
		return ast.Value{}, fmt.Errorf("unsupported action parameter value")
	}
}

func (p *actionParser) readNumericLikeValue() (ast.Value, error) {
	start := p.pos
	for !p.done() && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
		p.pos++
	}

	if !p.done() && p.input[p.pos] == '.' {
		p.pos++
		for !p.done() && p.input[p.pos] >= '0' && p.input[p.pos] <= '9' {
			p.pos++
		}
	}

	kind := ast.NumberKind
	switch {
	case strings.HasPrefix(p.input[p.pos:], "px"):
		p.pos += 2
		kind = ast.UnitKind
	case strings.HasPrefix(p.input[p.pos:], "%"):
		p.pos++
		kind = ast.UnitKind
	case strings.HasPrefix(p.input[p.pos:], "*"):
		p.pos++
		kind = ast.UnitKind
	case strings.HasPrefix(p.input[p.pos:], "auto"):
		p.pos += 4
		kind = ast.UnitKind
	}

	literal := p.input[start:p.pos]
	if kind == ast.UnitKind {
		return ast.Value{Kind: ast.UnitKind, Data: literal}, nil
	}

	if strings.Contains(literal, ".") {
		number, err := strconv.ParseFloat(literal, 64)
		if err != nil {
			return ast.Value{}, fmt.Errorf("invalid numeric action parameter")
		}
		return ast.Value{Kind: ast.NumberKind, Data: number}, nil
	}

	integer, err := strconv.ParseInt(literal, 10, 64)
	if err != nil {
		return ast.Value{}, fmt.Errorf("invalid integer action parameter")
	}

	return ast.Value{Kind: ast.IntKind, Data: integer}, nil
}

func (p *actionParser) readQuotedString() (string, error) {
	if !p.consume('"') {
		return "", fmt.Errorf("expected quoted string")
	}

	var builder strings.Builder
	for !p.done() {
		ch := p.input[p.pos]
		p.pos++
		if ch == '"' {
			return builder.String(), nil
		}
		if ch == '\\' {
			if p.done() {
				return "", fmt.Errorf("unterminated escape sequence")
			}
			next := p.input[p.pos]
			p.pos++
			switch next {
			case '"', '\\':
				builder.WriteByte(next)
			case 'n':
				builder.WriteByte('\n')
			case 't':
				builder.WriteByte('\t')
			default:
				return "", fmt.Errorf("invalid escape sequence")
			}
			continue
		}
		builder.WriteByte(ch)
	}

	return "", fmt.Errorf("unterminated quoted string")
}

func (p *actionParser) readIdentifier() (string, error) {
	p.skipSpaces()
	if p.done() || !isActionIdentifierStart(p.input[p.pos]) {
		return "", fmt.Errorf("expected identifier")
	}

	start := p.pos
	for !p.done() && isActionIdentifierPart(p.input[p.pos]) {
		p.pos++
	}

	return p.input[start:p.pos], nil
}

func (p *actionParser) consume(expected byte) bool {
	p.skipSpaces()
	if p.done() || p.input[p.pos] != expected {
		return false
	}
	p.pos++
	return true
}

func (p *actionParser) skipSpaces() {
	for !p.done() {
		switch p.input[p.pos] {
		case ' ', '\t', '\n', '\r':
			p.pos++
		default:
			return
		}
	}
}

func (p *actionParser) done() bool {
	return p.pos >= len(p.input)
}

func isActionIdentifierStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isActionIdentifierPart(ch byte) bool {
	return isActionIdentifierStart(ch) || (ch >= '0' && ch <= '9')
}

func isActionNumberStart(ch byte) bool {
	return ch >= '0' && ch <= '9'
}
