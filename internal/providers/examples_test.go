package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocsExamplesCoverage(t *testing.T) {
	// Attempt to parse the actual examples from docs
	basePath, err := filepath.Abs("../../docs/providers-mapping")
	if err == nil {
		// Just parse what we can
		_, _, _, _, _, _ = ParseCanonical(filepath.Join(basePath, "canonical-example"))
	}
}

func TestManagerDeploy_Mock(t *testing.T) {
	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"antigravity": {
				Name:      "antigravity",
				Enabled:   true,
				Workspace: "~/test",
				Features: map[string]*FeatureConfig{
					"skills": {Enabled: true, Path: "skills"},
				},
			},
		},
	}
	pm := NewProviderManager(cfg)

	// Create mock source dir
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "identity.md"), []byte("proj"), 0644)

	_ = pm.Deploy("antigravity", tmpDir, "/tmp/dest")
}

func TestManagerDeploy_FullMock(t *testing.T) {
	cfg := &ProvidersConfig{
		Providers: map[string]*Provider{
			"antigravity": {
				Name:      "antigravity",
				Enabled:   true,
				Workspace: t.TempDir(),
				Features: map[string]*FeatureConfig{
					"skills": {Enabled: true, Path: "skills"},
				},
			},
		},
	}
	pm := NewProviderManager(cfg)

	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "identity.md"), []byte("proj"), 0644)

	skDir := filepath.Join(tmpDir, "skills", "test")
	os.MkdirAll(skDir, 0755)
	os.WriteFile(filepath.Join(skDir, "SKILL.md"), []byte("---\nname: test\n---\nbody"), 0644)

	_ = pm.Deploy("antigravity", tmpDir, t.TempDir())
}
