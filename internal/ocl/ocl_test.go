package ocl

import (
	"testing"
)

func TestParse_SimpleQuery(t *testing.T) {
	source := `SELECT * FROM Orders WHERE Status == 'OPEN' LIMIT 10`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(prog.AST.Statements) != 1 {
		t.Errorf("expected 1 statement, got %d", len(prog.AST.Statements))
	}
}

func TestParse_CrossServiceQuery(t *testing.T) {
	source := `
SELECT po.ID, po.Vendor, items.Description, items.Amount
FROM MM.PurchaseOrders AS po
JOIN MM.PurchaseOrderItems AS items ON po.ID == items.OrderID
WHERE po.Status == 'OPEN' AND items.Amount > 1000
ORDER BY items.Amount DESC
LIMIT 50
`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	if len(prog.AST.Statements) != 1 {
		t.Errorf("expected 1 statement, got %d", len(prog.AST.Statements))
	}

	errors := prog.Validate()
	if len(errors) != 0 {
		t.Errorf("unexpected validation errors: %v", errors)
	}
}

func TestParseQuery(t *testing.T) {
	source := `SELECT ID, Vendor FROM Orders WHERE Amount > 100`

	query, err := ParseQuery(source)
	if err != nil {
		t.Fatalf("ParseQuery error: %v", err)
	}

	if query.From.Entity != "Orders" {
		t.Errorf("expected entity 'Orders', got %q", query.From.Entity)
	}

	if len(query.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(query.Fields))
	}
}

func TestParseWorkflow(t *testing.T) {
	source := `
WORKFLOW UpdateDeliveryDate($poNumber, $newDate)
  STEP read_po:
    READ PurchaseOrders WHERE PONumber == $poNumber $po

  STEP update_date:
    UPDATE PurchaseOrders SET DeliveryDate = $newDate
`

	workflow, err := ParseWorkflow(source)
	if err != nil {
		t.Fatalf("ParseWorkflow error: %v", err)
	}

	if workflow.Name != "UpdateDeliveryDate" {
		t.Errorf("expected name 'UpdateDeliveryDate', got %q", workflow.Name)
	}

	if len(workflow.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %d", len(workflow.Parameters))
	}
}

func TestParse_CompleteProgram(t *testing.T) {
	source := `
-- Service definitions
SERVICE MM "https://sap.example.com/odata/MM"
SERVICE SD "https://sap.example.com/odata/SD"

-- Workflow definition
WORKFLOW ProcessOrder($orderId)
  STEP read:
    READ MM.Orders WHERE OrderID == $orderId $order

  STEP create_delivery:
    CREATE SD.Deliveries SET OrderID = $orderId, Status = 'NEW' $delivery

  STEP update_status:
    UPDATE MM.Orders SET Status = 'PROCESSING'

-- Expose as MCP tool
EXPOSE ProcessOrder AS mcp
`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	// Should have: 2 services + 1 workflow + 1 expose
	if len(prog.AST.Statements) < 3 {
		t.Errorf("expected at least 3 statements, got %d", len(prog.AST.Statements))
	}
}

func TestParse_ControlFlow(t *testing.T) {
	source := `
WORKFLOW BatchProcess($items)
  FOREACH $item IN $items
    IF $amount > 1000 THEN
      CREATE Approval SET Amount = $amount
    ELSE
      UPDATE Items SET Status = 'APPROVED'
    ENDIF
  ENDFOR
`

	workflow, err := ParseWorkflow(source)
	if err != nil {
		t.Fatalf("ParseWorkflow error: %v", err)
	}

	if workflow.Name != "BatchProcess" {
		t.Errorf("expected name 'BatchProcess', got %q", workflow.Name)
	}
}

func TestParse_Aggregation(t *testing.T) {
	source := `
SELECT Vendor, COUNT(ID) AS OrderCount, SUM(Amount) AS TotalAmount
FROM PurchaseOrders
WHERE Status == 'CLOSED'
GROUP BY Vendor
HAVING SUM(Amount) > 100000
ORDER BY TotalAmount DESC
`

	query, err := ParseQuery(source)
	if err != nil {
		t.Fatalf("ParseQuery error: %v", err)
	}

	if len(query.Fields) != 3 {
		t.Errorf("expected 3 fields, got %d", len(query.Fields))
	}

	if len(query.GroupBy) != 1 {
		t.Errorf("expected 1 group by, got %d", len(query.GroupBy))
	}

	if query.Having == nil {
		t.Error("expected HAVING clause")
	}
}

func TestParse_InvalidSyntax(t *testing.T) {
	source := `SELECT FROM WHERE`

	_, err := Parse(source)
	if err == nil {
		t.Error("expected parse error for invalid syntax")
	}
}

func TestValidate_EmptyWorkflow(t *testing.T) {
	source := `WORKFLOW Empty()`

	prog, err := Parse(source)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	errors := prog.Validate()
	if len(errors) == 0 {
		t.Error("expected validation error for empty workflow")
	}
}

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}
