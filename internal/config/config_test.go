package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasDigestAuth(t *testing.T) {
	cfg := &Config{Username: "admin", Password: "secret", AuthType: "digest"}
	assert.True(t, cfg.HasDigestAuth())
	assert.False(t, cfg.HasBasicAuth(), "should not be basic when auth type is digest")
}

func TestHasBasicAuthDefault(t *testing.T) {
	cfg := &Config{Username: "admin", Password: "secret", AuthType: "basic"}
	assert.True(t, cfg.HasBasicAuth())
	assert.False(t, cfg.HasDigestAuth())
}

func TestHasBasicAuthEmptyAuthType(t *testing.T) {
	// When AuthType is empty string (not "digest"), HasBasicAuth should return true
	cfg := &Config{Username: "admin", Password: "secret"}
	assert.True(t, cfg.HasBasicAuth())
	assert.False(t, cfg.HasDigestAuth())
}

func TestNoAuthWhenCredentialsMissing(t *testing.T) {
	cfg := &Config{AuthType: "digest"}
	assert.False(t, cfg.HasDigestAuth(), "no digest without credentials")
	assert.False(t, cfg.HasBasicAuth(), "no basic without credentials")
}
