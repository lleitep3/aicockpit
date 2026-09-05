package providers

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

const codexProviderName = "codex"

// CodexCompiler compiles canonical AICockpit assets into Codex's project-local
// instruction, skill, permission, and custom-agent formats.
type CodexCompiler struct{}

// NewCodexCompiler creates a compiler for the Codex provider.
func NewCodexCompiler() *CodexCompiler {
	return &CodexCompiler{}
}

// Name returns the provider identifier used by providers.yaml.
func (c *CodexCompiler) Name() string {
	return codexProviderName
}

// CompileEntrypoint renders the canonical context and golden rules into
// AGENTS.md. The ProviderManager adds its managed block around this content.
func (c *CodexCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	feature, enabled, err := codexFeature(provider, "rules")
	if err != nil {
		return nil, err
	}
	if !enabled {
		return map[string]string{}, nil
	}
	if entrypoint == nil {
		return nil, fmt.Errorf("codex entrypoint cannot be nil")
	}

	var content strings.Builder
	if projectContext := strings.TrimSpace(entrypoint.ProjectContext); projectContext != "" {
		content.WriteString(projectContext)
		content.WriteString("\n\n")
	}

	if len(entrypoint.GoldenRules) > 0 {
		content.WriteString("## AICockpit Gold Rules\n\n")
		for _, rule := range entrypoint.GoldenRules {
			if rule = strings.TrimSpace(rule); rule != "" {
				content.WriteString(rule)
				content.WriteString("\n\n")
			}
		}
	}

	return map[string]string{
		feature.Path: AddGeneratedHeader(strings.TrimSpace(content.String()), codexProviderName),
	}, nil
}

// CompileRules appends canonical rules to the Codex AGENTS.md target.
func (c *CodexCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
	feature, enabled, err := codexFeature(provider, "rules")
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

// CompileSkills renders each canonical skill using Codex's SKILL.md format.
func (c *CodexCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	feature, enabled, err := codexFeature(provider, "skills")
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

		skillPath := filepath.Join(feature.Path, skill.Name)
		content := renderCodexSkill(skill.Name, skill.Description, skill.Content)
		files[filepath.Join(skillPath, "SKILL.md")] = AddGeneratedHeader(content, codexProviderName)

		scriptNames := make([]string, 0, len(skill.ScriptFiles))
		for scriptName := range skill.ScriptFiles {
			scriptNames = append(scriptNames, scriptName)
		}
		sort.Strings(scriptNames)
		for _, scriptName := range scriptNames {
			if err := validateCodexRelativePath(scriptName, "skill script"); err != nil {
				return nil, err
			}
			files[filepath.Join(skillPath, scriptName)] = skill.ScriptFiles[scriptName]
		}
	}

	return files, nil
}

// CompileWorkflows exposes canonical workflows as Codex skills. Codex has no
// separate project workflow directory, so a skill is the most portable target.
func (c *CodexCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	feature, enabled, err := codexFeature(provider, "workflows")
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

		var body strings.Builder
		body.WriteString("# Workflow: ")
		body.WriteString(workflow.Name)
		body.WriteString("\n\n")
		if description := strings.TrimSpace(workflow.Description); description != "" {
			body.WriteString(description)
			body.WriteString("\n\n")
		}
		body.WriteString("## Steps\n\n")
		for i, step := range workflow.Steps {
			fmt.Fprintf(&body, "%d. %s\n", i+1, strings.TrimSpace(step))
		}

		content := renderCodexSkill(workflow.Name, workflow.Description, body.String())
		files[filepath.Join(feature.Path, workflow.Name, "SKILL.md")] = AddGeneratedHeader(content, codexProviderName)
	}

	return files, nil
}

