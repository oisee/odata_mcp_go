package config

// Config holds all configuration options for the OData MCP bridge
type Config struct {
	// Service configuration
	ServiceURL string `mapstructure:"service_url"`

	// Authentication
	Username     string            `mapstructure:"username"`
	Password     string            `mapstructure:"password"`
	CookieFile   string            `mapstructure:"cookie_file"`
	CookieString string            `mapstructure:"cookie_string"`
	Cookies      map[string]string // Parsed cookies

	// Tool naming options
	ToolPrefix  string `mapstructure:"tool_prefix"`
	ToolPostfix string `mapstructure:"tool_postfix"`
	NoPostfix   bool   `mapstructure:"no_postfix"`
	ToolShrink  bool   `mapstructure:"tool_shrink"`

	// Entity and function filtering
	Entities         string   `mapstructure:"entities"`
	Functions        string   `mapstructure:"functions"`
	AllowedEntities  []string // Parsed from Entities
	AllowedFunctions []string // Parsed from Functions

	// Output and debugging
	Verbose   bool `mapstructure:"verbose"`
	Debug     bool `mapstructure:"debug"`
	SortTools bool `mapstructure:"sort_tools"`
	Trace     bool `mapstructure:"trace"`
}

// HasBasicAuth returns true if username and password are configured
func (c *Config) HasBasicAuth() bool {
	return c.Username != "" && c.Password != ""
}

// HasCookieAuth returns true if cookies are configured
func (c *Config) HasCookieAuth() bool {
	return len(c.Cookies) > 0
}

// UsePostfix returns true if tool postfix should be used instead of prefix
func (c *Config) UsePostfix() bool {
	return !c.NoPostfix
}