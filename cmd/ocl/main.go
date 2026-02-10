// OCL - OData Composer Language CLI
//
// Usage:
//   ocl parse <file.ocl>     Parse and validate OCL file
//   ocl parse -              Parse from stdin
//   ocl run <file.ocl>       Parse and execute OCL file
//   ocl repl                 Interactive REPL
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/zmcp/odata-mcp/internal/ocl"
	"github.com/zmcp/odata-mcp/internal/ocl/ast"
	"github.com/zmcp/odata-mcp/internal/ocl/executor"
)

var verbose bool

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Check for -v flag
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "-v" || arg == "--verbose" {
			verbose = true
			args = append(args[:i], args[i+1:]...)
			break
		}
	}

	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	switch args[0] {
	case "parse":
		if len(args) < 2 {
			fmt.Println("Usage: ocl parse <file.ocl> or ocl parse -")
			os.Exit(1)
		}
		if args[1] == "-" {
			parseStdin()
		} else {
			parseFile(args[1])
		}
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: ocl run <file.ocl>")
			os.Exit(1)
		}
		if args[1] == "-" {
			runStdin()
		} else {
			runFile(args[1])
		}
	case "repl":
		runREPL()
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`OCL - OData Composer Language v` + ocl.Version + `

Usage:
  ocl parse <file.ocl>   Parse and validate an OCL file
  ocl parse -            Parse from stdin
  ocl run <file.ocl>     Parse and execute an OCL file
  ocl run -v <file.ocl>  Execute with verbose output
  ocl repl               Interactive REPL

Examples:
  echo "SELECT * FROM Orders" | ocl parse -
  ocl run examples/ocl/northwind-queries.ocl
  ocl repl`)
}

func parseStdin() {
	var input strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input.WriteString(scanner.Text())
		input.WriteString("\n")
	}
	parseAndPrint(input.String())
}

func parseFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	parseAndPrint(string(data))
}

func runStdin() {
	var input strings.Builder
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input.WriteString(scanner.Text())
		input.WriteString("\n")
	}
	runSource(input.String(), "")
}

func runFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
	}
	runSource(string(data), path)
}

