package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

func main() {
	// Start the MCP server
	cmd := exec.Command("./odata-mcp", "https://services.odata.org/V2/Northwind/Northwind.svc/")
	
	// Get stdin/stdout pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	
	// Start reading stderr in a goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[STDERR] %s\n", scanner.Text())
		}
	}()
	
	// Create a scanner for reading responses with larger buffer
	scanner := bufio.NewScanner(stdout)
	const maxScanTokenSize = 10 * 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	
	// Send initialize request
	fmt.Fprintln(os.Stderr, "\n=== Sending initialize request ===")
	initReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "0.1.0",
			"capabilities": map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name": "test-client",
				"version": "1.0.0",
			},
		},
	}
	
	if err := sendRequest(stdin, initReq); err != nil {
		panic(err)
	}
	
	// Read initialize response
	if scanner.Scan() {
		fmt.Fprintf(os.Stderr, "Response: %s\n", scanner.Text())
		
		// Parse and pretty print
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(scanner.Text()), &resp); err == nil {
			pretty, _ := json.MarshalIndent(resp, "", "  ")
			fmt.Fprintf(os.Stderr, "\nParsed response:\n%s\n", pretty)
		}
	}
	
	// Send initialized notification
	fmt.Fprintln(os.Stderr, "\n=== Sending initialized notification ===")
	initializedNotif := map[string]interface{}{
		"jsonrpc": "2.0",
		"method": "initialized",
	}
	
	if err := sendRequest(stdin, initializedNotif); err != nil {
		panic(err)
	}
	
	// Give it a moment to process
	time.Sleep(100 * time.Millisecond)
	
	// Send tools/list request
	fmt.Fprintln(os.Stderr, "\n=== Sending tools/list request ===")
	toolsReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id": 2,
		"method": "tools/list",
		"params": map[string]interface{}{},
	}
	
	if err := sendRequest(stdin, toolsReq); err != nil {
		panic(err)
	}
	
	// Read tools/list response with timeout
	fmt.Fprintln(os.Stderr, "Waiting for tools/list response...")
	
	done := make(chan string)
	go func() {
		if scanner.Scan() {
			done <- scanner.Text()
		} else {
			done <- ""
		}
	}()
	
	select {
	case response := <-done:
		if response == "" {
			fmt.Fprintln(os.Stderr, "Failed to read response - scanner returned empty")
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "Scanner error: %v\n", err)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Response: %s\n", response)
			
			// Parse and show tool count
			var resp map[string]interface{}
			if err := json.Unmarshal([]byte(response), &resp); err == nil {
				if result, ok := resp["result"].(map[string]interface{}); ok {
					if tools, ok := result["tools"].([]interface{}); ok {
						fmt.Fprintf(os.Stderr, "\nFound %d tools\n", len(tools))
						
						// Show first few tool names
						for i, tool := range tools {
							if i >= 5 {
								fmt.Fprintf(os.Stderr, "... and %d more tools\n", len(tools)-5)
								break
							}
							if t, ok := tool.(map[string]interface{}); ok {
								fmt.Fprintf(os.Stderr, "  - %s\n", t["name"])
							}
						}
					}
				}
			}
		}
	case <-time.After(5 * time.Second):
		fmt.Fprintln(os.Stderr, "Timeout waiting for response")
	}
	
	// Close stdin to signal we're done
	stdin.Close()
	
	// Wait for the command to exit
	cmd.Wait()
	
	fmt.Fprintln(os.Stderr, "\n=== Test completed ===")
}

func sendRequest(w io.Writer, req interface{}) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	
	fmt.Fprintf(os.Stderr, "Sending: %s\n", data)
	_, err = fmt.Fprintf(w, "%s\n", data)
	return err
}