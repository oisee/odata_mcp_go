# From Fiori Capture to DSL: The Complete Pipeline

## The Vision: Record Once, Replay Everywhere

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    FIORI AUTOMATOR (Capture)                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐               │
│  │  Clicks  │  │  Inputs  │  │  OData   │  │   UI5    │               │
│  │  Events  │  │  Values  │  │ Requests │  │ Context  │               │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘               │
│       └──────────────┼──────────────┼──────────────┘                    │
│                      ▼                                                   │
│              Raw Session Log (JSON)                                      │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    AI ANALYZER (Understand)                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │   Identify   │  │   Extract    │  │   Infer      │                  │
│  │   Atomic     │  │   Business   │  │   Control    │                  │
│  │   Operations │  │   Entities   │  │   Flow       │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
│                              │                                           │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐                  │
│  │  Assertions  │  │   Pre-      │  │  Fallbacks   │                  │
│  │  (Expected)  │  │  requisites │  │  (On Error)  │                  │
│  └──────────────┘  └──────────────┘  └──────────────┘                  │
└─────────────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    OCL COMPILER (Generate)                               │
│                                                                          │
│  WORKFLOW UpdatePurchaseOrder                                           │
│    PARAMS order_id: String, new_date: Date                              │
│                                                                          │
│    -- Prerequisites (from UI validation)                                │
│    LET order = FROM sap.PurchaseOrders WHERE ID = $order_id SINGLE     │
│    ASSERT order.Status IN ('Draft', 'Pending')                          │
│      ERROR "Cannot modify order in status {order.Status}"               │
│                                                                          │
│    -- Main operation (from captured OData PATCH)                        │
│    UPDATE sap.PurchaseOrders                                            │
│      WHERE ID = $order_id                                               │
│      SET DeliveryDate = $new_date                                       │
│      ON ERROR THROW "Update failed: {error}"                            │
│                                                                          │
│    RETURN order                                                          │
│  END WORKFLOW                                                            │
└─────────────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│   EXECUTE via   │ │  TRANSPILE to   │ │   EXPOSE as     │
│   odata-mcp     │ │     ABAP        │ │   OData/MCP     │
│   (Runtime)     │ │   (Native)      │ │   (Service)     │
└─────────────────┘ └─────────────────┘ └─────────────────┘
```

---

## Why Not CAP?

### The License Problem

CAP (Cloud Application Programming Model) has restrictive licensing:

> **SAP's CAP license explicitly restricts AI/LLM usage.**

This means:
- Cannot train models on CAP code
- Cannot use LLMs to generate CAP artifacts
- Cannot integrate CAP with AI assistants
- Legally problematic for our vision

### Our Approach: Fully Open

```
MIT License

Permission is hereby granted, free of charge, to any person...
```

- AI-friendly by design
- LLM generation encouraged
- Community contributions welcome
- No legal barriers to innovation

---

## What Fiori Automator Captures

From `content.js` analysis:

### Event Structure
```javascript
{
  type: 'click' | 'input' | 'submit' | 'keyboard',

  // Element identification
  element: {
    tagName: 'BUTTON',
    id: 'saveButton',
    className: 'sapMBtn sapMBtnAccept',
    attributes: {...}
  },

  // UI5 Context (the gold!)
  ui5Context: {
    controlType: 'sap.m.Button',
    controlId: '__button0',
    properties: {
      text: 'Save',
      type: 'Accept',
      enabled: true
    },
    bindingInfo: {
      path: '/PurchaseOrder',
      model: 'mainModel'
    }
  },

  // Coordinates for replay
  coordinates: { x: 150, y: 300 },

  // Page context
  pageUrl: 'https://sap.../app#/PO/123',
  pageTitle: 'Edit Purchase Order'
}
```

### OData Request Correlation
```javascript
{
  type: 'odata_request',
  method: 'PATCH',
  url: '/sap/opu/odata/sap/API_PURCHASEORDER_PROCESS_SRV/PurchaseOrder(\'4500000123\')',
  requestBody: {
    DeliveryDate: '2025-07-15'
  },
  responseStatus: 200,
  responseBody: {...},
  correlatedEvent: 'click-on-save-button'  // Links to UI event
}
```

---

## AI Analysis: From Raw to Structured

### Input: Raw Session Log
```json
[
  {"type": "click", "element": {"id": "editBtn"}, "ui5Context": {"controlType": "sap.m.Button"}},
  {"type": "input", "element": {"id": "dateField"}, "value": "2025-07-15", "ui5Context": {"bindingInfo": {"path": "/DeliveryDate"}}},
  {"type": "click", "element": {"id": "saveBtn"}, "ui5Context": {"controlType": "sap.m.Button", "properties": {"text": "Save"}}},
  {"type": "odata_request", "method": "PATCH", "url": "...PurchaseOrder('123')", "requestBody": {"DeliveryDate": "2025-07-15"}}
]
```

### AI Analysis Prompt
```
Analyze this Fiori session recording and extract:

