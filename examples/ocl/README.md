# OCL Testing Guide

## Quick Start

```bash
# Build the OCL CLI
go build -o ocl ./cmd/ocl

# Parse a file
./ocl parse examples/ocl/northwind-queries.ocl

# Interactive REPL
./ocl repl

# Parse from stdin
echo "SELECT * FROM Orders LIMIT 10" | ./ocl parse -
```

## Test Services

### 1. Northwind (Read-Only Queries)
**URL:** `https://services.odata.org/V4/Northwind/Northwind.svc/`

Best for: Testing SELECT, JOIN, ORDER BY, aggregations

```bash
# Test queries
./ocl parse examples/ocl/northwind-queries.ocl

# Try in REPL
./ocl repl
ocl> SELECT ProductName, UnitPrice FROM Northwind.Products \
...> WHERE UnitPrice > 20 ORDER BY UnitPrice DESC LIMIT 10
```

### 2. TripPin (Full CRUD)
**URL:** `https://services.odata.org/TripPinRESTierService/`

Best for: Testing CREATE, UPDATE, DELETE, workflows

```bash
./ocl parse examples/ocl/trippin-crud.ocl
```

### 3. SAP ES5 (Enterprise Patterns)
**URL:** `https://sapes5.sapdevcenter.com/sap/opu/odata/sap/`

Best for: SAP-specific patterns, real-world scenarios

**Setup:**
1. Create free account at https://www.sap.com/developer/tutorials/gateway-demo-signup.html
2. Get ES5 credentials

```bash
./ocl parse examples/ocl/sap-es5-demo.ocl
```

## Current Limitations

The parser is v0.1.0 - some features are parsed but not yet executed:

| Feature | Parse | Execute |
|---------|-------|---------|
| Queries (SELECT/JOIN/WHERE) | ✅ | 🔜 |
| Aggregations (COUNT/SUM) | ✅ | 🔜 |
| CRUD (CREATE/UPDATE/DELETE) | ✅ | 🔜 |
| Workflows (WORKFLOW/STEP) | ✅ | 🔜 |
| Variables ($var) | ✅ | 🔜 |
| Field access ($var.field) | 🔜 | 🔜 |
| Transactions (BEGIN/COMMIT) | ✅ | 🔜 |

## Example OCL Syntax

### Simple Query
```sql
SELECT * FROM Products WHERE Price > 100 LIMIT 10
```

### Cross-Service Query
```sql
SELECT po.ID, items.Amount
FROM MM.PurchaseOrders AS po
JOIN MM.Items AS items ON po.ID == items.OrderID
WHERE po.Status == 'OPEN'
ORDER BY items.Amount DESC
LIMIT 50
```

### Workflow
```sql
WORKFLOW UpdateOrder($orderId, $status)
  STEP read:
    READ Orders WHERE ID == $orderId $order
    ASSERT $order != NULL "Order not found"

  STEP update:
    UPDATE Orders SET Status = $status WHERE ID == $orderId

EXPOSE UpdateOrder AS mcp
```

## Testing Tips

1. **Start with queries** - They're stateless and safe
2. **Use LIMIT** - Don't fetch entire datasets
3. **Check parsing first** - Use `./ocl parse` before execution
4. **Use REPL** - Great for iterating on syntax
