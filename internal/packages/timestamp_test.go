package packages

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestSavePackageAddsTimestamps(t *testing.T) {
	dir := t.TempDir()

	pkg := &Package{
		Name:         "testpkg",
		Version:      "0.1.0",
		Description:  "Test package",
		Author:       "tester",
		License:      "MIT",
		Requirements: Requirements{Cockpit: ">=0.1.0"},
		Installation: Installation{SupportedProviders: []string{"test"}, ProviderFeatures: map[string][]string{"test": {}}, Method: "copy"},
		Features:     Features{},
		Metadata:     Metadata{},
	}

	if err := SavePackage(dir, pkg); err != nil {
		t.Fatalf("SavePackage failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "cockpit-package.yml"))
	if err != nil {
		t.Fatalf("failed to read manifest: %v", err)
	}
	var loaded Package
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal manifest: %v", err)
	}
	if loaded.Metadata.CreationDate == "" || loaded.Metadata.LastModified == "" {
		t.Fatalf("timestamps not set in metadata")
	}
	if _, err := time.Parse(time.RFC3339, loaded.Metadata.CreationDate); err != nil {
		t.Fatalf("CreationDate not RFC3339: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, loaded.Metadata.LastModified); err != nil {
		t.Fatalf("LastModified not RFC3339: %v", err)
	}
}
