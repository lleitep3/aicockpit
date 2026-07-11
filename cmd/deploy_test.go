package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func TestNewDeployCommand(t *testing.T) {
	// Create mock cockpit home dir
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Save and restore working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(cwd)

	// Change working directory to tmpDir so command deploys there
	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	// Write mock config.yaml and providers.yaml
	cockpitHome := filepath.Join(tmpDir, ".cockpit")
	err = os.MkdirAll(filepath.Join(cockpitHome, "rules"), 0755)
	if err != nil {
		t.Fatal(err)
	}

	// Write mock providers.yaml
	providersYaml := `
version: "1.0"
providers:
  antigravity:
    enabled: true
    name: "Antigravity"
    workspace: "` + tmpDir + `"
    features:
      rules:
        enabled: true
        path: ".gemini/rules/rule.md"
`
	err = os.WriteFile(filepath.Join(cockpitHome, "providers.yaml"), []byte(providersYaml), 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = os.WriteFile(filepath.Join(cockpitHome, "rules", "rule.md"), []byte("rule"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	logMgr, err := logging.NewManager(filepath.Join(tmpDir, "logs"))
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Version:          "1.0.0",
		Language:         "en-us",
		EnabledProviders: []string{"antigravity"},
	}
	translator := i18n.New("en-us")

	deployCmd := NewDeployCommand(logMgr, cfg, translator)
	if deployCmd == nil {
		t.Fatal("expected deploy command to be non-nil")
	}

	// Run deploy command
	var buf bytes.Buffer
	deployCmd.SetOut(&buf)
	deployCmd.SetErr(&buf)

	err = deployCmd.Execute()
	if err != nil {
		t.Fatalf("deployCmd.Execute() failed: %v", err)
	}

	// Verify rule was written
	rulePath := filepath.Join(tmpDir, ".gemini/rules/rule.md")
	if _, err := os.Stat(rulePath); err != nil {
		t.Errorf("expected rules file to be created at %s: %v", rulePath, err)
	}
}

func TestNewDeployCommand_NoProviders(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	logMgr, _ := logging.NewManager(filepath.Join(tmpDir, "logs"))
	cfg := &config.Config{
		Version:          "1.0.0",
		Language:         "en-us",
		EnabledProviders: []string{}, // no providers
	}
	tr := i18n.New("en-us")

	cmd := NewDeployCommand(logMgr, cfg, tr)
	if err := cmd.Execute(); err == nil {
		t.Error("deploy with no providers should return error")
	}
}

func TestNewDeployCommand_NoProvidersYaml(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create cockpit dir but no providers.yaml
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	logMgr, _ := logging.NewManager(filepath.Join(tmpDir, "logs"))
	cfg := &config.Config{
		Version:          "1.0.0",
		Language:         "en-us",
		EnabledProviders: []string{"antigravity"},
	}
	tr := i18n.New("en-us")

	cmd := NewDeployCommand(logMgr, cfg, tr)
	if err := cmd.Execute(); err == nil {
		t.Error("deploy with missing providers.yaml should return error")
	}
}

func TestNewDeployCommand_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(tmpDir)

	cockpitHome := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(filepath.Join(cockpitHome, "rules"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write providers.yaml with a provider that has bad workspace
	providersYaml := `
version: "1.0"
providers:
  bad-provider:
    enabled: true
    name: "Bad Provider"
    workspace: "/nonexistent/path/xyz"
    features:
      rules:
        enabled: true
        path: ".bad-provider/rules.md"
`
	if err := os.WriteFile(filepath.Join(cockpitHome, "providers.yaml"), []byte(providersYaml), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "rules", "rule.md"), []byte("rule"), 0644); err != nil {
		t.Fatal(err)
	}

	logMgr, _ := logging.NewManager(filepath.Join(tmpDir, "logs"))
	cfg := &config.Config{
		Version:          "1.0.0",
		Language:         "en-us",
		EnabledProviders: []string{"bad-provider"},
	}
	tr := i18n.New("en-us")

	cmd := NewDeployCommand(logMgr, cfg, tr)
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Should fail because bad workspace path
	err := cmd.Execute()
	if err == nil {
		t.Error("deploy with bad workspace should return error")
	}
}
