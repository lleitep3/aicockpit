package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantKeys    []string
		wantBody    string
		expectEmpty bool
	}{
		{
			name: "With Valid Frontmatter",
			content: `---
title: "Test Title"
description: Simple test
---
Body Content
Line 2`,
			wantKeys: []string{"title", "description"},
			wantBody: "Body Content\nLine 2",
		},
		{
			name: "No Frontmatter",
			content: `Some basic content
without frontmatter`,
			wantKeys:    nil,
			wantBody:    "Some basic content\nwithout frontmatter",
			expectEmpty: true,
		},
		{
			name: "Unterminated Frontmatter",
			content: `---
title: Unterminated
no closure`,
			wantKeys:    nil,
			wantBody:    "---\ntitle: Unterminated\nno closure",
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body := StripFrontmatter(tt.content)
			if tt.expectEmpty {
				if len(fm) != 0 {
					t.Errorf("expected empty frontmatter map, got %v", fm)
				}
			} else {
				for _, k := range tt.wantKeys {
					if _, exists := fm[k]; !exists {
						t.Errorf("expected key %s to exist in frontmatter", k)
					}
				}
			}
			if body != tt.wantBody {
				t.Errorf("expected body %q, got %q", tt.wantBody, body)
			}
		})
	}
}

func TestAddGeneratedHeader(t *testing.T) {
	content := "some content"
	mdResult := AddGeneratedHeader(content, "test-adapter.md")
	if !strings.Contains(mdResult, "GENERATED") || !strings.Contains(mdResult, "<!--") {
		t.Errorf("expected html comment style generated warning, got %q", mdResult)
	}

	yamlResult := AddGeneratedHeader(content, "config.yaml")
	if !strings.Contains(yamlResult, "GENERATED") || !strings.Contains(yamlResult, "#") {
		t.Errorf("expected bash comment style generated warning, got %q", yamlResult)
	}
}

