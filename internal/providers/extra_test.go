package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvidersConfig_NilMaps(t *testing.T) {
	// Test methods on config with nil Providers map
	cfg := &ProvidersConfig{Version: "1.0"}

	if p := cfg.GetProvider("devin"); p != nil {
		t.Errorf("GetProvider should return nil, got %v", p)
	}

	if eps := cfg.GetEnabledProviders(); eps != nil {
		t.Errorf("GetEnabledProviders should return nil, got %v", eps)
	}

	if names := cfg.GetProviderNames(); names != nil {
		t.Errorf("GetProviderNames should return nil, got %v", names)
	}

	if err := cfg.EnableProvider("devin"); err == nil {
		t.Error("EnableProvider should fail with no providers")
	}

	if err := cfg.DisableProvider("devin"); err == nil {
		t.Error("DisableProvider should fail with no providers")
	}

	if enabled := cfg.IsProviderEnabled("devin"); enabled {
		t.Error("IsProviderEnabled should return false")
	}

	if opts := cfg.GetProviderOptions(); opts != nil {
		t.Errorf("GetProviderOptions should return nil, got %v", opts)
	}

	// Test methods on provider with nil Features map
	pNilFeatures := &Provider{Features: nil}
	if feats := pNilFeatures.GetSupportedFeatures(); feats != nil {
		t.Errorf("GetSupportedFeatures should return nil, got %v", feats)
	}

	if supports := pNilFeatures.SupportsFeature("rules"); supports {
		t.Error("SupportsFeature should return false")
	}

	// Test AddProvider when Providers map is nil (initializes it)
	p := &Provider{Name: "New Devin", Workspace: "/tmp"}
	if err := cfg.AddProvider("devin", p); err != nil {
		t.Fatalf("AddProvider failed on nil map: %v", err)
	}
	if len(cfg.Providers) != 1 {
		t.Errorf("expected 1 provider, got %d", len(cfg.Providers))
	}
}

func TestProvider_EmptyWorkspace(t *testing.T) {
	p := &Provider{Workspace: ""}
	if ws := p.GetWorkspacePath(); ws != "" {
		t.Errorf("expected empty workspace path, got %q", ws)
	}
}

func TestManager_Deploy_UserHomeError(t *testing.T) {
	// Unset HOME env variables to force os.UserHomeDir to fail
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Unsetenv("HOME")

	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"antigravity": {
				Enabled:   true,
				Workspace: "~", // triggering home dir resolution
				Name:      "Antigravity",
			},
		},
	}

	pm := NewProviderManager(cfg)
	pm.Register(NewAntigravityAdapter())

	err := pm.Deploy("antigravity", "/tmp/cockpit", "")
	if err == nil {
		t.Error("expected error deploying with unset HOME")
	}
}

func TestManager_Deploy_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"antigravity": {
				Enabled:   true,
				Workspace: filePath,
				Name:      "Antigravity",
				Features: map[string]*FeatureConfig{
					"rules": {Enabled: true, Path: "rules"},
				},
			},
		},
	}

	pm := NewProviderManager(cfg)
	pm.Register(NewAntigravityAdapter())

	cockpitHome := filepath.Join(tmpDir, "cockpit")
	err = os.MkdirAll(filepath.Join(cockpitHome, "rules"), 0755)
	if err != nil {
		t.Fatalf("failed to create cockpit rules dir: %v", err)
	}
	err = os.WriteFile(filepath.Join(cockpitHome, "rules", "dev-rules.md"), []byte("rules"), 0644)
	if err != nil {
		t.Fatalf("failed to write rule: %v", err)
	}

	err = pm.Deploy("antigravity", cockpitHome, filePath)
	if err == nil {
		t.Error("expected error during MkdirAll because parent is a file")
	}
}

func TestReadCanonicalDir_Error(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(filePath, []byte("content"), 0644)
	if err != nil {
		t.Fatalf("failed to write mock file: %v", err)
	}

	_, err = ReadCanonicalDir(filePath)
	if err == nil {
		t.Error("expected error reading file as directory")
	}
}
