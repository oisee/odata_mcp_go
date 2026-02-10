// Package parser implements the OCL parser
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zmcp/odata-mcp/internal/ocl/ast"
	"github.com/zmcp/odata-mcp/internal/ocl/lexer"
)

// Parser parses OCL source into an AST
type Parser struct {
	l         *lexer.Lexer
	curToken  lexer.Token
	peekToken lexer.Token
	errors    []string
}

// New creates a new Parser
func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}
	// Read two tokens to initialize curToken and peekToken
	p.nextToken()
	p.nextToken()
	return p
}

// Errors returns parsing errors
func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) curTokenIs(t lexer.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t lexer.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t lexer.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t lexer.TokenType) {
	msg := fmt.Sprintf("line %d:%d: expected %s, got %s",
		p.peekToken.Line, p.peekToken.Column, t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) addError(format string, args ...interface{}) {
	msg := fmt.Sprintf("line %d:%d: %s",
		p.curToken.Line, p.curToken.Column, fmt.Sprintf(format, args...))
	p.errors = append(p.errors, msg)
}

// Parse parses the input and returns the AST
func (p *Parser) Parse() *ast.Program {
	program := &ast.Program{Statements: []ast.Statement{}}

	for !p.curTokenIs(lexer.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case lexer.SELECT:
		return p.parseQueryStatement()
	case lexer.FROM:
		return p.parseQueryStatementFromFrom()
	case lexer.WORKFLOW:
		return p.parseWorkflowStatement()
	case lexer.STEP:
		return p.parseStepStatement()
	case lexer.CREATE:
		return p.parseCreateStatement()
	case lexer.UPDATE:
		return p.parseUpdateStatement()
	case lexer.DELETE:
		return p.parseDeleteStatement()
	case lexer.READ:
		return p.parseReadStatement()
	case lexer.CALL:
		return p.parseCallStatement()
	case lexer.IF:
		return p.parseIfStatement()
	case lexer.FOREACH:
		return p.parseForEachStatement()
	case lexer.RETRY:
		return p.parseRetryStatement()
	case lexer.ASSERT:
		return p.parseAssertStatement()
	case lexer.EXPECT:
		return p.parseExpectStatement()
	case lexer.FALLBACK:
		return p.parseFallbackStatement()
	case lexer.ROLLBACK:
		return &ast.RollbackStatement{}
	case lexer.THROW:
		return p.parseThrowStatement()
	case lexer.BEGIN:
		return p.parseBeginTransactionStatement()
	case lexer.COMMIT:
		return &ast.CommitStatement{}
	case lexer.SERVICE:
		return p.parseServiceStatement()
	case lexer.EXPOSE:
		return p.parseExposeStatement()
	case lexer.VARIABLE:
		return p.parseAssignmentStatement()
	default:
		return nil
	}
}

// ============================================================================
// Query Parsing
// ============================================================================

func (p *Parser) parseQueryStatement() *ast.QueryStatement {
	stmt := &ast.QueryStatement{}

	// Parse SELECT fields
	p.nextToken()
	stmt.Fields = p.parseSelectFields()

	// Parse FROM
	if !p.expectPeek(lexer.FROM) {
		return nil
	}
	p.nextToken()
	stmt.From = p.parseSource()

	// Parse optional clauses
	for {
		switch p.peekToken.Type {
		case lexer.JOIN:
			p.nextToken()
			stmt.Joins = append(stmt.Joins, p.parseJoinClause())
		case lexer.WHERE:
			p.nextToken()
			p.nextToken()
			stmt.Where = p.parseExpression(LOWEST)
		case lexer.GROUP:
			p.nextToken()
			if !p.expectPeek(lexer.BY) {
				return nil
			}
			p.nextToken()
			stmt.GroupBy = p.parseExpressionList()
		case lexer.HAVING:
			p.nextToken()
			p.nextToken()
			stmt.Having = p.parseExpression(LOWEST)
		case lexer.ORDER:
			p.nextToken()
			if !p.expectPeek(lexer.BY) {
				return nil
			}
			p.nextToken()
			stmt.OrderBy = p.parseOrderByList()
		case lexer.LIMIT:
			p.nextToken()
			p.nextToken()
			if limit, err := strconv.Atoi(p.curToken.Literal); err == nil {
				stmt.Limit = limit
			}
		case lexer.OFFSET:
			p.nextToken()
			p.nextToken()
			if offset, err := strconv.Atoi(p.curToken.Literal); err == nil {
				stmt.Offset = offset
			}
		default:
			return stmt
		}
	}
}

func (p *Parser) parseQueryStatementFromFrom() *ast.QueryStatement {
	stmt := &ast.QueryStatement{}

	// Parse FROM source first
	p.nextToken()
	stmt.From = p.parseSource()

	// Check for SELECT
	if p.peekTokenIs(lexer.SELECT) {
		p.nextToken()
		p.nextToken()
		stmt.Fields = p.parseSelectFields()
	}

	// Parse optional clauses (same as above)
	for {
		switch p.peekToken.Type {
		case lexer.WHERE:
			p.nextToken()
			p.nextToken()
			stmt.Where = p.parseExpression(LOWEST)
		case lexer.ORDER:
			p.nextToken()
			if !p.expectPeek(lexer.BY) {
				return nil
			}
			p.nextToken()
			stmt.OrderBy = p.parseOrderByList()
		case lexer.LIMIT:
			p.nextToken()
			p.nextToken()
			if limit, err := strconv.Atoi(p.curToken.Literal); err == nil {
				stmt.Limit = limit
			}
		default:
			return stmt
		}
	}
}

func (p *Parser) parseSelectFields() []*ast.SelectField {
	fields := []*ast.SelectField{}

	// Handle SELECT *
	if p.curTokenIs(lexer.MULTIPLY) {
		fields = append(fields, &ast.SelectField{
			Expression: &ast.Identifier{Value: "*"},
		})
		return fields
	}

	for {
		field := &ast.SelectField{}
		field.Expression = p.parseExpression(LOWEST)

		// Check for AS alias
		if p.peekTokenIs(lexer.AS) {
			p.nextToken()
			p.nextToken()
			field.Alias = p.curToken.Literal
		}

		fields = append(fields, field)

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken() // consume comma
		p.nextToken() // move to next field
	}

	return fields
}

func (p *Parser) parseSource() *ast.Source {
	source := &ast.Source{}

	// Parse service.entity or just entity
	source.Entity = p.curToken.Literal

	if p.peekTokenIs(lexer.DOT) {
		source.Service = source.Entity
		p.nextToken() // consume dot
		p.nextToken() // move to entity
		source.Entity = p.curToken.Literal
	}

	// Check for AS alias
	if p.peekTokenIs(lexer.AS) {
		p.nextToken()
		p.nextToken()
		source.Alias = p.curToken.Literal
	}

	return source
}

func (p *Parser) parseJoinClause() *ast.JoinClause {
	join := &ast.JoinClause{Type: "JOIN"}

	p.nextToken()
	join.Source = p.parseSource()

	if !p.expectPeek(lexer.ON) {
		return nil
	}
	p.nextToken()
	join.Condition = p.parseExpression(LOWEST)

	return join
}

func (p *Parser) parseOrderByList() []*ast.OrderByClause {
	clauses := []*ast.OrderByClause{}

	for {
		clause := &ast.OrderByClause{}
		clause.Field = p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.DESC) {
			p.nextToken()
			clause.Descending = true
		} else if p.peekTokenIs(lexer.ASC) {
			p.nextToken()
		}

		clauses = append(clauses, clause)

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
	}

	return clauses
}

