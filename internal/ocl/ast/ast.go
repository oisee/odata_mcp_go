// Package ast defines the Abstract Syntax Tree for OData Composer Language
package ast

// Node is the base interface for all AST nodes
type Node interface {
	TokenLiteral() string
}

// Statement represents a statement node
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of every OCL program
type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// ============================================================================
// Literals
// ============================================================================

// Identifier represents a name (field, entity, service, etc.)
type Identifier struct {
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Value }

// StringLiteral represents a string value
type StringLiteral struct {
	Value string
}

func (s *StringLiteral) expressionNode()      {}
func (s *StringLiteral) TokenLiteral() string { return s.Value }

// NumberLiteral represents a numeric value
type NumberLiteral struct {
	Value string // Keep as string to preserve precision
}

func (n *NumberLiteral) expressionNode()      {}
func (n *NumberLiteral) TokenLiteral() string { return n.Value }

// BooleanLiteral represents TRUE or FALSE
type BooleanLiteral struct {
	Value bool
}

func (b *BooleanLiteral) expressionNode()      {}
func (b *BooleanLiteral) TokenLiteral() string {
	if b.Value {
		return "TRUE"
	}
	return "FALSE"
}

// NullLiteral represents NULL
type NullLiteral struct{}

func (n *NullLiteral) expressionNode()      {}
func (n *NullLiteral) TokenLiteral() string { return "NULL" }

// Variable represents a $variable reference
type Variable struct {
	Name string // includes the $ prefix
}

func (v *Variable) expressionNode()      {}
func (v *Variable) TokenLiteral() string { return v.Name }

// ============================================================================
// Expressions
// ============================================================================

// BinaryExpression represents a binary operation (a + b, a AND b, etc.)
type BinaryExpression struct {
	Left     Expression
	Operator string
	Right    Expression
}

func (b *BinaryExpression) expressionNode()      {}
func (b *BinaryExpression) TokenLiteral() string { return b.Operator }

// UnaryExpression represents a unary operation (NOT x, -x)
type UnaryExpression struct {
	Operator string
	Right    Expression
}

func (u *UnaryExpression) expressionNode()      {}
func (u *UnaryExpression) TokenLiteral() string { return u.Operator }

// FieldAccess represents a.b or service.entity.field
type FieldAccess struct {
	Object Expression
	Field  *Identifier
}

func (f *FieldAccess) expressionNode()      {}
func (f *FieldAccess) TokenLiteral() string { return "." }

// FunctionCall represents COUNT(x), SUM(x), etc.
type FunctionCall struct {
	Function  string
	Arguments []Expression
}

func (f *FunctionCall) expressionNode()      {}
func (f *FunctionCall) TokenLiteral() string { return f.Function }

// ArrayLiteral represents [a, b, c]
type ArrayLiteral struct {
	Elements []Expression
}

func (a *ArrayLiteral) expressionNode()      {}
func (a *ArrayLiteral) TokenLiteral() string { return "[" }

// ObjectLiteral represents { key: value, ... }
type ObjectLiteral struct {
	Pairs map[string]Expression
}

func (o *ObjectLiteral) expressionNode()      {}
func (o *ObjectLiteral) TokenLiteral() string { return "{" }

// ============================================================================
// Query Statements
// ============================================================================

// SelectField represents a field in SELECT clause with optional alias
type SelectField struct {
	Expression Expression
	Alias      string
}

// JoinClause represents a JOIN in a query
type JoinClause struct {
	Type      string     // "JOIN", "LEFT JOIN", etc.
	Source    *Source    // The entity/service to join
	Condition Expression // The ON condition
}

// Source represents a data source (service.entity AS alias)
type Source struct {
	Service string
	Entity  string
	Alias   string
}

// OrderByClause represents ORDER BY field ASC/DESC
type OrderByClause struct {
	Field      Expression
	Descending bool
}

// QueryStatement represents a SELECT query
type QueryStatement struct {
	Fields   []*SelectField
	From     *Source
	Joins    []*JoinClause
	Where    Expression
	GroupBy  []Expression
	Having   Expression
	OrderBy  []*OrderByClause
	Limit    int
	Offset   int
}

func (q *QueryStatement) statementNode()       {}
func (q *QueryStatement) TokenLiteral() string { return "SELECT" }

// ============================================================================
// Workflow Statements
// ============================================================================

// WorkflowStatement represents a WORKFLOW definition
type WorkflowStatement struct {
	Name       string
	Parameters []*Parameter
	Returns    *Identifier
	Steps      []Statement
}

func (w *WorkflowStatement) statementNode()       {}
func (w *WorkflowStatement) TokenLiteral() string { return "WORKFLOW" }

// Parameter represents a workflow parameter
type Parameter struct {
	Name    string
	Type    string
	Default Expression
}

// StepStatement represents a STEP in a workflow
type StepStatement struct {
	Name       string
	Action     Statement // CREATE, UPDATE, DELETE, READ, CALL
	Assertions []Statement
	Fallback   Statement
}

func (s *StepStatement) statementNode()       {}
func (s *StepStatement) TokenLiteral() string { return "STEP" }

