package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProvidersConfig(t *testing.T) {
	// Create a temporary providers config
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "providers.yaml")

	config := &ProvidersConfig{
		Version:     "1.0",
		Description: "Test providers",
		Providers: map[string]*Provider{
			"devin": {
				Enabled:     true,
				Name:        "Devin",
				Description: "Test provider",
				Workspace:   tmpDir,
				Version:     "1.0.0",
				Features: map[string]*FeatureConfig{
					"agents": {
						Enabled:     true,
						Path:        "agents",
						Description: "Agents",
					},
				},
			},
		},
	}

	// Save config
	if err := SaveProvidersConfig(configPath, config); err != nil {
		t.Fatalf("SaveProvidersConfig failed: %v", err)
	}

	// Load config
	loaded, err := LoadProvidersConfig(configPath)
	if err != nil {
		t.Fatalf("LoadProvidersConfig failed: %v", err)
	}

	if loaded.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", loaded.Version)
	}

	if len(loaded.Providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(loaded.Providers))
	}

	provider := loaded.GetProvider("devin")
	if provider == nil {
		t.Fatal("Expected devin provider to exist")
	}

	if !provider.Enabled {
		t.Error("Expected devin provider to be enabled")
	}
}

func TestGetProvider(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin": {
				Name: "Devin",
			},
			"goose": {
				Name: "Goose",
			},
		},
	}

	tests := []struct {
		name     string
		provider string
		found    bool
	}{
		{"devin", "devin", true},
		{"goose", "goose", true},
		{"unknown", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := config.GetProvider(tt.provider)
			if (provider != nil) != tt.found {
				t.Errorf("Expected found=%v, got %v", tt.found, provider != nil)
			}
		})
	}
}

func TestGetEnabledProviders(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin": {
				Enabled: true,
				Name:    "Devin",
			},
			"goose": {
				Enabled: false,
				Name:    "Goose",
			},
			"claude-code": {
				Enabled: true,
				Name:    "Claude Code",
			},
		},
	}

	enabled := config.GetEnabledProviders()
	if len(enabled) != 2 {
		t.Errorf("Expected 2 enabled providers, got %d", len(enabled))
	}

	// Check that only enabled providers are returned
	for _, provider := range enabled {
		if !provider.Enabled {
			t.Errorf("Expected enabled provider, got disabled: %s", provider.Name)
		}
	}
}

func TestEnableProvider(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin": {
				Enabled: false,
				Name:    "Devin",
			},
		},
	}

	if err := config.EnableProvider("devin"); err != nil {
		t.Fatalf("EnableProvider failed: %v", err)
	}

	if !config.IsProviderEnabled("devin") {
		t.Error("Expected devin to be enabled")
	}
}

func TestDisableProvider(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin": {
				Enabled: true,
				Name:    "Devin",
			},
		},
	}

	if err := config.DisableProvider("devin"); err != nil {
		t.Fatalf("DisableProvider failed: %v", err)
	}

	if config.IsProviderEnabled("devin") {
		t.Error("Expected devin to be disabled")
	}
}

func TestGetSupportedFeatures(t *testing.T) {
	provider := &Provider{
		Name: "Devin",
		Features: map[string]*FeatureConfig{
			"agents": {
				Enabled: true,
			},
			"skills": {
				Enabled: true,
			},
			"hooks": {
				Enabled: false,
			},
		},
	}

	features := provider.GetSupportedFeatures()
	if len(features) != 2 {
		t.Errorf("Expected 2 supported features, got %d", len(features))
	}

	// Check that only enabled features are returned
	for _, feature := range features {
		if feature == "hooks" {
			t.Error("Expected hooks to not be in supported features")
		}
	}
}

func TestSupportsFeature(t *testing.T) {
	provider := &Provider{
		Name: "Devin",
		Features: map[string]*FeatureConfig{
			"agents": {
				Enabled: true,
			},
			"skills": {
				Enabled: false,
			},
		},
	}

	tests := []struct {
		feature  string
		supports bool
	}{
		{"agents", true},
		{"skills", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.feature, func(t *testing.T) {
			if provider.SupportsFeature(tt.feature) != tt.supports {
				t.Errorf("Expected SupportsFeature(%s)=%v", tt.feature, tt.supports)
			}
		})
	}
}

func TestGetFeaturePath(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &Provider{
		Name:      "Devin",
		Workspace: tmpDir,
		Features: map[string]*FeatureConfig{
			"agents": {
				Enabled: true,
				Path:    "agents",
			},
		},
	}

	path := provider.GetFeaturePath("agents")
	expected := filepath.Join(tmpDir, "agents")
	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}

func TestGetWorkspacePath(t *testing.T) {
	tmpDir := t.TempDir()
	provider := &Provider{
		Name:      "Devin",
		Workspace: tmpDir,
	}

	path := provider.GetWorkspacePath()
	if path != tmpDir {
		t.Errorf("Expected path %s, got %s", tmpDir, path)
	}
}

func TestGetWorkspacePathWithTilde(t *testing.T) {
	provider := &Provider{
		Name:      "Devin",
		Workspace: "~/.cockpit",
	}

	path := provider.GetWorkspacePath()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".cockpit")

	if path != expected {
		t.Errorf("Expected path %s, got %s", expected, path)
	}
}

