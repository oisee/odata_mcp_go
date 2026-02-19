# OCL Development Session Notes

**Date:** 2026-02-19
**Branch:** `ocl`
**Status:** Parser + Executor working against live OData services

## What Was Built

### 1. Lexer (`internal/ocl/lexer/`)
- **token.go** - 60+ token types (keywords, operators, literals)
- **lexer.go** - Tokenizer with line/column tracking, comment support
- Case-sensitive keywords (UPPERCASE only) - allows `update` as identifier

### 2. AST (`internal/ocl/ast/`)
- **ast.go** - Complete AST node definitions:
  - Query: SELECT, FROM, JOIN, WHERE, ORDER BY, GROUP BY, HAVING
  - CRUD: CREATE, UPDATE, DELETE, READ
  - Workflow: WORKFLOW, STEP, parameters
  - Control flow: IF/THEN/ELSE, FOREACH, RETRY
  - Assertions: ASSERT, EXPECT, FALLBACK, ROLLBACK
  - Transactions: BEGIN, COMMIT
  - Config: SERVICE, EXPOSE

### 3. Parser (`internal/ocl/parser/`)
- **parser.go** - Recursive descent parser with Pratt expression parsing
- Handles complex nested expressions
- Good error messages with line/column

### 4. Executor (`internal/ocl/executor/`)
- **executor.go** - Executes OCL against real OData services:
  - Query execution with $filter, $select, $orderby, $top, $skip
  - CREATE/UPDATE/DELETE operations
  - Variable substitution
  - OData v2 and v4 response parsing
- **auth.go** - Authentication support:
  - Basic auth
  - Bearer token
  - API key (custom header)
- **config.go** - YAML config loading:
  - Service URLs and auth settings
  - Environment variable expansion (${VAR})
  - Auto-discovery from .ocl file directory

### 5. CLI (`cmd/ocl/`)
- `ocl parse <file>` - Parse and validate
- `ocl run <file>` - Execute against live services
- `ocl run -` - Execute from stdin
- `ocl repl` - Interactive REPL
- `-v` flag for verbose output

## Test Services

| Service | URL | Auth | Status |
|---------|-----|------|--------|
| Northwind | services.odata.org/V4/Northwind/ | None | ✅ Tested |
| TripPin | services.odata.org/TripPinRESTierService/ | None | Ready |
| SAP ES5 | sapes5.sapdevcenter.com | Basic | Ready (needs account) |

## Current Limitations

1. **Field access on variables** - `$item.ID` not yet supported
2. **JOINs** - Simplified $expand, needs metadata for nav property mapping
3. **Workflows** - Parsed but execution not implemented
4. **Transactions** - Parsed but not executed
5. **ABAP transpiler** - Not started

## Example Working Query

```sql
SERVICE Northwind "https://services.odata.org/V4/Northwind/Northwind.svc"

SELECT ProductName, UnitPrice
FROM Northwind.Products
WHERE UnitPrice > 50
ORDER BY UnitPrice DESC
LIMIT 5
```

Output:
```
✓ 5 result(s) in 862ms
  [1] { "ProductName": "Côte de Blaye", "UnitPrice": 263.5 }
  [2] { "ProductName": "Thüringer Rostbratwurst", "UnitPrice": 123.79 }
  ...
```

## Files Structure

```
internal/ocl/
├── ast/
│   └── ast.go           # AST node definitions
├── lexer/
│   ├── token.go         # Token types
│   ├── lexer.go         # Tokenizer
│   └── lexer_test.go
├── parser/
│   ├── parser.go        # Parser implementation
│   └── parser_test.go
├── executor/
│   ├── executor.go      # Query/CRUD execution
│   ├── auth.go          # Authentication
│   └── config.go        # Config file loading
├── ocl.go               # Main package entry
├── ocl_test.go
└── SESSION-NOTES.md     # This file

cmd/ocl/
└── main.go              # CLI tool

examples/ocl/
├── README.md            # Testing guide
├── ocl.config.yaml      # Example config
├── northwind-queries.ocl
├── trippin-crud.ocl
└── sap-es5-demo.ocl
```

## Next Steps

1. **Field access** - Support `$item.ID` in expressions
2. **Workflow execution** - Execute STEP sequences with variable binding
3. **Better JOINs** - Fetch metadata to map navigation properties
4. **ABAP transpiler** - Generate native ABAP from OCL workflows
5. **Capture integration** - Parse Fiori Automator JSON → OCL

## Design Decisions

- **Hand-written parser** (not Participle/ANTLR) - Full control, better errors
- **Case-sensitive keywords** - UPPERCASE only, allows lowercase identifiers
- **YAML config** - Simple, supports env vars, familiar format
- **Separate executor** - Parser is pure, executor handles I/O
