package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// gooseRequiredExtensions lists extensions that must be enabled for cockpit to
// work correctly inside Goose. Each key maps to an extension config block that
// will be merged into ~/.config/goose/config.yaml if not already present.
var gooseRequiredExtensions = []string{
	"developer", // shell + filesystem access
	"skills",    // skill discovery from filesystem
	"summon",    // subagent delegation
}

// GooseAdapter compiles assets for the Goose provider.
type GooseAdapter struct{}

// NewGooseAdapter creates a new GooseAdapter.
func NewGooseAdapter() *GooseAdapter {
	return &GooseAdapter{}
}

// Name returns the provider name.
func (g *GooseAdapter) Name() string {
	return "goose"
}

// Compile compiles .goosehints and skills, and optionally updates the Goose
// global config to ensure required extensions are enabled.
func (g *GooseAdapter) Compile(cockpitHomeDir string, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)

	// 1. Build .goosehints (rules feature)
	rulesConfig, hasRules := provider.Features["rules"]
	if hasRules && rulesConfig.Enabled {
		goosehints, err := buildGoosehints(cockpitHomeDir)
		if err != nil {
			return nil, fmt.Errorf("failed to build .goosehints: %w", err)
		}
		files[rulesConfig.Path] = goosehints
	}

	// 2. Build `.agents/skills/` files (skills feature)
	skillsConfig, hasSkills := provider.Features["skills"]
	if hasSkills && skillsConfig.Enabled {
		skillsSrcDir := filepath.Join(cockpitHomeDir, "skills")
		if _, err := os.Stat(skillsSrcDir); err == nil {
			err = filepath.Walk(skillsSrcDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(skillsSrcDir, path)
				if err != nil {
					return err
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				targetPath := filepath.Join(skillsConfig.Path, relPath)
				files[targetPath] = string(content)
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk skills: %w", err)
			}
		}
	}

	// 3. Apply permissions: ensure required extensions are enabled in the Goose
	//    global config (~/.config/goose/config.yaml).
	permConfig, hasPerms := provider.Features["permissions"]
	if hasPerms && permConfig.Enabled {
		if err := g.applyPermissions(permConfig.Path); err != nil {
			return nil, fmt.Errorf("failed to apply goose permissions: %w", err)
		}
	}

	return files, nil
}

// applyPermissions reads ~/.config/goose/config.yaml, ensures required
// extensions are set to enabled: true (without disabling others), and writes
// the file back.
//
// Goose config structure (YAML):
//
//	extensions:
//	  developer:
//	    enabled: true
//	    type: platform
//	    name: developer
//	    ...
//	  skills:
//	    enabled: true
//	    ...
func (g *GooseAdapter) applyPermissions(configPath string) error {
	expanded, err := expandHome(configPath)
	if err != nil {
		return err
	}

	// Read existing YAML as raw text to preserve comments and ordering.
	// We use a line-based approach to avoid mangling the existing YAML structure.
	data, err := os.ReadFile(expanded)
	if os.IsNotExist(err) {
		// File doesn't exist yet; nothing to do — user hasn't installed Goose globally.
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", expanded, err)
	}

	updated := ensureGooseExtensionsEnabled(string(data), gooseRequiredExtensions)

	if err := os.MkdirAll(filepath.Dir(expanded), 0o755); err != nil {
		return err
	}
	return os.WriteFile(expanded, []byte(updated), 0o644)
}

// ensureGooseExtensionsEnabled parses the Goose config YAML text and makes sure
// each extension in required has "enabled: true". It works line-by-line to
// preserve the rest of the config (comments, ordering, model settings etc.).
//
// For each required extension:
//   - If the extension block already exists with "enabled: false", flip it to true.
//   - If the extension block doesn't exist at all, append a minimal enabled block.
func ensureGooseExtensionsEnabled(content string, required []string) string {
	lines := strings.Split(content, "\n")
	found := make(map[string]bool)

	// Pass 1: flip existing "enabled: false" under each required extension.
	inExtensions := false
	currentExt := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "extensions:" {
			inExtensions = true
			continue
		}
		if inExtensions {
			// Detect a top-level extension key (2-space indent, ends with colon)
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && strings.HasSuffix(trimmed, ":") {
				currentExt = strings.TrimSuffix(trimmed, ":")
			}
			// Detect a non-indented line — we've left the extensions block
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "#") {
				inExtensions = false
				currentExt = ""
				continue
			}
			// If we're inside a required extension and see "enabled: false", flip it
			if currentExt != "" && isRequired(currentExt, required) {
				if strings.TrimSpace(line) == "enabled: false" {
					lines[i] = strings.Replace(line, "enabled: false", "enabled: true", 1)
					found[currentExt] = true
				}
				if strings.TrimSpace(line) == "enabled: true" {
					found[currentExt] = true
				}
			}
		}
	}

	// Pass 2: append blocks for required extensions that were not found at all.
	sort.Strings(required)
	for _, ext := range required {
		if !found[ext] {
			block := fmt.Sprintf("\n  %s:\n    enabled: true\n    type: platform\n    name: %s\n    description: 'Required by AICockpit'\n    bundled: true\n    available_tools: []", ext, ext)
			// Find the extensions: line and insert after it, or append at end.
			extIdx := -1
			for i, l := range lines {
				if strings.TrimSpace(l) == "extensions:" {
					extIdx = i
					break
				}
			}
			if extIdx >= 0 {
				lines[extIdx] = lines[extIdx] + block
			} else {
				lines = append(lines, "extensions:"+block)
			}
		}
	}

	return strings.Join(lines, "\n")
}

// isRequired returns true if ext is in the required slice.
func isRequired(ext string, required []string) bool {
	for _, r := range required {
		if r == ext {
			return true
		}
	}
	return false
}

func buildGoosehints(cockpitHomeDir string) (string, error) {
	identityPath := filepath.Join(cockpitHomeDir, "identity.md")
	identityContent := ""
	if _, err := os.Stat(identityPath); err == nil {
		data, err := os.ReadFile(identityPath)
		if err != nil {
			return "", err
		}
		identityContent = string(data)
	}

	rulesDir := filepath.Join(cockpitHomeDir, "rules")
	var rules []string
	if _, err := os.Stat(rulesDir); err == nil {
		entries, err := os.ReadDir(rulesDir)
		if err != nil {
			return "", err
		}

		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})

		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
				data, err := os.ReadFile(filepath.Join(rulesDir, entry.Name()))
				if err != nil {
					return "", err
				}
				rules = append(rules, string(data))
			}
		}
	}

	var builder strings.Builder
	builder.WriteString(AddGeneratedHeader("", ".goosehints"))

	if identityContent != "" {
		builder.WriteString(identityContent)
		builder.WriteString("\n\n---\n\n")
	}

	builder.WriteString("## 🏅 Gold Rules & Project Guidelines\n\n")

	builder.WriteString(strings.Join(rules, "\n\n---\n\n"))
	return builder.String(), nil
}
