package bridge

import (
	"strings"
	"testing"

	"github.com/zmcp/odata-mcp/internal/config"
	"github.com/zmcp/odata-mcp/internal/hint"
	"github.com/zmcp/odata-mcp/internal/mcp"
	"github.com/zmcp/odata-mcp/internal/models"
)

func TestParameterTransformation(t *testing.T) {
	tests := []struct {
		name               string
		claudeCodeFriendly bool
		inputParam         string
		expectedOutput     string
	}{
		{
			name:               "Standard mode - keeps $ prefix",
			claudeCodeFriendly: false,
			inputParam:         "$filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Claude-friendly mode - removes $ prefix",
			claudeCodeFriendly: true,
			inputParam:         "$filter",
			expectedOutput:     "filter",
		},
		{
			name:               "Claude-friendly mode - handles select",
			claudeCodeFriendly: true,
			inputParam:         "$select",
			expectedOutput:     "select",
		},
		{
			name:               "Claude-friendly mode - handles expand",
			claudeCodeFriendly: true,
			inputParam:         "$expand",
			expectedOutput:     "expand",
		},
		{
			name:               "Claude-friendly mode - handles orderby",
			claudeCodeFriendly: true,
			inputParam:         "$orderby",
			expectedOutput:     "orderby",
		},
		{
			name:               "Claude-friendly mode - handles top",
			claudeCodeFriendly: true,
			inputParam:         "$top",
			expectedOutput:     "top",
		},
		{
			name:               "Claude-friendly mode - handles skip",
			claudeCodeFriendly: true,
			inputParam:         "$skip",
			expectedOutput:     "skip",
		},
		{
			name:               "Claude-friendly mode - handles count",
			claudeCodeFriendly: true,
			inputParam:         "$count",
			expectedOutput:     "count",
		},
		{
			name:               "Standard mode - non-OData param unchanged",
			claudeCodeFriendly: false,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
		{
			name:               "Claude-friendly mode - non-OData param unchanged",
			claudeCodeFriendly: true,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ClaudeCodeFriendly: tt.claudeCodeFriendly,
			}
			bridge := &ODataMCPBridge{
				config: cfg,
			}

			result := bridge.getParameterName(tt.inputParam)
			if result != tt.expectedOutput {
				t.Errorf("getParameterName() = %v, want %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestParameterMapping(t *testing.T) {
	tests := []struct {
		name               string
		claudeCodeFriendly bool
		inputParam         string
		expectedOutput     string
	}{
		// Claude-friendly mode tests
		{
			name:               "Maps filter to $filter",
			claudeCodeFriendly: true,
			inputParam:         "filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Maps select to $select",
			claudeCodeFriendly: true,
			inputParam:         "select",
			expectedOutput:     "$select",
		},
		{
			name:               "Maps expand to $expand",
			claudeCodeFriendly: true,
			inputParam:         "expand",
			expectedOutput:     "$expand",
		},
		{
			name:               "Maps orderby to $orderby",
			claudeCodeFriendly: true,
			inputParam:         "orderby",
			expectedOutput:     "$orderby",
		},
		{
			name:               "Maps top to $top",
			claudeCodeFriendly: true,
			inputParam:         "top",
			expectedOutput:     "$top",
		},
		{
			name:               "Maps skip to $skip",
			claudeCodeFriendly: true,
			inputParam:         "skip",
			expectedOutput:     "$skip",
		},
		{
			name:               "Maps count to $count",
			claudeCodeFriendly: true,
			inputParam:         "count",
			expectedOutput:     "$count",
		},
		{
			name:               "Keeps $filter as is",
			claudeCodeFriendly: true,
			inputParam:         "$filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Non-OData param unchanged",
			claudeCodeFriendly: true,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
		// Standard mode tests
		{
			name:               "Standard mode - keeps $filter",
			claudeCodeFriendly: false,
			inputParam:         "$filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Standard mode - adds $ to filter",
			claudeCodeFriendly: false,
			inputParam:         "filter",
			expectedOutput:     "$filter",
		},
		{
			name:               "Standard mode - adds $ to select",
			claudeCodeFriendly: false,
			inputParam:         "select",
			expectedOutput:     "$select",
		},
		{
			name:               "Standard mode - non-OData unchanged",
			claudeCodeFriendly: false,
			inputParam:         "customParam",
			expectedOutput:     "customParam",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				ClaudeCodeFriendly: tt.claudeCodeFriendly,
			}
			bridge := &ODataMCPBridge{
				config: cfg,
			}

			result := bridge.mapParameterToOData(tt.inputParam)
			if result != tt.expectedOutput {
				t.Errorf("mapParameterToOData() = %v, want %v", result, tt.expectedOutput)
			}
		})
	}
}

func TestUniversalToolDescription(t *testing.T) {
	cfg := &config.Config{
		ServiceURL:    "https://example.com/odata",
		UniversalTool: true,
	}

	bridge := &ODataMCPBridge{
		config:      cfg,
		tools:       make(map[string]*models.ToolInfo),
		hintManager: hint.NewManager(),
		metadata: &models.ODataMetadata{
			EntitySets: map[string]*models.EntitySet{
				"Products": {
					Name:       "Products",
					EntityType: "Product",
					Creatable:  true,
					Updatable:  true,
					Deletable:  true,
					Searchable: true,
				},
				"Categories": {
					Name:       "Categories",
					EntityType: "Category",
					Creatable:  false,
					Updatable:  true,
					Deletable:  false,
					Searchable: false,
				},
			},
			FunctionImports: map[string]*models.FunctionImport{
				"GetTopProducts": {
					Name:       "GetTopProducts",
					HTTPMethod: "GET",
					Parameters: []*models.FunctionParameter{
						{Name: "count", Type: "Edm.Int32", Mode: "In"},
					},
				},
			},
		},
	}

	description := bridge.generateUniversalDescription()

	// Check service URL is included
	if !strings.Contains(description, "https://example.com/odata") {
		t.Error("Description should contain service URL")
	}

	// Check entities are listed
	if !strings.Contains(description, "Products") {
		t.Error("Description should list Products entity")
	}
	if !strings.Contains(description, "Categories") {
		t.Error("Description should list Categories entity")
	}

	// Check capabilities are shown
	if !strings.Contains(description, "search") {
		t.Error("Description should show search capability for Products")
	}

	// Check functions are listed
	if !strings.Contains(description, "GetTopProducts") {
		t.Error("Description should list GetTopProducts function")
	}

	// Check action examples are included
	if !strings.Contains(description, "list") || !strings.Contains(description, "get") {
		t.Error("Description should include action examples")
	}
}

func TestUniversalToolGeneration(t *testing.T) {
	cfg := &config.Config{
		ServiceURL:    "https://example.com/odata",
		UniversalTool: true,
	}

	mcpServer := mcp.NewServer("test", "1.0")

	bridge := &ODataMCPBridge{
		config:      cfg,
		server:      mcpServer,
		tools:       make(map[string]*models.ToolInfo),
		hintManager: hint.NewManager(),
		metadata: &models.ODataMetadata{
			EntitySets: map[string]*models.EntitySet{
				"Products": {
					Name:       "Products",
					EntityType: "Product",
				},
			},
			FunctionImports: map[string]*models.FunctionImport{},
		},
	}

	bridge.generateUniversalTool()

	// Should generate exactly 1 tool
	if len(bridge.tools) != 1 {
		t.Errorf("Universal mode should generate 1 tool, got %d", len(bridge.tools))
	}

	// Check the tool exists with correct name
	var toolName string
	for name := range bridge.tools {
		toolName = name
	}

	if !strings.Contains(toolName, "OData") {
		t.Errorf("Universal tool should have 'OData' in name, got %s", toolName)
	}
}

func TestUniversalVsStandardToolCount(t *testing.T) {
	// Create metadata with multiple entities
	metadata := &models.ODataMetadata{
		EntitySets: map[string]*models.EntitySet{
			"Products":   {Name: "Products", EntityType: "Product", Creatable: true, Updatable: true, Deletable: true},
			"Categories": {Name: "Categories", EntityType: "Category", Creatable: true, Updatable: true, Deletable: true},
			"Orders":     {Name: "Orders", EntityType: "Order", Creatable: true, Updatable: true, Deletable: true},
		},
		EntityTypes: map[string]*models.EntityType{
			"Product":  {Name: "Product", KeyProperties: []string{"ID"}, Properties: []*models.EntityProperty{{Name: "ID", Type: "Edm.Int32", IsKey: true}}},
			"Category": {Name: "Category", KeyProperties: []string{"ID"}, Properties: []*models.EntityProperty{{Name: "ID", Type: "Edm.Int32", IsKey: true}}},
			"Order":    {Name: "Order", KeyProperties: []string{"ID"}, Properties: []*models.EntityProperty{{Name: "ID", Type: "Edm.Int32", IsKey: true}}},
		},
		FunctionImports: map[string]*models.FunctionImport{},
	}

	// Standard mode
	standardCfg := &config.Config{
		ServiceURL:    "https://example.com/odata",
		UniversalTool: false,
	}
	standardServer := mcp.NewServer("test", "1.0")
	standardBridge := &ODataMCPBridge{
		config:      standardCfg,
		server:      standardServer,
		tools:       make(map[string]*models.ToolInfo),
		hintManager: hint.NewManager(),
		metadata:    metadata,
	}
	standardBridge.generateTools()
	standardToolCount := len(standardBridge.tools)

	// Universal mode
	universalCfg := &config.Config{
		ServiceURL:    "https://example.com/odata",
		UniversalTool: true,
	}
	universalServer := mcp.NewServer("test", "1.0")
	universalBridge := &ODataMCPBridge{
		config:      universalCfg,
		server:      universalServer,
		tools:       make(map[string]*models.ToolInfo),
		hintManager: hint.NewManager(),
		metadata:    metadata,
	}
	universalBridge.generateTools()
	universalToolCount := len(universalBridge.tools)

	// Standard mode should have many more tools (1 service_info + ~5 per entity * 3 entities = ~16)
	// Universal mode should have exactly 1 tool
	if universalToolCount != 1 {
		t.Errorf("Universal mode should have 1 tool, got %d", universalToolCount)
	}

	if standardToolCount <= universalToolCount {
		t.Errorf("Standard mode (%d tools) should have more tools than universal mode (%d)", standardToolCount, universalToolCount)
	}

	t.Logf("Standard mode: %d tools, Universal mode: %d tools (reduction: %.1f%%)",
		standardToolCount, universalToolCount,
		float64(standardToolCount-universalToolCount)/float64(standardToolCount)*100)
}
