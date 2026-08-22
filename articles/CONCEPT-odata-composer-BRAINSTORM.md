# OData Composer Language - Deep Brainstorm

## The Core Insight

From the Fiori article comments, John Patterson made a critical point:
> "Fiori OData services are built for humans - paging, inline counts, UI annotations... When you run automation through those, you're pushing AI into an interface meant for eyeballs, not intent."

**The gap:** OData services are designed for UI consumption, not for analytical queries.

**The opportunity:** A composition layer that transforms intent into optimized multi-source queries.

---

## Problem Statement (Refined)

### Today's Pain Points

1. **Fragmented Data**
   - SAP has financials
   - Dynamics has CRM
   - ServiceNow has tickets
   - Each query is isolated

2. **AI Context Explosion**
   - AI makes 10 calls to get related data
   - Intermediate results consume tokens
   - Math/aggregations done in context (error-prone)
   - Multiple round-trips = latency

3. **No Aggregation Layer**
   - OData supports $filter, $select, $expand
   - But no GROUP BY, SUM, JOIN across entities
   - OData v4 has $apply but adoption is spotty

4. **Temporal Blindness**
   - "Compare this quarter to last" is simple ask
   - Requires manual calculation of date ranges
   - Then two queries, then diff logic

---

## Design Goals

### Must Have
- [ ] Stateless execution (no persistence)
- [ ] Works with existing OData services (no backend changes)
- [ ] Single query → single result (no multi-turn)
- [ ] Push-down optimization where possible
- [ ] Fail gracefully (partial results on errors)

### Should Have
- [ ] Schema discovery/introspection
- [ ] Parameterized queries (reusable templates)
- [ ] Caching hints (for repeated queries)
- [ ] Explain plan (show execution strategy)

### Nice to Have
- [ ] Materialized views (ephemeral, time-limited)
- [ ] Query history/versioning
- [ ] Cost estimation

---

## Syntax Deep Dive

### Why Not Just SQL?

SQL is familiar, but:
- Assumes single database
- No concept of "services"
- No native temporal operators
- No service discovery

### Why Not GraphQL?

GraphQL is modern, but:
- Requires schema stitching server
- No aggregations
- Verbose for analytics
- Different mental model

### OCL Design Principles

1. **SQL-like** - Familiar to enterprise developers
2. **Service-aware** - First-class concept of data sources
3. **Temporal-native** - Built-in period operators
4. **AI-friendly** - Easy to generate programmatically

---

## Grammar Sketch (EBNF)

```ebnf
query       = select_query | compare_query | describe_query | union_query ;

select_query =
    "FROM" source_ref [ "AS" alias ]
    { "JOIN" source_ref [ "AS" alias ] "ON" condition }
    [ "WHERE" condition ]
    [ "GROUP BY" field_list ]
    "SELECT" select_list
    [ "HAVING" condition ]
    [ "ORDER BY" order_list ]
    [ "LIMIT" number ] ;

source_ref  = service_name "." entity_name ;
service_name = identifier ;
entity_name  = identifier ;

compare_query =
    "COMPARE" "(" select_query ")" [ "AS" "Current" ]
    "WITH" "(" select_query ")" [ "AS" "Previous" ]
    "ON" field_list
    "CALCULATE" calc_list ;

union_query =
    "UNION" [ "ALL" ] "(" query { "," query } ")" ;

describe_query =
    "DESCRIBE" ( "SERVICES" | service_ref | source_ref ) ;

condition   = expr ( "AND" | "OR" ) expr | "NOT" condition | "(" condition ")" ;
expr        = field_ref ( "=" | "!=" | ">" | "<" | ">=" | "<=" | "LIKE" | "IN" ) value ;

select_list = select_item { "," select_item } ;
select_item = ( field_ref | aggregate_fn | case_expr ) [ "AS" alias ] ;

aggregate_fn = ( "SUM" | "AVG" | "COUNT" | "MIN" | "MAX" ) "(" field_ref ")" ;

case_expr   = "CASE" { "WHEN" condition "THEN" value } [ "ELSE" value ] "END" ;

field_ref   = [ alias "." ] field_name ;
temporal    = "@" ( "Today" | "Yesterday" | "ThisWeek" | "LastWeek"
                  | "ThisMonth" | "LastMonth" | "ThisQuarter" | "LastQuarter"
                  | "ThisYear" | "LastYear" | "Last" number "Days" ) ;
```

