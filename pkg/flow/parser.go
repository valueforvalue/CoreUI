package flow

import "fmt"

// Parser parses a CoreFlow (.flow) source string into a FlowDocument.
type Parser struct {
	lexer   *flowLexer
	current token
	peek    token
}

// NewParser returns a Parser ready to parse src.
func NewParser(src string) *Parser {
	l := newLexer(src)
	p := &Parser{lexer: l}
	p.current = l.Next()
	p.peek = l.Next()
	return p
}

// ParseDocument parses the full .flow source and returns the document AST.
func (p *Parser) ParseDocument() (*FlowDocument, error) {
	if p.current.Type == ttIllegal {
		return nil, p.illegalErr()
	}

	doc := &FlowDocument{}

	for p.current.Type != ttEOF {
		if p.current.Type == ttIllegal {
			return nil, p.illegalErr()
		}

		if p.current.Type != ttIdent {
			return nil, p.errorf("expected block keyword (State, On, Compute), got %q", p.current.Literal)
		}

		switch p.current.Literal {
		case "State":
			if doc.State != nil {
				return nil, p.errorf("only one State block is allowed per .flow file")
			}
			block, err := p.parseStateBlock()
			if err != nil {
				return nil, err
			}
			doc.State = block
		case "On":
			block, err := p.parseOnBlock()
			if err != nil {
				return nil, err
			}
			doc.OnBlocks = append(doc.OnBlocks, block)
		case "Compute":
			block, err := p.parseComputeBlock()
			if err != nil {
				return nil, err
			}
			doc.Computes = append(doc.Computes, block)
		default:
			return nil, p.errorf("unknown block type %q; expected State, On, or Compute", p.current.Literal)
		}
	}

	return doc, nil
}

// parseStateBlock parses: State { VarDecl* }
func (p *Parser) parseStateBlock() (*StateBlock, error) {
	block := &StateBlock{Line: p.current.Line, Col: p.current.Col}
	p.advance() // consume "State"

	if err := p.expectAdvance(ttLBrace, "expected '{' after State"); err != nil {
		return nil, err
	}

	for p.current.Type != ttRBrace {
		if p.current.Type == ttEOF {
			return nil, p.errorf("unterminated State block")
		}
		if p.current.Type == ttIllegal {
			return nil, p.illegalErr()
		}

		decl, err := p.parseVarDecl()
		if err != nil {
			return nil, err
		}
		block.Vars = append(block.Vars, decl)
	}

	p.advance() // consume "}"
	return block, nil
}

// parseVarDecl parses a variable declaration:
//   - var IDENT = Expr
//   - list IDENT            (initial value is an implicit empty list)
//   - map IDENT             (initial value is an implicit empty map)
func (p *Parser) parseVarDecl() (*VarDecl, error) {
	line, col := p.current.Line, p.current.Col

	var kind VarKind
	switch p.current.Literal {
	case "var":
		kind = VarKindVar
	case "list":
		kind = VarKindList
	case "map":
		kind = VarKindMap
	default:
		return nil, p.errorf("expected var, list, or map; got %q", p.current.Literal)
	}
	p.advance()

	if p.current.Type != ttIdent {
		return nil, p.errorf("expected variable name after %s", kind)
	}
	name := p.current.Literal
	p.advance()

	// list and map declarations do not require an initial value expression.
	if kind == VarKindList || kind == VarKindMap {
		return &VarDecl{Line: line, Col: col, Kind: kind, Name: name, Init: Expr{}}, nil
	}

	if err := p.expectAdvance(ttEq, fmt.Sprintf("expected '=' after variable name %q", name)); err != nil {
		return nil, err
	}

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}

	return &VarDecl{Line: line, Col: col, Kind: kind, Name: name, Init: expr}, nil
}

