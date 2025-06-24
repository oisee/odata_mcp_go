package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/odata-mcp/go/internal/bridge"
	"github.com/odata-mcp/go/internal/config"
)

var cfg *config.Config

var rootCmd = &cobra.Command{
	Use:   "odata-mcp [service-url]",
	Short: "OData to MCP Bridge - Universal OData v2 to Model Context Protocol bridge",
	Long: `OData to MCP Bridge - Universal OData v2 to Model Context Protocol bridge.

This tool creates a bridge between OData v2 services and the Model Context Protocol
(MCP), dynamically generating MCP tools based on OData metadata.

Examples:
  odata-mcp https://services.odata.org/V2/Northwind/Northwind.svc/
  odata-mcp --service https://my-sap-service.com/sap/opu/odata/sap/SERVICE_NAME/
  odata-mcp --user admin --password secret https://my-service.com/odata/
  odata-mcp --cookie-file cookies.txt https://my-service.com/odata/`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBridge,
}

func init() {
	// Load .env file if it exists
	godotenv.Load()

	// Initialize config
	cfg = &config.Config{}

	// Service URL
	rootCmd.Flags().StringVar(&cfg.ServiceURL, "service", "", "URL of the OData service (overrides positional argument and ODATA_SERVICE_URL env var)")

	// Authentication flags (mutually exclusive handled in validation)
	rootCmd.Flags().StringVarP(&cfg.Username, "user", "u", "", "Username for basic authentication (overrides ODATA_USERNAME env var)")
	rootCmd.Flags().StringVarP(&cfg.Password, "password", "p", "", "Password for basic authentication (overrides ODATA_PASSWORD env var)")
	rootCmd.Flags().StringVar(&cfg.Password, "pass", "", "Password for basic authentication (alias for --password)")
	rootCmd.Flags().StringVar(&cfg.CookieFile, "cookie-file", "", "Path to cookie file in Netscape format")
	rootCmd.Flags().StringVar(&cfg.CookieString, "cookie-string", "", "Cookie string (key1=val1; key2=val2)")

	// Tool naming options
	rootCmd.Flags().StringVar(&cfg.ToolPrefix, "tool-prefix", "", "Custom prefix for tool names (use with --no-postfix)")
	rootCmd.Flags().StringVar(&cfg.ToolPostfix, "tool-postfix", "", "Custom postfix for tool names (default: _for_<service_id>)")
	rootCmd.Flags().BoolVar(&cfg.NoPostfix, "no-postfix", false, "Use prefix instead of postfix for tool naming")
	rootCmd.Flags().BoolVar(&cfg.ToolShrink, "tool-shrink", false, "Use shortened tool names (create_, get_, upd_, del_, search_, filter_)")

	// Entity and function filtering
	rootCmd.Flags().StringVar(&cfg.Entities, "entities", "", "Comma-separated list of entities to generate tools for (e.g., 'Products,Categories,Orders'). Supports wildcards: 'Product*,Order*'")
	rootCmd.Flags().StringVar(&cfg.Functions, "functions", "", "Comma-separated list of function imports to generate tools for (e.g., 'GetProducts,CreateOrder'). Supports wildcards: 'Get*,Create*'")

	// Output and debugging options
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose output to stderr")
	rootCmd.Flags().BoolVar(&cfg.Debug, "debug", false, "Alias for --verbose")
	rootCmd.Flags().BoolVar(&cfg.SortTools, "sort-tools", true, "Sort tools alphabetically in the output")
	rootCmd.Flags().BoolVar(&cfg.Trace, "trace", false, "Initialize MCP service and print all tools and parameters, then exit (useful for debugging)")
	
	// Response enhancement options
	rootCmd.Flags().BoolVar(&cfg.PaginationHints, "pagination-hints", false, "Add pagination support with suggested_next_call and has_more indicators")
	rootCmd.Flags().BoolVar(&cfg.LegacyDates, "legacy-dates", true, "Support epoch timestamp format (/Date(1234567890000)/) - enabled by default for SAP")
	rootCmd.Flags().BoolVar(&cfg.NoLegacyDates, "no-legacy-dates", false, "Disable legacy date format conversion")
	rootCmd.Flags().BoolVar(&cfg.VerboseErrors, "verbose-errors", false, "Provide detailed error context and debugging information")
	rootCmd.Flags().BoolVar(&cfg.ResponseMetadata, "response-metadata", false, "Include detailed __metadata blocks in entity responses")
	
	// Response size limits
	rootCmd.Flags().IntVar(&cfg.MaxResponseSize, "max-response-size", 5*1024*1024, "Maximum response size in bytes (default: 5MB)")
	rootCmd.Flags().IntVar(&cfg.MaxItems, "max-items", 100, "Maximum number of items in response (default: 100)")

	// Bind flags to viper for environment variable support
	viper.BindPFlag("service", rootCmd.Flags().Lookup("service"))
	viper.BindPFlag("username", rootCmd.Flags().Lookup("user"))
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))

	// Set up environment variable mapping
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.SetEnvPrefix("ODATA")
}