---

## Execution Engine Design

### Query Planning

```
OCL Query
    │
    ▼
┌─────────────────────┐
│  Parser             │ → AST
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│  Analyzer           │ → Typed AST + Schema Info
│  - Resolve aliases  │
│  - Type check       │
│  - Validate fields  │
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│  Optimizer          │ → Execution Plan
│  - Push-down        │
│  - Join ordering    │
│  - Parallelization  │
└─────────────────────┘
    │
    ▼
┌─────────────────────┐
│  Executor           │ → Results
│  - Parallel fetch   │
│  - Stream join      │
│  - Aggregate        │
└─────────────────────┘
```

### Push-Down Rules

| Operation | Push-Down? | Notes |
|-----------|------------|-------|
| `WHERE field = value` | ✅ Yes | → `$filter=field eq value` |
| `SELECT field1, field2` | ✅ Yes | → `$select=field1,field2` |
| `ORDER BY field` | ⚠️ Maybe | Only if single source, no aggregation |
| `LIMIT n` | ⚠️ Maybe | → `$top=n` only if no post-processing |
| `SUM(field)` | ❌ No* | OData v4 `$apply` only |
| `JOIN` | ❌ No | Always in-memory |
| `HAVING` | ❌ No | Post-aggregation filter |

### Join Strategies

1. **Nested Loop** - Simple, O(n*m), good for small datasets
2. **Hash Join** - Build hash table, O(n+m), memory-intensive
3. **Sort-Merge** - Pre-sort, O(n log n + m log m), good for ordered data
4. **Broadcast** - Small table to all executors (if distributed)

For OCL Phase 1: **Hash Join** with configurable memory limits.

---

## Service Registry

### Configuration
```yaml
services:
  sap:
    url: https://my-sap.com/sap/opu/odata/sap/API_BUSINESS_PARTNER
    auth:
      type: basic
      username: ${SAP_USER}
      password: ${SAP_PASS}

  dynamics:
    url: https://my-org.crm.dynamics.com/api/data/v9.2
    auth:
      type: oauth2
      tenant: ${AZURE_TENANT}
      client_id: ${AZURE_CLIENT}
      client_secret: ${AZURE_SECRET}

  servicenow:
    url: https://my-instance.service-now.com/api/now/table
    auth:
      type: basic
      username: ${SNOW_USER}
      password: ${SNOW_PASS}
```

### Runtime Discovery
```ocl
DESCRIBE SERVICES
-- Returns:
-- | Service    | URL                              | Entities |
-- |------------|----------------------------------|----------|
-- | sap        | https://my-sap.com/...           | 156      |
-- | dynamics   | https://my-org.crm.dynamics.com  | 89       |
-- | servicenow | https://my-instance.service-now  | 42       |
```

---

## MCP Integration Options

### Option A: New Tool
```json
{
  "name": "odata_compose",
  "parameters": {
    "query": "FROM sap.Orders WHERE Total > 1000..."
  }
}
```

**Pros:** Clean separation, explicit
**Cons:** Yet another tool, token overhead

### Option B: Universal Tool Extension
```json
{
  "action": "compose",
  "query": "FROM sap.Orders...",
  "target": null  // Not needed for compose
}
```

**Pros:** Reuses existing universal tool pattern
**Cons:** Overloads the tool

### Option C: Separate MCP Server
```
odata-composer-mcp --config services.yaml
```

**Pros:** Clean architecture, independent scaling
**Cons:** Another server to deploy

### Recommendation: **Option C** for production, **Option B** for quick wins.

---

## Security Considerations

### Query Injection
```ocl
-- User input: "'; DROP TABLE Users; --"
FROM sap.Orders WHERE CustomerName = '{{ user_input }}'
```

**Mitigation:** Parameterized queries only
```ocl
FROM sap.Orders WHERE CustomerName = $customer_name
-- Parameters passed separately: {"customer_name": "..."}
```

### Cross-System Data Leakage
```ocl
-- Should HR salary data be joinable with CRM?
FROM hr.Employees AS e
JOIN crm.Opportunities AS o ON e.ID = o.OwnerID
```

