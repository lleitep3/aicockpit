package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDevinAdapter_WriteGoldRules(t *testing.T) {
	tmpDir := t.TempDir()
	cockpitHome := filepath.Join(tmpDir, ".cockpit")
	rulesDir := filepath.Join(cockpitHome, "rules")
	
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Create some gold rules
	if err := os.WriteFile(filepath.Join(rulesDir, "rtk-gold-rules.md"), []byte("# Rule 1\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "caveman-gold-rules.md"), []byte("# Rule 2\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	// Create a non-gold rule file to ensure it gets skipped
	if err := os.WriteFile(filepath.Join(rulesDir, "other-rules.md"), []byte("# Rule 3\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	targetPath := filepath.Join(tmpDir, "global_rules.md")
	
	adapter := NewDevinAdapter()
	if err := adapter.writeGoldRules(cockpitHome, targetPath); err != nil {
		t.Fatalf("writeGoldRules failed: %v", err)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read target file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "# 🏅 AICockpit Gold Rules") {
		t.Errorf("expected header, got:\n%s", content)
	}
	if !strings.Contains(content, "# Rule 1") {
		t.Errorf("expected Rule 1, got:\n%s", content)
	}
	if !strings.Contains(content, "# Rule 2") {
		t.Errorf("expected Rule 2, got:\n%s", content)
	}
	if strings.Contains(content, "# Rule 3") {
		t.Errorf("did not expect Rule 3 (not a gold rule file), got:\n%s", content)
	}
}

func TestDevinAdapter_WriteGoldRules_NoRules(t *testing.T) {
	tmpDir := t.TempDir()
	cockpitHome := filepath.Join(tmpDir, ".cockpit")
	targetPath := filepath.Join(tmpDir, "global_rules.md")
	
	adapter := NewDevinAdapter()
	if err := adapter.writeGoldRules(cockpitHome, targetPath); err != nil {
		t.Fatalf("writeGoldRules failed for empty dir: %v", err)
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("expected no file to be created when no rules exist")
	}
}
