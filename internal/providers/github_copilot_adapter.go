package providers

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const githubCopilotProviderName = "github-copilot"

// GitHubCopilotCompiler compiles canonical AICockpit assets into the user-level
// GitHub Copilot CLI locations rooted in ~/.copilot.
type GitHubCopilotCompiler struct{}

// NewGitHubCopilotCompiler creates a GitHub Copilot compiler.
func NewGitHubCopilotCompiler() *GitHubCopilotCompiler {
	return &GitHubCopilotCompiler{}
}

// Name returns the provider identifier used by providers.yaml.
func (c *GitHubCopilotCompiler) Name() string {
	return githubCopilotProviderName
}

// CompileEntrypoint renders project context and golden rules into the user's
// Copilot CLI instructions file.
func (c *GitHubCopilotCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	feature, enabled, err := githubCopilotFeature(provider, "rules")
	if err != nil {
		return nil, err
	}
	if !enabled {
		return map[string]string{}, nil
	}
	if entrypoint == nil {
		return nil, fmt.Errorf("github-copilot entrypoint cannot be nil")
	}

	var content strings.Builder
	if projectContext := strings.TrimSpace(entrypoint.ProjectContext); projectContext != "" {
		content.WriteString(projectContext)
		content.WriteString("\n\n")
	}
	if len(entrypoint.GoldenRules) > 0 {
		content.WriteString("## AICockpit Gold Rules\n\n")
		for _, rule := range entrypoint.GoldenRules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}
			content.WriteString(rule)
			content.WriteString("\n\n")
		}
	}

	return map[string]string{
		feature.Path: AddGeneratedHeader(strings.TrimSpace(content.String()), githubCopilotProviderName),
	}, nil
}

