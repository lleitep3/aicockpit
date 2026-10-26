package providers

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	claudeProviderName = "claude"
	geminiProviderName = "gemini"

	claudeMCPPath  = ".mcp.json"
	geminiMCPPath  = ".gemini/settings.json"
	codexMCPPath   = ".codex/config.toml"
	copilotMCPPath = "~/.copilot/mcp-config.json"
)

// ClaudeCompiler writes Claude Code's project MCP config. Canonical rules,
// skills, and permissions are not translated for Claude in this phase.
type ClaudeCompiler struct{}

// NewClaudeCompiler creates a compiler for Claude Code.
func NewClaudeCompiler() *ClaudeCompiler {
	return &ClaudeCompiler{}
}

// Name returns the provider identifier used by providers.yaml.
func (c *ClaudeCompiler) Name() string {
	return claudeProviderName
}

// CompileEntrypoint returns no files. Claude rules are not compiled yet.
func (c *ClaudeCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(claudeProviderName, provider)
}

// CompileSkills returns no files. Claude skills are not compiled yet.
func (c *ClaudeCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(claudeProviderName, provider)
}

// CompileRules returns no files. Claude rules are not compiled yet.
func (c *ClaudeCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(claudeProviderName, provider)
}

// CompileWorkflows returns no files. Claude workflows are not compiled yet.
func (c *ClaudeCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(claudeProviderName, provider)
}

// CompilePermissions returns no files. Claude permissions are not compiled yet.
func (c *ClaudeCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(claudeProviderName, provider)
}

// CompileAgents returns no files. Claude agents are not compiled yet.
func (c *ClaudeCompiler) CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(claudeProviderName, provider)
}

// CompileMCP writes .mcp.json with one streamable HTTP server per bank.
// Claude Code requires type "http". A url without type is ignored.
func (c *ClaudeCompiler) CompileMCP(servers []MCPServer, provider *Provider) (map[string]string, error) {
	return compileJSONMCP(claudeProviderName, provider, claudeMCPPath, servers, claudeMCPBody)
}

// GeminiCompiler writes the Gemini CLI project MCP config. This is separate
// from the Antigravity compiler, which does not accept this HTTP document.
type GeminiCompiler struct{}

// NewGeminiCompiler creates a compiler for the Gemini CLI.
func NewGeminiCompiler() *GeminiCompiler {
	return &GeminiCompiler{}
}

// Name returns the provider identifier used by providers.yaml.
func (c *GeminiCompiler) Name() string {
	return geminiProviderName
}

// CompileEntrypoint returns no files. Gemini CLI rules are not compiled yet.
func (c *GeminiCompiler) CompileEntrypoint(entrypoint *CanonicalEntrypoint, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(geminiProviderName, provider)
}

// CompileSkills returns no files. Gemini CLI skills are not compiled yet.
func (c *GeminiCompiler) CompileSkills(skills []CanonicalSkill, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(geminiProviderName, provider)
}

// CompileRules returns no files. Gemini CLI rules are not compiled yet.
func (c *GeminiCompiler) CompileRules(rules []CanonicalRule, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(geminiProviderName, provider)
}

// CompileWorkflows returns no files. Gemini CLI workflows are not compiled yet.
func (c *GeminiCompiler) CompileWorkflows(workflows []CanonicalWorkflow, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(geminiProviderName, provider)
}

// CompilePermissions returns no files. Gemini CLI permissions are not compiled yet.
func (c *GeminiCompiler) CompilePermissions(perms *CanonicalPermissions, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(geminiProviderName, provider)
}

// CompileAgents returns no files. Gemini CLI agents are not compiled yet.
func (c *GeminiCompiler) CompileAgents(agents []CanonicalAgent, provider *Provider) (map[string]string, error) {
	return emptyMCPProvider(geminiProviderName, provider)
}

// CompileMCP writes .gemini/settings.json. Gemini CLI uses httpUrl for the
// streamable HTTP transport. The url field is the legacy SSE transport.
func (c *GeminiCompiler) CompileMCP(servers []MCPServer, provider *Provider) (map[string]string, error) {
	return compileJSONMCP(geminiProviderName, provider, geminiMCPPath, servers, geminiMCPBody)
}

// CompileMCP writes the project Codex config.toml MCP tables. Codex selects
// streamable HTTP when a server table has url and no command.
func (c *CodexCompiler) CompileMCP(servers []MCPServer, provider *Provider) (map[string]string, error) {
	path, enabled, err := mcpOutputPath(codexProviderName, provider, codexMCPPath)
	if err != nil || !enabled {
		return nil, err
	}
	ordered, err := orderedMCPServers(servers)
	if err != nil || len(ordered) == 0 {
		return nil, err
	}
	return map[string]string{path: renderCodexMCP(ordered)}, nil
}

// CompileMCP writes ~/.copilot/mcp-config.json. Copilot CLI uses type "http"
// for streamable HTTP. type "sse" is the legacy transport.
func (c *GitHubCopilotCompiler) CompileMCP(servers []MCPServer, provider *Provider) (map[string]string, error) {
	return compileJSONMCP(githubCopilotProviderName, provider, copilotMCPPath, servers, copilotMCPBody)
}

func emptyMCPProvider(name string, provider *Provider) (map[string]string, error) {
	if provider == nil {
		return nil, fmt.Errorf("%s provider configuration cannot be nil", name)
	}
	return map[string]string{}, nil
}

type mcpJSONServer func(MCPServer) any

func compileJSONMCP(providerName string, provider *Provider, defaultPath string, servers []MCPServer, body mcpJSONServer) (map[string]string, error) {
	path, enabled, err := mcpOutputPath(providerName, provider, defaultPath)
	if err != nil || !enabled {
		return nil, err
	}
	ordered, err := orderedMCPServers(servers)
	if err != nil || len(ordered) == 0 {
		if err != nil {
			return nil, err
		}
		return map[string]string{}, nil
	}
	serversDoc := make(map[string]any, len(ordered))
	for _, server := range ordered {
		serversDoc[server.Name] = body(server)
	}
	encoded, err := encodeJSON(map[string]any{"mcpServers": serversDoc})
	if err != nil {
		return nil, err
	}
	return map[string]string{path: encoded}, nil
}

type claudeMCPServer struct {
	Type string `json:"type"`
	URL  string `json:"url"`
}

func claudeMCPBody(server MCPServer) any {
	return claudeMCPServer{Type: "http", URL: server.URL}
}

type geminiMCPServer struct {
	HTTPURL string `json:"httpUrl"`
}

func geminiMCPBody(server MCPServer) any {
	return geminiMCPServer{HTTPURL: server.URL}
}

type copilotMCPServer struct {
	Type  string   `json:"type"`
	URL   string   `json:"url"`
	Tools []string `json:"tools"`
}

func copilotMCPBody(server MCPServer) any {
	return copilotMCPServer{Type: "http", URL: server.URL, Tools: []string{"*"}}
}

func renderCodexMCP(servers []MCPServer) string {
	var b strings.Builder
	for i, server := range servers {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "[mcp_servers.%s]\n", strconv.Quote(server.Name))
		fmt.Fprintf(&b, "url = %s\n", strconv.Quote(server.URL))
	}
	return b.String()
}
