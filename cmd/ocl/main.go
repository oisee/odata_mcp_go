// OCL - OData Composer Language CLI
//
// Usage:
//   ocl parse <file.ocl>     Parse and validate OCL file
//   ocl parse -              Parse from stdin
//   ocl repl                 Interactive REPL
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/zmcp/odata-mcp/internal/ocl"
	"github.com/zmcp/odata-mcp/internal/ocl/ast"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "parse":
		if len(os.Args) < 3 {
			fmt.Println("Usage: ocl parse <file.ocl> or ocl parse -")
			os.Exit(1)
		}
		if os.Args[2] == "-" {
			parseStdin()
		} else {
			parseFile(os.Args[2])
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
  ocl repl               Interactive REPL

Examples:
  echo "SELECT * FROM Orders" | ocl parse -
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