func TestCanonicalHelpers(t *testing.T) {
	tmpDir := t.TempDir()

	// Write mock file
	err := os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("hello"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Test read helper
	content, err := ReadCanonicalFile(tmpDir, "test.txt")
	if err != nil || content != "hello" {
		t.Errorf("expected hello and no error, got %s and %v", content, err)
	}

	// Test read error
	_, err = ReadCanonicalFile(tmpDir, "nonexistent.txt")
	if err == nil {
		t.Error("expected error reading nonexistent file")
	}

	// Test list helper
	files, err := ReadCanonicalDir(tmpDir)
	if err != nil || len(files) != 1 || files[0] != "test.txt" {
		t.Errorf("expected test.txt, got %v, error: %v", files, err)
	}

	// Test list non-existent dir
	emptyFiles, err := ReadCanonicalDir(tmpDir, "nonexistent")
	if err != nil || len(emptyFiles) != 0 {
		t.Errorf("expected empty list and no error for nonexistent dir, got %v, %v", emptyFiles, err)
	}
}

func TestAdaptersAndManager(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup directories
	err := os.MkdirAll(filepath.Join(tmpDir, "rules"), 0755)
	if err != nil {
		t.Fatalf("failed to create mock rules directory: %v", err)
	}
	err = os.MkdirAll(filepath.Join(tmpDir, "skills", "test-skill"), 0755)
	if err != nil {
		t.Fatalf("failed to create mock skills directory: %v", err)
	}

	// Write mock files
	err = os.WriteFile(filepath.Join(tmpDir, "identity.md"), []byte("# Identity\nI am developer"), 0644)
	if err != nil {
		t.Fatalf("failed to write identity.md: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "rules", "01-rules.md"), []byte("## General Rules\nRule 1"), 0644)
	if err != nil {
		t.Fatalf("failed to write rules file: %v", err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "skills", "test-skill", "SKILL.md"), []byte("---\nname: test-skill\n---\nSkill content"), 0644)
	if err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}

	projectDir := t.TempDir()

	cfg := &ProvidersConfig{
		Version:     "1.0",
		Description: "Mock config",
		Providers: map[string]*Provider{
			"devin": {
				Enabled:     true,
				Name:        "Devin",
				Description: "Devin Mock",
				Workspace:   projectDir,
				Version:     "1.0.0",
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    "AGENTS.md",
					},
					"skills": {
						Enabled: true,
						Path:    ".agents/skills",
					},
					"workflows": {
						Enabled: true,
						Path:    ".devin/config.yaml",
					},
				},
			},
			"goose": {
				Enabled:     true,
				Name:        "Goose",
				Description: "Goose Mock",
				Workspace:   projectDir,
				Version:     "1.0.0",
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    ".goosehints",
					},
					"skills": {
						Enabled: true,
						Path:    ".agents/skills",
					},
				},
			},
			"antigravity": {
				Enabled:     true,
				Name:        "Antigravity",
				Description: "Antigravity Mock",
				Workspace:   projectDir,
				Version:     "1.0.0",
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    ".gemini/config/rules",
					},
					"skills": {
						Enabled: true,
						Path:    ".gemini/config/skills",
					},
				},
			},
			"disabled-provider": {
				Enabled:   false,
				Name:      "Disabled",
				Workspace: projectDir,
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    "rules",
					},
				},
			},
		},
	}

	pm := NewProviderManager(cfg)

	// Deploy Devin
	t.Run("Deploy Devin", func(t *testing.T) {
		err = pm.Deploy("devin", tmpDir, projectDir)
		if err != nil {
			t.Fatalf("devin deploy failed: %v", err)
		}

		// Verify AGENTS.md was created
		agentsPath := filepath.Join(projectDir, "AGENTS.md")
		if _, err := os.Stat(agentsPath); err != nil {
			t.Errorf("expected AGENTS.md to be created: %v", err)
		}

		// Verify .devin/config.yaml was created
		configPath := filepath.Join(projectDir, ".devin/config.yaml")
		if _, err := os.Stat(configPath); err != nil {
			t.Errorf("expected .devin/config.yaml to be created: %v", err)
		}

		// Verify skill was compiled
		skillPath := filepath.Join(projectDir, ".agents/skills/test-skill/SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			t.Errorf("expected SKILL.md to be compiled: %v", err)
		}
	})

	// Deploy Goose
	t.Run("Deploy Goose", func(t *testing.T) {
		err = pm.Deploy("goose", tmpDir, projectDir)
		if err != nil {
			t.Fatalf("goose deploy failed: %v", err)
		}

		// Verify .goosehints was created
		hintsPath := filepath.Join(projectDir, ".goosehints")
		if _, err := os.Stat(hintsPath); err != nil {
			t.Errorf("expected .goosehints to be created: %v", err)
		}
	})

	// Deploy Antigravity
	t.Run("Deploy Antigravity", func(t *testing.T) {
		err = pm.Deploy("antigravity", tmpDir, projectDir)
		if err != nil {
			t.Fatalf("antigravity deploy failed: %v", err)
		}

		// Verify rule was created under .gemini/config/rules
		rulePath := filepath.Join(projectDir, ".gemini/config/rules/01-rules.md")
		if _, err := os.Stat(rulePath); err != nil {
			t.Errorf("expected rules file to be created: %v", err)
		}
	})

	// Deploy nonexistent
	t.Run("Deploy nonexistent", func(t *testing.T) {
		err = pm.Deploy("nonexistent", tmpDir, projectDir)
		if err == nil {
			t.Fatal("expected error deploying nonexistent provider")
		}
	})

	// Deploy disabled
	t.Run("Deploy disabled", func(t *testing.T) {
		err = pm.Deploy("disabled-provider", tmpDir, projectDir)
		if err == nil {
			t.Fatal("expected error deploying provider with no registered adapter")
		}
	})

	// Test GetProviderOptions coverage
	t.Run("GetProviderOptions", func(t *testing.T) {
		opts := cfg.GetProviderOptions()
		if len(opts) != 3 { // devin, goose, antigravity
			t.Errorf("expected 3 options, got %d", len(opts))
		}
	})
}

