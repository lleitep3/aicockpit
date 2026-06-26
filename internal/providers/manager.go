package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProviderManager orchestrates compilation and deployment of adapter files.
type ProviderManager struct {
	compilers map[string]Compiler
	config    *ProvidersConfig
}

// NewProviderManager creates a new ProviderManager and registers default compilers.
func NewProviderManager(config *ProvidersConfig) *ProviderManager {
	pm := &ProviderManager{
		compilers: make(map[string]Compiler),
		config:    config,
	}

	pm.Register(NewAntigravityCompiler())
	pm.Register(NewDevinCompiler())
	pm.Register(NewGooseCompiler())

	return pm
}

// Register registers a new compiler strategy.
func (pm *ProviderManager) Register(compiler Compiler) {
	pm.compilers[compiler.Name()] = compiler
}

// Deploy compiles rules/skills for a provider and deploys them to target locations.
func (pm *ProviderManager) Deploy(providerName string, cockpitHomeDir string, projectDir string) error {
	provider := pm.config.GetProvider(providerName)
	if provider == nil {
		return fmt.Errorf("provider not configured: %s", providerName)
	}

	compiler, exists := pm.compilers[providerName]
	if !exists {
		return fmt.Errorf("no compiler registered for: %s", providerName)
	}

	// 1. Parse canonical structure from ~/.cockpit
	entrypoint, skills, rules, workflows, perms, agents, err := ParseCanonical(cockpitHomeDir)
	if err != nil {
		return fmt.Errorf("failed to parse canonical structures: %w", err)
	}

	// 2. Compile files using the adapter strategy
	allFiles := make(map[string]string)

	if files, err := compiler.CompileEntrypoint(entrypoint, provider); err == nil {
		mergeMap(allFiles, files)
	} else {
		return fmt.Errorf("failed to compile entrypoint: %w", err)
	}

	if files, err := compiler.CompileSkills(skills, provider); err == nil {
		mergeMap(allFiles, files)
	} else {
		return fmt.Errorf("failed to compile skills: %w", err)
	}

	if files, err := compiler.CompileRules(rules, provider); err == nil {
		mergeMap(allFiles, files)
	} else {
		return fmt.Errorf("failed to compile rules: %w", err)
	}

	if files, err := compiler.CompileWorkflows(workflows, provider); err == nil {
		mergeMap(allFiles, files)
	} else {
		return fmt.Errorf("failed to compile workflows: %w", err)
	}

	if files, err := compiler.CompilePermissions(perms, provider); err == nil {
		mergeMap(allFiles, files)
	} else {
		return fmt.Errorf("failed to compile permissions: %w", err)
	}

	if files, err := compiler.CompileAgents(agents, provider); err == nil {
		mergeMap(allFiles, files)
	} else {
		return fmt.Errorf("failed to compile agents: %w", err)
	}

	// 3. Determine target base directory
	baseDir := projectDir
	if provider.Workspace == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		baseDir = home
	}

	// 4. Write compiled files to their relative target paths
	for relPath, content := range allFiles {
		relPath = filepath.Clean(relPath)

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

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
		}

		// Read existing file if it exists
		existingContent := ""
		if data, err := os.ReadFile(destPath); err == nil {
			existingContent = string(data)
		}

		finalContent := content
		baseName := filepath.Base(destPath)

		// Only use markers for global entrypoint files to preserve user edits
		if baseName == "AGENTS.md" || baseName == ".goosehints" {
			startMarker := "<!-- cockpit:global -->"
			endMarker := "<!-- /cockpit:global -->"

			if existingContent != "" {
				startIdx := strings.Index(existingContent, startMarker)
				endIdx := strings.Index(existingContent, endMarker)

				if startIdx != -1 && endIdx != -1 {
					// Replace the block
					finalContent = existingContent[:startIdx] + startMarker + "\n" + content + "\n" + endMarker + existingContent[endIdx+len(endMarker):]
				} else {
					// Append with markers
					finalContent = existingContent + "\n\n" + startMarker + "\n" + content + "\n" + endMarker + "\n"
				}
			} else {
				// Write new file with markers
				finalContent = startMarker + "\n" + content + "\n" + endMarker + "\n"
			}
		}

		if err := os.WriteFile(destPath, []byte(finalContent), 0o644); err != nil {
			return fmt.Errorf("failed to write compiled file %s: %w", destPath, err)
		}
	}

	return nil
}

func mergeMap(dest map[string]string, src map[string]string) {
	for k, v := range src {
		if existing, ok := dest[k]; ok {
			// If both write to the same file (e.g. AGENTS.md), append them
			dest[k] = existing + "\n\n" + v
		} else {
			dest[k] = v
		}
	}
}
