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

type MCPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <service-url>\n", os.Args[0])
		os.Exit(1)
	}

	serviceURL := os.Args[1]
	
	// Start the MCP server
	cmd := exec.Command("./odata-mcp", serviceURL)
	
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
	
	if err := cmd.Start(); err != nil {
		panic(err)
	}
	
	// Read stderr in background
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[SERVER] %s\n", scanner.Text())
		}
	}()
	
	// Give server time to start
	time.Sleep(1 * time.Second)
	
	fmt.Println("=== Testing MCP Server ===")
	
	// Test 1: Initialize
	fmt.Println("\n1. Testing initialize...")
	initReq := MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo": map[string]interface{}{
				"name":    "test-client",
				"version": "1.0.0",
			},
		},
	}
	
	response := sendRequest(stdin, stdout, initReq)
	fmt.Printf("Initialize response: %s\n", string(response))
	
	// Test 2: Send initialized notification
	fmt.Println("\n2. Sending initialized notification...")
	initedReq := MCPRequest{
		JSONRPC: "2.0",
		Method:  "initialized",
	}
	
	reqBytes, _ := json.Marshal(initedReq)
	stdin.Write(reqBytes)
	stdin.Write([]byte("\n"))
	
	// Test 3: List tools
	fmt.Println("\n3. Testing tools/list...")
	toolsReq := MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	
	response = sendRequest(stdin, stdout, toolsReq)
	fmt.Printf("Tools list response: %s\n", string(response))
	
	// Cleanup
	stdin.Close()
	cmd.Wait()
}

func sendRequest(stdin *os.File, stdout *os.File, req MCPRequest) []byte {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}
	
	fmt.Printf("Sending: %s\n", string(reqBytes))
	
	stdin.Write(reqBytes)
	stdin.Write([]byte("\n"))
	
	// Read response
	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		line := scanner.Text()
		fmt.Printf("Received: %s\n", line)
		return []byte(line)
	}
	
	return []byte{}
}