func runBridge(cmd *cobra.Command, args []string) error {
	// Handle --debug as alias for --verbose
	if cfg.Debug {
		cfg.Verbose = true
	}
	
	// Handle legacy dates flags
	if cfg.NoLegacyDates {
		cfg.LegacyDates = false
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Legacy date format conversion disabled.\n")
		}
	} else if !cmd.Flags().Changed("legacy-dates") {
		// Default to legacy dates for SAP compatibility
		cfg.LegacyDates = true
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Legacy date format enabled by default for SAP compatibility. Use --no-legacy-dates to disable.\n")
		}
	}

	// Determine service URL with priority: --service flag > positional arg > env vars
	if cfg.ServiceURL == "" && len(args) > 0 {
		cfg.ServiceURL = args[0]
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using OData service URL from positional argument.\n")
		}
	}

	if cfg.ServiceURL == "" {
		cfg.ServiceURL = viper.GetString("URL")
		if cfg.ServiceURL == "" {
			cfg.ServiceURL = viper.GetString("SERVICE_URL")
		}
		if cfg.ServiceURL != "" && cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using ODATA_URL from environment.\n")
		}
	}

	if cfg.ServiceURL == "" {
		return fmt.Errorf("OData service URL not provided. Use --service flag, positional argument, or ODATA_URL environment variable")
	}

	// Validate and process authentication
	if err := processAuthentication(cfg); err != nil {
		return err
	}

	// Parse entity and function filters
	if cfg.Entities != "" {
		cfg.AllowedEntities = parseCommaSeparated(cfg.Entities)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Filtering tools to only these entities: %v\n", cfg.AllowedEntities)
		}
	}

	if cfg.Functions != "" {
		cfg.AllowedFunctions = parseCommaSeparated(cfg.Functions)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Filtering tools to only these functions: %v\n", cfg.AllowedFunctions)
		}
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create and initialize bridge
	bridge, err := bridge.NewODataMCPBridge(cfg)
	if err != nil {
		return fmt.Errorf("failed to create OData MCP bridge: %w", err)
	}

	// Handle trace mode
	if cfg.Trace {
		return printTraceInfo(bridge)
	}

	// Start bridge in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- bridge.Run()
	}()

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		fmt.Fprintf(os.Stderr, "\n%s received, shutting down server...\n", sig)
		bridge.Stop()
		return nil
	case err := <-errChan:
		return err
	}
}

