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

// GooseCompiler compiles assets for the Goose provider.
type GooseCompiler struct{}

// NewGooseCompiler creates a new GooseCompiler.
func NewGooseCompiler() *GooseCompiler {
	return &GooseCompiler{}
}

// Name returns the provider name.
func (g *GooseCompiler) Name() string {
	return "goose"
}

// CompileEntrypoint writes .goosehints and project context.
func (g *GooseCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	rulesConfig, hasRules := provider.Features["rules"]

	if hasRules && rulesConfig.Enabled {
		var builder strings.Builder
		builder.WriteString(AddGeneratedHeader("", ".goosehints"))

		if entrypoint.ProjectContext != "" {
			builder.WriteString(entrypoint.ProjectContext)
			builder.WriteString("\n\n---\n\n")
		}

		builder.WriteString("## 🏅 Gold Rules & Project Guidelines\n\n")
		for _, gr := range entrypoint.GoldenRules {
			builder.WriteString(gr)
			builder.WriteString("\n\n")
		}

		files[rulesConfig.Path] = builder.String()
	}
	return files, nil
}

// CompileRules appends standard rules to .goosehints
func (g *GooseCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	rulesConfig, hasRules := provider.Features["rules"]

	if hasRules && rulesConfig.Enabled && len(rules) > 0 {
		var builder strings.Builder
		for i, r := range rules {
			builder.WriteString(r.Content)
			if i < len(rules)-1 {
				builder.WriteString("\n\n---\n\n")
			}
		}
		files[rulesConfig.Path] = builder.String()
	}
	return files, nil
}

// CompileSkills writes skills to the `.agents/skills` folder.
func (g *GooseCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	skillsConfig, hasSkills := provider.Features["skills"]

	if hasSkills && skillsConfig.Enabled {
		for _, skill := range skills {
			files[filepath.Join(skillsConfig.Path, skill.Name, "SKILL.md")] = fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n%s", skill.Name, skill.Description, skill.Content)

			for relPath, content := range skill.ScriptFiles {
				files[filepath.Join(skillsConfig.Path, skill.Name, relPath)] = content
			}
		}
	}
	return files, nil
}

// CompileWorkflows translates CanonicalWorkflows to Goose Recipes (YAML).
func (g *GooseCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	wfConfig, hasWorkflows := provider.Features["workflows"]

	if hasWorkflows && wfConfig.Enabled {
		for _, wf := range workflows {
			safeName := strings.ReplaceAll(wf.Name, "-", "_")
			var content strings.Builder

			content.WriteString(fmt.Sprintf("name: %s\ndescription: %s\nsteps:\n", safeName, wf.Description))
			for _, step := range wf.Steps {
				// simple yaml encoding hack
				safeStep := strings.ReplaceAll(step, `"`, `\"`)
				content.WriteString(fmt.Sprintf("  - \"%s\"\n", safeStep))
			}

			// Goose Recipes are just YAML files in .goose/recipes
			files[filepath.Join(wfConfig.Path, safeName+".yaml")] = content.String()
		}
	}
	return files, nil
}

// CompilePermissions reads ~/.config/goose/config.yaml and merges required extensions
func (g *GooseCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	permConfig, hasPerms := provider.Features["permissions"]

	if hasPerms && permConfig.Enabled {
		expanded, err := expandHome(permConfig.Path)
		if err != nil {
			return nil, err
		}

		data, err := os.ReadFile(expanded)
		if os.IsNotExist(err) {
			return nil, nil // Not installed globally
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", expanded, err)
		}

		updated := ensureGooseExtensionsEnabled(string(data), gooseRequiredExtensions)

		err = os.WriteFile(expanded, []byte(updated), 0o644)
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}

func ensureGooseExtensionsEnabled(content string, required []string) string {
	lines := strings.Split(content, "\n")
	found := make(map[string]bool)

	inExtensions := false
	currentExt := ""
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "extensions:" {
			inExtensions = true
			continue
		}
		if inExtensions {
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "   ") && strings.HasSuffix(trimmed, ":") {
				currentExt = strings.TrimSuffix(trimmed, ":")
			}
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && !strings.HasPrefix(line, "#") {
				inExtensions = false
				currentExt = ""
				continue
			}
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

	sort.Strings(required)
	for _, ext := range required {
		if !found[ext] {
			block := fmt.Sprintf("\n  %s:\n    enabled: true\n    type: platform\n    name: %s\n    description: 'Required by AICockpit'\n    bundled: true\n    available_tools: []", ext, ext)
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

func isRequired(ext string, required []string) bool {
	for _, r := range required {
		if r == ext {
			return true
		}
	}
	return false
}

// CompileAgents is a no-op for Goose (does not support custom subagents).
func (g *GooseCompiler) CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error) {
	return make(map[string]string), nil
}
