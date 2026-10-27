package providers

// CanonicalSkill represents a modular capability.
type CanonicalSkill struct {
	Name          string
	Directory     string // directory name used in provider output; defaults to Name
	Description   string
	Content       string
	Frontmatter   string            // raw YAML fields, excluding --- delimiters
	ResourceFiles map[string]string // relative path -> file contents
	ResourceModes map[string]uint32 // relative path -> permission bits
	ScriptFiles   map[string]string // legacy alias for script resources
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

// MCPCompiler is implemented by adapters that can declare MCP servers in the
// agent's native config. Adapters that cannot represent an HTTP MCP server
// omit the interface. Deploy then skips them instead of writing a file the
// agent would ignore.
type MCPCompiler interface {
	CompileMCP(servers []MCPServer, provider *Provider) (map[string]string, error)
}
