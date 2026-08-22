# OData Composer V2: The Full Vision

## Beyond Queries: A Complete Orchestration Layer

### The Expanded Vision

```
┌─────────────────────────────────────────────────────────────────────┐
│                    OData Composer DSL                                │
│  "One language to query, orchestrate, and automate"                 │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   ┌──────────────┐    ┌──────────────┐    ┌──────────────┐         │
│   │   QUERIES    │    │  WORKFLOWS   │    │  COMPOSITE   │         │
│   │  (Stateless) │    │  (Stateful)  │    │    TOOLS     │         │
│   │              │    │              │    │              │         │
│   │  SELECT      │    │  CREATE →    │    │  MCP Tool    │         │
│   │  JOIN        │    │  UPDATE →    │    │  exposed as  │         │
│   │  AGGREGATE   │    │  ASSERT →    │    │  single op   │         │
│   │  COMPARE     │    │  ROLLBACK    │    │              │         │
│   └──────────────┘    └──────────────┘    └──────────────┘         │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
                              │
           ┌──────────────────┼──────────────────┐
           ▼                  ▼                  ▼
   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐
   │  Execute on  │   │  Expose as   │   │  Transpile   │
   │  MCP/JSON-RPC│   │  OData       │   │  to ABAP     │
   └──────────────┘   └──────────────┘   └──────────────┘
```

---

## Why Not CAP?

| Aspect | CAP | OData Composer |
|--------|-----|----------------|
| **Size** | Full framework, Node.js/Java runtime | Tiny DSL, single binary |
| **Setup** | Project scaffolding, build pipeline | Zero setup, point at services |
| **Backend** | Needs deployment | Runs anywhere (CLI, MCP, cloud) |
| **Learning** | CDS, annotations, projections... | One DSL, SQL-like |
| **Existing services** | Must wrap/proxy | Works directly |
| **ABAP replay** | Not possible | First-class target |

**CAP's strength:** Building new services from scratch
**Our strength:** Orchestrating EXISTING services without touching backends

---

## The Two Modes

### Mode 1: Stateless (Queries)
```
INPUT → QUERY → OUTPUT
```
- No side effects
- No transactions
- Cacheable
- Safe to retry

### Mode 2: Stateful (Workflows)
```
INPUT → STEP 1 → STEP 2 → ... → STEP N → OUTPUT
              ↓ fail      ↓ fail
           ROLLBACK    ROLLBACK
```
- Side effects (CRUD)
- Transaction boundaries
- Compensation logic
- State tracking

---

## Workflow DSL Syntax

### Basic Workflow
```ocl
WORKFLOW CreateSalesOrder
  PARAMS
    customer_id: String,
    items: Array<{product_id: String, quantity: Int}>

  -- Step 1: Validate customer
  LET customer = FROM sap.Customers
                 WHERE ID = $customer_id
                 SELECT *
                 SINGLE

  ASSERT customer IS NOT NULL
    ERROR "Customer not found: {$customer_id}"

  ASSERT customer.CreditStatus = 'OK'
    ERROR "Customer credit blocked: {customer.CreditStatus}"

  -- Step 2: Check inventory for all items
  FOR item IN $items DO
    LET stock = FROM sap.Stock
                WHERE ProductID = item.product_id
                SELECT Quantity
                SINGLE

    ASSERT stock.Quantity >= item.quantity
      ERROR "Insufficient stock for {item.product_id}: need {item.quantity}, have {stock.Quantity}"
  END FOR

  -- Step 3: Create order header
  LET order = CREATE sap.SalesOrders WITH {
    CustomerID: $customer_id,
    Status: 'Draft',
    CreatedAt: @Now
  }
  ON ERROR THROW "Failed to create order"

  -- Step 4: Create order items
  FOR item IN $items DO
    CREATE sap.SalesOrderItems WITH {
      OrderID: order.ID,
      ProductID: item.product_id,
      Quantity: item.quantity
    }
    ON ERROR ROLLBACK order  -- Compensating action
  END FOR

  -- Step 5: Submit order
  UPDATE sap.SalesOrders
    WHERE ID = order.ID
    SET Status = 'Submitted'

  RETURN order
END WORKFLOW
```

