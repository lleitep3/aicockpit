package providers

// CanonicalSkill represents a modular capability.
type CanonicalSkill struct {
	Name        string
	Description string
	Content     string
	ScriptFiles map[string]string // e.g. "scripts/deploy.sh" -> "#!/bin/bash..."
}

// CanonicalRule represents a behavioral guideline or constraint.
type CanonicalRule struct {
	Name    string
	Content string
}

// CanonicalWorkflow represents a step-by-step procedure.
type CanonicalWorkflow struct {
	Name        string
	Description string
	Steps       []string
}

// CanonicalPermissions represents system restrictions.
type CanonicalPermissions struct {
	AllowedCommands []string
	DeniedCommands  []string
	AllowedDirs     []string
	// Devin-specific permissions
	Allow []string // Examples: "Read(src/**)", "Exec(git)", "Write(tests/**)"
	Deny  []string // Examples: "Exec(rm)", "Write(.env*)"
	Ask   []string // Examples: "Write(**)", "exec"
}

// CanonicalEntrypoint represents the initial bootstrap instructions.
type CanonicalEntrypoint struct {
	GoldenRules    []string
	ProjectContext string
}

// CanonicalAgent represents a custom subagent profile.
type CanonicalAgent struct {
	Name         string
	Description  string
	Model        string
	AllowedTools []string
	Permissions  map[string]interface{} // allow, deny, ask
	Content      string                 // System prompt
	MaxNesting   int                    // Maximum nesting depth
}

// Compiler defines the strategy each AI provider adapter must implement
// to translate canonical AST models into their specific structures.
type Compiler interface {
	Name() string
	CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error)
	CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error)
	CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error)
	CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error)
	CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error)
	CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error)
}