// CompileRules appends canonical rules to the user's Copilot CLI instructions file.
func (c *GitHubCopilotCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
	feature, enabled, err := githubCopilotFeature(provider, "rules")
	if err != nil {
		return nil, err
	}
	if !enabled || len(rules) == 0 {
		return map[string]string{}, nil
	}

	ordered := append([]CanonicalRule(nil), rules...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	var content strings.Builder
	for _, rule := range ordered {
		body := strings.TrimSpace(rule.Content)
		if body == "" {
			continue
		}
		if content.Len() > 0 {
			content.WriteString("\n\n---\n\n")
		}
		content.WriteString(body)
	}
	if content.Len() == 0 {
		return map[string]string{}, nil
	}
	return map[string]string{feature.Path: content.String()}, nil
}

// CompileSkills renders each canonical skill as a directory under ~/.copilot/skills.
func (c *GitHubCopilotCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	feature, enabled, err := githubCopilotFeature(provider, "skills")
	if err != nil {
		return nil, err
	}
	if !enabled || len(skills) == 0 {
		return map[string]string{}, nil
	}

	ordered := append([]CanonicalSkill(nil), skills...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	files := make(map[string]string)
	for _, skill := range ordered {
		if err := validateCodexArtifactName(skill.Name, "skill"); err != nil {
			return nil, err
		}
		directory := canonicalSkillDirectory(skill)
		if err := validateCodexArtifactName(directory, "skill directory"); err != nil {
			return nil, err
		}
		skillDir := filepath.Join(feature.Path, directory)
		content := renderCanonicalSkill(skill)
		files[filepath.Join(skillDir, "SKILL.md")] = AddGeneratedHeader(content, githubCopilotProviderName)

		resources := canonicalSkillResources(skill)
		resourceNames := make([]string, 0, len(resources))
		for resourceName := range resources {
			resourceNames = append(resourceNames, resourceName)
		}
		sort.Strings(resourceNames)
		for _, resourceName := range resourceNames {
			if err := validateCodexRelativePath(resourceName, "skill resource"); err != nil {
				return nil, err
			}
			files[filepath.Join(skillDir, filepath.FromSlash(resourceName))] = resources[resourceName]
		}
	}
	return files, nil
}

// CompileWorkflows publishes each canonical workflow as an invocable skill.
func (c *GitHubCopilotCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	feature, enabled, err := githubCopilotFeature(provider, "workflows")
	if err != nil {
		return nil, err
	}
	if !enabled || len(workflows) == 0 {
		return map[string]string{}, nil
	}

	ordered := append([]CanonicalWorkflow(nil), workflows...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	files := make(map[string]string)
	for _, workflow := range ordered {
		if err := validateCodexArtifactName(workflow.Name, "workflow"); err != nil {
			return nil, err
		}
		files[filepath.Join(feature.Path, "cockpit-workflow-"+workflow.Name, "SKILL.md")] = AddGeneratedHeader(renderGitHubCopilotWorkflow(workflow), githubCopilotProviderName)
	}
	return files, nil
}

// CompilePermissions renders the canonical permission model to a JSON policy.
func (c *GitHubCopilotCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	_, enabled, err := githubCopilotFeature(provider, "permissions")
	if err != nil {
		return nil, err
	}
	if !enabled || perms == nil {
		return map[string]string{}, nil
	}

	if len(dedupeStrings(perms.AllowedCommands, perms.DeniedCommands, perms.AllowedDirs, perms.Allow, perms.Deny, perms.Ask)) == 0 {
		return map[string]string{}, nil
	}

	return nil, fmt.Errorf("github-copilot cannot compile canonical permissions: Copilot CLI approvals are repository-scoped and do not represent the canonical allow/deny/ask policy")
}

// CompileAgents renders canonical agents as user-level Copilot CLI agent files.
func (c *GitHubCopilotCompiler) CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error) {
	feature, enabled, err := githubCopilotFeature(provider, "agents")
	if err != nil {
		return nil, err
	}
	if !enabled || len(agents) == 0 {
		return map[string]string{}, nil
	}

	ordered := append([]CanonicalAgent(nil), agents...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Name < ordered[j].Name
	})

	files := make(map[string]string)
	for _, agent := range ordered {
		if err := validateCodexArtifactName(agent.Name, "agent"); err != nil {
			return nil, err
		}
		if len(agent.Permissions) > 0 {
			return nil, fmt.Errorf("github-copilot agent %q has canonical permissions that cannot be represented by a Copilot CLI agent", agent.Name)
		}
		if agent.MaxNesting != 0 {
			return nil, fmt.Errorf("github-copilot agent %q has a max_nesting value that cannot be represented by a Copilot CLI agent", agent.Name)
		}
		files[filepath.Join(feature.Path, agent.Name+".agent.md")] = AddGeneratedHeader(renderGitHubCopilotAgent(agent), githubCopilotProviderName)
	}
	return files, nil
}

func githubCopilotFeature(provider *Provider, name string) (*FeatureConfig, bool, error) {
	if provider == nil {
		return nil, false, fmt.Errorf("github-copilot provider configuration cannot be nil")
	}
	feature, exists := provider.Features[name]
	if !exists || feature == nil || !feature.Enabled {
		return nil, false, nil
	}
	if strings.TrimSpace(feature.Path) == "" {
		return nil, false, fmt.Errorf("github-copilot feature %q is enabled but has no path", name)
	}
	featureCopy := *feature
	featureCopy.Path = strings.TrimSpace(featureCopy.Path)
	return &featureCopy, true, nil
}

func renderGitHubCopilotWorkflow(workflow CanonicalWorkflow) string {
	var body strings.Builder
	body.WriteString("# Workflow: ")
	body.WriteString(workflow.Name)
	body.WriteString("\n\n## Steps\n\n")
	for i, step := range workflow.Steps {
		fmt.Fprintf(&body, "%d. %s\n", i+1, strings.TrimSpace(step))
	}

	description := strings.TrimSpace(workflow.Description)
	if description == "" {
		description = "Run the " + workflow.Name + " workflow."
	}
	return renderCodexSkill("cockpit-workflow-"+workflow.Name, description, body.String())
}

func renderGitHubCopilotAgent(agent CanonicalAgent) string {
	var content strings.Builder
	content.WriteString("---\n")
	fmt.Fprintf(&content, "name: %s\n", strconv.Quote(agent.Name))
	fmt.Fprintf(&content, "description: %s\n", strconv.Quote(agent.Description))
	if agent.Model != "" {
		fmt.Fprintf(&content, "model: %s\n", strconv.Quote(agent.Model))
	}
	if allowedTools := dedupeStrings(agent.AllowedTools); len(allowedTools) > 0 {
		content.WriteString("tools:\n")
		for _, tool := range allowedTools {
			fmt.Fprintf(&content, "  - %s\n", strconv.Quote(tool))
		}
	}
	content.WriteString("include-custom-instructions: true\n")
	content.WriteString("---\n")
	if body := strings.TrimSpace(agent.Content); body != "" {
		content.WriteString(body)
		content.WriteByte('\n')
	}
	return content.String()
}

func dedupeStrings(values ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, slice := range values {
		for _, item := range slice {
			trimmed := strings.TrimSpace(item)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}