1. ATOMIC OPERATIONS
   - What business operations were performed?
   - Map each to OData service calls

2. PARAMETERS
   - What values were dynamic (user input)?
   - What values were static (UI defaults)?

3. PREREQUISITES
   - What state must exist before the operation?
   - What validations occurred in the UI?

4. ASSERTIONS
   - What success indicators were shown?
   - What error handling was present?

5. CONTROL FLOW
   - What was the sequence of operations?
   - Were there conditional branches?

Output as structured JSON for DSL compilation.
```

### AI Output: Structured Analysis
```json
{
  "workflow_name": "UpdatePurchaseOrderDeliveryDate",
  "description": "Update delivery date on a purchase order",

  "parameters": [
    {"name": "order_id", "type": "String", "source": "url_path", "example": "4500000123"},
    {"name": "new_date", "type": "Date", "source": "user_input", "field": "dateField"}
  ],

  "prerequisites": [
    {"type": "entity_exists", "entity": "PurchaseOrder", "key": "$order_id"},
    {"type": "status_check", "field": "Status", "allowed": ["Draft", "Pending"], "inferred_from": "edit_button_enabled"}
  ],

  "operations": [
    {
      "step": 1,
      "type": "update",
      "entity": "PurchaseOrder",
      "key": "$order_id",
      "fields": {"DeliveryDate": "$new_date"},
      "odata_method": "PATCH"
    }
  ],

  "assertions": [
    {"type": "response_status", "expected": 200},
    {"type": "ui_message", "pattern": "saved successfully", "inferred_from": "message_toast"}
  ],

  "fallbacks": [
    {"on_error": "status_409", "action": "throw", "message": "Concurrent modification detected"}
  ]
}
```

---

## DSL Compilation

### From Analysis to OCL

```ocl
-- Auto-generated from Fiori session: fs-2025-07-10-1430-edit-purchase-order
-- Captured by: Fiori Automator v1.1
-- Analyzed by: Claude

WORKFLOW UpdatePurchaseOrderDeliveryDate
  DESCRIPTION "Update delivery date on a purchase order"

  PARAMS
    order_id: String REQUIRED,
    new_date: Date REQUIRED

  -- Prerequisites (inferred from UI state)
  LET order = FROM sap.PurchaseOrders
              WHERE PurchaseOrder = $order_id
              SELECT Status, DeliveryDate
              SINGLE

  ASSERT order IS NOT NULL
    ERROR "Purchase order not found: {$order_id}"

  ASSERT order.Status IN ('Draft', 'Pending')
    ERROR "Cannot modify order in status: {order.Status}"

  -- Main operation (from captured PATCH)
  UPDATE sap.PurchaseOrders
    WHERE PurchaseOrder = $order_id
    SET DeliveryDate = $new_date

  -- Assertion (from success message)
  ON SUCCESS
    LOG "Delivery date updated to {$new_date}"

  ON ERROR 409
    THROW "Concurrent modification - please refresh and retry"

  ON ERROR
    THROW "Update failed: {error.message}"

  RETURN {
    order_id: $order_id,
    new_date: $new_date,
    previous_date: order.DeliveryDate
  }
END WORKFLOW
```

---

## Execution Targets

### 1. odata-mcp Runtime
```bash
# Execute directly via MCP
odata-mcp --workflow UpdatePurchaseOrderDeliveryDate \
  --param order_id=4500000123 \
  --param new_date=2025-07-15