// CreateStatement represents CREATE entity SET field = value, ...
type CreateStatement struct {
	Service string
	Entity  string
	Fields  map[string]Expression
	Into    string // Variable to store result
}

func (c *CreateStatement) statementNode()       {}
func (c *CreateStatement) TokenLiteral() string { return "CREATE" }

// UpdateStatement represents UPDATE entity SET field = value WHERE condition
type UpdateStatement struct {
	Service   string
	Entity    string
	Fields    map[string]Expression
	Where     Expression
	KeyFields map[string]Expression // For single-entity updates by key
}

func (u *UpdateStatement) statementNode()       {}
func (u *UpdateStatement) TokenLiteral() string { return "UPDATE" }

// DeleteStatement represents DELETE FROM entity WHERE condition
type DeleteStatement struct {
	Service   string
	Entity    string
	Where     Expression
	KeyFields map[string]Expression
}

func (d *DeleteStatement) statementNode()       {}
func (d *DeleteStatement) TokenLiteral() string { return "DELETE" }

// ReadStatement represents READ entity WHERE condition
type ReadStatement struct {
	Service string
	Entity  string
	Fields  []string
	Where   Expression
	Into    string
}

func (r *ReadStatement) statementNode()       {}
func (r *ReadStatement) TokenLiteral() string { return "READ" }

// CallStatement represents CALL service.function(args)
type CallStatement struct {
	Service   string
	Function  string
	Arguments map[string]Expression
	Into      string
}

func (c *CallStatement) statementNode()       {}
func (c *CallStatement) TokenLiteral() string { return "CALL" }

// ============================================================================
// Control Flow
// ============================================================================

// IfStatement represents IF condition THEN ... ELSE ... ENDIF
type IfStatement struct {
	Condition   Expression
	Consequence []Statement
	Alternative []Statement
}

func (i *IfStatement) statementNode()       {}
func (i *IfStatement) TokenLiteral() string { return "IF" }

// ForEachStatement represents FOREACH $item IN collection ... ENDFOR
type ForEachStatement struct {
	Variable   string
	Collection Expression
	Body       []Statement
}

func (f *ForEachStatement) statementNode()       {}
func (f *ForEachStatement) TokenLiteral() string { return "FOREACH" }

// RetryStatement represents RETRY n TIMES DELAY d
type RetryStatement struct {
	Times int
	Delay string // Duration string like "1s", "500ms"
	Body  Statement
}

func (r *RetryStatement) statementNode()       {}
func (r *RetryStatement) TokenLiteral() string { return "RETRY" }

// ============================================================================
// Assertions & Error Handling
// ============================================================================

// AssertStatement represents ASSERT condition
type AssertStatement struct {
	Condition Expression
	Message   string
}

func (a *AssertStatement) statementNode()       {}
func (a *AssertStatement) TokenLiteral() string { return "ASSERT" }

// ExpectStatement represents EXPECT field operator value
type ExpectStatement struct {
	Field    Expression
	Operator string
	Value    Expression
}

func (e *ExpectStatement) statementNode()       {}
func (e *ExpectStatement) TokenLiteral() string { return "EXPECT" }

// FallbackStatement represents FALLBACK => action
type FallbackStatement struct {
	Action Statement
}

func (f *FallbackStatement) statementNode()       {}
func (f *FallbackStatement) TokenLiteral() string { return "FALLBACK" }

// RollbackStatement represents ROLLBACK
type RollbackStatement struct{}

func (r *RollbackStatement) statementNode()       {}
func (r *RollbackStatement) TokenLiteral() string { return "ROLLBACK" }

// ThrowStatement represents THROW "error message"
type ThrowStatement struct {
	Message Expression
}

func (t *ThrowStatement) statementNode()       {}
func (t *ThrowStatement) TokenLiteral() string { return "THROW" }

// ============================================================================
// Transaction Control
// ============================================================================

// BeginTransactionStatement represents BEGIN TRANSACTION
type BeginTransactionStatement struct {
	Name string
}

func (b *BeginTransactionStatement) statementNode()       {}
func (b *BeginTransactionStatement) TokenLiteral() string { return "BEGIN" }

// CommitStatement represents COMMIT
type CommitStatement struct{}

func (c *CommitStatement) statementNode()       {}
func (c *CommitStatement) TokenLiteral() string { return "COMMIT" }

// ============================================================================
// Service & Exposure
// ============================================================================

// ServiceStatement represents SERVICE declaration for configuration
type ServiceStatement struct {
	Name       string
	URL        string
	Properties map[string]Expression
}

func (s *ServiceStatement) statementNode()       {}
func (s *ServiceStatement) TokenLiteral() string { return "SERVICE" }

// ExposeStatement represents EXPOSE workflow AS odata/mcp
type ExposeStatement struct {
	Workflow string
	As       string // "odata" or "mcp"
	Name     string
}

func (e *ExposeStatement) statementNode()       {}
func (e *ExposeStatement) TokenLiteral() string { return "EXPOSE" }

// ============================================================================
// Variable Assignment
// ============================================================================

// AssignmentStatement represents $var = expression
type AssignmentStatement struct {
	Variable string
	Value    Expression
}

func (a *AssignmentStatement) statementNode()       {}
func (a *AssignmentStatement) TokenLiteral() string { return "=" }
