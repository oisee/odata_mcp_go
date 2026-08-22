// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidateHTTPSecurity tests the security validation for HTTP transport
func TestValidateHTTPSecurity(t *testing.T) {
	tests := []struct {
		name        string
		config      SecurityConfig
		wantErr     bool
		errContains string
	}{
		// Localhost without token - rejected
		{
			name: "localhost:8080 without token - rejected",
			config: SecurityConfig{
				Addr: "localhost:8080",
			},
			wantErr:     true,
			errContains: "mcp-token required",
		},
		{
			name: "127.0.0.1:8080 without token - rejected",
			config: SecurityConfig{
				Addr: "127.0.0.1:8080",
			},
			wantErr:     true,
			errContains: "mcp-token required",
		},
		{
			name: "[::1]:8080 without token - rejected",
			config: SecurityConfig{
				Addr: "[::1]:8080",
			},
			wantErr:     true,
			errContains: "mcp-token required",
		},

		// Localhost with token - allowed
		{
			name: "localhost:8080 with token - allowed",
			config: SecurityConfig{
				Addr:  "localhost:8080",
				Token: "secret-token",
			},
			wantErr: false,
		},

		// Non-localhost without token - rejected
		{
			name: "192.168.1.100:8080 without token - rejected",
			config: SecurityConfig{
				Addr: "192.168.1.100:8080",
			},
			wantErr:     true,
			errContains: "mcp-token required",
		},

		// Non-localhost with token but no TLS - rejected
		{
			name: "192.168.1.100:8080 with token but no TLS - rejected",
			config: SecurityConfig{
				Addr:  "192.168.1.100:8080",
				Token: "secret-token",
			},
			wantErr:     true,
			errContains: "tls required",
		},

		// Non-localhost with token and TLS - allowed
		{
			name: "192.168.1.100:8080 with token and TLS - allowed",
			config: SecurityConfig{
				Addr:       "192.168.1.100:8080",
				Token:      "secret-token",
				TLSEnabled: true,
				TLSCert:    "/path/to/cert.pem",
				TLSKey:     "/path/to/key.pem",
			},
			wantErr: false,
		},

		// 0.0.0.0 without explicit allow flag - rejected
		{
			name: "0.0.0.0:8080 without explicit allow - rejected",
			config: SecurityConfig{
				Addr: "0.0.0.0:8080",
			},
			wantErr:     true,
			errContains: "all interfaces",
		},

		// 0.0.0.0 with flag but no token - rejected
		{
			name: "0.0.0.0:8080 with flag but no token - rejected",
			config: SecurityConfig{
				Addr:               "0.0.0.0:8080",
				AllowAllInterfaces: true,
			},
			wantErr:     true,
			errContains: "mcp-token required",
		},

		// 0.0.0.0 with flag and token but no TLS - rejected
		{
			name: "0.0.0.0:8080 with flag and token but no TLS - rejected",
			config: SecurityConfig{
				Addr:               "0.0.0.0:8080",
				AllowAllInterfaces: true,
				Token:              "secret-token",
			},
			wantErr:     true,
			errContains: "tls required",
		},

		// 0.0.0.0 with all requirements - allowed
		{
			name: "0.0.0.0:8080 with flag, token, and TLS - allowed",
			config: SecurityConfig{
				Addr:               "0.0.0.0:8080",
				Token:              "secret-token",
				TLSEnabled:         true,
				TLSCert:            "/path/to/cert.pem",
				TLSKey:             "/path/to/key.pem",
				AllowAllInterfaces: true,
			},
			wantErr: false,
		},

		// :8080 (implicit all interfaces) - rejected
		{
			name: ":8080 (implicit all) without flag - rejected",
			config: SecurityConfig{
				Addr: ":8080",
			},
			wantErr:     true,
			errContains: "all interfaces",
		},

		// [::]:8080 (IPv6 all interfaces) - rejected
		{
			name: "[::]:8080 without flag - rejected",
			config: SecurityConfig{
				Addr: "[::]:8080",
			},
			wantErr:     true,
			errContains: "all interfaces",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateHTTPSecurity(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestIsLoopback tests the loopback address detection
func TestIsLoopback(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"localhost:8080", true},
		{"127.0.0.1:8080", true},
		{"127.0.0.2:8080", true},
		{"127.1:8080", true},
		{"[::1]:8080", true},
		{"192.168.1.1:8080", false},
		{"10.0.0.1:8080", false},
		{"0.0.0.0:8080", false},
		{"[::]:8080", false},
		{":8080", false},
		{"example.com:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := IsLoopbackAddr(tt.addr)
			assert.Equal(t, tt.expected, result, "IsLoopbackAddr(%q)", tt.addr)
		})
	}
}

// TestIsUnspecified tests detection of addresses that bind to all interfaces
func TestIsUnspecified(t *testing.T) {
	tests := []struct {
		addr     string
		expected bool
	}{
		{"0.0.0.0:8080", true},
		{"[::]:8080", true},
		{":8080", true},
		{"localhost:8080", false},
		{"127.0.0.1:8080", false},
		{"192.168.1.1:8080", false},
		{"[::1]:8080", false},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := IsUnspecifiedAddr(tt.addr)
			assert.Equal(t, tt.expected, result, "IsUnspecifiedAddr(%q)", tt.addr)
		})
	}
}

// TestValidateToken tests constant-time token comparison
func TestValidateToken(t *testing.T) {
	tests := []struct {
		name     string
		provided string
		expected string
		valid    bool
	}{
		{"exact match", "secret-token-123", "secret-token-123", true},
		{"wrong token", "wrong-token", "secret-token-123", false},
		{"empty provided", "", "secret-token-123", false},
		{"empty expected", "secret-token-123", "", false},
		{"both empty", "", "", true},
		{"prefix match only", "secret", "secret-token-123", false},
		{"suffix match only", "token-123", "secret-token-123", false},
		{"case sensitive", "SECRET-TOKEN-123", "secret-token-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateToken(tt.provided, tt.expected)
			assert.Equal(t, tt.valid, result)
		})
	}
}