func runSource(source, configHint string) {
	prog, err := ocl.Parse(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Parsed %d statement(s)\n\n", len(prog.AST.Statements))

	// Create executor
	exec := executor.New()
	exec.SetVerbose(verbose)

	// Load config if available
	if configPath := executor.FindConfig(configHint); configPath != "" {
		config, err := executor.LoadConfig(configPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: config load error: %v\n", err)
		} else {
			exec.ApplyConfig(config)
			if verbose {
				fmt.Printf("✓ Loaded config from %s\n\n", configPath)
			}
		}
	}

	ctx := context.Background()

	// Execute each statement
	for i, stmt := range prog.AST.Statements {
		fmt.Printf("─── Statement %d: %s ───\n", i+1, describeStatement(stmt))

		result, err := exec.Execute(ctx, stmt)
		if err != nil {
			fmt.Printf("✗ Error: %v\n\n", err)
			continue
		}

		printResult(result)
		fmt.Println()
	}
}

func printResult(r *executor.Result) {
	switch r.Type {
	case "service":
		fmt.Printf("✓ %v\n", r.Data)

	case "query":
		fmt.Printf("✓ %d result(s) in %v\n", r.Count, r.Elapsed)
		if verbose {
			fmt.Printf("  URL: %s\n", r.URL)
		}
		if results, ok := r.Data.([]any); ok && len(results) > 0 {
			// Pretty print first few results
			limit := 5
			if len(results) < limit {
				limit = len(results)
			}
			for i := 0; i < limit; i++ {
				jsonData, _ := json.MarshalIndent(results[i], "  ", "  ")
				fmt.Printf("  [%d] %s\n", i+1, string(jsonData))
			}
			if len(results) > limit {
				fmt.Printf("  ... and %d more\n", len(results)-limit)
			}
		}

	case "create", "update":
		fmt.Printf("✓ %s completed in %v\n", r.Type, r.Elapsed)
		if r.Data != nil {
			jsonData, _ := json.MarshalIndent(r.Data, "  ", "  ")
			fmt.Printf("  %s\n", string(jsonData))
		}

	case "delete":
		fmt.Printf("✓ Deleted in %v\n", r.Elapsed)

	default:
		fmt.Printf("✓ %s: %v\n", r.Type, r.Data)
	}
}

func parseAndPrint(source string) {
	prog, err := ocl.Parse(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse error:\n%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Parsed %d statement(s)\n\n", len(prog.AST.Statements))

	for i, stmt := range prog.AST.Statements {
		fmt.Printf("[%d] %s\n", i+1, describeStatement(stmt))
	}

	// Validate
	errors := prog.Validate()
	if len(errors) > 0 {
		fmt.Println("\nValidation warnings:")
		for _, e := range errors {
			fmt.Printf("  ⚠ %v\n", e)
		}
	}
}

func describeStatement(stmt ast.Statement) string {
	switch s := stmt.(type) {
	case *ast.QueryStatement:
		desc := "QUERY"
		if s.From != nil {
			if s.From.Service != "" {
				desc += fmt.Sprintf(" from %s.%s", s.From.Service, s.From.Entity)
			} else {
				desc += fmt.Sprintf(" from %s", s.From.Entity)
			}
		}
		if len(s.Fields) > 0 {
			desc += fmt.Sprintf(" (%d fields)", len(s.Fields))
		}
		if s.Where != nil {
			desc += " [filtered]"
		}
		if len(s.Joins) > 0 {
			desc += fmt.Sprintf(" [%d joins]", len(s.Joins))
		}
		if s.Limit > 0 {
			desc += fmt.Sprintf(" LIMIT %d", s.Limit)
		}
		return desc

	case *ast.WorkflowStatement:
		desc := fmt.Sprintf("WORKFLOW %s", s.Name)
		if len(s.Parameters) > 0 {
			params := make([]string, len(s.Parameters))
			for i, p := range s.Parameters {
				params[i] = p.Name
			}
			desc += fmt.Sprintf("(%s)", strings.Join(params, ", "))
		}
		desc += fmt.Sprintf(" [%d steps]", countSteps(s.Steps))
		return desc

	case *ast.ServiceStatement:
		return fmt.Sprintf("SERVICE %s = %q", s.Name, s.URL)

	case *ast.ExposeStatement:
		return fmt.Sprintf("EXPOSE %s AS %s", s.Workflow, s.As)

	case *ast.CreateStatement:
		entity := s.Entity
		if s.Service != "" {
			entity = s.Service + "." + entity
		}
		return fmt.Sprintf("CREATE %s (%d fields)", entity, len(s.Fields))

	case *ast.UpdateStatement:
		entity := s.Entity
		if s.Service != "" {
			entity = s.Service + "." + entity
		}
		return fmt.Sprintf("UPDATE %s (%d fields)", entity, len(s.Fields))

	case *ast.DeleteStatement:
		entity := s.Entity
		if s.Service != "" {
			entity = s.Service + "." + entity
		}
		return fmt.Sprintf("DELETE %s", entity)

	case *ast.StepStatement:
		return fmt.Sprintf("STEP %s", s.Name)

	default:
		return fmt.Sprintf("%T", stmt)
	}
}

func countSteps(stmts []ast.Statement) int {
	count := 0
	for _, s := range stmts {
		if _, ok := s.(*ast.StepStatement); ok {
			count++
		}
	}
	return count
}

func runREPL() {
	fmt.Println("OCL REPL v" + ocl.Version)
	fmt.Println("Type OCL statements. Use Ctrl+D to exit, '\\' for multiline.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var multiline strings.Builder
	inMultiline := false

	for {
		if inMultiline {
			fmt.Print("...> ")
		} else {
			fmt.Print("ocl> ")
		}

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()

		// Handle multiline input
		if strings.HasSuffix(line, "\\") {
			multiline.WriteString(strings.TrimSuffix(line, "\\"))
			multiline.WriteString("\n")
			inMultiline = true
			continue
		}

		if inMultiline {
			multiline.WriteString(line)
			line = multiline.String()
			multiline.Reset()
			inMultiline = false
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Special commands
		if line == "help" || line == "?" {
			printREPLHelp()
			continue
		}
		if line == "exit" || line == "quit" {
			break
		}

		// Parse the input
		prog, err := ocl.Parse(line)
		if err != nil {
			fmt.Printf("Error: %v\n\n", err)
			continue
		}

		fmt.Printf("✓ Parsed %d statement(s)\n", len(prog.AST.Statements))
		for i, stmt := range prog.AST.Statements {
			fmt.Printf("  [%d] %s\n", i+1, describeStatement(stmt))
		}
		fmt.Println()
	}

	fmt.Println("\nBye!")
}

func printREPLHelp() {
	fmt.Println(`
OCL REPL Commands:
  help, ?     Show this help
  exit, quit  Exit the REPL
  \           Continue on next line (multiline input)

Example OCL:
  SELECT * FROM Orders WHERE Status == 'OPEN' LIMIT 10

  SELECT po.ID, items.Amount \
  FROM MM.PurchaseOrders AS po \
  JOIN MM.Items AS items ON po.ID == items.OrderID \
  WHERE po.Status == 'OPEN'

  WORKFLOW UpdateOrder($orderId, $status)
    STEP update:
      UPDATE Orders SET Status = $status WHERE ID == $orderId
`)
}
