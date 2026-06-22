package providers

import (
	"fmt"
	"path/filepath"
	"strings"
)

// antigravityCockpitPermissions lists the minimum permission grants that cockpit
// needs to operate inside Antigravity sessions.
var antigravityCockpitPermissions = []string{
	"command(cockpit)",
	"command(make)",
	"command(git)",
	"command(go)",
	"command(gh)",
	"read_file(~/.cockpit)",
}

// AntigravityCompiler compiles assets for the Antigravity provider.
type AntigravityCompiler struct{}

// NewAntigravityCompiler creates a new AntigravityCompiler.
func NewAntigravityCompiler() *AntigravityCompiler {
	return &AntigravityCompiler{}
}

// Name returns the provider name.
func (a *AntigravityCompiler) Name() string {
	return "antigravity"
}

// CompileEntrypoint writes the AGENTS.md file
func (a *AntigravityCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	rulesConfig, hasRules := provider.Features["rules"]

	if hasRules && rulesConfig.Enabled {
		var builder strings.Builder
		builder.WriteString(AddGeneratedHeader("", "antigravity"))

		if entrypoint.ProjectContext != "" {
			builder.WriteString(entrypoint.ProjectContext)
			builder.WriteString("\n\n---\n\n")
		}

		if len(entrypoint.GoldenRules) > 0 {
			builder.WriteString("## 🏅 Gold Rules & Project Guidelines\n\n")
			for _, gr := range entrypoint.GoldenRules {
				builder.WriteString(gr)
				builder.WriteString("\n\n")
			}
		}

		files[rulesConfig.Path] = builder.String()
	}
	return files, nil
}

// CompileRules appends rules to the AGENTS.md file
func (a *AntigravityCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
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

// CompileSkills writes skills using YAML frontmatter in SKILL.md
func (a *AntigravityCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	skillsConfig, hasSkills := provider.Features["skills"]

	if hasSkills && skillsConfig.Enabled {
		for _, skill := range skills {
			content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n%s", skill.Name, skill.Description, skill.Content)
			files[filepath.Join(skillsConfig.Path, skill.Name, "SKILL.md")] = AddGeneratedHeader(content, "antigravity")

			for relPath, scriptContent := range skill.ScriptFiles {
				files[filepath.Join(skillsConfig.Path, skill.Name, relPath)] = scriptContent
			}
		}
	}
	return files, nil
}

// CompileWorkflows translates workflows into Antigravity skills.
func (a *AntigravityCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	wfConfig, hasWorkflows := provider.Features["workflows"] // Actually, for Antigravity, workflows just become skills or subagents

	if hasWorkflows && wfConfig.Enabled {
		for _, wf := range workflows {
			var content strings.Builder
			content.WriteString(fmt.Sprintf("# Workflow: %s\n\n%s\n\n## Steps\n", wf.Name, wf.Description))
			for i, step := range wf.Steps {
				content.WriteString(fmt.Sprintf("%d. %s\n", i+1, step))
			}

			fileContent := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n%s", wf.Name, wf.Description, content.String())
			files[filepath.Join(wfConfig.Path, wf.Name, "SKILL.md")] = AddGeneratedHeader(fileContent, "antigravity")
		}
	}
	return files, nil
}

// CompilePermissions reads the config.json and merges it
func (a *AntigravityCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	files := make(map[string]string)
	permConfig, hasPerms := provider.Features["permissions"]

	if hasPerms && permConfig.Enabled {
		expanded, err := expandHome(permConfig.Path)
		if err != nil {
			return nil, err
		}

		m, err := readJSONFile(expanded)
		if err != nil {
			m = make(map[string]interface{})
		}

		userSettings := getNestedMap(m, "userSettings")
		grants := getNestedMap(userSettings, "globalPermissionGrants")

		existing := getStringSliceFromMap(grants, "allow")
		merged := mergeStringSlice(existing, antigravityCockpitPermissions)
		if perms != nil {
			// Convert CanonicalPermissions to Antigravity format: "command(x)", "read_file(x)", etc
			for _, cmd := range perms.AllowedCommands {
				merged = mergeStringSlice(merged, []string{fmt.Sprintf("command(%s)", cmd)})
			}
			for _, dir := range perms.AllowedDirs {
				merged = mergeStringSlice(merged, []string{fmt.Sprintf("read_file(%s)", dir)})
				merged = mergeStringSlice(merged, []string{fmt.Sprintf("write_file(%s)", dir)})
			}
		}

		setStringSliceInMap(grants, "allow", merged)

		userSettings["globalPermissionGrants"] = grants
		m["userSettings"] = userSettings

		err = writeJSONFile(expanded, m)
		if err != nil {
			return nil, err
		}
	}

	return files, nil
}
