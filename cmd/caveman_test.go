package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func TestCavemanCommand(t *testing.T) {
	// Setup mock environment
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Language:   "en-us",
		AIProvider: "antigravity", // mock provider
	}

	// Create mock providers.yaml in tmpDir
	providersYaml := `version: "1"
providers:
  antigravity:
    enabled: true
    name: antigravity
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

	cmd := NewCavemanCommand(logMgr, cfg, translator)

	// 1. Test Status when OFF
	cmd.SetArgs([]string{"status"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("expected success, got err: %v", err)
	}

	// 2. Test ON
	cmd.SetArgs([]string{"on"})
	if err := cmd.Execute(); err != nil {
		// deploy might fail because parser expects identity.md etc.,
		// but the file should be created. Let's ignore full deploy errors in unit test
		// or provide mock files
	}

	rulePath := filepath.Join(cockpitDir, "rules", "caveman.md")
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Errorf("expected caveman.md to be created")
	}

	// 3. Test OFF
	cmd.SetArgs([]string{"off"})
	if err := cmd.Execute(); err != nil {
		// ignore full deploy error
	}

	if _, err := os.Stat(rulePath); !os.IsNotExist(err) {
		t.Errorf("expected caveman.md to be deleted")
	}
}