func processAuthentication(cfg *config.Config) error {
	// Check for mutually exclusive authentication options
	authMethods := 0
	if cfg.CookieFile != "" {
		authMethods++
	}
	if cfg.CookieString != "" {
		authMethods++
	}
	if cfg.Username != "" {
		authMethods++
	}

	if authMethods > 1 {
		return fmt.Errorf("only one authentication method can be used at a time")
	}

	// Process cookie file authentication
	if cfg.CookieFile != "" {
		if _, err := os.Stat(cfg.CookieFile); os.IsNotExist(err) {
			return fmt.Errorf("cookie file not found: %s", cfg.CookieFile)
		}

		cookies, err := loadCookiesFromFile(cfg.CookieFile)
		if err != nil {
			return fmt.Errorf("failed to load cookies from file: %w", err)
		}

		cfg.Cookies = cookies
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Loaded %d cookies from file: %s\n", len(cookies), cfg.CookieFile)
		}
	} else if cfg.CookieString != "" {
		// Process cookie string authentication
		cookies := parseCookieString(cfg.CookieString)
		if len(cookies) == 0 {
			return fmt.Errorf("failed to parse cookie string")
		}

		cfg.Cookies = cookies
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Parsed %d cookies from string\n", len(cookies))
		}
	} else {
		// Handle basic authentication from environment if not provided via flags
		if cfg.Username == "" {
			cfg.Username = viper.GetString("USER")
			if cfg.Username == "" {
				cfg.Username = viper.GetString("USERNAME")
			}
		}

		if cfg.Password == "" {
			cfg.Password = viper.GetString("PASS")
			if cfg.Password == "" {
				cfg.Password = viper.GetString("PASSWORD")
			}
		}

		// Check for cookie environment variables if no auth is configured
		if cfg.Username == "" {
			envCookieFile := viper.GetString("COOKIE_FILE")
			envCookieString := viper.GetString("COOKIE_STRING")

			if envCookieFile != "" {
				if _, err := os.Stat(envCookieFile); err == nil {
					cookies, err := loadCookiesFromFile(envCookieFile)
					if err == nil {
						cfg.Cookies = cookies
						if cfg.Verbose {
							fmt.Fprintf(os.Stderr, "[VERBOSE] Loaded %d cookies from environment ODATA_COOKIE_FILE\n", len(cookies))
						}
					}
				}
			} else if envCookieString != "" {
				cookies := parseCookieString(envCookieString)
				if len(cookies) > 0 {
					cfg.Cookies = cookies
					if cfg.Verbose {
						fmt.Fprintf(os.Stderr, "[VERBOSE] Parsed %d cookies from environment ODATA_COOKIE_STRING\n", len(cookies))
					}
				}
			}
		}

		// Set up basic auth if credentials are available
		if cfg.Username != "" && cfg.Password != "" {
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Using basic authentication for user: %s\n", cfg.Username)
			}
		} else if cfg.Verbose && len(cfg.Cookies) == 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] No authentication provided or configured. Attempting anonymous access.\n")
		}
	}

	return nil
}

func loadCookiesFromFile(cookieFile string) (map[string]string, error) {
	cookies := make(map[string]string)

	file, err := os.Open(cookieFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse Netscape format (7 fields separated by tabs)
		parts := strings.Split(line, "\t")
		if len(parts) >= 7 {
			// domain, flag, path, secure, expiration, name, value
			name := parts[5]
			value := parts[6]
			cookies[name] = value
		} else if strings.Contains(line, "=") {
			// Simple key=value format fallback
			kv := strings.SplitN(line, "=", 2)
			if len(kv) == 2 {
				cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}

	return cookies, scanner.Err()
}

func parseCookieString(cookieString string) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range strings.Split(cookieString, ";") {
		cookie = strings.TrimSpace(cookie)
		if strings.Contains(cookie, "=") {
			kv := strings.SplitN(cookie, "=", 2)
			if len(kv) == 2 {
				cookies[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
			}
		}
	}
	return cookies
}

func parseCommaSeparated(input string) []string {
	var result []string
	for _, item := range strings.Split(input, ",") {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func printTraceInfo(bridge *bridge.ODataMCPBridge) error {
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("🔍 OData MCP Bridge Trace Information")
	fmt.Println(strings.Repeat("=", 80))

	info, err := bridge.GetTraceInfo()
	if err != nil {
		return fmt.Errorf("failed to get trace info: %w", err)
	}

	// Print trace information as JSON for now
	// TODO: Implement pretty printing like the Python version
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal trace info: %w", err)
	}

	fmt.Println(string(data))

	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("✅ Trace complete - MCP bridge initialized successfully but not started")
	fmt.Println("💡 Use without --trace to start the actual MCP server")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "\n--- FATAL ERROR ---\n")
		fmt.Fprintf(os.Stderr, "An unexpected error occurred: %v\n", err)
		fmt.Fprintf(os.Stderr, "-------------------\n")
		os.Exit(1)
	}
}