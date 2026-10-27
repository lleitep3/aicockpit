package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockCompiler implements Compiler for testing
type mockCompiler struct {
	name               string
	compileEntrypoint  func(ep *CanonicalEntrypoint, p *Provider) (map[string]string, error)
	compileSkills      func(skills []CanonicalSkill, p *Provider) (map[string]string, error)
	compileRules       func(rules []CanonicalRule, p *Provider) (map[string]string, error)
	compileWorkflows   func(workflows []CanonicalWorkflow, p *Provider) (map[string]string, error)
	compilePermissions func(perms *CanonicalPermissions, p *Provider) (map[string]string, error)
}

func (m *mockCompiler) Name() string { return m.name }
func (m *mockCompiler) CompileEntrypoint(ep *CanonicalEntrypoint, p *Provider) (map[string]string, error) {
	if m.compileEntrypoint != nil {
		return m.compileEntrypoint(ep, p)
	}
	return make(map[string]string), nil
}
func (m *mockCompiler) CompileSkills(skills []CanonicalSkill, p *Provider) (map[string]string, error) {
	if m.compileSkills != nil {
		return m.compileSkills(skills, p)
	}
	return make(map[string]string), nil
}
func (m *mockCompiler) CompileRules(rules []CanonicalRule, p *Provider) (map[string]string, error) {
	if m.compileRules != nil {
		return m.compileRules(rules, p)
	}
	return make(map[string]string), nil
}
func (m *mockCompiler) CompileWorkflows(workflows []CanonicalWorkflow, p *Provider) (map[string]string, error) {
	if m.compileWorkflows != nil {
		return m.compileWorkflows(workflows, p)
	}
	return make(map[string]string), nil
}
func (m *mockCompiler) CompilePermissions(perms *CanonicalPermissions, p *Provider) (map[string]string, error) {
	if m.compilePermissions != nil {
		return m.compilePermissions(perms, p)
	}
	return make(map[string]string), nil
}
func (m *mockCompiler) CompileAgents(agents []CanonicalAgent, p *Provider) (map[string]string, error) {
	return make(map[string]string), nil
}

func TestManagerDeploy(t *testing.T) {
	tmpProjectDir := t.TempDir()

	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"mock": {
				Name:      "mock",
				Enabled:   true,
				Workspace: ".",
				Features: map[string]*FeatureConfig{
					"rules": {Enabled: true, Path: "test_rules.md"},
				},
			},
		},
	}

	pm := NewProviderManager(config)

	// Register our mock compiler
	pm.Register(&mockCompiler{
		name: "mock",
		compileRules: func(rules []CanonicalRule, p *Provider) (map[string]string, error) {
			return map[string]string{
				"test_rules.md": "Compiled Rule Content",
			}, nil
		},
	})

	// Setup mock cockpit dir using parser_test.go helper (which we just wrote)
	mockCockpitDir := t.TempDir()

	err := pm.Deploy("mock", mockCockpitDir, tmpProjectDir)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	// Verify the file was written
	targetFile := filepath.Join(tmpProjectDir, "test_rules.md")
	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read deployed file: %v", err)
	}

	if string(content) != "Compiled Rule Content" {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestManagerDeploy_HomeWorkspace(t *testing.T) {
	tmpProjectDir := t.TempDir()

	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"mock": {
				Name:      "mock",
				Enabled:   true,
				Workspace: "~",
				Features: map[string]*FeatureConfig{
					"rules": {Enabled: true, Path: "test_rules.md"},
				},
			},
		},
	}

	pm := NewProviderManager(config)
	pm.Register(&mockCompiler{
		name: "mock",
		compileRules: func(rules []CanonicalRule, p *Provider) (map[string]string, error) {
			return map[string]string{
				"~/.mock/test_rules.md": "Home Content",
			}, nil
		},
	})

	mockCockpitDir := t.TempDir()
	err := pm.Deploy("mock", mockCockpitDir, tmpProjectDir)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	home, _ := os.UserHomeDir()
	targetFile := filepath.Join(home, ".mock", "test_rules.md")

	// We just ensure it doesn't fail. Clean up immediately to not leave trash in user home.
	defer os.RemoveAll(filepath.Join(home, ".mock"))

	content, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("Failed to read home deployed file: %v", err)
	}

	if string(content) != "Home Content" {
		t.Errorf("Unexpected content: %s", string(content))
	}
}