// parseOnBlock parses: On(id="...", event="...") { Statement* }
func (p *Parser) parseOnBlock() (*OnBlock, error) {
	block := &OnBlock{Line: p.current.Line, Col: p.current.Col}
	p.advance() // consume "On"

	if err := p.expectAdvance(ttLParen, "expected '(' after On"); err != nil {
		return nil, err
	}

	targetID, event, err := p.parseOnParams()
	if err != nil {
		return nil, err
	}
	block.TargetID = targetID
	block.Event = event

	if err := p.expectAdvance(ttRParen, "expected ')' after On parameters"); err != nil {
		return nil, err
	}

	if err := p.expectAdvance(ttLBrace, "expected '{' after On(...)"); err != nil {
		return nil, err
	}

	for p.current.Type != ttRBrace {
		if p.current.Type == ttEOF {
			return nil, p.errorf("unterminated On block")
		}
		if p.current.Type == ttIllegal {
			return nil, p.illegalErr()
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return nil, err
		}
		block.Statements = append(block.Statements, stmt)
	}

	p.advance() // consume "}"
	return block, nil
}

// parseOnParams parses: id="...", event="..."
func (p *Parser) parseOnParams() (targetID, event string, err error) {
	for i := 0; i < 2; i++ {
		if p.current.Type != ttIdent {
			return "", "", p.errorf("expected parameter name in On(...)")
		}
		key := p.current.Literal
		p.advance()

		if err := p.expectAdvance(ttEq, fmt.Sprintf("expected '=' after %q", key)); err != nil {
			return "", "", err
		}

		if p.current.Type != ttString {
			return "", "", p.errorf("expected string value for %q", key)
		}
		val := p.current.Literal
		p.advance()

		switch key {
		case "id":
			targetID = val
		case "event":
			event = val
		default:
			return "", "", p.errorf("unknown On parameter %q; expected id or event", key)
		}

		if p.current.Type == ttComma {
			p.advance()
		}
	}

	if targetID == "" {
		return "", "", p.errorf("On block requires id parameter")
	}
	if event == "" {
		return "", "", p.errorf("On block requires event parameter")
	}
	return targetID, event, nil
}

// parseComputeBlock parses: Compute(target="...") { Expr }
func (p *Parser) parseComputeBlock() (*ComputeBlock, error) {
	block := &ComputeBlock{Line: p.current.Line, Col: p.current.Col}
	p.advance() // consume "Compute"

	if err := p.expectAdvance(ttLParen, "expected '(' after Compute"); err != nil {
		return nil, err
	}

	if p.current.Type != ttIdent || p.current.Literal != "target" {
		return nil, p.errorf("expected target= in Compute(...)")
	}
	p.advance()

	if err := p.expectAdvance(ttEq, "expected '=' after target"); err != nil {
		return nil, err
	}

	if p.current.Type != ttString {
		return nil, p.errorf("expected string value for Compute target")
	}
	block.Target = p.current.Literal
	p.advance()

	if err := p.expectAdvance(ttRParen, "expected ')' after Compute parameters"); err != nil {
		return nil, err
	}

	if err := p.expectAdvance(ttLBrace, "expected '{' after Compute(...)"); err != nil {
		return nil, err
	}

	expr, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	block.Expr = expr

	if err := p.expectAdvance(ttRBrace, "expected '}' to close Compute block"); err != nil {
		return nil, err
	}

	return block, nil
}

// parseStatement parses a single logic statement.
func (p *Parser) parseStatement() (Statement, error) {
	line, col := p.current.Line, p.current.Col

	if p.current.Type != ttIdent {
		return Statement{}, p.errorf("expected statement keyword (set, add, toggle, if, call_service)")
	}

	switch p.current.Literal {
	case "set":
		return p.parseSetStmt(line, col)
	case "add":
		return p.parseAddStmt(line, col)
	case "toggle":
		return p.parseToggleStmt(line, col)
	case "if":
		return p.parseIfStmt(line, col)
	case "call_service":
		return p.parseCallServiceStmt(line, col)
	default:
		return Statement{}, p.errorf("unknown statement %q; expected set, add, toggle, if, or call_service", p.current.Literal)
	}
}