### With Fallbacks
```ocl
WORKFLOW CreateOrderWithFallback
  PARAMS order_data: Object

  TRY
    -- Primary: SAP S/4HANA
    LET order = CALL sap.CreateOrder WITH $order_data
    RETURN {source: 'SAP', order: order}

  FALLBACK
    -- Secondary: Legacy ECC
    LET order = CALL ecc.ZCreateOrder WITH $order_data
    RETURN {source: 'ECC', order: order}

  FALLBACK
    -- Tertiary: Queue for manual processing
    CREATE queue.PendingOrders WITH $order_data
    RETURN {source: 'QUEUED', status: 'pending'}

  END TRY
END WORKFLOW
```

### With Parallel Execution
```ocl
WORKFLOW EnrichCustomerData
  PARAMS customer_id: String

  -- Fetch from multiple sources in parallel
  PARALLEL
    LET sap_data = FROM sap.Customers WHERE ID = $customer_id
    LET crm_data = FROM dynamics.Accounts WHERE ExternalID = $customer_id
    LET tickets = FROM servicenow.Incidents WHERE CustomerID = $customer_id
  END PARALLEL

  -- Merge results
  RETURN {
    master: sap_data,
    crm: crm_data,
    open_tickets: COUNT(tickets WHERE Status != 'Closed')
  }
END WORKFLOW
```

---

## Composite Tools (MCP Exposure)

### Definition
```ocl
TOOL quarterly_comparison
  DESCRIPTION "Compare financial metrics between quarters"

  PARAMS
    company_code: String REQUIRED,
    metric: Enum('Revenue', 'EBITDA', 'NetIncome') DEFAULT 'Revenue'

  RETURNS Object<{
    current: Number,
    previous: Number,
    delta: Number,
    delta_percent: Number
  }>

  BODY
    COMPARE
      (FROM sap.FinancialStatements
       WHERE CompanyCode = $company_code
         AND Period = @ThisQuarter
         AND Metric = $metric)
    WITH
      (FROM sap.FinancialStatements
       WHERE CompanyCode = $company_code
         AND Period = @LastQuarter
         AND Metric = $metric)
    ON Metric
    CALCULATE
      delta = Current.Amount - Previous.Amount,
      delta_percent = delta / Previous.Amount * 100
  END BODY
END TOOL
```

### Exposed via MCP
```json
{
  "name": "quarterly_comparison",
  "description": "Compare financial metrics between quarters",
  "inputSchema": {
    "type": "object",
    "properties": {
      "company_code": {"type": "string"},
      "metric": {"type": "string", "enum": ["Revenue", "EBITDA", "NetIncome"]}
    },
    "required": ["company_code"]
  }
}
```

### Exposed via OData
```http
POST /odata/CompositeTools/quarterly_comparison
Content-Type: application/json

{
  "company_code": "1000",
  "metric": "Revenue"
}
```

---

## ABAP Transpilation

### OCL Source
```ocl
WORKFLOW ApproveOrder
  PARAMS order_id: String

  LET order = FROM sap.SalesOrders WHERE ID = $order_id SINGLE

  ASSERT order.Status = 'Pending'
    ERROR "Order not pending: {order.Status}"

  ASSERT order.Total <= 50000
    ERROR "Order exceeds auto-approval limit"

  UPDATE sap.SalesOrders
    WHERE ID = $order_id
    SET Status = 'Approved', ApprovedAt = @Now

  RETURN order
END WORKFLOW
```

### Transpiled ABAP
```abap
CLASS zcl_wf_approve_order DEFINITION
  PUBLIC FINAL CREATE PUBLIC.

  PUBLIC SECTION.
    TYPES: BEGIN OF ty_params,
             order_id TYPE string,
           END OF ty_params.

    TYPES: BEGIN OF ty_result,
             order TYPE zsalesorder,
           END OF ty_result.

    METHODS execute
      IMPORTING is_params TYPE ty_params
      RETURNING VALUE(rs_result) TYPE ty_result
      RAISING zcx_workflow_error.

ENDCLASS.

CLASS zcl_wf_approve_order IMPLEMENTATION.
  METHOD execute.
    DATA: ls_order TYPE zsalesorder.

    " Step: Fetch order
    SELECT SINGLE * FROM zsalesorders
      INTO ls_order
      WHERE id = is_params-order_id.

    IF sy-subrc <> 0.
      RAISE EXCEPTION TYPE zcx_workflow_error
        EXPORTING message = |Order not found: { is_params-order_id }|.
    ENDIF.

    " Assert: Status check
    IF ls_order-status <> 'Pending'.
      RAISE EXCEPTION TYPE zcx_workflow_error
        EXPORTING message = |Order not pending: { ls_order-status }|.
    ENDIF.

    " Assert: Limit check
    IF ls_order-total > 50000.
      RAISE EXCEPTION TYPE zcx_workflow_error
        EXPORTING message = |Order exceeds auto-approval limit|.
    ENDIF.

    " Update
    UPDATE zsalesorders
      SET status = 'Approved'
          approved_at = sy-datum
      WHERE id = is_params-order_id.

    " Return
    rs_result-order = ls_order.
  ENDMETHOD.
ENDCLASS.
```

