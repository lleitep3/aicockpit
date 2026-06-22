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
		Version:    "1.0.0",
		Language:   "en-us",
		AIProvider: "antigravity",
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
