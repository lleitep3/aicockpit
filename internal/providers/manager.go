package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProviderManager orchestrates compilation and deployment of adapter files.
type ProviderManager struct {
	adapters map[string]Adapter
	config   *ProvidersConfig
}

// NewProviderManager creates a new ProviderManager and registers default adapters.
func NewProviderManager(config *ProvidersConfig) *ProviderManager {
	pm := &ProviderManager{
		adapters: make(map[string]Adapter),
		config:   config,
	}

	// Register default strategies
	pm.Register(NewAntigravityAdapter())
	pm.Register(NewDevinAdapter())
	pm.Register(NewGooseAdapter())

	return pm
}

// Register registers a new adapter strategy.
func (pm *ProviderManager) Register(adapter Adapter) {
	pm.adapters[adapter.Name()] = adapter
}

// Deploy compiles rules/skills for a provider and deploys them to target locations.
func (pm *ProviderManager) Deploy(providerName string, cockpitHomeDir string, projectDir string) error {
	provider := pm.config.GetProvider(providerName)
	if provider == nil {
		return fmt.Errorf("provider not configured: %s", providerName)
	}

	adapter, exists := pm.adapters[providerName]
	if !exists {
		return fmt.Errorf("no adapter registered for: %s", providerName)
	}

	// 1. Compile files using the adapter strategy
	files, err := adapter.Compile(cockpitHomeDir, provider)
	if err != nil {
		return fmt.Errorf("failed to compile rules for provider %s: %w", providerName, err)
	}

	// 2. Determine target base directory
	baseDir := projectDir
	if provider.Workspace == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = home
	}

	// 3. Write compiled files to their relative target paths
	for relPath, content := range files {
		// Clean the relative path to prevent directory traversal
		cleanRel := filepath.Clean(relPath)
		if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
			// Resolve relative to home if it tries to escape or looks like absolute
			// But usually config workspace paths are relative to baseDir
			// Let's handle absolute paths if the provider path starts with ~ or /
			if strings.HasPrefix(relPath, "~") {
				home, err := os.UserHomeDir()
				if err != nil {
					return fmt.Errorf("failed to get user home directory: %w", err)
				}
				cleanRel = filepath.Join(home, relPath[1:])
			}
		}

		// Check if the path starts with ~ or home
		var destPath string
		if strings.HasPrefix(relPath, "~") {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get user home directory: %w", err)
			}
			destPath = filepath.Join(home, relPath[1:])
		} else if filepath.IsAbs(relPath) {
			destPath = relPath
		} else {
			destPath = filepath.Join(baseDir, relPath)
		}

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		// Write file
		if err := os.WriteFile(destPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write compiled file %s: %w", destPath, err)
		}
	}

	return nil
}
