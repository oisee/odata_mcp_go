package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents OCL configuration
type Config struct {
	Services  map[string]ServiceConfig  `yaml:"services"`
	Variables map[string]any            `yaml:"variables"`
}

// ServiceConfig represents a service configuration
type ServiceConfig struct {
	URL      string `yaml:"url"`
	Auth     string `yaml:"auth"`     // none, basic, bearer, apikey
	Username string `yaml:"username"` // for basic auth
	Password string `yaml:"password"` // for basic auth
	Token    string `yaml:"token"`    // for bearer/apikey
	Header   string `yaml:"header"`   // for apikey (default: X-API-Key)
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Expand environment variables
	content := expandEnvVars(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(content), &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &config, nil
}

// FindConfig searches for config file in standard locations
func FindConfig(oclFilePath string) string {
	// Search order:
	// 1. Same directory as .ocl file: ocl.config.yaml
	// 2. Same directory as .ocl file: .ocl.yaml
	// 3. Current directory: ocl.config.yaml
	// 4. Home directory: ~/.ocl/config.yaml

	searchPaths := []string{}

	if oclFilePath != "" {
		dir := filepath.Dir(oclFilePath)
		searchPaths = append(searchPaths,
			filepath.Join(dir, "ocl.config.yaml"),
			filepath.Join(dir, ".ocl.yaml"),
		)
	}

	searchPaths = append(searchPaths, "ocl.config.yaml")

	if home, err := os.UserHomeDir(); err == nil {
		searchPaths = append(searchPaths, filepath.Join(home, ".ocl", "config.yaml"))
	}

	for _, path := range searchPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// ApplyConfig applies configuration to an executor
func (e *Executor) ApplyConfig(config *Config) {
	// Register services with auth
	for name, svc := range config.Services {
		e.RegisterService(name, svc.URL)

		if svc.Auth != "" && svc.Auth != "none" {
			auth := &ServiceAuth{}
			switch svc.Auth {
			case "basic":
				auth.Type = AuthBasic
				auth.Username = svc.Username
				auth.Password = svc.Password
			case "bearer":
				auth.Type = AuthBearer
				auth.Token = svc.Token
			case "apikey":
				auth.Type = AuthAPIKey
				auth.Token = svc.Token
				auth.Header = svc.Header
			}
			e.SetServiceAuth(name, auth)
		}
	}

	// Set variables
	for name, value := range config.Variables {
		e.SetVariable("$"+name, value)
	}
}

// expandEnvVars expands ${VAR} patterns in string
func expandEnvVars(s string) string {
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		varName := strings.TrimPrefix(strings.TrimSuffix(match, "}"), "${")
		if value := os.Getenv(varName); value != "" {
			return value
		}
		return match // Keep original if not found
	})
}