// ============================================================================
// Workflow Parsing
// ============================================================================

func (p *Parser) parseWorkflowStatement() *ast.WorkflowStatement {
	stmt := &ast.WorkflowStatement{}

	// WORKFLOW name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// Optional parameters (param1, param2)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		stmt.Parameters = p.parseParameters()
	}

	// Optional RETURNS type
	if p.peekTokenIs(lexer.RETURNS) {
		p.nextToken()
		p.nextToken()
		stmt.Returns = &ast.Identifier{Value: p.curToken.Literal}
	}

	// Parse body until we hit another WORKFLOW or EOF
	stmt.Steps = []ast.Statement{}
	for !p.peekTokenIs(lexer.EOF) && !p.peekTokenIs(lexer.WORKFLOW) {
		p.nextToken()
		step := p.parseStatement()
		if step != nil {
			stmt.Steps = append(stmt.Steps, step)
		}
	}

	return stmt
}

func (p *Parser) parseParameters() []*ast.Parameter {
	params := []*ast.Parameter{}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return params
	}

	p.nextToken()
	for {
		param := &ast.Parameter{}
		param.Name = p.curToken.Literal

		// Optional type annotation: name: Type
		if p.peekTokenIs(lexer.COLON) {
			p.nextToken()
			p.nextToken()
			param.Type = p.curToken.Literal
		}

		// Optional default: name = value
		if p.peekTokenIs(lexer.ASSIGN) {
			p.nextToken()
			p.nextToken()
			param.Default = p.parseExpression(LOWEST)
		}

		params = append(params, param)

		if !p.peekTokenIs(lexer.COMMA) {
			break
		}
		p.nextToken()
		p.nextToken()
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return params
}

