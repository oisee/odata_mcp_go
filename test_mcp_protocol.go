package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Test the MCP protocol by simulating a client
func main() {
	// Create pipes for communication
	stdinR, stdinW, _ := os.Pipe()
	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()

	// Start a goroutine to capture stderr
	stderrBuf := &bytes.Buffer{}
	go func() {
		scanner := bufio.NewScanner(stderrR)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuf.WriteString(line + "\n")
			fmt.Fprintf(os.Stderr, "STDERR: %s\n", line)
		}
	}()

	// Create test server
	os.Args = []string{"odata-mcp", "--service", "https://services.odata.org/V2/Northwind/Northwind.svc/"}
	
	// Replace stdin/stdout for the server
	oldStdin := os.Stdin
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	os.Stdin = stdinR
	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Start server in goroutine
	serverDone := make(chan error)
	go func() {
		// Import and run the main function
		// This would need to be adapted to actually run the server
		fmt.Fprintln(os.Stderr, "Server would start here")
		serverDone <- nil
	}()

	// Restore original streams for client
	os.Stdin = oldStdin
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	// Client side - send initialize request
	request := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "0.1.0",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}

	requestBytes, _ := json.Marshal(request)
	fmt.Fprintf(stdinW, "%s\n", requestBytes)

	// Read response
	scanner := bufio.NewScanner(stdoutR)
	responseReceived := make(chan string)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				responseReceived <- line
				break
			}
		}
	}()

	// Wait for response with timeout
	select {
	case response := <-responseReceived:
		fmt.Printf("Got response: %s\n", response)
		
		// Parse and validate response
		var resp map[string]interface{}
		if err := json.Unmarshal([]byte(response), &resp); err != nil {
			fmt.Printf("ERROR: Failed to parse response: %v\n", err)
		} else {
			fmt.Printf("Parsed response: %+v\n", resp)
		}
	case <-time.After(5 * time.Second):
		fmt.Println("ERROR: Timeout waiting for response")
	}

	// Check stderr
	fmt.Printf("\nStderr output:\n%s\n", stderrBuf.String())
}