// parseSetStmt parses: "set" IDENT "=" Expr
func (p *Parser) parseSetStmt(line, col int) (Statement, error) {
	p.advance() // consume "set"

	if p.current.Type != ttIdent {
		return Statement{}, p.errorf("expected variable name after set")
	}
	varName := p.current.Literal
	p.advance()

	if err := p.expectAdvance(ttEq, fmt.Sprintf("expected '=' after variable name %q in set statement", varName)); err != nil {
		return Statement{}, err
	}

	expr, err := p.parseExpr()
	if err != nil {
		return Statement{}, err
	}

	return Statement{Line: line, Col: col, Kind: StmtSet, VarName: varName, Value: expr}, nil
}

// parseAddStmt parses: "add" IDENT Expr
func (p *Parser) parseAddStmt(line, col int) (Statement, error) {
	p.advance() // consume "add"

	if p.current.Type != ttIdent {
		return Statement{}, p.errorf("expected variable name after add")
	}
	varName := p.current.Literal
	p.advance()

	amount, err := p.parseExpr()
	if err != nil {
		return Statement{}, err
	}

	return Statement{Line: line, Col: col, Kind: StmtAdd, VarName: varName, Amount: amount}, nil
}

// parseToggleStmt parses: "toggle" IDENT
func (p *Parser) parseToggleStmt(line, col int) (Statement, error) {
	p.advance() // consume "toggle"

	if p.current.Type != ttIdent {
		return Statement{}, p.errorf("expected variable name after toggle")
	}
	varName := p.current.Literal
	p.advance()

	return Statement{Line: line, Col: col, Kind: StmtToggle, VarName: varName}, nil
}

// parseIfStmt parses: "if" Condition "{" Statement* "}" ("else" "{" Statement* "}")?
func (p *Parser) parseIfStmt(line, col int) (Statement, error) {
	p.advance() // consume "if"

	cond, err := p.parseCondition()
	if err != nil {
		return Statement{}, err
	}

	if err := p.expectAdvance(ttLBrace, "expected '{' after if condition"); err != nil {
		return Statement{}, err
	}

	var thenStmts []Statement
	for p.current.Type != ttRBrace {
		if p.current.Type == ttEOF {
			return Statement{}, p.errorf("unterminated if block")
		}
		stmt, err := p.parseStatement()
		if err != nil {
			return Statement{}, err
		}
		thenStmts = append(thenStmts, stmt)
	}
	p.advance() // consume "}"

	var elseStmts []Statement
	if p.current.Type == ttIdent && p.current.Literal == "else" {
		p.advance() // consume "else"

		if err := p.expectAdvance(ttLBrace, "expected '{' after else"); err != nil {
			return Statement{}, err
		}

		for p.current.Type != ttRBrace {
			if p.current.Type == ttEOF {
				return Statement{}, p.errorf("unterminated else block")
			}
			stmt, err := p.parseStatement()
			if err != nil {
				return Statement{}, err
			}
			elseStmts = append(elseStmts, stmt)
		}
		p.advance() // consume "}"
	}

	return Statement{
		Line:      line,
		Col:       col,
		Kind:      StmtIf,
		Condition: cond,
		Then:      thenStmts,
		Else:      elseStmts,
	}, nil
}

// parseCallServiceStmt parses: "call_service" STRING ("(" ParamList? ")")?
func (p *Parser) parseCallServiceStmt(line, col int) (Statement, error) {
	p.advance() // consume "call_service"

	if p.current.Type != ttString {
		return Statement{}, p.errorf("expected service endpoint string after call_service")
	}
	service := p.current.Literal
	p.advance()

	stmt := Statement{Line: line, Col: col, Kind: StmtService, Service: service}

	if p.current.Type == ttLParen {
		p.advance() // consume "("

		params := map[string]Expr{}
		for p.current.Type != ttRParen {
			if p.current.Type == ttEOF {
				return Statement{}, p.errorf("unterminated call_service parameter list")
			}
			if p.current.Type != ttIdent {
				return Statement{}, p.errorf("expected parameter name in call_service")
			}
			key := p.current.Literal
			p.advance()

			if err := p.expectAdvance(ttEq, fmt.Sprintf("expected '=' after parameter name %q", key)); err != nil {
				return Statement{}, err
			}

			expr, err := p.parseExpr()
			if err != nil {
				return Statement{}, err
			}
			params[key] = expr

			if p.current.Type == ttComma {
				p.advance()
			}
		}
		p.advance() // consume ")"
		stmt.Params = params
	}

	return stmt, nil
}