func TestAddProvider(t *testing.T) {
	config := &ProvidersConfig{
		Providers: make(map[string]*Provider),
	}

	newProvider := &Provider{
		Enabled:   true,
		Name:      "NewProvider",
		Workspace: "~/.new-provider",
		Features: map[string]*FeatureConfig{
			"agents": {Enabled: true},
		},
	}

	if err := config.AddProvider("new-provider", newProvider); err != nil {
		t.Fatalf("AddProvider failed: %v", err)
	}

	provider := config.GetProvider("new-provider")
	if provider == nil {
		t.Fatal("Expected new-provider to exist")
	}

	if !provider.Enabled {
		t.Error("Expected new-provider to be enabled")
	}
}

func TestRemoveProvider(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin": {
				Name: "Devin",
			},
		},
	}

	if err := config.RemoveProvider("devin"); err != nil {
		t.Fatalf("RemoveProvider failed: %v", err)
	}

	if config.GetProvider("devin") != nil {
		t.Error("Expected devin provider to be removed")
	}
}

func TestValidateProvider(t *testing.T) {
	tests := []struct {
		name     string
		provider *Provider
		wantErr  bool
	}{
		{
			name: "valid provider",
			provider: &Provider{
				Name:      "Test",
				Workspace: "~/.test",
				Features: map[string]*FeatureConfig{
					"agents": {Enabled: true},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			provider: &Provider{
				Workspace: "~/.test",
				Features: map[string]*FeatureConfig{
					"agents": {Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "missing workspace",
			provider: &Provider{
				Name: "Test",
				Features: map[string]*FeatureConfig{
					"agents": {Enabled: true},
				},
			},
			wantErr: true,
		},
		{
			name: "no features",
			provider: &Provider{
				Name:      "Test",
				Workspace: "~/.test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.provider.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *ProvidersConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: &ProvidersConfig{
				Version: "1.0",
				Providers: map[string]*Provider{
					"devin": {
						Name:      "Devin",
						Workspace: "~/.cockpit",
						Features: map[string]*FeatureConfig{
							"agents": {Enabled: true},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing version",
			config: &ProvidersConfig{
				Providers: map[string]*Provider{
					"devin": {
						Name:      "Devin",
						Workspace: "~/.cockpit",
						Features: map[string]*FeatureConfig{
							"agents": {Enabled: true},
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no providers",
			config: &ProvidersConfig{
				Version: "1.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateConfig()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetProviderNames(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin":       {Name: "Devin"},
			"goose":       {Name: "Goose"},
			"claude-code": {Name: "Claude Code"},
		},
	}

	names := config.GetProviderNames()
	if len(names) != 3 {
		t.Errorf("Expected 3 provider names, got %d", len(names))
	}

	// Check that all names are present
	nameMap := make(map[string]bool)
	for _, name := range names {
		nameMap[name] = true
	}

	expectedNames := []string{"devin", "goose", "claude-code"}
	for _, expected := range expectedNames {
		if !nameMap[expected] {
			t.Errorf("Expected provider name %s not found", expected)
		}
	}
}

func TestMultiProviderFeatures(t *testing.T) {
	config := &ProvidersConfig{
		Version: "1.0",
		Providers: map[string]*Provider{
			"devin": {
				Enabled:   true,
				Name:      "Devin",
				Workspace: "~/.cockpit",
				Features: map[string]*FeatureConfig{
					"agents":    {Enabled: true, Path: "agents"},
					"skills":    {Enabled: true, Path: "skills"},
					"hooks":     {Enabled: true, Path: "hooks"},
					"workflows": {Enabled: true, Path: "workflows"},
					"memories":  {Enabled: true, Path: "memories"},
					"kb":        {Enabled: true, Path: "kb"},
				},
			},
			"goose": {
				Enabled:   true,
				Name:      "Goose",
				Workspace: "~/.goose",
				Features: map[string]*FeatureConfig{
					"agents":    {Enabled: true, Path: "agents"},
					"skills":    {Enabled: true, Path: "skills"},
					"hooks":     {Enabled: true, Path: "hooks"},
					"workflows": {Enabled: true, Path: "workflows"},
					"memories":  {Enabled: false, Path: "memories"},
					"kb":        {Enabled: true, Path: "kb"},
				},
			},
			"claude-code": {
				Enabled:   true,
				Name:      "Claude Code",
				Workspace: "~/.claude-code",
				Features: map[string]*FeatureConfig{
					"agents":    {Enabled: false, Path: "agents"},
					"skills":    {Enabled: true, Path: "skills"},
					"hooks":     {Enabled: true, Path: "hooks"},
					"workflows": {Enabled: false, Path: "workflows"},
					"memories":  {Enabled: true, Path: "memories"},
					"kb":        {Enabled: true, Path: "kb"},
				},
			},
		},
	}

	// Verify Devin supports all features
	devin := config.GetProvider("devin")
	if len(devin.GetSupportedFeatures()) != 6 {
		t.Errorf("Expected Devin to support 6 features, got %d", len(devin.GetSupportedFeatures()))
	}

	// Verify Goose doesn't support memories
	goose := config.GetProvider("goose")
	if goose.SupportsFeature("memories") {
		t.Error("Expected Goose to not support memories")
	}

	// Verify Claude Code doesn't support agents and workflows
	claudeCode := config.GetProvider("claude-code")
	if claudeCode.SupportsFeature("agents") {
		t.Error("Expected Claude Code to not support agents")
	}
	if claudeCode.SupportsFeature("workflows") {
		t.Error("Expected Claude Code to not support workflows")
	}
}
