# CONCEPT: OData Composer Language (OCL)

## The Vision: "0 Notifications"

From Alice's Fiori article:
> "9 AM Monday. CFO pings you: Need the latest P&L for company 1000 — compare this quarter to last, right now."

**The problem:** This requires data from multiple systems, aggregations, comparisons - things that today require:
- Custom ABAP reports
- BW queries
- Excel gymnastics
- Or... 23 clicks through Fiori

**The vision:** A declarative language that lets AI (or humans) compose queries across OData services without building new services.

---

## What Is OData Composer Language?

A **stateless DSL** that sits on top of the OData-MCP bridge, enabling:

1. **Cross-service queries** - Join data from SAP, Dynamics, Salesforce
2. **Aggregations** - SUM, AVG, COUNT, GROUP BY across entities
3. **Temporal comparisons** - This quarter vs last quarter
4. **Calculated fields** - Margins, deltas, percentages
5. **Filtering chains** - Complex business logic without code

All **read-only**, **stateless**, and **composable**.

---

## Syntax Concept (Strawman)

### Basic Query
```ocl
FROM sap.Products
WHERE Category = 'Electronics' AND Price > 100
SELECT ProductName, Price, Stock
ORDER BY Price DESC
LIMIT 10
```

### Cross-Service Join
```ocl
FROM sap.Orders AS o
JOIN dynamics.Customers AS c ON o.CustomerID = c.ID
WHERE o.OrderDate >= @ThisQuarter
SELECT c.Name, o.OrderID, o.Total
```

### Aggregation
```ocl
FROM sap.SalesOrderItems
WHERE CreatedAt >= @LastMonth
GROUP BY ProductCategory
SELECT
  ProductCategory,
  SUM(Amount) AS TotalSales,
  COUNT(*) AS OrderCount,
  AVG(Amount) AS AvgOrderValue
```

### Temporal Comparison (The CFO Query!)
```ocl
COMPARE
  (FROM sap.FinancialStatements
   WHERE CompanyCode = '1000' AND Period = @ThisQuarter)
WITH
  (FROM sap.FinancialStatements
   WHERE CompanyCode = '1000' AND Period = @LastQuarter)
ON Account
CALCULATE
  Delta = Current.Amount - Previous.Amount,
  DeltaPercent = (Current.Amount - Previous.Amount) / Previous.Amount * 100
```

### Conditional Aggregation
```ocl
FROM sap.Customers AS c
LEFT JOIN sap.Orders AS o ON c.ID = o.CustomerID
WHERE c.Segment = 'Enterprise'
GROUP BY c.ID, c.Name
SELECT
  c.Name,
  COUNT(o.OrderID) AS OrderCount,
  SUM(o.Total) AS TotalRevenue,
  CASE
    WHEN COUNT(o.OrderID) = 0 THEN 'Churned'
    WHEN MAX(o.OrderDate) < @ThreeMonthsAgo THEN 'At Risk'
    ELSE 'Active'
  END AS Status
HAVING TotalRevenue > 10000 OR Status = 'Churned'
```

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      OCL Query                               │
│  "FROM sap.Orders JOIN dynamics.Customers..."               │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   OCL Parser & Planner                       │
│  - Parse DSL                                                 │
│  - Build execution plan                                      │
│  - Optimize (push-down where possible)                       │
└─────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  OData-MCP      │ │  OData-MCP      │ │  OData-MCP      │
│  (SAP)          │ │  (Dynamics)     │ │  (Salesforce)   │
└─────────────────┘ └─────────────────┘ └─────────────────┘
              │               │               │
              ▼               ▼               ▼
┌─────────────────┐ ┌─────────────────┐ ┌─────────────────┐
│  SAP S/4HANA    │ │  Dynamics 365   │ │  Salesforce     │
└─────────────────┘ └─────────────────┘ └─────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   OCL Executor                               │
│  - Fetch data from each source                              │
│  - Join in memory (stateless)                               │
│  - Apply aggregations                                       │
│  - Return unified result                                    │
└─────────────────────────────────────────────────────────────┘
```

---

## Key Design Principles

### 1. Stateless
- No persistent storage
- No caching (or optional ephemeral cache)
- Every query is fresh
- No ETL, no data warehousing

### 2. Read-Only (Initially)
- Phase 1: Only queries
- Phase 2: Mutations with transaction semantics (harder)

### 3. Push-Down Optimization
```ocl
-- This filter can be pushed to SAP
FROM sap.Orders WHERE Status = 'Open'

-- This filter CANNOT be pushed (cross-service)
FROM sap.Orders AS o
JOIN dynamics.Customers AS c ON o.CustomerID = c.ID
WHERE c.Region = 'EMEA'  -- Must fetch first, then filter
```

### 4. Lazy Evaluation
- Don't fetch all data
- Use pagination/streaming
- Only materialize what's needed

### 5. Schema Discovery
```ocl
-- Explore available services
DESCRIBE SERVICES

-- Explore entities in a service
DESCRIBE sap.*

