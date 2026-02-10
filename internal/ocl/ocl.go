// Package ocl implements the OData Composer Language (OCL)
//
// OCL is a domain-specific language for composing OData operations across
// multiple services. It supports:
//
//   - Cross-service queries with SQL-like syntax
//   - Workflows with CRUD operations, transactions, and error handling
//   - Variables, control flow, and assertions
//   - Exposure as OData or MCP endpoints
//   - Transpilation to ABAP for native SAP execution
//
// Example query:
//
//	SELECT po.ID, po.Vendor, items.Description, items.Amount
//	FROM MM.PurchaseOrders AS po
//	JOIN MM.PurchaseOrderItems AS items ON po.ID == items.OrderID
//	WHERE po.Status == 'OPEN' AND items.Amount > 1000
//	ORDER BY items.Amount DESC
//	LIMIT 50
//
// Example workflow:
//
//	WORKFLOW UpdateDeliveryDate($poNumber, $newDate)
//	  STEP read_po:
//	    READ PurchaseOrders WHERE PONumber == $poNumber $po
//	    ASSERT $po != NULL "Purchase order not found"
//
//	  STEP update_date:
//	    UPDATE PurchaseOrders($po.ID) SET DeliveryDate = $newDate
//	    EXPECT status == 200
//	    FALLBACK => THROW "Update failed"
package ocl

import (
	"fmt"

	"github.com/zmcp/odata-mcp/internal/ocl/ast"
	"github.com/zmcp/odata-mcp/internal/ocl/lexer"
	"github.com/zmcp/odata-mcp/internal/ocl/parser"
)

// Version of the OCL implementation
const Version = "0.1.0"

// Program represents a parsed OCL program
type Program struct {
	AST    *ast.Program
	Source string
}

// Parse parses OCL source code into a Program
func Parse(source string) (*Program, error) {
	l := lexer.New(source)
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		return nil, fmt.Errorf("parse errors:\n%v", p.Errors())
	}

	return &Program{
		AST:    program,
		Source: source,
	}, nil
}

// ParseQuery is a convenience function for parsing a single query
func ParseQuery(source string) (*ast.QueryStatement, error) {
	prog, err := Parse(source)
	if err != nil {
		return nil, err
	}

	if len(prog.AST.Statements) == 0 {
		return nil, fmt.Errorf("no statements found")
	}

	query, ok := prog.AST.Statements[0].(*ast.QueryStatement)
	if !ok {
		return nil, fmt.Errorf("expected query statement, got %T", prog.AST.Statements[0])
	}

	return query, nil
}

// ParseWorkflow is a convenience function for parsing a workflow
func ParseWorkflow(source string) (*ast.WorkflowStatement, error) {
	prog, err := Parse(source)
	if err != nil {
		return nil, err
	}

	if len(prog.AST.Statements) == 0 {
		return nil, fmt.Errorf("no statements found")
	}

	workflow, ok := prog.AST.Statements[0].(*ast.WorkflowStatement)
	if !ok {
		return nil, fmt.Errorf("expected workflow statement, got %T", prog.AST.Statements[0])
	}

	return workflow, nil
}

// Validate checks an OCL program for semantic errors
func (p *Program) Validate() []error {
	var errors []error

	for _, stmt := range p.AST.Statements {
		switch s := stmt.(type) {
		case *ast.QueryStatement:
			errors = append(errors, validateQuery(s)...)
		case *ast.WorkflowStatement:
			errors = append(errors, validateWorkflow(s)...)
		}
	}

	return errors
}

func validateQuery(q *ast.QueryStatement) []error {
	var errors []error

	if q.From == nil {
		errors = append(errors, fmt.Errorf("query missing FROM clause"))
	}

	return errors
}

func validateWorkflow(w *ast.WorkflowStatement) []error {
	var errors []error

	if w.Name == "" {
		errors = append(errors, fmt.Errorf("workflow missing name"))
	}

	if len(w.Steps) == 0 {
		errors = append(errors, fmt.Errorf("workflow '%s' has no steps", w.Name))
	}

	return errors
}