func TestDeployEdgeCases(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := t.TempDir()

	cfg := &ProvidersConfig{
		Version: "1.0",
		Providers: map[string]*Provider{
			"antigravity": {
				Enabled:   true,
				Name:      "Antigravity",
				Workspace: "~",
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    ".gemini/rules",
					},
				},
			},
			"invalid-workspace": {
				Enabled:   true,
				Name:      "Invalid Workspace",
				Workspace: "",
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    "rules",
					},
				},
			},
		},
	}

	// Write mock rules
	err := os.MkdirAll(filepath.Join(tmpDir, "rules"), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(tmpDir, "rules", "rule.md"), []byte("rule"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	pm := NewProviderManager(cfg)

	// Deploy with tilde workspace
	t.Run("Tilde Workspace", func(t *testing.T) {
		cfg.Providers["antigravity"].Workspace = projectDir
		cfg.Providers["antigravity"].Features["rules"].Path = "~/mock-rules"
		err := pm.Deploy("antigravity", tmpDir, projectDir)
		if err != nil {
			t.Logf("deploy returned: %v", err)
		}
	})

	// Test compiler when files are missing
	t.Run("Empty Devin Compilation", func(t *testing.T) {
		emptyDir := t.TempDir()
		devinAdapter := NewDevinAdapter()
		provider := cfg.Providers["antigravity"] // any provider
		files, err := devinAdapter.Compile(emptyDir, provider)
		if err != nil {
			t.Fatalf("devin compile failed on empty dir: %v", err)
		}
		if len(files) == 0 {
			t.Error("expected compiled files map to be populated")
		}
	})

	// Test absolute path target in Deploy
	t.Run("Absolute Path Target", func(t *testing.T) {
		cfg.Providers["antigravity"].Workspace = projectDir
		absPath := filepath.Join(projectDir, "my-absolute-rules-file.md")
		cfg.Providers["antigravity"].Features["rules"].Path = absPath
		err := pm.Deploy("antigravity", tmpDir, projectDir)
		if err != nil {
			t.Fatalf("expected deploy to succeed with absolute path: %v", err)
		}
		if _, err := os.Stat(absPath); err != nil {
			t.Errorf("expected absolute path file to be created: %v", err)
		}
	})
}

func TestProvidersConfigValidationErrors(t *testing.T) {
	// 1. Invalid YAML
	t.Run("Invalid YAML", func(t *testing.T) {
		tmpDir := t.TempDir()
		invalidYamlPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidYamlPath, []byte("invalid: yaml: :"), 0644)
		if err != nil {
			t.Fatal(err)
		}
		_, err = LoadProvidersConfig(invalidYamlPath)
		if err == nil {
			t.Error("expected error loading invalid YAML")
		}
	})

	// 2. Save to invalid path
	t.Run("Save Error", func(t *testing.T) {
		cfg := &ProvidersConfig{Version: "1.0"}
		err := SaveProvidersConfig("/nonexistent-dir/config.yaml", cfg)
		if err == nil {
			t.Error("expected error saving to nonexistent absolute dir")
		}
	})
}

func TestTildePathExpansion(t *testing.T) {
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", tmpDir)

	cfg := &ProvidersConfig{
		Version: "1.0",
		Providers: map[string]*Provider{
			"antigravity": {
				Enabled:   true,
				Workspace: "~", // test workspace tilde expansion
				Name:      "Antigravity",
				Features: map[string]*FeatureConfig{
					"rules": {
						Enabled: true,
						Path:    "rules",
					},
				},
			},
		},
	}

	// Test Save with tilde
	configPath := "~/providers-test-tilde.yaml"
	err := SaveProvidersConfig(configPath, cfg)
	if err != nil {
		t.Fatalf("failed to save config with tilde: %v", err)
	}

	// Verify file was written inside tmpDir
	expectedFile := filepath.Join(tmpDir, "providers-test-tilde.yaml")
	if _, err := os.Stat(expectedFile); err != nil {
		t.Errorf("expected config file to be written to %s: %v", expectedFile, err)
	}

	// Test Load with tilde
	loaded, err := LoadProvidersConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config with tilde: %v", err)
	}
	if loaded.Version != "1.0" {
		t.Errorf("expected version 1.0, got %s", loaded.Version)
	}

	// Test GetWorkspacePath tilde expansion
	p := loaded.GetProvider("antigravity")
	wsPath := p.GetWorkspacePath()
	if wsPath != tmpDir {
		t.Errorf("expected workspace path to be %s, got %s", tmpDir, wsPath)
	}

	// Test GetFeaturePath tilde expansion
	fPath := p.GetFeaturePath("rules")
	expectedFPath := filepath.Join(tmpDir, "rules")
	if fPath != expectedFPath {
		t.Errorf("expected feature path to be %s, got %s", expectedFPath, fPath)
	}
}

func TestProviderConfigEdgeCases(t *testing.T) {
	// 1. GetFeaturePath on nonexistent feature
	p := &Provider{
		Workspace: "/tmp",
		Features: map[string]*FeatureConfig{
			"rules": {
				Enabled: true,
				Path:    "rules",
			},
		},
	}
	if p.GetFeaturePath("nonexistent") != "" {
		t.Error("expected empty string for nonexistent feature")
	}

	// 2. GetFeaturePath on nil features
	pNil := &Provider{Workspace: "/tmp"}
	if pNil.GetFeaturePath("rules") != "" {
		t.Error("expected empty string for nil features")
	}

	// 3. AddProvider already exists
	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin": {Name: "Devin"},
		},
	}
	err := cfg.AddProvider("devin", &Provider{Name: "Devin2"})
	if err == nil {
		t.Error("expected error adding provider that already exists")
	}

	// 4. RemoveProvider nonexistent
	err = cfg.RemoveProvider("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent provider")
	}

	// 5. RemoveProvider on nil map
	cfgNil := &ProvidersConfig{}
	err = cfgNil.RemoveProvider("devin")
	if err == nil {
		t.Error("expected error removing provider from nil map")
	}
}
