package providers

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AdapterManifest defines the schema of provider-adapter.yaml files.
type AdapterManifest struct {
	// Name is the provider identifier (must match providers.yaml key)
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Version     string `yaml:"version"`
	// Artifacts maps canonical artifact type to its output rules.
	// Keys: "entrypoint", "skills", "rules", "workflows", "permissions", "agents"
	Artifacts map[string]ArtifactConfig `yaml:"artifacts"`
}

// ArtifactConfig describes how one artifact type is rendered.
type ArtifactConfig struct {
	Enabled bool `yaml:"enabled"`
	// Path is the destination file path. Supports: relative (to workspace),
	// absolute, or ~ prefix for home dir.
	Path string `yaml:"path"`
	// Template is a Go text/template string. Available template variables vary
	// by artifact type — see GenericYAMLCompiler for details.
	Template string `yaml:"template"`
}

// LoadAdapterManifest reads and parses a provider-adapter.yaml file.
func LoadAdapterManifest(path string) (*AdapterManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read adapter manifest: %w", err)
	}
	var m AdapterManifest
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse adapter manifest: %w", err)
	}
	if m.Name == "" {
		return nil, fmt.Errorf("adapter manifest missing required field: name")
	}
	return &m, nil
}