---

## Transport Layer: JSON-RPC Native

### Why JSON-RPC?
- Already used by MCP
- Bidirectional (notifications, progress)
- Stateful sessions possible
- Batch requests built-in

### Protocol Extension
```json
// Execute query (stateless)
{
  "jsonrpc": "2.0",
  "method": "ocl/query",
  "params": {
    "query": "FROM sap.Orders WHERE Total > 1000 SELECT *"
  },
  "id": 1
}

// Execute workflow (stateful)
{
  "jsonrpc": "2.0",
  "method": "ocl/workflow",
  "params": {
    "workflow": "CreateSalesOrder",
    "args": {
      "customer_id": "C001",
      "items": [{"product_id": "P001", "quantity": 10}]
    }
  },
  "id": 2
}

// Progress notification (server → client)
{
  "jsonrpc": "2.0",
  "method": "ocl/progress",
  "params": {
    "workflow_id": "wf-123",
    "step": 3,
    "total_steps": 5,
    "message": "Creating order items..."
  }
}

// Workflow result
{
  "jsonrpc": "2.0",
  "result": {
    "workflow_id": "wf-123",
    "status": "completed",
    "result": {"order_id": "SO-456"},
    "steps_executed": 5,
    "duration_ms": 1234
  },
  "id": 2
}
```

---

## State Management

### Workflow State Machine
```
                    ┌─────────────┐
                    │   CREATED   │
                    └──────┬──────┘
                           │ start
                           ▼
    ┌──────────────────────────────────────────┐
    │                 RUNNING                   │
    │  ┌────────┐  ┌────────┐  ┌────────┐     │
    │  │ Step 1 │→ │ Step 2 │→ │ Step N │     │
    │  └────────┘  └────────┘  └────────┘     │
    └──────────────────┬───────────────────────┘
           │           │           │
      success      failure     timeout
           │           │           │
           ▼           ▼           ▼
    ┌──────────┐ ┌──────────┐ ┌──────────┐
    │COMPLETED │ │ ROLLING  │ │ TIMED    │
    │          │ │   BACK   │ │   OUT    │
    └──────────┘ └────┬─────┘ └──────────┘
                      │
                      ▼
               ┌──────────┐
               │  FAILED  │
               │(cleaned) │
               └──────────┘
```

### State Storage Options
1. **In-memory** - Fast, but lost on restart
2. **Redis** - Fast, persistent, good for short workflows
3. **PostgreSQL** - Durable, queryable history
4. **OData service** - Meta! Store state in SAP itself

---

## Execution Targets

```
                    OCL Source
                         │
         ┌───────────────┼───────────────┐
         ▼               ▼               ▼
┌─────────────────┐ ┌─────────────┐ ┌─────────────┐
│   INTERPRETER   │ │  TRANSPILER │ │  COMPILER   │
│   (Runtime)     │ │  (Codegen)  │ │  (Binary)   │
└────────┬────────┘ └──────┬──────┘ └──────┬──────┘
         │                 │               │
         ▼                 ▼               ▼
┌─────────────────┐ ┌─────────────┐ ┌─────────────┐
│  Execute via    │ │  Generate   │ │  Native     │
│  odata-mcp      │ │  ABAP/Go/TS │ │  WASM       │
│  (MCP/JSON-RPC) │ │  code       │ │  module     │
└─────────────────┘ └─────────────┘ └─────────────┘
```

### Target: ABAP
- For SAP-native execution
- Runs in same process as data
- No network overhead
- Full transaction support

### Target: Go
- For edge/cloud execution
- Compile to single binary
- Embed in microservices

### Target: TypeScript
- For browser/Node execution
- UI5/React integration
- Serverless functions

---

## The "Compact" Promise

### Minimal Runtime
```
┌────────────────────────────────────┐
│        odata-composer              │
│  ┌──────────────────────────────┐  │
│  │  Parser     (~2K LOC)        │  │
│  │  Executor   (~3K LOC)        │  │
│  │  Transport  (~1K LOC)        │  │
│  │  Transpiler (~2K LOC)        │  │
│  └──────────────────────────────┘  │
│                                    │
│  Total: ~8-10K LOC                 │
│  Binary: ~15MB                     │
│  Dependencies: 0 (pure Go)        │
└────────────────────────────────────┘
```

