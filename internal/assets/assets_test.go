package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreAssets(t *testing.T) {
	tmpDir := t.TempDir()

	err := RestoreAssets(tmpDir)
	if err != nil {
		t.Fatalf("RestoreAssets failed: %v", err)
	}

	// Verify expected files were created
	expectedFiles := []string{
		"config.yaml",
		"providers.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", file)
		}
	}
}
