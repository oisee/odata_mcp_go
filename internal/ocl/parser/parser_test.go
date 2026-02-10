package parser

import (
	"testing"

	"github.com/zmcp/odata-mcp/internal/ocl/ast"
	"github.com/zmcp/odata-mcp/internal/ocl/lexer"
)

func TestParseSimpleQuery(t *testing.T) {
	input := `SELECT * FROM Orders`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	query, ok := program.Statements[0].(*ast.QueryStatement)
	if !ok {
		t.Fatalf("expected QueryStatement, got %T", program.Statements[0])
	}

	if query.From == nil {
		t.Fatal("query.From is nil")
	}

	if query.From.Entity != "Orders" {
		t.Errorf("expected entity 'Orders', got %q", query.From.Entity)
	}
}

func TestParseQueryWithServiceAndAlias(t *testing.T) {
	input := `SELECT po.ID, po.Vendor
FROM MM.PurchaseOrders AS po
WHERE po.Status == 'OPEN'
LIMIT 50`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	query, ok := program.Statements[0].(*ast.QueryStatement)
	if !ok {
		t.Fatalf("expected QueryStatement, got %T", program.Statements[0])
	}

	// Check FROM clause
	if query.From.Service != "MM" {
		t.Errorf("expected service 'MM', got %q", query.From.Service)
	}
	if query.From.Entity != "PurchaseOrders" {
		t.Errorf("expected entity 'PurchaseOrders', got %q", query.From.Entity)
	}
	if query.From.Alias != "po" {
		t.Errorf("expected alias 'po', got %q", query.From.Alias)
	}

	// Check fields
	if len(query.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(query.Fields))
	}

	// Check WHERE clause exists
	if query.Where == nil {
		t.Error("expected WHERE clause")
	}

	// Check LIMIT
	if query.Limit != 50 {
		t.Errorf("expected LIMIT 50, got %d", query.Limit)
	}
}

func TestParseQueryWithJoin(t *testing.T) {
	input := `SELECT po.ID, items.Description
FROM MM.PurchaseOrders AS po
JOIN MM.POItems AS items ON po.ID == items.OrderID
WHERE po.Status == 'OPEN'`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	query := program.Statements[0].(*ast.QueryStatement)

	if len(query.Joins) != 1 {
		t.Fatalf("expected 1 join, got %d", len(query.Joins))
	}

	join := query.Joins[0]
	if join.Source.Entity != "POItems" {
		t.Errorf("expected join entity 'POItems', got %q", join.Source.Entity)
	}
	if join.Source.Alias != "items" {
		t.Errorf("expected join alias 'items', got %q", join.Source.Alias)
	}
}

func TestParseQueryWithOrderBy(t *testing.T) {
	input := `SELECT * FROM Orders ORDER BY Amount DESC, CreatedAt ASC`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	query := program.Statements[0].(*ast.QueryStatement)

	if len(query.OrderBy) != 2 {
		t.Fatalf("expected 2 order by clauses, got %d", len(query.OrderBy))
	}

	if !query.OrderBy[0].Descending {
		t.Error("first order by should be DESC")
	}
	if query.OrderBy[1].Descending {
		t.Error("second order by should be ASC")
	}
}

func TestParseQueryWithAggregation(t *testing.T) {
	input := `SELECT Vendor, COUNT(ID), SUM(Amount)
FROM PurchaseOrders
GROUP BY Vendor
HAVING SUM(Amount) > 10000`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	query := program.Statements[0].(*ast.QueryStatement)

	if len(query.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(query.Fields))
	}

	if len(query.GroupBy) != 1 {
		t.Errorf("expected 1 group by field, got %d", len(query.GroupBy))
	}

	if query.Having == nil {
		t.Error("expected HAVING clause")
	}
}