-- Explore fields in an entity
DESCRIBE sap.Orders
```

---

## Integration with MCP

### New Tool: `odata_compose`

```json
{
  "name": "odata_compose",
  "description": "Execute OCL queries across OData services",
  "parameters": {
    "query": {
      "type": "string",
      "description": "OCL query to execute"
    },
    "services": {
      "type": "object",
      "description": "Map of service aliases to OData URLs"
    }
  }
}
```

### Example MCP Call

```json
{
  "action": "compose",
  "query": "FROM sap.Orders WHERE Total > 1000 SELECT CustomerID, Total",
  "services": {
    "sap": "https://my-sap.com/odata/",
    "dynamics": "https://my-dynamics.com/odata/"
  }
}
```

---

## Use Cases

### 1. CFO Dashboard (The 0 Notifications Dream)
```ocl
-- Morning briefing: everything that needs attention
UNION ALL (
  -- Overdue invoices
  FROM sap.Invoices WHERE DueDate < @Today AND Status = 'Open',

  -- High-value orders pending approval
  FROM sap.Orders WHERE Status = 'Pending' AND Total > 50000,

  -- Customer escalations
  FROM servicenow.Incidents WHERE Priority = 'P1' AND Status = 'Open'
)
ORDER BY Priority, Amount DESC
```

### 2. Cross-System Customer 360
```ocl
FROM sap.Customers AS c
LEFT JOIN dynamics.Opportunities AS o ON c.ID = o.CustomerID
LEFT JOIN servicenow.Tickets AS t ON c.ID = t.CustomerID
GROUP BY c.ID, c.Name
SELECT
  c.Name,
  SUM(o.Value) AS PipelineValue,
  COUNT(t.ID) AS OpenTickets,
  MAX(t.Priority) AS HighestPriority
```

### 3. Inventory Optimization
```ocl
FROM sap.Materials AS m
JOIN sap.Stock AS s ON m.ID = s.MaterialID
JOIN sap.SalesHistory AS h ON m.ID = h.MaterialID
WHERE h.Date >= @Last90Days
GROUP BY m.ID, m.Name, s.Quantity
SELECT
  m.Name,
  s.Quantity AS CurrentStock,
  AVG(h.DailySales) AS AvgDailySales,
  s.Quantity / AVG(h.DailySales) AS DaysOfStock
HAVING DaysOfStock < 14
ORDER BY DaysOfStock ASC
```

---

## Implementation Phases

### Phase 1: Single-Service Aggregations
- Parser for basic OCL syntax
- Aggregations on single OData service
- Push-down optimization for filters
- Temporal variables (@Today, @ThisMonth, etc.)

### Phase 2: Cross-Service Joins
- Multiple service registration
- In-memory hash joins
- Result streaming

### Phase 3: Advanced Features
- Window functions
- CTEs (Common Table Expressions)
- Parameterized queries (stored procedures equivalent)
- Query optimization hints

### Phase 4: Mutations (Careful!)
- Transaction boundaries
- Rollback semantics
- Cross-service transactions (saga pattern?)

---

## Why This Matters

### The Current State
```
User → AI → MCP → OData → One Entity at a Time
```

AI has to:
1. Understand the question
2. Figure out which entities
3. Make multiple calls
4. Join in its context
5. Aggregate manually
6. Hope it fits in context window

### The OCL State
```
User → AI → OCL → Optimized Multi-Source Query → Result
```

AI only needs to:
1. Understand the question
2. Write one OCL query
3. Get the answer

**Token savings:** Instead of 10 tool calls with intermediate results, 1 query.

**Accuracy:** Instead of AI doing math, the engine does it.

**Speed:** Parallel fetches, optimized execution.

---

## Open Questions

1. **Authentication** - How to handle credentials for multiple services?
2. **Rate limiting** - Cross-service queries could hammer backends
3. **Schema conflicts** - Same field names, different meanings
4. **Error handling** - What if one service fails mid-query?
5. **Timeouts** - Large cross-service joins could take forever
6. **Governance** - Who approves cross-system queries?

---

## Competition / Prior Art

- **GraphQL Federation** - Similar concept, different protocol
- **Presto/Trino** - SQL over multiple data sources (heavy)
- **Microsoft Fabric** - Unified analytics (requires data copy)
- **SAP BW/4HANA** - Aggregations, but SAP-only and heavy

**OCL differentiator:** Stateless, OData-native, AI-first, zero infrastructure.

---

## Next Steps

1. [ ] Define formal grammar (EBNF)
2. [ ] Build parser (Go, using parser combinator or ANTLR)
3. [ ] Implement single-service aggregations
4. [ ] Add to odata-mcp as `--composer` mode
5. [ ] Test with real enterprise scenarios
6. [ ] Write "OData Composer: The Missing Link" article

---

## Tagline Ideas

- "SQL for the Enterprise Cloud"
- "GraphQL met OData. They had a baby."
- "The query language enterprises deserve"
- "Zero ETL. Zero infrastructure. All answers."
- "What if you could query your entire enterprise in one statement?"

---

*Draft: February 2026*
*Status: Concept / Brainstorm*
