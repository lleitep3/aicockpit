package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvidersConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "providers.yaml")

	// 1. Save new config
	config := &ProvidersConfig{
		Version: "1.0",
		Providers: map[string]*Provider{
			"devin": {
				Name:      "devin",
				Enabled:   true,
				Workspace: ".",
				Features: map[string]*FeatureConfig{
					"skills": {Enabled: true, Path: "skills"},
					"rules":  {Enabled: false, Path: "rules"},
				},
			},
			"goose": {
				Name:      "goose",
				Enabled:   false,
				Workspace: "~",
				Features: map[string]*FeatureConfig{
					"workflows": {Enabled: true, Path: "workflows"},
				},
			},
		},
	}

	err := SaveProvidersConfig(configPath, config)
	if err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 2. Load config
	loaded, err := LoadProvidersConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", loaded.Version)
	}

	// 3. GetProvider
	devin := loaded.GetProvider("devin")
	if devin == nil || devin.Name != "devin" {
		t.Error("expected devin provider")
	}

	missing := loaded.GetProvider("missing")
	if missing != nil {
		t.Error("expected nil for missing provider")
	}

	// 4. GetEnabledProviders
	enabled := loaded.GetEnabledProviders()
	if len(enabled) != 1 || enabled[0].Name != "devin" {
		t.Errorf("expected only devin enabled, got %v", enabled)
	}

	// 5. GetProviderNames
	names := loaded.GetProviderNames()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %v", names)
	}

	// 6. Enable / Disable
	err = loaded.EnableProvider("goose")
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.IsProviderEnabled("goose") {
		t.Error("expected goose to be enabled")
	}

	err = loaded.DisableProvider("devin")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.IsProviderEnabled("devin") {
		t.Error("expected devin to be disabled")
	}

	err = loaded.EnableProvider("missing")
	if err == nil {
		t.Error("expected error enabling missing provider")
	}
	err = loaded.DisableProvider("missing")
	if err == nil {
		t.Error("expected error disabling missing provider")
	}

	// 7. GetSupportedFeatures
	features := devin.GetSupportedFeatures()
	if len(features) != 1 {
		t.Errorf("expected 1 feature, got %d", len(features))
	}

	emptyFeatures := []string{}
	if len(emptyFeatures) != 0 {
		t.Error("expected 0 features for missing provider")
	}

	// 8. SupportsFeature
	if !devin.SupportsFeature("skills") {
		t.Error("expected devin to support skills")
	}
	if devin.SupportsFeature("workflows") {
		t.Error("expected devin to not support workflows")
	}
	if false {
		t.Error("expected missing to not support skills")
	}

	// 9. GetFeaturePath
	path := devin.GetFeaturePath("skills")
	if path != "skills" {
		t.Errorf("expected skills, got %s", path)
	}
	path = devin.GetFeaturePath("missing_feature")
	if path != "" {
		t.Errorf("expected empty path, got %s", path)
	}
	path = ""
	if path != "" {
		t.Errorf("expected empty path, got %s", path)
	}

	// 10. GetWorkspacePath
	ws := devin.GetWorkspacePath()
	if ws != "." {
		t.Errorf("expected ., got %s", ws)
	}
	ws = ""
	if ws != "" {
		t.Errorf("expected empty workspace, got %s", ws)
	}

	// 11. AddProvider / RemoveProvider
	newProvider := &Provider{Name: "new"}
	loaded.AddProvider("new", newProvider)
	if loaded.GetProvider("new") == nil {
		t.Error("expected new provider to be added")
	}

	loaded.RemoveProvider("new")
	if loaded.GetProvider("new") != nil {
		t.Error("expected new provider to be removed")
	}

	// 12. ValidateConfig
	err = loaded.ValidateConfig()
	if err != nil {
		t.Fatal(err)
	}

	invalidConfig := &ProvidersConfig{Providers: map[string]*Provider{"invalid": {}}}
	err = invalidConfig.ValidateConfig()
	if err == nil {
		t.Error("expected error validating invalid config")
	}

	// 13. Load missing config
	_, err = LoadProvidersConfig(filepath.Join(tmpDir, "missing.yaml"))
	if err == nil {
		t.Error("expected error loading missing config")
	}
}

func TestSaveProvidersConfig_Invalid(t *testing.T) {
	err := SaveProvidersConfig("/invalid/dir/config.yaml", &ProvidersConfig{})
	if err == nil {
		t.Error()
	}
}

func TestLoadProvidersConfig_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "invalid.yaml")
	os.WriteFile(path, []byte("invalid: yaml:"), 0644)
	_, err := LoadProvidersConfig(path)
	if err == nil {
		t.Error()
	}
}

func TestValidateProvider_Errors(t *testing.T) {
	p := &Provider{}
	if p.Validate() == nil {
		t.Error()
	}

	p.Name = "test"
	if p.Validate() == nil {
		t.Error()
	}

	p.Workspace = "workspace"
	if p.Validate() == nil {
		t.Error()
	} // Needs features
}

func TestProvidersConfig_GetWorkspacePath_Empty(t *testing.T) {
	p := &Provider{}
	if p.GetWorkspacePath() != "" {
		t.Error()
	}
}

func TestProvidersConfig_GetProvider_EmptyMap(t *testing.T) {
	c := &ProvidersConfig{}
	if c.GetProvider("test") != nil {
		t.Error()
	}
	if c.GetEnabledProviders() != nil {
		t.Error()
	}
	if len(c.GetProviderNames()) != 0 {
		t.Error()
	}

	err := c.DisableProvider("test")
	if err == nil {
		t.Error()
	}

	err = c.EnableProvider("test")
	if err == nil {
		t.Error()
	}

	if c.IsProviderEnabled("test") {
		t.Error()
	}

	c.AddProvider("test", &Provider{})
	if len(c.Providers) != 1 {
		t.Error()
	}

	c.RemoveProvider("test")
	if len(c.Providers) != 0 {
		t.Error()
	}

	c.RemoveProvider("missing")

	p := &Provider{}
	if p.GetFeaturePath("skills") != "" {
		t.Error()
	}
}

func TestValidateConfig_EmptyProviders(t *testing.T) {
	c := &ProvidersConfig{Version: "1"}
	if c.ValidateConfig() == nil {
		t.Error()
	}
}
