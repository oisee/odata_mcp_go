package debug

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TraceLogger provides detailed tracing for debugging
type TraceLogger struct {
	mu       sync.Mutex
	file     *os.File
	enabled  bool
	filename string
}

// NewTraceLogger creates a new trace logger
func NewTraceLogger(enabled bool) (*TraceLogger, error) {
	if !enabled {
		return &TraceLogger{enabled: false}, nil
	}

	// Create trace file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(os.TempDir(), fmt.Sprintf("mcp_trace_%s.log", timestamp))

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace file: %w", err)
	}

	logger := &TraceLogger{
		file:     file,
		enabled:  enabled,
		filename: filename,
	}

	// Log initial message
	logger.Log("TRACE", "Trace logging started", map[string]interface{}{
		"filename": filename,
		"pid":      os.Getpid(),
		"time":     time.Now().Format(time.RFC3339),
	})

	return logger, nil
}

// Log writes a trace entry
func (t *TraceLogger) Log(level, message string, data interface{}) {
	if !t.enabled || t.file == nil {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	entry := map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339Nano),
		"level":     level,
		"message":   message,
	}

	if data != nil {
		entry["data"] = data
	}

	jsonData, _ := json.Marshal(entry)
	fmt.Fprintf(t.file, "%s\n", jsonData)
	_ = t.file.Sync() // Best-effort sync to disk
}

// LogRequest logs an incoming JSON-RPC request
func (t *TraceLogger) LogRequest(raw string, parsed interface{}) {
	t.Log("REQUEST", "Incoming request", map[string]interface{}{
		"raw":    raw,
		"parsed": parsed,
	})
}

// LogResponse logs an outgoing JSON-RPC response
func (t *TraceLogger) LogResponse(response interface{}, err error) {
	data := map[string]interface{}{
		"response": response,
	}
	if err != nil {
		data["error"] = err.Error()
	}
	t.Log("RESPONSE", "Outgoing response", data)
}

// LogError logs an error with context
func (t *TraceLogger) LogError(context string, err error, data interface{}) {
	t.Log("ERROR", context, map[string]interface{}{
		"error": err.Error(),
		"data":  data,
	})
}

// GetFilename returns the trace filename
func (t *TraceLogger) GetFilename() string {
	return t.filename
}

// Close closes the trace file
func (t *TraceLogger) Close() error {
	if t.file != nil {
		t.Log("TRACE", "Trace logging stopped", nil)
		return t.file.Close()
	}
	return nil
}