func (p *Parser) parseStepStatement() *ast.StepStatement {
	stmt := &ast.StepStatement{}

	// STEP name:
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	if p.peekTokenIs(lexer.COLON) {
		p.nextToken()
	}

	// Parse action
	p.nextToken()
	stmt.Action = p.parseStatement()

	// Parse assertions
	for p.peekTokenIs(lexer.ASSERT) || p.peekTokenIs(lexer.EXPECT) {
		p.nextToken()
		stmt.Assertions = append(stmt.Assertions, p.parseStatement())
	}

	// Parse fallback
	if p.peekTokenIs(lexer.FALLBACK) {
		p.nextToken()
		stmt.Fallback = p.parseStatement()
	}

	return stmt
}

// ============================================================================
// CRUD Operations
// ============================================================================

func (p *Parser) parseCreateStatement() *ast.CreateStatement {
	stmt := &ast.CreateStatement{Fields: make(map[string]ast.Expression)}

	// CREATE entity
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Entity = p.curToken.Literal

	// Check for service.entity
	if p.peekTokenIs(lexer.DOT) {
		stmt.Service = stmt.Entity
		p.nextToken()
		p.nextToken()
		stmt.Entity = p.curToken.Literal
	}

	// SET field = value, ...
	if p.peekTokenIs(lexer.SET) {
		p.nextToken()
		stmt.Fields = p.parseFieldAssignments()
	}

	// INTO $variable
	if p.peekTokenIs(lexer.VARIABLE) {
		p.nextToken()
		stmt.Into = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseUpdateStatement() *ast.UpdateStatement {
	stmt := &ast.UpdateStatement{
		Fields:    make(map[string]ast.Expression),
		KeyFields: make(map[string]ast.Expression),
	}

	// UPDATE entity
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Entity = p.curToken.Literal

	// Check for service.entity
	if p.peekTokenIs(lexer.DOT) {
		stmt.Service = stmt.Entity
		p.nextToken()
		p.nextToken()
		stmt.Entity = p.curToken.Literal
	}

	// Optional key in parentheses: (Key = 'value')
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		stmt.KeyFields = p.parseFieldAssignmentsUntil(lexer.RPAREN)
	}

	// SET field = value, ...
	if p.peekTokenIs(lexer.SET) {
		p.nextToken()
		stmt.Fields = p.parseFieldAssignments()
	}

	// WHERE condition
	if p.peekTokenIs(lexer.WHERE) {
		p.nextToken()
		p.nextToken()
		stmt.Where = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseDeleteStatement() *ast.DeleteStatement {
	stmt := &ast.DeleteStatement{KeyFields: make(map[string]ast.Expression)}

	// DELETE FROM entity or DELETE entity
	if p.peekTokenIs(lexer.FROM) {
		p.nextToken()
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Entity = p.curToken.Literal

	// Check for service.entity
	if p.peekTokenIs(lexer.DOT) {
		stmt.Service = stmt.Entity
		p.nextToken()
		p.nextToken()
		stmt.Entity = p.curToken.Literal
	}

	// Optional key in parentheses
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		stmt.KeyFields = p.parseFieldAssignmentsUntil(lexer.RPAREN)
	}

	// WHERE condition
	if p.peekTokenIs(lexer.WHERE) {
		p.nextToken()
		p.nextToken()
		stmt.Where = p.parseExpression(LOWEST)
	}

	return stmt
}

func (p *Parser) parseReadStatement() *ast.ReadStatement {
	stmt := &ast.ReadStatement{}

	// READ entity
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Entity = p.curToken.Literal

	// Check for service.entity
	if p.peekTokenIs(lexer.DOT) {
		stmt.Service = stmt.Entity
		p.nextToken()
		p.nextToken()
		stmt.Entity = p.curToken.Literal
	}

	// WHERE condition
	if p.peekTokenIs(lexer.WHERE) {
		p.nextToken()
		p.nextToken()
		stmt.Where = p.parseExpression(LOWEST)
	}

	// INTO $variable
	if p.peekTokenIs(lexer.VARIABLE) {
		p.nextToken()
		stmt.Into = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseCallStatement() *ast.CallStatement {
	stmt := &ast.CallStatement{Arguments: make(map[string]ast.Expression)}

	// CALL function or service.function
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Function = p.curToken.Literal

	// Check for service.function
	if p.peekTokenIs(lexer.DOT) {
		stmt.Service = stmt.Function
		p.nextToken()
		p.nextToken()
		stmt.Function = p.curToken.Literal
	}

	// Arguments (arg = value, ...)
	if p.peekTokenIs(lexer.LPAREN) {
		p.nextToken()
		stmt.Arguments = p.parseFieldAssignmentsUntil(lexer.RPAREN)
	}

	// INTO $variable
	if p.peekTokenIs(lexer.VARIABLE) {
		p.nextToken()
		stmt.Into = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseFieldAssignments() map[string]ast.Expression {
	return p.parseFieldAssignmentsUntil(lexer.EOF)
}

func (p *Parser) parseFieldAssignmentsUntil(end lexer.TokenType) map[string]ast.Expression {
	fields := make(map[string]ast.Expression)

	p.nextToken()
	for !p.curTokenIs(end) && !p.curTokenIs(lexer.EOF) && !p.curTokenIs(lexer.WHERE) {
		name := p.curToken.Literal

		if !p.expectPeek(lexer.ASSIGN) && !p.expectPeek(lexer.COLON) {
			break
		}

		p.nextToken()
		fields[name] = p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
		} else if p.peekTokenIs(end) {
			p.nextToken()
			break
		} else {
			break
		}
	}

	return fields
}

// ============================================================================
// Control Flow
// ============================================================================

func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{}

	// IF condition
	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// THEN
	if p.peekTokenIs(lexer.THEN) {
		p.nextToken()
	}

	// Parse consequence until ELSE or ENDIF
	stmt.Consequence = []ast.Statement{}
	for !p.peekTokenIs(lexer.ELSE) && !p.peekTokenIs(lexer.ENDIF) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		s := p.parseStatement()
		if s != nil {
			stmt.Consequence = append(stmt.Consequence, s)
		}
	}

	// ELSE
	if p.peekTokenIs(lexer.ELSE) {
		p.nextToken()
		stmt.Alternative = []ast.Statement{}
		for !p.peekTokenIs(lexer.ENDIF) && !p.peekTokenIs(lexer.EOF) {
			p.nextToken()
			s := p.parseStatement()
			if s != nil {
				stmt.Alternative = append(stmt.Alternative, s)
			}
		}
	}

	// ENDIF
	if p.peekTokenIs(lexer.ENDIF) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseForEachStatement() *ast.ForEachStatement {
	stmt := &ast.ForEachStatement{}

	// FOREACH $variable
	if !p.expectPeek(lexer.VARIABLE) {
		return nil
	}
	stmt.Variable = p.curToken.Literal

	// IN
	if !p.expectPeek(lexer.IN) {
		return nil
	}

	// collection
	p.nextToken()
	stmt.Collection = p.parseExpression(LOWEST)

	// Body until ENDFOR
	stmt.Body = []ast.Statement{}
	for !p.peekTokenIs(lexer.ENDFOR) && !p.peekTokenIs(lexer.EOF) {
		p.nextToken()
		s := p.parseStatement()
		if s != nil {
			stmt.Body = append(stmt.Body, s)
		}
	}

	if p.peekTokenIs(lexer.ENDFOR) {
		p.nextToken()
	}

	return stmt
}

func (p *Parser) parseRetryStatement() *ast.RetryStatement {
	stmt := &ast.RetryStatement{Times: 3}

	// RETRY n TIMES
	if p.peekTokenIs(lexer.NUMBER) {
		p.nextToken()
		if n, err := strconv.Atoi(p.curToken.Literal); err == nil {
			stmt.Times = n
		}
	}

	if p.peekTokenIs(lexer.TIMES) {
		p.nextToken()
	}

	// DELAY duration
	if p.peekTokenIs(lexer.DELAY) {
		p.nextToken()
		p.nextToken()
		stmt.Delay = p.curToken.Literal
	}

	// Body (single statement)
	p.nextToken()
	stmt.Body = p.parseStatement()

	return stmt
}

// ============================================================================
// Assertions & Error Handling
// ============================================================================

func (p *Parser) parseAssertStatement() *ast.AssertStatement {
	stmt := &ast.AssertStatement{}

	p.nextToken()
	stmt.Condition = p.parseExpression(LOWEST)

	// Optional message
	if p.peekTokenIs(lexer.STRING) {
		p.nextToken()
		stmt.Message = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseExpectStatement() *ast.ExpectStatement {
	stmt := &ast.ExpectStatement{}

	p.nextToken()
	stmt.Field = p.parseExpression(LOWEST)

	// Operator
	p.nextToken()
	stmt.Operator = p.curToken.Literal

	// Value
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

func (p *Parser) parseFallbackStatement() *ast.FallbackStatement {
	stmt := &ast.FallbackStatement{}

	// FALLBACK => action
	if p.peekTokenIs(lexer.FATARROW) {
		p.nextToken()
	}

	p.nextToken()
	stmt.Action = p.parseStatement()

	return stmt
}

func (p *Parser) parseThrowStatement() *ast.ThrowStatement {
	stmt := &ast.ThrowStatement{}

	p.nextToken()
	stmt.Message = p.parseExpression(LOWEST)

	return stmt
}

// ============================================================================
// Transactions
// ============================================================================

func (p *Parser) parseBeginTransactionStatement() *ast.BeginTransactionStatement {
	stmt := &ast.BeginTransactionStatement{}

	if p.peekTokenIs(lexer.TRANSACTION) {
		p.nextToken()
	}

	if p.peekTokenIs(lexer.IDENT) || p.peekTokenIs(lexer.STRING) {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	}

	return stmt
}

// ============================================================================
// Service & Exposure
// ============================================================================

func (p *Parser) parseServiceStatement() *ast.ServiceStatement {
	stmt := &ast.ServiceStatement{Properties: make(map[string]ast.Expression)}

	// SERVICE name
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Name = p.curToken.Literal

	// URL or properties
	if p.peekTokenIs(lexer.STRING) {
		p.nextToken()
		stmt.URL = p.curToken.Literal
	}

	// Optional WITH properties
	if p.peekTokenIs(lexer.WITH) {
		p.nextToken()
		if p.peekTokenIs(lexer.LBRACE) {
			p.nextToken()
			stmt.Properties = p.parseFieldAssignmentsUntil(lexer.RBRACE)
		}
	}

	return stmt
}

func (p *Parser) parseExposeStatement() *ast.ExposeStatement {
	stmt := &ast.ExposeStatement{}

	// EXPOSE workflow
	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.Workflow = p.curToken.Literal

	// AS odata/mcp
	if !p.expectPeek(lexer.AS) {
		return nil
	}

	if !p.expectPeek(lexer.IDENT) {
		return nil
	}
	stmt.As = strings.ToLower(p.curToken.Literal)

	// Optional name
	if p.peekTokenIs(lexer.IDENT) {
		p.nextToken()
		stmt.Name = p.curToken.Literal
	}

	return stmt
}

func (p *Parser) parseAssignmentStatement() *ast.AssignmentStatement {
	stmt := &ast.AssignmentStatement{}
	stmt.Variable = p.curToken.Literal

	if !p.expectPeek(lexer.ASSIGN) {
		return nil
	}

	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)

	return stmt
}

// ============================================================================
// Expression Parsing (Pratt Parser)
// ============================================================================

const (
	_ int = iota
	LOWEST
	OR_PREC
	AND_PREC
	EQUALS
	LESSGREATER
	SUM
	PRODUCT
	PREFIX
	CALL
	INDEX
)

var precedences = map[lexer.TokenType]int{
	lexer.OR:       OR_PREC,
	lexer.AND:      AND_PREC,
	lexer.EQ:       EQUALS,
	lexer.NEQ:      EQUALS,
	lexer.LT:       LESSGREATER,
	lexer.GT:       LESSGREATER,
	lexer.LTE:      LESSGREATER,
	lexer.GTE:      LESSGREATER,
	lexer.PLUS:     SUM,
	lexer.MINUS:    SUM,
	lexer.MULTIPLY: PRODUCT,
	lexer.DIVIDE:   PRODUCT,
	lexer.LPAREN:   CALL,
	lexer.DOT:      INDEX,
	lexer.LBRACKET: INDEX,
}

func (p *Parser) peekPrecedence() int {
	if prec, ok := precedences[p.peekToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if prec, ok := precedences[p.curToken.Type]; ok {
		return prec
	}
	return LOWEST
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	var left ast.Expression

	// Prefix parsing
	switch p.curToken.Type {
	case lexer.IDENT:
		left = &ast.Identifier{Value: p.curToken.Literal}
	case lexer.STRING:
		left = &ast.StringLiteral{Value: p.curToken.Literal}
	case lexer.NUMBER:
		left = &ast.NumberLiteral{Value: p.curToken.Literal}
	case lexer.TRUE:
		left = &ast.BooleanLiteral{Value: true}
	case lexer.FALSE:
		left = &ast.BooleanLiteral{Value: false}
	case lexer.NULL:
		left = &ast.NullLiteral{}
	case lexer.VARIABLE:
		left = &ast.Variable{Name: p.curToken.Literal}
	case lexer.NOT:
		p.nextToken()
		left = &ast.UnaryExpression{Operator: "NOT", Right: p.parseExpression(PREFIX)}
	case lexer.MINUS:
		p.nextToken()
		left = &ast.UnaryExpression{Operator: "-", Right: p.parseExpression(PREFIX)}
	case lexer.LPAREN:
		p.nextToken()
		left = p.parseExpression(LOWEST)
		if !p.expectPeek(lexer.RPAREN) {
			return nil
		}
	case lexer.LBRACKET:
		left = p.parseArrayLiteral()
	case lexer.LBRACE:
		left = p.parseObjectLiteral()
	case lexer.COUNT, lexer.SUM, lexer.AVG, lexer.MIN, lexer.MAX:
		left = p.parseFunctionCall()
	default:
		p.addError("no prefix parse function for %s", p.curToken.Type)
		return nil
	}

	// Infix parsing
	for !p.peekTokenIs(lexer.EOF) && precedence < p.peekPrecedence() {
		switch p.peekToken.Type {
		case lexer.PLUS, lexer.MINUS, lexer.MULTIPLY, lexer.DIVIDE,
			lexer.EQ, lexer.NEQ, lexer.LT, lexer.GT, lexer.LTE, lexer.GTE,
			lexer.AND, lexer.OR:
			p.nextToken()
			left = p.parseInfixExpression(left)
		case lexer.DOT:
			p.nextToken()
			left = p.parseFieldAccess(left)
		case lexer.LPAREN:
			p.nextToken()
			left = p.parseCallExpression(left)
		case lexer.LBRACKET:
			p.nextToken()
			left = p.parseIndexExpression(left)
		default:
			return left
		}
	}

	return left
}

func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	expr := &ast.BinaryExpression{
		Left:     left,
		Operator: p.curToken.Literal,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)

	return expr
}

func (p *Parser) parseFieldAccess(left ast.Expression) ast.Expression {
	p.nextToken()
	return &ast.FieldAccess{
		Object: left,
		Field:  &ast.Identifier{Value: p.curToken.Literal},
	}
}

func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	var funcName string
	if ident, ok := function.(*ast.Identifier); ok {
		funcName = ident.Value
	}

	call := &ast.FunctionCall{
		Function:  funcName,
		Arguments: []ast.Expression{},
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return call
	}

	p.nextToken()
	call.Arguments = append(call.Arguments, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		call.Arguments = append(call.Arguments, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return call
}

func (p *Parser) parseIndexExpression(left ast.Expression) ast.Expression {
	p.nextToken()
	index := p.parseExpression(LOWEST)

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return &ast.BinaryExpression{
		Left:     left,
		Operator: "[]",
		Right:    index,
	}
}

func (p *Parser) parseFunctionCall() ast.Expression {
	call := &ast.FunctionCall{
		Function:  p.curToken.Literal,
		Arguments: []ast.Expression{},
	}

	if !p.expectPeek(lexer.LPAREN) {
		return nil
	}

	if p.peekTokenIs(lexer.RPAREN) {
		p.nextToken()
		return call
	}

	p.nextToken()
	call.Arguments = append(call.Arguments, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		call.Arguments = append(call.Arguments, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.RPAREN) {
		return nil
	}

	return call
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	arr := &ast.ArrayLiteral{Elements: []ast.Expression{}}

	if p.peekTokenIs(lexer.RBRACKET) {
		p.nextToken()
		return arr
	}

	p.nextToken()
	arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		arr.Elements = append(arr.Elements, p.parseExpression(LOWEST))
	}

	if !p.expectPeek(lexer.RBRACKET) {
		return nil
	}

	return arr
}

func (p *Parser) parseObjectLiteral() ast.Expression {
	obj := &ast.ObjectLiteral{Pairs: make(map[string]ast.Expression)}

	if p.peekTokenIs(lexer.RBRACE) {
		p.nextToken()
		return obj
	}

	p.nextToken()
	for !p.curTokenIs(lexer.RBRACE) && !p.curTokenIs(lexer.EOF) {
		key := p.curToken.Literal

		if !p.expectPeek(lexer.COLON) && !p.expectPeek(lexer.ASSIGN) {
			return nil
		}

		p.nextToken()
		obj.Pairs[key] = p.parseExpression(LOWEST)

		if p.peekTokenIs(lexer.COMMA) {
			p.nextToken()
			p.nextToken()
		} else if p.peekTokenIs(lexer.RBRACE) {
			p.nextToken()
			break
		} else {
			break
		}
	}

	return obj
}

func (p *Parser) parseExpressionList() []ast.Expression {
	exprs := []ast.Expression{}

	exprs = append(exprs, p.parseExpression(LOWEST))

	for p.peekTokenIs(lexer.COMMA) {
		p.nextToken()
		p.nextToken()
		exprs = append(exprs, p.parseExpression(LOWEST))
	}

	return exprs
}