// parseCondition parses: Expr CompOp Expr
// where CompOp is one of: ==  !=  >  <  >=  <=
func (p *Parser) parseCondition() (Condition, error) {
	left, err := p.parseSingleExprToken()
	if err != nil {
		return Condition{}, err
	}

	op, err := p.parseCompOp()
	if err != nil {
		return Condition{}, err
	}

	right, err := p.parseSingleExprToken()
	if err != nil {
		return Condition{}, err
	}

	return Condition{Left: Expr{left}, Op: op, Right: Expr{right}}, nil
}

func (p *Parser) parseCompOp() (string, error) {
	switch p.current.Type {
	case ttEqEq, ttNeq, ttGt, ttLt, ttGte, ttLte:
		op := p.current.Literal
		p.advance()
		return op, nil
	default:
		return "", p.errorf("expected comparison operator (==, !=, >, <, >=, <=), got %q", p.current.Literal)
	}
}

// parseExpr parses a flat expression: Atom (Op Atom)*
// Atoms are: identifier, string literal, integer, float, bool literal.
func (p *Parser) parseExpr() (Expr, error) {
	first, err := p.parseSingleExprToken()
	if err != nil {
		return nil, err
	}
	expr := Expr{first}

	for p.isExprOp() {
		op := ExprToken{Kind: TokOp, Val: p.current.Literal}
		p.advance()
		right, err := p.parseSingleExprToken()
		if err != nil {
			return nil, err
		}
		expr = append(expr, op, right)
	}

	return expr, nil
}

// parseSingleExprToken parses one atom: IDENT | STRING | INT | FLOAT | BOOL.
func (p *Parser) parseSingleExprToken() (ExprToken, error) {
	switch p.current.Type {
	case ttIdent:
		tok := ExprToken{Kind: TokIdent, Val: p.current.Literal}
		p.advance()
		return tok, nil
	case ttString:
		tok := ExprToken{Kind: TokStr, Val: p.current.Literal}
		p.advance()
		return tok, nil
	case ttInt:
		tok := ExprToken{Kind: TokInt, Val: p.current.Literal}
		p.advance()
		return tok, nil
	case ttFloat:
		tok := ExprToken{Kind: TokFloat, Val: p.current.Literal}
		p.advance()
		return tok, nil
	case ttTrue:
		tok := ExprToken{Kind: TokBool, Val: "true"}
		p.advance()
		return tok, nil
	case ttFalse:
		tok := ExprToken{Kind: TokBool, Val: "false"}
		p.advance()
		return tok, nil
	default:
		return ExprToken{}, p.errorf("expected expression value, got %q", p.current.Literal)
	}
}

// isExprOp reports whether the current token is an arithmetic or concat operator.
func (p *Parser) isExprOp() bool {
	switch p.current.Type {
	case ttPlus, ttMinus, ttStar, ttSlash:
		return true
	}
	return false
}

// expectAdvance expects the current token to have the given type and advances.
func (p *Parser) expectAdvance(tt TokenType, msg string) error {
	if p.current.Type == ttIllegal {
		return p.illegalErr()
	}
	if p.current.Type != tt {
		return p.errorf("%s", msg)
	}
	p.advance()
	return nil
}

func (p *Parser) advance() {
	p.current = p.peek
	p.peek = p.lexer.Next()
}

func (p *Parser) errorf(format string, args ...any) error {
	return fmt.Errorf("[Flow Error] Line %d, Col %d: %s",
		p.current.Line, p.current.Col, fmt.Sprintf(format, args...))
}

func (p *Parser) illegalErr() error {
	return fmt.Errorf("[Flow Error] Line %d, Col %d: %s",
		p.current.Line, p.current.Col, p.current.Literal)
}

// ParseFlow is the primary entry point for parsing a CoreFlow source string.
func ParseFlow(src string) (*FlowDocument, error) {
	return NewParser(src).ParseDocument()
}
