package packages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPackage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test package manifest
	manifestPath := filepath.Join(tmpDir, "cockpit-package.yml")
	manifestContent := `
name: "test-package"
version: "1.0.0"
description: "Test package"
author: "Test Author"
license: "MIT"
type: "utility"

requirements:
  cockpit: "0.2.0"

features:
  skills:
    - path: "skills/test.go"
      name: "test-skill"
      description: "Test skill"

installation:
  supported_providers:
    - devin
  provider_features:
    devin:
      - skills
  method: "symlink"
`

	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("Failed to create test manifest: %v", err)
	}

	pkg, err := LoadPackage(tmpDir)
	if err != nil {
		t.Fatalf("LoadPackage failed: %v", err)
	}

	if pkg.Name != "test-package" {
		t.Errorf("Expected name 'test-package', got '%s'", pkg.Name)
	}

	if pkg.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", pkg.Version)
	}

	if pkg.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got '%s'", pkg.Author)
	}
}

func TestValidatePackage(t *testing.T) {
	tests := []struct {
		name    string
		pkg     *Package
		wantErr bool
	}{
		{
			name: "valid package",
			pkg: &Package{
				Name:        "test",
				Version:     "1.0.0",
				Description: "Test",
				Author:      "Author",
				License:     "MIT",
				Type:        "utility",
				Requirements: Requirements{
					Cockpit: "0.2.0",
				},
				Features: Features{
					Skills: []Feature{
						{Path: "skills/test.go", Name: "test"},
					},
				},
				Installation: Installation{
					SupportedProviders: []string{"devin"},
					ProviderFeatures: map[string][]string{
						"devin": {"skills"},
					},
					Method: "symlink",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			pkg: &Package{
				Version:     "1.0.0",
				Description: "Test",
				Author:      "Author",
				License:     "MIT",
				Requirements: Requirements{
					Cockpit: "0.2.0",
				},
				Features: Features{
					Skills: []Feature{
						{Path: "skills/test.go", Name: "test"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "missing version",
			pkg: &Package{
				Name:        "test",
				Description: "Test",
				Author:      "Author",
				License:     "MIT",
				Requirements: Requirements{
					Cockpit: "0.2.0",
				},
				Features: Features{
					Skills: []Feature{
						{Path: "skills/test.go", Name: "test"},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "no features",
			pkg: &Package{
				Name:        "test",
				Version:     "1.0.0",
				Description: "Test",
				Author:      "Author",
				License:     "MIT",
				Requirements: Requirements{
					Cockpit: "0.2.0",
				},
				Features: Features{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.pkg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFeaturesByType(t *testing.T) {
	pkg := &Package{
		Features: Features{
			Agents: []Feature{
				{Path: "agents/test.go", Name: "test-agent"},
			},
			Skills: []Feature{
				{Path: "skills/test.go", Name: "test-skill"},
			},
		},
	}

	tests := []struct {
		featureType string
		expected    int
	}{
		{"agents", 1},
		{"skills", 1},
		{"modules", 0},
		{"workflows", 0},
	}

	for _, tt := range tests {
		features := pkg.GetFeaturesByType(tt.featureType)
		if len(features) != tt.expected {
			t.Errorf("GetFeaturesByType(%s) returned %d features, expected %d",
				tt.featureType, len(features), tt.expected)
		}
	}
}

func TestSupportsProvider(t *testing.T) {
	pkg := &Package{
		Installation: Installation{
			SupportedProviders: []string{"devin", "goose"},
		},
	}

	tests := []struct {
		provider string
		expected bool
	}{
		{"devin", true},
		{"goose", true},
		{"claude-code", false},
		{"github-copilot", false},
	}

	for _, tt := range tests {
		result := pkg.SupportsProvider(tt.provider)
		if result != tt.expected {
			t.Errorf("SupportsProvider(%s) = %v, expected %v",
				tt.provider, result, tt.expected)
		}
	}
}

func TestGetProviderFeatures(t *testing.T) {
	pkg := &Package{
		Installation: Installation{
			ProviderFeatures: map[string][]string{
				"devin": {"agents", "skills", "modules"},
				"goose": {"skills", "modules"},
			},
		},
	}

	tests := []struct {
		provider string
		expected int
	}{
		{"devin", 3},
		{"goose", 2},
		{"claude-code", 0},
	}

	for _, tt := range tests {
		features := pkg.GetProviderFeatures(tt.provider)
		if len(features) != tt.expected {
			t.Errorf("GetProviderFeatures(%s) returned %d features, expected %d",
				tt.provider, len(features), tt.expected)
		}
	}
}

func TestHasDependencies(t *testing.T) {
	tests := []struct {
		name     string
		pkg      *Package
		expected bool
	}{
		{
			name: "with dependencies",
			pkg: &Package{
				Dependencies: []Dependency{
					{Name: "dep1", Version: "1.0.0"},
				},
			},
			expected: true,
		},
		{
			name: "without dependencies",
			pkg: &Package{
				Dependencies: []Dependency{},
			},
			expected: false,
		},
		{
			name: "nil dependencies",
			pkg: &Package{
				Dependencies: nil,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pkg.HasDependencies()
			if result != tt.expected {
				t.Errorf("HasDependencies() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHasExternalDependencies(t *testing.T) {
	tests := []struct {
		name     string
		pkg      *Package
		expected bool
	}{
		{
			name: "with go dependencies",
			pkg: &Package{
				ExternalDependencies: ExternalDeps{
					Go: []string{"github.com/test/pkg@v1.0.0"},
				},
			},
			expected: true,
		},
		{
			name: "with node dependencies",
			pkg: &Package{
				ExternalDependencies: ExternalDeps{
					Node: []string{"express@^4.0.0"},
				},
			},
			expected: true,
		},
		{
			name: "with system dependencies",
			pkg: &Package{
				ExternalDependencies: ExternalDeps{
					System: []string{"pandoc>=2.0"},
				},
			},
			expected: true,
		},
		{
			name: "without external dependencies",
			pkg: &Package{
				ExternalDependencies: ExternalDeps{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.pkg.HasExternalDependencies()
			if result != tt.expected {
				t.Errorf("HasExternalDependencies() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestGetDefaultConfig(t *testing.T) {
	pkg := &Package{
		Configuration: Configuration{
			Defaults: map[string]interface{}{
				"output_dir": "articles",
				"theme":      "light",
			},
		},
	}

	config := pkg.GetDefaultConfig()
	if len(config) != 2 {
		t.Errorf("GetDefaultConfig() returned %d items, expected 2", len(config))
	}

	if config["output_dir"] != "articles" {
		t.Errorf("Expected output_dir='articles', got '%v'", config["output_dir"])
	}

	if config["theme"] != "light" {
		t.Errorf("Expected theme='light', got '%v'", config["theme"])
	}
}

func TestSavePackage(t *testing.T) {
	tmpDir := t.TempDir()

	pkg := &Package{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Author:      "Test Author",
		License:     "MIT",
		Type:        "utility",
		Requirements: Requirements{
			Cockpit: "0.2.0",
		},
		Features: Features{
			Skills: []Feature{
				{Path: "skills/test.go", Name: "test-skill"},
			},
		},
		Installation: Installation{
			SupportedProviders: []string{"devin"},
			ProviderFeatures: map[string][]string{
				"devin": {"skills"},
			},
			Method: "symlink",
		},
	}

	if err := SavePackage(tmpDir, pkg); err != nil {
		t.Fatalf("SavePackage failed: %v", err)
	}

	// Verify manifest was created
	manifestPath := filepath.Join(tmpDir, "cockpit-package.yml")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Error("Manifest file was not created")
	}

	// Load and verify
	loaded, err := LoadPackage(tmpDir)
	if err != nil {
		t.Fatalf("Failed to load saved package: %v", err)
	}

	if loaded.Name != pkg.Name {
		t.Errorf("Expected name '%s', got '%s'", pkg.Name, loaded.Name)
	}

	if loaded.Version != pkg.Version {
		t.Errorf("Expected version '%s', got '%s'", pkg.Version, loaded.Version)
	}
}
