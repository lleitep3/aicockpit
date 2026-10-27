package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProviderManager orchestrates compilation and deployment of adapter files.
type ProviderManager struct {
	compilers    map[string]Compiler
	config       *ProvidersConfig
	hindsightMCP []MCPServer
}

// NewProviderManager creates a new ProviderManager and registers default compilers.
func NewProviderManager(config *ProvidersConfig) *ProviderManager {
	pm := &ProviderManager{
		compilers: make(map[string]Compiler),
		config:    config,
	}

	pm.Register(NewAntigravityCompiler())
	pm.Register(NewCodexCompiler())
	pm.Register(NewClaudeCompiler())
	pm.Register(NewGeminiCompiler())
	pm.Register(NewGitHubCopilotCompiler())
	pm.Register(NewDevinCompiler())
	pm.Register(NewGooseCompiler())

	return pm
}

// NewProviderManagerWithPlugins creates a ProviderManager, registers default
// compilers, and additionally discovers and registers any YAML-based provider
// adapters from <cockpitDir>/providers/.
func NewProviderManagerWithPlugins(config *ProvidersConfig, cockpitDir string) *ProviderManager {
	pm := NewProviderManager(config)
	pluginsDir := filepath.Join(cockpitDir, "providers")
	loader := NewPluginLoader(pluginsDir)
	loader.DiscoverAndRegister(pm) //nolint:errcheck // plugin load errors are non-fatal
	return pm
}

// Register registers a new compiler strategy.
func (pm *ProviderManager) Register(compiler Compiler) {
	pm.compilers[compiler.Name()] = compiler
}

// SetHindsightMCP stores the per-bank Hindsight MCP servers to inject.
// A nil or empty list leaves every compiled file byte-identical to a deploy
// that does not know about Hindsight. The manager does not delete servers
// written by an earlier deploy.
func (pm *ProviderManager) SetHindsightMCP(servers []MCPServer) {
	if len(servers) == 0 {
		pm.hindsightMCP = nil
		return
	}
	pm.hindsightMCP = append([]MCPServer(nil), servers...)
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
	mcpFiles, err := pm.compileHindsightMCP(compiler, provider)
	if err != nil {
		return err
	}
	resourceModes := make(map[string]os.FileMode)
	if skillsFeature, ok := provider.Features["skills"]; ok && skillsFeature.Enabled {
		for _, skill := range skills {
			for relPath, mode := range skill.ResourceModes {
				compiledPath := filepath.Clean(filepath.Join(skillsFeature.Path, canonicalSkillDirectory(skill), filepath.FromSlash(relPath)))
				resourceModes[compiledPath] = os.FileMode(mode).Perm()
			}
		}
	}
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
		destPath, err := resolveCompiledPath(relPath, baseDir)
		if err != nil {
			return err
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
		if baseName == "AGENTS.md" || baseName == ".goosehints" || baseName == "copilot-instructions.md" {
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

		mode := os.FileMode(0o644)
		if resourceMode, ok := resourceModes[relPath]; ok {
			mode = resourceMode
		}
		if err := atomicWriteFile(destPath, []byte(finalContent), mode); err != nil {
			return fmt.Errorf("failed to write compiled file %s: %w", destPath, err)
		}
	}

	for relPath, content := range mcpFiles {
		if err := writeMergedMCP(relPath, content, baseDir); err != nil {
			return err
		}
	}

	return nil
}

func (pm *ProviderManager) compileHindsightMCP(compiler Compiler, provider *Provider) (map[string]string, error) {
	if len(pm.hindsightMCP) == 0 {
		return nil, nil
	}
	mcpCompiler, ok := compiler.(MCPCompiler)
	if !ok {
		return nil, nil
	}
	files, err := mcpCompiler.CompileMCP(pm.hindsightMCP, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to compile hindsight mcp: %w", err)
	}
	return files, nil
}

func resolveCompiledPath(relPath, baseDir string) (string, error) {
	relPath = filepath.Clean(relPath)
	if strings.HasPrefix(relPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home directory: %w", err)
		}
		return filepath.Join(home, relPath[1:]), nil
	}
	if filepath.IsAbs(relPath) {
		return relPath, nil
	}
	return filepath.Join(baseDir, relPath), nil
}

func writeMergedMCP(relPath, content, baseDir string) error {
	if strings.TrimSpace(content) == "" {
		return nil
	}
	destPath, err := resolveCompiledPath(relPath, baseDir)
	if err != nil {
		return err
	}
	existing := ""
	data, readErr := os.ReadFile(destPath)
	switch {
	case readErr == nil:
		existing = string(data)
	case os.IsNotExist(readErr):
	default:
		return fmt.Errorf("failed to read mcp config %s: %w", destPath, readErr)
	}
	merged, err := mergeMCPDocument(relPath, existing, content)
	if err != nil {
		return err
	}
	if existing == merged {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", destPath, err)
	}
	if err := atomicWriteFile(destPath, []byte(merged), 0o644); err != nil {
		return fmt.Errorf("failed to write mcp config %s: %w", destPath, err)
	}
	return nil
}

// atomicWriteFile writes data to path atomically: it first writes to a
// temporary file in the same directory, then renames it to path.
// This prevents readers from ever observing a partially-written file and
// preserves the original when a write fails mid-way.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := os.Chmod(tmpPath, perm); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath) //nolint:errcheck // best-effort cleanup
		return fmt.Errorf("failed to rename temp file to destination: %w", err)
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