func TestParseWorkflow(t *testing.T) {
	input := `WORKFLOW UpdateDeliveryDate($poNumber, $newDate)
  STEP read_po:
    READ PurchaseOrders WHERE PONumber == $poNumber $po

  STEP update:
    UPDATE PurchaseOrders SET DeliveryDate = $newDate`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	workflow, ok := program.Statements[0].(*ast.WorkflowStatement)
	if !ok {
		t.Fatalf("expected WorkflowStatement, got %T", program.Statements[0])
	}

	if workflow.Name != "UpdateDeliveryDate" {
		t.Errorf("expected workflow name 'UpdateDeliveryDate', got %q", workflow.Name)
	}

	if len(workflow.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(workflow.Parameters))
	}

	if len(workflow.Steps) < 2 {
		t.Errorf("expected at least 2 steps, got %d", len(workflow.Steps))
	}
}

func TestParseCreateStatement(t *testing.T) {
	input := `CREATE PurchaseOrder SET Vendor = 'ACME', Amount = 1000 $newPO`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	create, ok := program.Statements[0].(*ast.CreateStatement)
	if !ok {
		t.Fatalf("expected CreateStatement, got %T", program.Statements[0])
	}

	if create.Entity != "PurchaseOrder" {
		t.Errorf("expected entity 'PurchaseOrder', got %q", create.Entity)
	}

	if len(create.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(create.Fields))
	}

	if create.Into != "$newPO" {
		t.Errorf("expected into '$newPO', got %q", create.Into)
	}
}

func TestParseUpdateStatement(t *testing.T) {
	input := `UPDATE PurchaseOrders(ID = '123') SET Status = 'CLOSED'`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	update, ok := program.Statements[0].(*ast.UpdateStatement)
	if !ok {
		t.Fatalf("expected UpdateStatement, got %T", program.Statements[0])
	}

	if update.Entity != "PurchaseOrders" {
		t.Errorf("expected entity 'PurchaseOrders', got %q", update.Entity)
	}

	if len(update.KeyFields) != 1 {
		t.Errorf("expected 1 key field, got %d", len(update.KeyFields))
	}

	if len(update.Fields) != 1 {
		t.Errorf("expected 1 field, got %d", len(update.Fields))
	}
}

func TestParseDeleteStatement(t *testing.T) {
	input := `DELETE FROM PurchaseOrders WHERE Status == 'CANCELLED'`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	del, ok := program.Statements[0].(*ast.DeleteStatement)
	if !ok {
		t.Fatalf("expected DeleteStatement, got %T", program.Statements[0])
	}

	if del.Entity != "PurchaseOrders" {
		t.Errorf("expected entity 'PurchaseOrders', got %q", del.Entity)
	}

	if del.Where == nil {
		t.Error("expected WHERE clause")
	}
}

func TestParseIfStatement(t *testing.T) {
	input := `IF $amount > 1000 THEN
  CREATE Approval SET Amount = $amount
ELSE
  UPDATE Order SET Status = 'APPROVED'
ENDIF`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	ifStmt, ok := program.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", program.Statements[0])
	}

	if ifStmt.Condition == nil {
		t.Error("expected condition")
	}

	if len(ifStmt.Consequence) != 1 {
		t.Errorf("expected 1 consequence statement, got %d", len(ifStmt.Consequence))
	}

	if len(ifStmt.Alternative) != 1 {
		t.Errorf("expected 1 alternative statement, got %d", len(ifStmt.Alternative))
	}
}

func TestParseForEachStatement(t *testing.T) {
	input := `FOREACH $item IN $items
  UPDATE Items SET Processed = TRUE
ENDFOR`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	forEach, ok := program.Statements[0].(*ast.ForEachStatement)
	if !ok {
		t.Fatalf("expected ForEachStatement, got %T", program.Statements[0])
	}

	if forEach.Variable != "$item" {
		t.Errorf("expected variable '$item', got %q", forEach.Variable)
	}

	if len(forEach.Body) != 1 {
		t.Errorf("expected 1 body statement, got %d", len(forEach.Body))
	}
}

