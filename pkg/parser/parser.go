package parser

import (
	"strconv"
	"strings"
	"unicode"

	"coreui/pkg/ast"
	"coreui/pkg/diag"
	"coreui/pkg/lexer"
	"coreui/pkg/registry"
)

type Parser struct {
	source  string
	lexer   *lexer.Lexer
	current lexer.Token
	peek    lexer.Token
	ids     map[string]bool
}

func New(source string) *Parser {
	lex := lexer.New(source)
	parser := &Parser{
		source: source,
		lexer:  lex,
		ids:    map[string]bool{},
	}
	parser.current = lex.NextToken()
	parser.peek = lex.NextToken()
	return parser
}

func (p *Parser) Parse() (*ast.Node, error) {
	if p.current.Type == lexer.Illegal {
		return nil, diag.New(p.current.Line, p.current.Col, p.current.Literal)
	}

	node, err := p.parseComponent()
	if err != nil {
		return nil, err
	}

	if p.current.Type != lexer.EOF {
		return nil, diag.Newf(p.current.Line, p.current.Col, "unexpected token %q", p.current.Literal)
	}

	return node, nil
}

func (p *Parser) parseComponent() (*ast.Node, error) {
	if p.current.Type == lexer.Illegal {
		return nil, diag.New(p.current.Line, p.current.Col, p.current.Literal)
	}
	if p.current.Type != lexer.Identifier || !startsWithUpper(p.current.Literal) {
		return nil, diag.New(p.current.Line, p.current.Col, "expected component type")
	}

	componentToken := p.current
	componentType := componentToken.Literal
	if _, ok := registry.GetComponent(componentType); !ok {
		return nil, diag.Newf(componentToken.Line, componentToken.Col, "unknown component %q", componentType)
	}

	p.advance()
	if _, err := p.expect(lexer.LParen, "expected '(' after component type"); err != nil {
		return nil, err
	}
	p.advance()

	attributes := map[string]ast.Value{}
	if p.current.Type != lexer.RParen {
		for {
			key, value, err := p.parseAttribute(componentType)
			if err != nil {
				return nil, err
			}
			if _, exists := attributes[key]; exists {
				return nil, diag.Newf(p.current.Line, p.current.Col, "duplicate attribute %q on %s", key, componentType)
			}
			attributes[key] = value

			if p.current.Type == lexer.Comma {
				p.advance()
				continue
			}
			break
		}
	}

	if _, err := p.expect(lexer.RParen, "expected ')' after attributes"); err != nil {
		return nil, err
	}
	p.advance()

	if err := p.validateRequiredAttributes(componentType, attributes, componentToken.Line, componentToken.Col); err != nil {
		return nil, err
	}
	if err := p.registerID(attributes["id"], componentToken.Line, componentToken.Col); err != nil {
		return nil, err
	}

	node := &ast.Node{
		Type:       componentType,
		Attributes: attributes,
		Position: ast.Position{
			Line: componentToken.Line,
			Col:  componentToken.Col,
		},
	}

	if p.current.Type == lexer.LBrace {
		p.advance()
		for p.current.Type != lexer.RBrace {
			if p.current.Type == lexer.EOF {
				return nil, diag.New(componentToken.Line, componentToken.Col, "unterminated children block")
			}
			child, err := p.parseComponent()
			if err != nil {
				return nil, err
			}
			node.Children = append(node.Children, child)
		}
		p.advance()
	}

	return node, nil
}

func (p *Parser) parseAttribute(componentType string) (string, ast.Value, error) {
	if p.current.Type == lexer.Illegal {
		return "", ast.Value{}, diag.New(p.current.Line, p.current.Col, p.current.Literal)
	}
	if p.current.Type != lexer.Identifier || startsWithUpper(p.current.Literal) {
		return "", ast.Value{}, diag.New(p.current.Line, p.current.Col, "expected attribute name")
	}

	keyToken := p.current
	key := keyToken.Literal

	if !registry.IsAttributeAllowed(componentType, key) {
		return "", ast.Value{}, diag.Newf(keyToken.Line, keyToken.Col, "unknown attribute %q for %s", key, componentType)
	}

	p.advance()
	if _, err := p.expect(lexer.Equal, "expected '=' after attribute name"); err != nil {
		return "", ast.Value{}, err
	}
	p.advance()

	var value ast.Value
	var err error
	if key == "action" {
		value, err = p.parseActionValue(keyToken.Line, keyToken.Col)
	} else {
		value, err = p.parseValue()
	}
	if err != nil {
		return "", ast.Value{}, err
	}

	if err := registry.ValidateValue(componentType, key, value); err != nil {
		return "", ast.Value{}, diag.New(keyToken.Line, keyToken.Col, err.Error())
	}

	return key, value, nil
}