```

### 2. ABAP Transpilation
```abap
*----------------------------------------------------------------------*
* Auto-generated from OCL workflow: UpdatePurchaseOrderDeliveryDate
* Source: fs-2025-07-10-1430-edit-purchase-order
*----------------------------------------------------------------------*
CLASS zcl_wf_update_po_date DEFINITION
  PUBLIC FINAL CREATE PUBLIC.

  PUBLIC SECTION.
    TYPES: BEGIN OF ty_params,
             order_id TYPE ebeln,
             new_date TYPE lfdat,
           END OF ty_params.

    TYPES: BEGIN OF ty_result,
             order_id TYPE ebeln,
             new_date TYPE lfdat,
             previous_date TYPE lfdat,
           END OF ty_result.

    METHODS execute
      IMPORTING is_params TYPE ty_params
      RETURNING VALUE(rs_result) TYPE ty_result
      RAISING zcx_workflow_error.

ENDCLASS.

CLASS zcl_wf_update_po_date IMPLEMENTATION.
  METHOD execute.
    DATA: ls_order TYPE bapimepoheader.

    " Prerequisite: Fetch order
    CALL FUNCTION 'BAPI_PO_GETDETAIL'
      EXPORTING
        purchaseorder = is_params-order_id
      IMPORTING
        po_header     = ls_order.

    " Assert: Order exists
    IF sy-subrc <> 0.
      RAISE EXCEPTION TYPE zcx_workflow_error
        EXPORTING message = |Purchase order not found: { is_params-order_id }|.
    ENDIF.

    " Assert: Status allows modification
    IF ls_order-status NOT IN ('Draft', 'Pending').
      RAISE EXCEPTION TYPE zcx_workflow_error
        EXPORTING message = |Cannot modify order in status: { ls_order-status }|.
    ENDIF.

    " Main operation: Update
    rs_result-previous_date = ls_order-deliv_date.

    CALL FUNCTION 'BAPI_PO_CHANGE'
      EXPORTING
        purchaseorder = is_params-order_id
        poheader      = VALUE #( deliv_date = is_params-new_date )
        poheaderx     = VALUE #( deliv_date = abap_true ).

    IF sy-subrc <> 0.
      RAISE EXCEPTION TYPE zcx_workflow_error
        EXPORTING message = |Update failed|.
    ENDIF.

    " Return result
    rs_result-order_id = is_params-order_id.
    rs_result-new_date = is_params-new_date.
  ENDMETHOD.
ENDCLASS.
```

### 3. MCP Tool Exposure
```json
{
  "name": "update_po_delivery_date",
  "description": "Update delivery date on a purchase order",
  "inputSchema": {
    "type": "object",
    "properties": {
      "order_id": {"type": "string", "description": "Purchase order number"},
      "new_date": {"type": "string", "format": "date", "description": "New delivery date"}
    },
    "required": ["order_id", "new_date"]
  }
}
```

---

## Multi-Case Learning

### The Power of Multiple Recordings

```
Session 1: Create Purchase Order (happy path)
Session 2: Create Purchase Order (validation error)
Session 3: Create Purchase Order (duplicate detection)
Session 4: Create Purchase Order (approval workflow)
```

### AI Consolidation
```
Analyze these 4 sessions of the same operation.
Identify:
- Common happy path
- Edge cases and error handling
- Conditional branches
- Required vs optional fields

Generate a comprehensive workflow covering all scenarios.
```

### Result: Robust Workflow
```ocl
WORKFLOW CreatePurchaseOrder
  PARAMS
    vendor_id: String REQUIRED,
    items: Array<OrderItem> REQUIRED,
    delivery_date: Date,
    approval_required: Boolean DEFAULT false

  -- Validate vendor (from session 2 error)
  LET vendor = FROM sap.Vendors WHERE ID = $vendor_id SINGLE
  ASSERT vendor IS NOT NULL
    ERROR "Vendor not found"
  ASSERT vendor.Status = 'Active'
    ERROR "Vendor is blocked"

  -- Check for duplicates (from session 3)
  LET existing = FROM sap.PurchaseOrders
    WHERE VendorID = $vendor_id
      AND CreatedAt >= @Today
      AND Status = 'Draft'
  ASSERT COUNT(existing) = 0
    WARNING "Similar draft order exists: {existing[0].ID}"
    PROMPT "Continue anyway?"

  -- Create order
  LET order = CREATE sap.PurchaseOrders WITH {
    VendorID: $vendor_id,
    DeliveryDate: $delivery_date ?? @NextWeek,
    Status: $approval_required ? 'PendingApproval' : 'Draft'
  }

  -- Create items
  FOR item IN $items DO
    CREATE sap.PurchaseOrderItems WITH {
      OrderID: order.ID,
      ProductID: item.product_id,
      Quantity: item.quantity,
      Price: item.price
    }
  END FOR

  -- Trigger approval if needed (from session 4)
  IF $approval_required THEN
    CALL sap.TriggerApprovalWorkflow WITH {
      ObjectType: 'PO',
      ObjectID: order.ID
    }
  END IF

  RETURN order