// CompilePermissions translates command permissions into Codex prefix rules.
// Directory permissions and provider-specific expressions are retained as
// warnings in the generated file because Codex's exec policy has no equivalent
// for them.
func (c *CodexCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	feature, enabled, err := codexFeature(provider, "permissions")
	if err != nil {
		return nil, err
	}
	if !enabled || perms == nil {
		return map[string]string{}, nil
	}

	rules, warnings := buildCodexPermissionRules(perms)
	if len(rules) == 0 && len(warnings) == 0 {
		return map[string]string{}, nil
	}

	var content strings.Builder
	for _, warning := range warnings {
		fmt.Fprintf(&content, "# WARNING: %s\n", warning)
	}
	if len(warnings) > 0 && len(rules) > 0 {
		content.WriteByte('\n')
	}

	for _, rule := range rules {
		pattern, err := splitCodexCommandPrefix(rule.command)
		if err != nil {
			return nil, fmt.Errorf("invalid Codex permission command %q: %w", rule.command, err)
		}
		content.WriteString("prefix_rule(\n    pattern = [")
		for i, token := range pattern {
			if i > 0 {
				content.WriteString(", ")
			}
			content.WriteString(strconv.Quote(token))
		}
		content.WriteString("],\n    decision = ")
		content.WriteString(strconv.Quote(rule.decision))
		content.WriteString(",\n)\n\n")
	}

	return map[string]string{
		codexPermissionFilePath(feature.Path): AddGeneratedHeader(strings.TrimSpace(content.String()), ".rules"),
	}, nil
}

// CompileAgents renders canonical agents using Codex's TOML custom-agent
// schema. Unsupported canonical fields are surfaced as comments in the file.
func (c *CodexCompiler) CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error) {
	feature, enabled, err := codexFeature(provider, "agents")
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

		var content strings.Builder
		for _, warning := range codexAgentWarnings(agent) {
			fmt.Fprintf(&content, "# WARNING: %s\n", warning)
		}
		if content.Len() > 0 {
			content.WriteByte('\n')
		}

		fmt.Fprintf(&content, "name = %s\n", strconv.Quote(agent.Name))
		fmt.Fprintf(&content, "description = %s\n", strconv.Quote(agent.Description))
		if agent.Model != "" {
			fmt.Fprintf(&content, "model = %s\n", strconv.Quote(agent.Model))
		}
		fmt.Fprintf(&content, "developer_instructions = %s\n", strconv.Quote(agent.Content))

		path := filepath.Join(feature.Path, agent.Name+".toml")
		files[path] = AddGeneratedHeader(strings.TrimSpace(content.String()), ".toml")
	}

	return files, nil
}

func codexFeature(provider *Provider, name string) (*FeatureConfig, bool, error) {
	if provider == nil {
		return nil, false, fmt.Errorf("codex provider configuration cannot be nil")
	}
	feature, exists := provider.Features[name]
	if !exists || feature == nil || !feature.Enabled {
		return nil, false, nil
	}
	if strings.TrimSpace(feature.Path) == "" {
		return nil, false, fmt.Errorf("codex feature %q is enabled but has no path", name)
	}

	copy := *feature
	copy.Path = strings.TrimSpace(copy.Path)
	return &copy, true, nil
}

func renderCodexSkill(name, description, body string) string {
	var content strings.Builder
	content.WriteString("---\n")
	fmt.Fprintf(&content, "name: %s\n", strconv.Quote(name))
	fmt.Fprintf(&content, "description: %s\n", strconv.Quote(description))
	content.WriteString("---\n")
	if body = strings.TrimSpace(body); body != "" {
		content.WriteString(body)
		content.WriteByte('\n')
	}
	return content.String()
}

func validateCodexArtifactName(name, kind string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("codex %s name cannot be empty", kind)
	}
	if name != strings.TrimSpace(name) || strings.ContainsAny(name, `/\\`) {
		return fmt.Errorf("invalid Codex %s name %q: path separators are not allowed", kind, name)
	}
	if name == "." || name == ".." {
		return fmt.Errorf("invalid Codex %s name %q", kind, name)
	}
	return nil
}

