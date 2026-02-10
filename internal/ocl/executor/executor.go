// Package executor runs OCL programs against OData services
package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/zmcp/odata-mcp/internal/ocl/ast"
)

// Executor runs OCL statements against OData services
type Executor struct {
	client   *http.Client
	services map[string]string       // service name -> URL
	auths    map[string]*ServiceAuth // service name -> auth config
	vars     map[string]any          // runtime variables
	verbose  bool
}

// New creates a new Executor
func New() *Executor {
	return &Executor{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		services: make(map[string]string),
		auths:    make(map[string]*ServiceAuth),
		vars:     make(map[string]any),
	}
}

// SetVerbose enables verbose output
func (e *Executor) SetVerbose(v bool) {
	e.verbose = v
}

// RegisterService registers a service URL
func (e *Executor) RegisterService(name, url string) {
	e.services[name] = strings.TrimSuffix(url, "/")
}

// SetVariable sets a runtime variable
func (e *Executor) SetVariable(name string, value any) {
	e.vars[name] = value
}

// Result represents an execution result
type Result struct {
	Type    string         // "query", "create", "update", "delete", etc.
	Data    any            // Result data
	Count   int            // Number of results
	URL     string         // The OData URL used
	Elapsed time.Duration  // Execution time
	Error   error          // Any error
}

// Execute runs a single statement
func (e *Executor) Execute(ctx context.Context, stmt ast.Statement) (*Result, error) {
	start := time.Now()

	switch s := stmt.(type) {
	case *ast.ServiceStatement:
		e.RegisterService(s.Name, s.URL)
		return &Result{
			Type:    "service",
			Data:    fmt.Sprintf("Registered service %s = %s", s.Name, s.URL),
			Elapsed: time.Since(start),
		}, nil

	case *ast.QueryStatement:
		return e.executeQuery(ctx, s, start)

	case *ast.CreateStatement:
		return e.executeCreate(ctx, s, start)

	case *ast.UpdateStatement:
		return e.executeUpdate(ctx, s, start)

	case *ast.DeleteStatement:
		return e.executeDelete(ctx, s, start)

	default:
		return nil, fmt.Errorf("execution not implemented for %T", stmt)
	}
}