END WORKFLOW
```

---

## Architecture: The Complete System

```
┌─────────────────────────────────────────────────────────────────────┐
│                         USER INTERFACE                               │
│  ┌────────────────┐  ┌────────────────┐  ┌────────────────┐        │
│  │ Fiori Automator│  │  CLI / VSCode  │  │   Web Editor   │        │
│  │   (Capture)    │  │   (Author)     │  │    (Manage)    │        │
│  └───────┬────────┘  └───────┬────────┘  └───────┬────────┘        │
└──────────┼───────────────────┼───────────────────┼──────────────────┘
           │                   │                   │
           ▼                   ▼                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     WORKFLOW REPOSITORY                              │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  workflows/                                                  │   │
│  │  ├── purchase_orders/                                       │   │
│  │  │   ├── create.ocl                                        │   │
│  │  │   ├── update_delivery_date.ocl                          │   │
│  │  │   └── approve.ocl                                       │   │
│  │  ├── sales_orders/                                          │   │
│  │  └── master_data/                                           │   │
│  └─────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
           │
           ▼
┌─────────────────────────────────────────────────────────────────────┐
│                     EXECUTION ENGINE                                 │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │    Parser    │→ │   Planner    │→ │   Executor   │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│         │                                    │                       │
│         ▼                                    ▼                       │
│  ┌──────────────┐                    ┌──────────────┐              │
│  │  Transpiler  │                    │   Runtime    │              │
│  │  (ABAP, Go)  │                    │  (odata-mcp) │              │
│  └──────────────┘                    └──────────────┘              │
└─────────────────────────────────────────────────────────────────────┘
           │                                   │
           ▼                                   ▼
┌─────────────────────────────────────────────────────────────────────┐
│                      TARGET SYSTEMS                                  │
│                                                                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │  SAP S/4HANA │  │   Dynamics   │  │ ServiceNow   │              │
│  │   (OData)    │  │    (OData)   │  │   (REST)     │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 0: Foundation
- [x] Fiori Automator (capture)
- [x] odata-mcp (OData access)
- [ ] Session log format specification
- [ ] AI analysis prompt templates

### Phase 1: Capture → Analysis
- [ ] Session log importer
- [ ] AI analysis pipeline
- [ ] Structured output schema
- [ ] Validation and testing framework

### Phase 2: Analysis → DSL
- [ ] OCL compiler (analysis → OCL)
- [ ] DSL validation
- [ ] Round-trip verification (DSL → execute → compare with original)

### Phase 3: DSL → Execution
- [ ] OCL interpreter in odata-mcp
- [ ] MCP tool generation
- [ ] State management for workflows

### Phase 4: DSL → ABAP
- [ ] ABAP code generator
- [ ] BAPI/RFC integration patterns
- [ ] Transport request generation

### Phase 5: Enterprise Features
- [ ] Workflow versioning
- [ ] Approval workflows for workflow changes
- [ ] Audit logging
- [ ] Performance monitoring

---

## The Compact Promise (Revisited)

| Component | Size | Purpose |
|-----------|------|---------|
| Fiori Automator | ~200KB | Chrome extension |
| OCL Parser | ~3K LOC | Parse DSL |
| OCL Executor | ~5K LOC | Run workflows |
| OCL→ABAP | ~2K LOC | Generate ABAP |
| **Total** | **~10K LOC** | **Full pipeline** |

Compare to:
- CAP: ~200MB+ runtime, restrictive license
- SAP BPA: Complex setup, expensive
- Custom ABAP: Months of development per workflow

---

## Summary

> **Record a Fiori session. Get a workflow that runs anywhere.**

The pipeline:
1. **Capture** with Fiori Automator (existing)
2. **Analyze** with AI (identify operations, assertions, fallbacks)
3. **Compile** to OCL (structured, portable DSL)
4. **Execute** via odata-mcp OR transpile to ABAP

All open source. All AI-friendly. All yours.

---

*Concept: February 2026*
*Status: Architecture defined, ready for Phase 0 completion*