**Mitigation:** Service-level ACLs in registry
```yaml
services:
  hr:
    acl:
      allow_join_with: [hr]  # Only join with itself
```

### Query Cost Attacks
```ocl
-- Cartesian product of two large tables
FROM sap.Orders
JOIN dynamics.Customers  -- No ON clause = full cross join
```

**Mitigation:**
- Require explicit ON clause for JOINs
- Query cost estimator with limits
- Timeout enforcement

---

## Performance Considerations

### Pagination Strategy
```
Problem: sap.Orders has 10M rows
Solution: Stream with cursor

Query:
FROM sap.Orders WHERE Year = 2025
JOIN sap.Customers ON ...

Execution:
1. Fetch Orders page 1 (1000 rows)
2. Extract unique CustomerIDs
3. Fetch matching Customers
4. Join in memory
5. Emit partial results
6. Repeat for page 2...
```

### Caching Strategy
```
Levels:
1. Schema cache (long TTL) - entity definitions
2. Reference data cache (medium TTL) - countries, currencies
3. Query result cache (short TTL, opt-in) - explicit CACHE hint

CACHE FOR 5 MINUTES
FROM sap.ExchangeRates
WHERE ValidFrom <= @Today AND ValidTo >= @Today
```

### Parallel Execution
```
FROM sap.Orders AS o
JOIN dynamics.Customers AS c ON o.CustomerID = c.ID
JOIN servicenow.Tickets AS t ON c.ID = t.CustomerID

Execution:
├── Thread 1: Fetch sap.Orders
├── Thread 2: Fetch dynamics.Customers
└── Thread 3: Fetch servicenow.Tickets
    │
    ▼
Barrier: Wait for all
    │
    ▼
Join: Orders ⋈ Customers ⋈ Tickets
```

---

## Roadmap

### Phase 1: Foundation (v0.1)
- [ ] OCL parser (Go)
- [ ] Single-service queries
- [ ] Basic aggregations (SUM, COUNT, AVG)
- [ ] Temporal variables
- [ ] Integration with odata-mcp

### Phase 2: Multi-Source (v0.2)
- [ ] Service registry
- [ ] Cross-service JOINs
- [ ] Hash join implementation
- [ ] Query explain

### Phase 3: Optimization (v0.3)
- [ ] Push-down optimization
- [ ] Query cost estimation
- [ ] Parallel execution
- [ ] Result streaming

### Phase 4: Advanced (v0.4)
- [ ] Window functions
- [ ] CTEs
- [ ] Parameterized queries (procedures)
- [ ] Query caching

### Phase 5: Enterprise (v1.0)
- [ ] ACLs and governance
- [ ] Audit logging
- [ ] Query history
- [ ] Admin UI

---

## Competitive Analysis

| Feature | OCL | Presto | GraphQL | SAP BW |
|---------|-----|--------|---------|--------|
| Stateless | ✅ | ❌ | ✅ | ❌ |
| No infrastructure | ✅ | ❌ | ❌ | ❌ |
| OData native | ✅ | ❌ | ❌ | ⚠️ |
| Aggregations | ✅ | ✅ | ❌ | ✅ |
| Cross-source | ✅ | ✅ | ✅ | ❌ |
| AI-friendly | ✅ | ⚠️ | ⚠️ | ❌ |
| Zero setup | ✅ | ❌ | ❌ | ❌ |

---

## Open Research Questions

1. **Query Equivalence** - Is OCL query A equivalent to query B? (optimization correctness)

2. **Cardinality Estimation** - Without statistics, how to estimate result sizes?

3. **Distributed Transactions** - If we add mutations, how to ensure consistency?

4. **Schema Evolution** - What happens when underlying OData schema changes?

5. **Query Federation** - Can OCL queries be themselves federated? (OCL over OCL)

---

## Article Angle: "OData Composer: The Missing Link"

**Hook:**
> "I promised 0 notifications. Here's how to actually get there."

**Story:**
1. The CFO query that started it all
2. Why 10 MCP calls isn't the answer
3. Introducing OCL - one query, all your data
4. Live demo with SAP + Dynamics + ServiceNow
5. The architecture that makes it possible
6. Call for contributors

**Tagline:**
> "What if your entire enterprise was queryable in one statement?"

---

*Brainstorm Session: February 2026*
*Next: Prototype parser, validate with 3 real use cases*
