// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package http

import (
	"crypto/subtle"
	"fmt"
	"net"
	"strings"
)

// SecurityConfig holds security settings for HTTP transport
type SecurityConfig struct {
	Addr               string // HTTP server address (e.g., "localhost:8080")
	Token              string // MCP token for authentication
	TLSEnabled         bool   // Whether TLS is enabled
	TLSCert            string // Path to TLS certificate
	TLSKey             string // Path to TLS key
	AllowAllInterfaces bool   // Explicit flag to allow 0.0.0.0/::
}

// ValidateHTTPSecurity validates security configuration for HTTP transport.
// Returns an error if the configuration is insecure.
//
// Security model (strict by default):
// - Token is always required unless --no-token-localhost is set for loopback addresses
// - Non-localhost requires token + TLS, no exceptions
// - Binding to all interfaces (0.0.0.0/::) requires explicit flag + token + TLS
func ValidateHTTPSecurity(cfg SecurityConfig) error {
	// Check for unspecified addresses (0.0.0.0, ::, or empty host)
	if IsUnspecifiedAddr(cfg.Addr) {
		if !cfg.AllowAllInterfaces {
			return fmt.Errorf("binding to all interfaces (0.0.0.0/::) requires --allow-all-interfaces flag")
		}
		if cfg.Token == "" {
			return fmt.Errorf("--mcp-token required when binding to all interfaces")
		}
		if !cfg.TLSEnabled {
			return fmt.Errorf("--tls required when binding to all interfaces")
		}
		return nil
	}

	// Localhost bindings: token always required
	if IsLoopbackAddr(cfg.Addr) {
		if cfg.Token == "" {
			return fmt.Errorf("--mcp-token required for HTTP transport")
		}
		return nil
	}

	// Non-localhost bindings: token + TLS required, no exceptions
	if cfg.Token == "" {
		return fmt.Errorf("--mcp-token required for non-localhost binding")
	}
	if !cfg.TLSEnabled {
		return fmt.Errorf("--tls required for non-localhost binding")
	}

	return nil
}

// IsLoopbackAddr checks if an address is a loopback address (localhost/127.x.x.x/::1)
func IsLoopbackAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	host = strings.Trim(host, "[]")

	if host == "localhost" {
		return true
	}

	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsLoopback()
	}

	// Check for 127.x shorthand (e.g., "127.1" = "127.0.0.1")
	if strings.HasPrefix(host, "127.") {
		return true
	}

	return false
}

// IsUnspecifiedAddr checks if an address binds to all interfaces (0.0.0.0, ::, or empty)
func IsUnspecifiedAddr(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		if strings.HasPrefix(addr, ":") {
			return true
		}
		host = addr
	}

	if host == "" {
		return true
	}

	host = strings.Trim(host, "[]")

	ip := net.ParseIP(host)
	if ip != nil {
		return ip.IsUnspecified()
	}

	return false
}

// ValidateToken performs constant-time comparison of tokens to prevent timing attacks
func ValidateToken(provided, expected string) bool {
	if provided == "" && expected == "" {
		return true
	}
	return subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}