### Compare to CAP
```
CAP Runtime:
- Node.js: ~100MB
- @sap/cds: ~50MB node_modules
- Database drivers, etc.
- Total: ~200MB+
```

---

## Use Case: The "0 Notifications" Dashboard

```ocl
TOOL morning_briefing
  DESCRIPTION "CFO morning briefing - everything that needs attention"

  PARAMS
    company_code: String DEFAULT '1000'

  BODY
    PARALLEL
      -- Overdue invoices
      LET overdue = FROM sap.Invoices
        WHERE CompanyCode = $company_code
          AND DueDate < @Today
          AND Status = 'Open'
        SELECT CustomerName, Amount, DaysPastDue
        ORDER BY Amount DESC
        LIMIT 10

      -- Large pending approvals
      LET approvals = FROM sap.PurchaseOrders
        WHERE CompanyCode = $company_code
          AND Status = 'Pending'
          AND Total > 50000
        SELECT Vendor, Total, RequestedBy
        ORDER BY Total DESC

      -- Customer escalations
      LET escalations = FROM servicenow.Incidents
        WHERE CompanyCode = $company_code
          AND Priority IN ('P1', 'P2')
          AND Status = 'Open'
        SELECT Customer, Summary, Priority, Age
        ORDER BY Priority, Age DESC

      -- Budget variances
      LET variances = COMPARE
        (FROM sap.CostCenters WHERE Period = @ThisMonth)
        WITH
        (FROM sap.Budgets WHERE Period = @ThisMonth)
        ON CostCenter
        CALCULATE variance_pct = (Actual - Budget) / Budget * 100
        WHERE variance_pct > 10 OR variance_pct < -10
    END PARALLEL

    RETURN {
      summary: {
        overdue_count: COUNT(overdue),
        overdue_total: SUM(overdue.Amount),
        pending_approvals: COUNT(approvals),
        pending_value: SUM(approvals.Total),
        open_escalations: COUNT(escalations),
        budget_alerts: COUNT(variances)
      },
      details: {
        overdue: overdue,
        approvals: approvals,
        escalations: escalations,
        variances: variances
      }
    }
  END BODY
END TOOL
```

**Result:** One tool call. All the data. Zero notifications needed.

---

## Implementation Roadmap

### Phase 0: Parser Foundation
- [ ] EBNF grammar finalization
- [ ] Lexer/Parser in Go (no dependencies)
- [ ] AST representation
- [ ] Pretty-printer (OCL → OCL)

### Phase 1: Query Execution
- [ ] Stateless query execution
- [ ] Integration with odata-mcp
- [ ] Basic aggregations
- [ ] Cross-service joins

### Phase 2: Workflows
- [ ] Workflow state machine
- [ ] ASSERT/ERROR handling
- [ ] FOR loops
- [ ] Transaction boundaries

### Phase 3: Composite Tools
- [ ] TOOL definitions
- [ ] MCP tool generation
- [ ] OData exposure

### Phase 4: Transpilation
- [ ] ABAP codegen
- [ ] Go codegen
- [ ] TypeScript codegen

### Phase 5: Production
- [ ] State persistence
- [ ] Monitoring/tracing
- [ ] Admin UI
- [ ] RBAC

---

## Names Considered

| Name | Vibe |
|------|------|
| **OCL** - OData Composer Language | Technical, clear |
| **OFlow** | Workflow focus |
| **OQL** - OData Query Language | SQL echo |
| **Weave** | Connecting systems |
| **Stitch** | Sewing data together |
| **Mesh** | Service mesh vibes |
| **Conduit** | Data pipeline |
| **Bridge DSL** | Extension of current project |

**Current favorite:** **OCL** (OData Composer Language) or **OFlow**

---

## Summary: What We're Building

> **A compact, cross-platform DSL that lets you query, orchestrate, and automate across OData services - executable via MCP, exposable as OData, and transpilable to ABAP.**

The three pillars:
1. **Queries** - Stateless, aggregations, joins, comparisons
2. **Workflows** - Stateful, CRUD, transactions, rollbacks
3. **Tools** - Composite operations as first-class MCP/OData endpoints

All in ~10K lines of Go. No framework. No infrastructure. Just point and compose.

---

*Brainstorm V2: February 2026*
*Status: Concept expansion*