// executeQuery runs a SELECT query
func (e *Executor) executeQuery(ctx context.Context, q *ast.QueryStatement, start time.Time) (*Result, error) {
	// Build OData URL
	odataURL, err := e.buildQueryURL(q)
	if err != nil {
		return nil, err
	}

	if e.verbose {
		fmt.Printf("[VERBOSE] GET %s\n", odataURL)
	}

	// Execute request
	req, err := http.NewRequestWithContext(ctx, "GET", odataURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	// Apply authentication if configured
	if q.From != nil && q.From.Service != "" {
		e.ApplyAuth(req, q.From.Service)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}

	// Extract results (OData v4 uses "value", v2 uses "d.results")
	var results []any
	if value, ok := data["value"].([]any); ok {
		results = value
	} else if d, ok := data["d"].(map[string]any); ok {
		if r, ok := d["results"].([]any); ok {
			results = r
		}
	}

	return &Result{
		Type:    "query",
		Data:    results,
		Count:   len(results),
		URL:     odataURL,
		Elapsed: time.Since(start),
	}, nil
}

// buildQueryURL converts a QueryStatement to an OData URL
func (e *Executor) buildQueryURL(q *ast.QueryStatement) (string, error) {
	if q.From == nil {
		return "", fmt.Errorf("query missing FROM clause")
	}

	// Get base URL
	var baseURL string
	if q.From.Service != "" {
		svcURL, ok := e.services[q.From.Service]
		if !ok {
			return "", fmt.Errorf("unknown service: %s", q.From.Service)
		}
		baseURL = svcURL
	} else {
		// Use first registered service as default
		for _, url := range e.services {
			baseURL = url
			break
		}
	}

	if baseURL == "" {
		return "", fmt.Errorf("no service URL configured")
	}

	// Build entity URL
	entityURL := fmt.Sprintf("%s/%s", baseURL, q.From.Entity)

	// Build query parameters
	params := url.Values{}

	// $select
	if len(q.Fields) > 0 && !isSelectAll(q.Fields) {
		fields := make([]string, 0, len(q.Fields))
		for _, f := range q.Fields {
			fieldName := extractFieldName(f.Expression)
			if fieldName != "" && fieldName != "*" {
				fields = append(fields, fieldName)
			}
		}
		if len(fields) > 0 {
			params.Set("$select", strings.Join(fields, ","))
		}
	}

	// $filter
	if q.Where != nil {
		filter := e.buildFilter(q.Where)
		if filter != "" {
			params.Set("$filter", filter)
		}
	}

	// $orderby
	if len(q.OrderBy) > 0 {
		orderParts := make([]string, len(q.OrderBy))
		for i, o := range q.OrderBy {
			fieldName := extractFieldName(o.Field)
			if o.Descending {
				orderParts[i] = fieldName + " desc"
			} else {
				orderParts[i] = fieldName + " asc"
			}
		}
		params.Set("$orderby", strings.Join(orderParts, ","))
	}

	// $top (LIMIT)
	if q.Limit > 0 {
		params.Set("$top", fmt.Sprintf("%d", q.Limit))
	}

	// $skip (OFFSET)
	if q.Offset > 0 {
		params.Set("$skip", fmt.Sprintf("%d", q.Offset))
	}

	// $expand for JOINs (simplified - real JOINs need more work)
	if len(q.Joins) > 0 {
		expands := make([]string, len(q.Joins))
		for i, j := range q.Joins {
			expands[i] = j.Source.Entity
		}
		params.Set("$expand", strings.Join(expands, ","))
	}

	if len(params) > 0 {
		return entityURL + "?" + params.Encode(), nil
	}
	return entityURL, nil
}

// buildFilter converts an expression to OData $filter syntax
func (e *Executor) buildFilter(expr ast.Expression) string {
	switch ex := expr.(type) {
	case *ast.BinaryExpression:
		left := e.buildFilter(ex.Left)
		right := e.buildFilter(ex.Right)
		op := convertOperator(ex.Operator)
		return fmt.Sprintf("%s %s %s", left, op, right)

	case *ast.Identifier:
		return ex.Value

	case *ast.FieldAccess:
		return extractFieldName(ex)

	case *ast.StringLiteral:
		return fmt.Sprintf("'%s'", ex.Value)

	case *ast.NumberLiteral:
		return ex.Value

	case *ast.BooleanLiteral:
		if ex.Value {
			return "true"
		}
		return "false"

	case *ast.Variable:
		// Substitute variable value
		if val, ok := e.vars[ex.Name]; ok {
			switch v := val.(type) {
			case string:
				return fmt.Sprintf("'%s'", v)
			default:
				return fmt.Sprintf("%v", v)
			}
		}
		return ex.Name

	default:
		return fmt.Sprintf("%v", expr)
	}
}

func convertOperator(op string) string {
	switch op {
	case "==":
		return "eq"
	case "!=":
		return "ne"
	case ">":
		return "gt"
	case "<":
		return "lt"
	case ">=":
		return "ge"
	case "<=":
		return "le"
	case "AND":
		return "and"
	case "OR":
		return "or"
	default:
		return op
	}
}

func extractFieldName(expr ast.Expression) string {
	switch e := expr.(type) {
	case *ast.Identifier:
		return e.Value
	case *ast.FieldAccess:
		// For alias.field, just return field for $select
		return e.Field.Value
	default:
		return ""
	}
}

func isSelectAll(fields []*ast.SelectField) bool {
	if len(fields) == 1 {
		if ident, ok := fields[0].Expression.(*ast.Identifier); ok {
			return ident.Value == "*"
		}
	}
	return false
}

// executeCreate handles CREATE statements
func (e *Executor) executeCreate(ctx context.Context, c *ast.CreateStatement, start time.Time) (*Result, error) {
	// Build URL
	var baseURL string
	if c.Service != "" {
		svcURL, ok := e.services[c.Service]
		if !ok {
			return nil, fmt.Errorf("unknown service: %s", c.Service)
		}
		baseURL = svcURL
	} else {
		for _, url := range e.services {
			baseURL = url
			break
		}
	}

	entityURL := fmt.Sprintf("%s/%s", baseURL, c.Entity)

	// Build body
	body := make(map[string]any)
	for name, expr := range c.Fields {
		body[name] = e.evaluateExpr(expr)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if e.verbose {
		fmt.Printf("[VERBOSE] POST %s\n[VERBOSE] Body: %s\n", entityURL, string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", entityURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var data any
	json.Unmarshal(respBody, &data)

	// Store in variable if specified
	if c.Into != "" {
		e.vars[c.Into] = data
	}

	return &Result{
		Type:    "create",
		Data:    data,
		Count:   1,
		URL:     entityURL,
		Elapsed: time.Since(start),
	}, nil
}

// executeUpdate handles UPDATE statements
func (e *Executor) executeUpdate(ctx context.Context, u *ast.UpdateStatement, start time.Time) (*Result, error) {
	var baseURL string
	if u.Service != "" {
		svcURL, ok := e.services[u.Service]
		if !ok {
			return nil, fmt.Errorf("unknown service: %s", u.Service)
		}
		baseURL = svcURL
	} else {
		for _, url := range e.services {
			baseURL = url
			break
		}
	}

	// Build key for URL
	entityURL := fmt.Sprintf("%s/%s", baseURL, u.Entity)
	if len(u.KeyFields) > 0 {
		keys := make([]string, 0, len(u.KeyFields))
		for name, expr := range u.KeyFields {
			val := e.evaluateExpr(expr)
			switch v := val.(type) {
			case string:
				keys = append(keys, fmt.Sprintf("%s='%s'", name, v))
			default:
				keys = append(keys, fmt.Sprintf("%s=%v", name, v))
			}
		}
		entityURL = fmt.Sprintf("%s(%s)", entityURL, strings.Join(keys, ","))
	}

	// Build body
	body := make(map[string]any)
	for name, expr := range u.Fields {
		body[name] = e.evaluateExpr(expr)
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	if e.verbose {
		fmt.Printf("[VERBOSE] PATCH %s\n[VERBOSE] Body: %s\n", entityURL, string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", entityURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return &Result{
		Type:    "update",
		Data:    string(respBody),
		Count:   1,
		URL:     entityURL,
		Elapsed: time.Since(start),
	}, nil
}

// executeDelete handles DELETE statements
func (e *Executor) executeDelete(ctx context.Context, d *ast.DeleteStatement, start time.Time) (*Result, error) {
	var baseURL string
	if d.Service != "" {
		svcURL, ok := e.services[d.Service]
		if !ok {
			return nil, fmt.Errorf("unknown service: %s", d.Service)
		}
		baseURL = svcURL
	} else {
		for _, url := range e.services {
			baseURL = url
			break
		}
	}

	entityURL := fmt.Sprintf("%s/%s", baseURL, d.Entity)
	if len(d.KeyFields) > 0 {
		keys := make([]string, 0, len(d.KeyFields))
		for name, expr := range d.KeyFields {
			val := e.evaluateExpr(expr)
			switch v := val.(type) {
			case string:
				keys = append(keys, fmt.Sprintf("%s='%s'", name, v))
			default:
				keys = append(keys, fmt.Sprintf("%s=%v", name, v))
			}
		}
		entityURL = fmt.Sprintf("%s(%s)", entityURL, strings.Join(keys, ","))
	}

	if e.verbose {
		fmt.Printf("[VERBOSE] DELETE %s\n", entityURL)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", entityURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return &Result{
		Type:    "delete",
		Count:   1,
		URL:     entityURL,
		Elapsed: time.Since(start),
	}, nil
}

func (e *Executor) evaluateExpr(expr ast.Expression) any {
	switch ex := expr.(type) {
	case *ast.StringLiteral:
		return ex.Value
	case *ast.NumberLiteral:
		return ex.Value
	case *ast.BooleanLiteral:
		return ex.Value
	case *ast.Variable:
		if val, ok := e.vars[ex.Name]; ok {
			return val
		}
		return nil
	case *ast.ArrayLiteral:
		arr := make([]any, len(ex.Elements))
		for i, el := range ex.Elements {
			arr[i] = e.evaluateExpr(el)
		}
		return arr
	default:
		return nil
	}
}