func (p *Parser) parseActionValue(line, col int) (ast.Value, error) {
	if p.current.Type == lexer.String {
		raw := p.current.Literal
		p.advance()
		action, err := parseAction(raw)
		if err != nil {
			return ast.Value{}, diag.New(line, col, err.Error())
		}
		return ast.Value{Kind: ast.ActionKind, Data: action}, nil
	}

	if p.current.Type == lexer.Comma || p.current.Type == lexer.RParen || p.current.Type == lexer.EOF {
		return ast.Value{}, diag.New(line, col, "missing action value")
	}

	start := p.current.Offset
	depthParen := 0
	depthBracket := 0

	for {
		if p.current.Type == lexer.EOF {
			break
		}

		switch p.current.Type {
		case lexer.LParen:
			depthParen++
		case lexer.RParen:
			if depthParen == 0 && depthBracket == 0 {
				raw := strings.TrimSpace(p.source[start:p.current.Offset])
				if raw == "" {
					return ast.Value{}, diag.New(line, col, "missing action value")
				}
				action, err := parseAction(raw)
				if err != nil {
					return ast.Value{}, diag.New(line, col, err.Error())
				}
				return ast.Value{Kind: ast.ActionKind, Data: action}, nil
			}
			depthParen--
		case lexer.LBracket:
			depthBracket++
		case lexer.RBracket:
			depthBracket--
		case lexer.Comma:
			if depthParen == 0 && depthBracket == 0 {
				raw := strings.TrimSpace(p.source[start:p.current.Offset])
				if raw == "" {
					return ast.Value{}, diag.New(line, col, "missing action value")
				}
				action, err := parseAction(raw)
				if err != nil {
					return ast.Value{}, diag.New(line, col, err.Error())
				}
				return ast.Value{Kind: ast.ActionKind, Data: action}, nil
			}
		}

		p.advance()
	}

	raw := strings.TrimSpace(p.source[start:p.current.Offset])
	if raw == "" {
		return ast.Value{}, diag.New(line, col, "missing action value")
	}

	action, err := parseAction(raw)
	if err != nil {
		return ast.Value{}, diag.New(line, col, err.Error())
	}

	return ast.Value{Kind: ast.ActionKind, Data: action}, nil
}

func (p *Parser) parseValue() (ast.Value, error) {
	if p.current.Type == lexer.Illegal {
		return ast.Value{}, diag.New(p.current.Line, p.current.Col, p.current.Literal)
	}

	switch p.current.Type {
	case lexer.String:
		value := ast.Value{Kind: ast.StringKind, Data: p.current.Literal}
		p.advance()
		return value, nil
	case lexer.Boolean:
		value := ast.Value{Kind: ast.BoolKind, Data: p.current.Literal == "true"}
		p.advance()
		return value, nil
	case lexer.Unit:
		value := ast.Value{Kind: ast.UnitKind, Data: p.current.Literal}
		p.advance()
		return value, nil
	case lexer.Number:
		literal := p.current.Literal
		p.advance()
		if strings.Contains(literal, ".") {
			number, err := strconv.ParseFloat(literal, 64)
			if err != nil {
				return ast.Value{}, diag.Newf(p.current.Line, p.current.Col, "invalid number %q", literal)
			}
			return ast.Value{Kind: ast.NumberKind, Data: number}, nil
		}
		integer, err := strconv.ParseInt(literal, 10, 64)
		if err != nil {
			return ast.Value{}, diag.Newf(p.current.Line, p.current.Col, "invalid integer %q", literal)
		}
		return ast.Value{Kind: ast.IntKind, Data: integer}, nil
	case lexer.LBracket:
		return p.parseArray()
	default:
		return ast.Value{}, diag.New(p.current.Line, p.current.Col, "expected attribute value")
	}
}

func (p *Parser) parseArray() (ast.Value, error) {
	p.advance()
	values := []ast.Value{}

	if p.current.Type != lexer.RBracket {
		for {
			value, err := p.parseValue()
			if err != nil {
				return ast.Value{}, err
			}
			values = append(values, value)

			if p.current.Type == lexer.Comma {
				p.advance()
				continue
			}
			break
		}
	}

	if _, err := p.expect(lexer.RBracket, "expected ']' after array"); err != nil {
		return ast.Value{}, err
	}
	p.advance()

	return ast.Value{Kind: ast.ArrayKind, Data: values}, nil
}

func (p *Parser) validateRequiredAttributes(component string, attributes map[string]ast.Value, line, col int) error {
	for _, key := range registry.RequiredAttributes(component) {
		if _, ok := attributes[key]; !ok {
			return diag.New(line, col, "Duplicate/Missing ID")
		}
	}
	return nil
}

func (p *Parser) registerID(value ast.Value, line, col int) error {
	id, ok := value.Data.(string)
	if !ok || id == "" {
		return diag.New(line, col, "Duplicate/Missing ID")
	}
	if p.ids[id] {
		return diag.New(line, col, "Duplicate/Missing ID")
	}
	p.ids[id] = true
	return nil
}

func (p *Parser) expect(expected lexer.TokenType, message string) (lexer.Token, error) {
	if p.current.Type == lexer.Illegal {
		return lexer.Token{}, diag.New(p.current.Line, p.current.Col, p.current.Literal)
	}
	if p.current.Type != expected {
		return lexer.Token{}, diag.New(p.current.Line, p.current.Col, message)
	}
	return p.current, nil
}

func (p *Parser) advance() {
	p.current = p.peek
	p.peek = p.lexer.NextToken()
}

func startsWithUpper(value string) bool {
	if value == "" {
		return false
	}
	return unicode.IsUpper(rune(value[0]))
}
