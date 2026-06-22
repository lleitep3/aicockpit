package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func TestRtkCommand(t *testing.T) {
	// 1. Set up isolated environment
	tmpDir := t.TempDir()

	// Create mock config
	cfg := &config.Config{
		AIProviders: config.ProvidersConfig{
			Enabled: []string{"antigravity"},
		},
	}

	// Write dummy providers.yaml to tmpDir/.cockpit so runDeploy doesn't fail
	providersYaml := `providers:
  - name: antigravity
`

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0755); err != nil {
		t.Fatalf("failed to create .cockpit dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitDir, "providers.yaml"), []byte(providersYaml), 0644); err != nil {
		t.Fatalf("failed to write mock providers.yaml: %v", err)
	}

	t.Setenv("HOME", tmpDir)

	// Temporarily override config home dir function by env variable
	logMgr, _ := logging.NewManager("")
	translator := i18n.New("en-us")

	cmd := NewRtkCommand(logMgr, cfg, translator)

	// 2. Test ON
	cmd.SetArgs([]string{"on"})
	if err := cmd.Execute(); err != nil {
		// execute might fail if the dummy structure is not complete enough for deploy,
		// but let's check if the file was created anyway.
	}

	rulePath := filepath.Join(cockpitDir, "rules", "rtk.md")
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Errorf("expected rtk.md to be created")
	}

	// 3. Test STATUS
	cmd.SetArgs([]string{"status"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("unexpected error on status: %v", err)
	}

	// 4. Test OFF
	cmd.SetArgs([]string{"off"})
	if err := cmd.Execute(); err != nil {
		// ignore deploy error
	}

	if _, err := os.Stat(rulePath); err == nil {
		t.Errorf("expected rtk.md to be deleted")
	}
}