func TestManagerDeploy_UnconfiguredProvider(t *testing.T) {
	config := &ProvidersConfig{Providers: map[string]*Provider{}}
	pm := NewProviderManager(config)

	err := pm.Deploy("unknown", t.TempDir(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "provider not configured") {
		t.Errorf("expected error for unconfigured provider, got %v", err)
	}
}

func TestManagerDeploy_UnregisteredCompiler(t *testing.T) {
	config := &ProvidersConfig{
		Providers: map[string]*Provider{
			"unregistered": {Name: "unregistered", Enabled: true},
		},
	}
	pm := NewProviderManager(config)

	err := pm.Deploy("unregistered", t.TempDir(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "no compiler registered") {
		t.Errorf("expected error for unregistered compiler, got %v", err)
	}
}

func TestDeploy_CompileErrors(t *testing.T) {
	tmpDir := t.TempDir()

	// Create canonical directories so ParseCanonical doesn't fail parsing completely
	os.MkdirAll(filepath.Join(tmpDir, "skills"), 0755)

	readonlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readonlyDir, 0555)

	features := map[string]*FeatureConfig{
		"entrypoint":  {Path: "entry.md", Enabled: true},
		"skills":      {Path: "skills", Enabled: true},
		"rules":       {Path: "rules", Enabled: true},
		"workflows":   {Path: "flows", Enabled: true},
		"permissions": {Path: "perms.json", Enabled: true},
	}

	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"devin":       {Name: "devin", Enabled: true, Workspace: readonlyDir, Features: features},
			"antigravity": {Name: "antigravity", Enabled: true, Workspace: "/invalid/path", Features: features},
			"goose":       {Name: "goose", Enabled: true, Workspace: "/invalid/path", Features: features},
		},
	}

	m := NewProviderManager(cfg)

	m.Deploy("devin", tmpDir, "")
	m.Deploy("antigravity", tmpDir, "")
	m.Deploy("goose", tmpDir, "")
}

func TestDeploy_WithGlobalMarkers(t *testing.T) {
	tmpDir := t.TempDir()

	// Create canonical struct
	os.MkdirAll(filepath.Join(tmpDir, "skills"), 0755)

	readonlyDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(readonlyDir, 0755)

	agentsPath := filepath.Join(readonlyDir, "AGENTS.md")
	os.WriteFile(agentsPath, []byte("Existing Content\n"), 0644)

	features := map[string]*FeatureConfig{
		"entrypoint": {Path: "entry.md", Enabled: true},
	}

	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"mock": {Name: "mock", Enabled: true, Workspace: readonlyDir, Features: features},
		},
	}

	pm := &ProviderManager{
		config:    cfg,
		compilers: make(map[string]Compiler),
	}

	// mock compiler
	pm.compilers["mock"] = &mockCompiler{
		name: "mock",
		compileEntrypoint: func(ep *CanonicalEntrypoint, p *Provider) (map[string]string, error) {
			return map[string]string{
				"AGENTS.md": "New Global Content",
			}, nil
		},
		compileSkills: func(skills []CanonicalSkill, p *Provider) (map[string]string, error) {
			return nil, nil
		},
		compileRules: func(rules []CanonicalRule, p *Provider) (map[string]string, error) {
			return nil, nil
		},
		compileWorkflows: func(workflows []CanonicalWorkflow, p *Provider) (map[string]string, error) {
			return nil, nil
		},
		compilePermissions: func(perms *CanonicalPermissions, p *Provider) (map[string]string, error) {
			return nil, nil
		},
	}

	err := pm.Deploy("mock", tmpDir, readonlyDir)
	if err != nil {
		t.Fatalf("Deploy failed: %v", err)
	}

	content, err := os.ReadFile(agentsPath)
	if err != nil {
		t.Fatalf("failed to read agents: %v", err)
	}

	if !strings.Contains(string(content), "<!-- cockpit:global -->") {
		t.Errorf("missing start marker in: %s", string(content))
	}
	if !strings.Contains(string(content), "New Global Content") {
		t.Errorf("missing new content in: %s", string(content))
	}
	if !strings.Contains(string(content), "Existing Content") {
		t.Errorf("missing existing content in: %s", string(content))
	}
}
