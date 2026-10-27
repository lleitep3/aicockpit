package providers

import (
	"fmt"
	"os"
	"path/filepath"
)

const adapterManifestFilename = "provider-adapter.yaml"

// PluginLoader discovers provider adapter manifests from a plugins directory
// and registers GenericYAMLCompiler instances into a ProviderManager.
type PluginLoader struct {
	pluginsDir string
}

// NewPluginLoader creates a PluginLoader that scans pluginsDir for adapters.
// pluginsDir is typically ~/.cockpit/providers/.
func NewPluginLoader(pluginsDir string) *PluginLoader {
	return &PluginLoader{pluginsDir: pluginsDir}
}

// Discover scans pluginsDir for subdirectories containing provider-adapter.yaml
// files and returns loaded manifests. Subdirectories without the manifest file
// are silently skipped.
func (pl *PluginLoader) Discover() ([]*AdapterManifest, error) {
	entries, err := os.ReadDir(pl.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No plugins directory — not an error, just no plugins
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read plugins directory: %w", err)
	}

	var manifests []*AdapterManifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(pl.pluginsDir, entry.Name(), adapterManifestFilename)
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			continue
		}
		manifest, err := LoadAdapterManifest(manifestPath)
		if err != nil {
			// Log but don't abort — other plugins should still load
			// (caller can decide what to do with the error)
			continue
		}
		manifests = append(manifests, manifest)
	}
	return manifests, nil
}

// DiscoverAndRegister loads all adapter manifests from pluginsDir and
// registers a GenericYAMLCompiler for each into the given ProviderManager.
// Returns the names of successfully registered plugins.
func (pl *PluginLoader) DiscoverAndRegister(pm *ProviderManager) ([]string, error) {
	manifests, err := pl.Discover()
	if err != nil {
		return nil, err
	}
	var registered []string
	for _, m := range manifests {
		pm.Register(NewGenericYAMLCompiler(m))
		registered = append(registered, m.Name)
	}
	return registered, nil
}