func TestParseAssertStatement(t *testing.T) {
	input := `ASSERT $result != NULL "Result should not be null"`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	assert, ok := program.Statements[0].(*ast.AssertStatement)
	if !ok {
		t.Fatalf("expected AssertStatement, got %T", program.Statements[0])
	}

	if assert.Condition == nil {
		t.Error("expected condition")
	}

	if assert.Message != "Result should not be null" {
		t.Errorf("expected message 'Result should not be null', got %q", assert.Message)
	}
}

func TestParseCallStatement(t *testing.T) {
	input := `CALL MM.CalculatePrice(OrderID = $orderId, Quantity = 10) $price`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	call, ok := program.Statements[0].(*ast.CallStatement)
	if !ok {
		t.Fatalf("expected CallStatement, got %T", program.Statements[0])
	}

	if call.Service != "MM" {
		t.Errorf("expected service 'MM', got %q", call.Service)
	}

	if call.Function != "CalculatePrice" {
		t.Errorf("expected function 'CalculatePrice', got %q", call.Function)
	}

	if len(call.Arguments) != 2 {
		t.Errorf("expected 2 arguments, got %d", len(call.Arguments))
	}

	if call.Into != "$price" {
		t.Errorf("expected into '$price', got %q", call.Into)
	}
}

func TestParseExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a == b", "=="},
		{"a != b", "!="},
		{"a > b", ">"},
		{"a < b", "<"},
		{"a >= b", ">="},
		{"a <= b", "<="},
		{"a AND b", "AND"},
		{"a OR b", "OR"},
		{"a + b", "+"},
		{"a - b", "-"},
		{"a * b", "*"},
		{"a / b", "/"},
	}

	for _, tt := range tests {
		input := "SELECT * FROM T WHERE " + tt.input

		l := lexer.New(input)
		p := New(l)
		program := p.Parse()

		checkParserErrors(t, p)

		query := program.Statements[0].(*ast.QueryStatement)
		binExpr, ok := query.Where.(*ast.BinaryExpression)
		if !ok {
			t.Errorf("expected BinaryExpression for %q", tt.input)
			continue
		}

		if binExpr.Operator != tt.expected {
			t.Errorf("expected operator %q, got %q", tt.expected, binExpr.Operator)
		}
	}
}

func TestParseServiceStatement(t *testing.T) {
	input := `SERVICE MM "https://sap.example.com/odata/MM"`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	svc, ok := program.Statements[0].(*ast.ServiceStatement)
	if !ok {
		t.Fatalf("expected ServiceStatement, got %T", program.Statements[0])
	}

	if svc.Name != "MM" {
		t.Errorf("expected name 'MM', got %q", svc.Name)
	}

	if svc.URL != "https://sap.example.com/odata/MM" {
		t.Errorf("expected URL 'https://sap.example.com/odata/MM', got %q", svc.URL)
	}
}

func TestParseExposeStatement(t *testing.T) {
	input := `EXPOSE UpdateDeliveryDate AS mcp`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	expose, ok := program.Statements[0].(*ast.ExposeStatement)
	if !ok {
		t.Fatalf("expected ExposeStatement, got %T", program.Statements[0])
	}

	if expose.Workflow != "UpdateDeliveryDate" {
		t.Errorf("expected workflow 'UpdateDeliveryDate', got %q", expose.Workflow)
	}

	if expose.As != "mcp" {
		t.Errorf("expected as 'mcp', got %q", expose.As)
	}
}

func TestParseVariableAssignment(t *testing.T) {
	input := `$total = $price * $quantity`

	l := lexer.New(input)
	p := New(l)
	program := p.Parse()

	checkParserErrors(t, p)

	assign, ok := program.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected AssignmentStatement, got %T", program.Statements[0])
	}

	if assign.Variable != "$total" {
		t.Errorf("expected variable '$total', got %q", assign.Variable)
	}
}

func checkParserErrors(t *testing.T, p *Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %s", msg)
	}
	t.FailNow()
}
