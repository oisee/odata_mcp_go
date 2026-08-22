package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/zmcp/odata-mcp/internal/debug"
	"github.com/zmcp/odata-mcp/internal/transport"
)

// StdioTransport implements the Transport interface for stdio communication
type StdioTransport struct {
	reader  *bufio.Reader
	writer  io.Writer
	handler transport.Handler
	tracer  *debug.TraceLogger
}

// New creates a new stdio transport
func New(handler transport.Handler) *StdioTransport {
	return &StdioTransport{
		reader:  bufio.NewReader(os.Stdin),
		writer:  os.Stdout,
		handler: handler,
	}
}

// SetTracer sets the trace logger
func (t *StdioTransport) SetTracer(tracer *debug.TraceLogger) {
	t.tracer = tracer
}

// Start begins processing messages from stdio
func (t *StdioTransport) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := t.ReadMessage()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				// Continue processing without logging to avoid stderr interference
				continue
			}

			// Process request if it has a method
			if msg.Method != "" && t.handler != nil {
				response, err := t.handler(ctx, msg)
				if err != nil {
					// Ensure ID is not null for error responses
					msgID := msg.ID
					if msgID == nil || string(msgID) == "null" {
						msgID = json.RawMessage("0")
					}

					// Send error response
					errorResponse := &transport.Message{
						JSONRPC: "2.0",
						ID:      msgID,
						Error: &transport.Error{
							Code:    -32603,
							Message: err.Error(),
						},
					}
					if writeErr := t.WriteMessage(errorResponse); writeErr != nil && t.tracer != nil {
						t.tracer.LogError("Failed to write error response", writeErr, errorResponse)
					}
				} else if response != nil {
					if writeErr := t.WriteMessage(response); writeErr != nil && t.tracer != nil {
						t.tracer.LogError("Failed to write response", writeErr, response)
					}
				}
			}
		}
	}
}

// ReadMessage reads a line-delimited JSON message from stdin
func (t *StdioTransport) ReadMessage() (*transport.Message, error) {
	line, err := t.reader.ReadBytes('\n')
	if err != nil {
		return nil, err
	}

	// Trace raw input
	if t.tracer != nil {
		t.tracer.Log("TRANSPORT_IN", "Raw message received", map[string]interface{}{
			"raw":  string(line),
			"size": len(line),
		})
	}

	var msg transport.Message
	if err := json.Unmarshal(line, &msg); err != nil {
		if t.tracer != nil {
			t.tracer.LogError("Failed to unmarshal message", err, map[string]interface{}{
				"raw": string(line),
			})
		}
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Trace parsed message
	if t.tracer != nil {
		t.tracer.Log("TRANSPORT_PARSED", "Message parsed", map[string]interface{}{
			"method":     msg.Method,
			"id":         msg.ID,
			"jsonrpc":    msg.JSONRPC,
			"has_params": len(msg.Params) > 0,
		})
	}

	return &msg, nil
}

// WriteMessage writes a JSON message to stdout
func (t *StdioTransport) WriteMessage(msg *transport.Message) error {
	// Trace outgoing message
	if t.tracer != nil {
		t.tracer.Log("TRANSPORT_OUT", "Sending message", map[string]interface{}{
			"id":         msg.ID,
			"has_result": msg.Result != nil,
			"has_error":  msg.Error != nil,
			"method":     msg.Method,
		})
	}

	data, err := json.Marshal(msg)
	if err != nil {
		if t.tracer != nil {
			t.tracer.LogError("Failed to marshal message", err, msg)
		}
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Trace raw output
	if t.tracer != nil {
		t.tracer.Log("TRANSPORT_RAW_OUT", "Raw output", map[string]interface{}{
			"raw":  string(data),
			"size": len(data),
		})
	}

	if _, err := t.writer.Write(data); err != nil {
		return err
	}

	if _, err := t.writer.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

// Close closes the transport (no-op for stdio)
func (t *StdioTransport) Close() error {
	return nil
}