func validateCodexRelativePath(path, kind string) error {
	if strings.TrimSpace(path) == "" || filepath.IsAbs(path) {
		return fmt.Errorf("invalid Codex %s path %q", kind, path)
	}
	if strings.Contains(path, `\`) {
		return fmt.Errorf("invalid Codex %s path %q: use platform-neutral relative paths", kind, path)
	}
	cleaned := filepath.Clean(path)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid Codex %s path %q: path escapes the skill directory", kind, path)
	}
	return nil
}

type codexPermissionRule struct {
	command  string
	decision string
}

func buildCodexPermissionRules(perms *CanonicalPermissions) ([]codexPermissionRule, []string) {
	var rules []codexPermissionRule
	var warnings []string
	seen := make(map[string]struct{})

	add := func(command, decision string) {
		command = strings.TrimSpace(command)
		if command == "" {
			return
		}
		key := decision + "\x00" + command
		if _, exists := seen[key]; exists {
			return
		}
		seen[key] = struct{}{}
		rules = append(rules, codexPermissionRule{command: command, decision: decision})
	}

	for _, command := range perms.AllowedCommands {
		add(command, "allow")
	}
	for _, command := range perms.DeniedCommands {
		add(command, "forbidden")
	}

	addStructured := func(values []string, decision, field string) {
		for _, value := range values {
			if command, ok := codexExecExpression(value); ok {
				add(command, decision)
				continue
			}
			if strings.TrimSpace(value) != "" {
				warnings = append(warnings, fmt.Sprintf("%s entry %q is not a Codex command and was not translated", field, value))
			}
		}
	}
	addStructured(perms.Allow, "allow", "allow")
	addStructured(perms.Deny, "forbidden", "deny")
	addStructured(perms.Ask, "prompt", "ask")

	if len(perms.AllowedDirs) > 0 {
		warnings = append(warnings, "allowed_dirs requires Codex sandbox configuration and was not translated to execpolicy rules")
	}
	return rules, warnings
}

func codexExecExpression(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "Exec(") || !strings.HasSuffix(value, ")") {
		return "", false
	}
	command := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "Exec("), ")"))
	return command, command != ""
}

func splitCodexCommandPrefix(command string) ([]string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	var tokens []string
	var token strings.Builder
	var quote rune
	escaped := false
	tokenStarted := false

	flush := func() {
		if tokenStarted {
			tokens = append(tokens, token.String())
			token.Reset()
			tokenStarted = false
		}
	}

	for _, char := range command {
		if escaped {
			token.WriteRune(char)
			escaped = false
			tokenStarted = true
			continue
		}

		if quote != 0 {
			if char == quote {
				quote = 0
			} else {
				token.WriteRune(char)
			}
			tokenStarted = true
			continue
		}

		switch {
		case char == '\\':
			escaped = true
			tokenStarted = true
		case char == '\'' || char == '"':
			quote = char
			tokenStarted = true
		case unicode.IsSpace(char):
			flush()
		default:
			token.WriteRune(char)
			tokenStarted = true
		}
	}

	if escaped {
		return nil, fmt.Errorf("unterminated escape")
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	flush()
	if len(tokens) == 0 {
		return nil, fmt.Errorf("command cannot be empty")
	}
	return tokens, nil
}

func codexPermissionFilePath(path string) string {
	if filepath.Ext(path) == ".rules" {
		return path
	}
	return filepath.Join(path, "aicockpit.rules")
}

func codexAgentWarnings(agent CanonicalAgent) []string {
	var warnings []string
	if len(agent.AllowedTools) > 0 {
		warnings = append(warnings, "allowed_tools is provider-specific and was not translated")
	}
	if len(agent.Permissions) > 0 {
		warnings = append(warnings, "agent permissions are provider-specific and were not translated")
	}
	if agent.MaxNesting > 0 {
		warnings = append(warnings, "max_nesting is not supported by the Codex custom-agent schema")
	}
	return warnings
}
