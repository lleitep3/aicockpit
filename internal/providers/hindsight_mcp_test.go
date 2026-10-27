package providers

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/memoryhindsight"
)

func TestHindsightServers(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module github.com/acme/Demo_App\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		settings  hindsight.Settings
		id        memoryhindsight.Identity
		wantErr   string
		wantNil   bool
		wantBanks []string
		wantURLs  []string
	}{
		{
			name:     "disabled does not invent an endpoint",
			settings: hindsight.Settings{Banks: hindsight.BanksConfig{Project: "demo-app", User: "ada"}, APIURL: "http://127.0.0.1:8888"},
			wantNil:  true,
		},
		{
			name: "configured banks and explicit url",
			settings: hindsight.Settings{
				Enabled: true,
				APIURL:  " http://127.0.0.1:8888/ ",
				Banks:   hindsight.BanksConfig{Project: "demo-app", User: "ada"},
			},
			wantBanks: []string{"demo-app", "ada"},
			wantURLs: []string{
				"http://127.0.0.1:8888/mcp/demo-app/",
				"http://127.0.0.1:8888/mcp/ada/",
			},
		},
		{
			name:      "derived banks and default port",
			settings:  hindsight.Settings{Enabled: true, APIPort: 9000, UIPort: 9001},
			id:        memoryhindsight.Identity{WorkDir: dir, Username: "Ada Lovelace"},
			wantBanks: []string{"project-demo_app", "user-ada-lovelace"},
			wantURLs: []string{
				"http://127.0.0.1:9000/mcp/project-demo_app/",
				"http://127.0.0.1:9000/mcp/user-ada-lovelace/",
			},
		},
		{
			name: "https base wins over the host port",
			settings: hindsight.Settings{
				Enabled: true,
				APIPort: 9000,
				UIPort:  9001,
				APIURL:  "https://memory.example",
				Banks:   hindsight.BanksConfig{Project: "demo-app", User: "ada"},
			},
			wantURLs: []string{
				"https://memory.example/mcp/demo-app/",
				"https://memory.example/mcp/ada/",
			},
		},
		{
			name: "path traversal in a configured name is sanitized",
			settings: hindsight.Settings{
				Enabled: true,
				Banks:   hindsight.BanksConfig{Project: "../secret", User: "ada"},
			},
			wantBanks: []string{"secret", "ada"},
			wantURLs: []string{
				"http://127.0.0.1:8888/mcp/secret/",
				"http://127.0.0.1:8888/mcp/ada/",
			},
		},
		{
			name:     "reserved bank",
			settings: hindsight.Settings{Enabled: true, Banks: hindsight.BanksConfig{Project: "project", User: "ada"}},
			wantErr:  "bank",
		},
		{
			name:     "empty sanitized bank",
			settings: hindsight.Settings{Enabled: true, Banks: hindsight.BanksConfig{Project: "..", User: "ada"}},
			wantErr:  "bank",
		},
		{
			name:     "same bank",
			settings: hindsight.Settings{Enabled: true, Banks: hindsight.BanksConfig{Project: "shared", User: "shared"}},
			wantErr:  "differ",
		},
		{
			name:     "ports must differ",
			settings: hindsight.Settings{Enabled: true, APIPort: 1, UIPort: 1},
			wantErr:  "ports",
		},
		{
			name:     "api url with userinfo",
			settings: hindsight.Settings{Enabled: true, APIURL: "http://user:pass@127.0.0.1:8888", Banks: hindsight.BanksConfig{Project: "demo-app", User: "ada"}},
			wantErr:  "url",
		},
		{
			name:     "api url with path",
			settings: hindsight.Settings{Enabled: true, APIURL: "http://127.0.0.1:8888/v1", Banks: hindsight.BanksConfig{Project: "demo-app", User: "ada"}},
			wantErr:  "path",
		},
		{
			name:     "api url with fragment",
			settings: hindsight.Settings{Enabled: true, APIURL: "http://127.0.0.1:8888/#mcp", Banks: hindsight.BanksConfig{Project: "demo-app", User: "ada"}},
			wantErr:  "fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HindsightServers(tt.settings, tt.id)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				if strings.Contains(err.Error(), "/mcp/") {
					t.Fatalf("error invented an endpoint: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantNil {
				if got != nil {
					t.Fatalf("servers = %#v, want nil", got)
				}
				return
			}
			if len(got) != 2 || got[0].Name != hindsightProjectServer || got[1].Name != hindsightUserServer {
				t.Fatalf("servers = %#v", got)
			}
			if tt.wantBanks != nil && (got[0].Bank != tt.wantBanks[0] || got[1].Bank != tt.wantBanks[1]) {
				t.Fatalf("banks = %s %s, want %v", got[0].Bank, got[1].Bank, tt.wantBanks)
			}
			for i, url := range tt.wantURLs {
				if got[i].URL != url {
					t.Fatalf("url[%d] = %q, want %q", i, got[i].URL, url)
				}
				if strings.Contains(got[i].URL, "..") {
					t.Fatalf("url escapes the mcp prefix: %s", got[i].URL)
				}
			}
		})
	}
}

func TestCompileMCPMatchesGolden(t *testing.T) {
	servers := testHindsightServers(t)
	providers := testMCPProviders()
	cases := []struct {
		name     string
		compiler MCPCompiler
		path     string
		golden   string
		needles  []string
		absent   []string
	}{
		{
			name:     "claude",
			compiler: NewClaudeCompiler(),
			path:     ".mcp.json",
			golden:   "claude.mcp.json",
			needles:  []string{`"type": "http"`, "http://127.0.0.1:8888/mcp/demo-app/", "http://127.0.0.1:8888/mcp/ada/"},
			absent:   []string{`"type": "sse"`, `"httpUrl"`},
		},
		{
			name:     "codex",
			compiler: NewCodexCompiler(),
			path:     ".codex/config.toml",
			golden:   "codex.config.toml",
			needles:  []string{`[mcp_servers."hindsight-project"]`, `[mcp_servers."hindsight-user"]`, `url = "http://127.0.0.1:8888/mcp/demo-app/"`},
			absent:   []string{"command =", "sse"},
		},
		{
			name:     "gemini",
			compiler: NewGeminiCompiler(),
			path:     ".gemini/settings.json",
			golden:   "gemini.settings.json",
			needles:  []string{`"httpUrl": "http://127.0.0.1:8888/mcp/demo-app/"`},
			absent:   []string{`"url":`, `"type": "sse"`},
		},
		{
			name:     "copilot",
			compiler: NewGitHubCopilotCompiler(),
			path:     "~/.copilot/mcp-config.json",
			golden:   "copilot.mcp-config.json",
			needles:  []string{`"type": "http"`, `"tools"`, `"*"`},
			absent:   []string{`"type": "sse"`, `"type": "local"`},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			files, err := tt.compiler.CompileMCP(servers, providers[tt.name])
			if err != nil {
				t.Fatal(err)
			}
			got, ok := files[tt.path]
			if !ok || len(files) != 1 {
				t.Fatalf("files = %#v", files)
			}
			for _, needle := range tt.needles {
				if !strings.Contains(got, needle) {
					t.Errorf("missing %q in\n%s", needle, got)
				}
			}
			for _, needle := range tt.absent {
				if strings.Contains(got, needle) {
					t.Errorf("unexpected %q in\n%s", needle, got)
				}
			}
			assertGoldenFile(t, filepath.Join("testdata", "hindsight", "enabled", tt.golden), got)
		})
	}
}

func TestCompileMCPEdges(t *testing.T) {
	servers := testHindsightServers(t)
	userOnly := []MCPServer{servers[1], servers[0]}
	claude := NewClaudeCompiler()
	provider := &Provider{}

	files, err := claude.CompileMCP(nil, provider)
	if err != nil || len(files) != 0 {
		t.Fatalf("empty servers files=%v err=%v", files, err)
	}
	if _, err := claude.CompileMCP(servers, nil); err == nil {
		t.Fatal("nil provider")
	}
	if _, err := claude.CompileMCP([]MCPServer{{Name: "other", URL: "http://127.0.0.1/mcp/x/"}}, provider); err == nil {
		t.Fatal("unknown server")
	}
	if _, err := claude.CompileMCP([]MCPServer{servers[0], servers[0]}, provider); err == nil {
		t.Fatal("duplicate server")
	}
	if _, err := claude.CompileMCP([]MCPServer{{Name: hindsightProjectServer, URL: "http://bad\n"}}, provider); err == nil {
		t.Fatal("newline url")
	}

	files, err = claude.CompileMCP(userOnly, provider)
	if err != nil {
		t.Fatal(err)
	}
	got := files[".mcp.json"]
	project := strings.Index(got, "hindsight-project")
	user := strings.Index(got, "hindsight-user")
	if project < 0 || user < 0 || project > user {
		t.Fatalf("servers were not ordered:\n%s", got)
	}

	disabled := &Provider{Features: map[string]*FeatureConfig{"mcp": {Enabled: false, Path: ".mcp.json"}}}
	files, err = claude.CompileMCP(servers, disabled)
	if err != nil || len(files) != 0 {
		t.Fatalf("disabled feature files=%v err=%v", files, err)
	}

	custom := &Provider{Features: map[string]*FeatureConfig{"mcp": {Enabled: true, Path: "custom/mcp.json"}}}
	files, err = NewCodexCompiler().CompileMCP(servers, custom)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files["custom/mcp.json"]; !ok {
		t.Fatalf("custom path files = %#v", files)
	}
	blankPath := &Provider{Features: map[string]*FeatureConfig{"mcp": {Enabled: true}}}
	files, err = NewGeminiCompiler().CompileMCP(servers, blankPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := files[".gemini/settings.json"]; !ok {
		t.Fatalf("default path files = %#v", files)
	}

	codex := NewCodexCompiler()
	if files, err = codex.CompileMCP(nil, provider); err != nil || len(files) != 0 {
		t.Fatalf("codex empty servers files=%v err=%v", files, err)
	}
	if _, err = codex.CompileMCP(servers, nil); err == nil {
		t.Fatal("codex nil provider")
	}
	if files, err = codex.CompileMCP(servers, disabled); err != nil || len(files) != 0 {
		t.Fatalf("codex disabled feature files=%v err=%v", files, err)
	}
	if _, err = NewGitHubCopilotCompiler().CompileMCP(servers, nil); err == nil {
		t.Fatal("copilot nil provider")
	}
}

func TestClaudeAndGeminiIgnoreCanonicalAssets(t *testing.T) {
	provider := &Provider{Features: map[string]*FeatureConfig{}}
	for _, compiler := range []Compiler{NewClaudeCompiler(), NewGeminiCompiler()} {
		t.Run(compiler.Name(), func(t *testing.T) {
			calls := []struct {
				name string
				fn   func() (map[string]string, error)
			}{
				{name: "entrypoint", fn: func() (map[string]string, error) {
					return compiler.CompileEntrypoint(&CanonicalEntrypoint{ProjectContext: "x"}, provider)
				}},
				{name: "skills", fn: func() (map[string]string, error) {
					return compiler.CompileSkills([]CanonicalSkill{{Name: "review"}}, provider)
				}},
				{name: "rules", fn: func() (map[string]string, error) {
					return compiler.CompileRules([]CanonicalRule{{Name: "style", Content: "x"}}, provider)
				}},
				{name: "workflows", fn: func() (map[string]string, error) {
					return compiler.CompileWorkflows([]CanonicalWorkflow{{Name: "ship"}}, provider)
				}},
				{name: "permissions", fn: func() (map[string]string, error) {
					return compiler.CompilePermissions(&CanonicalPermissions{Allow: []string{"Read"}}, provider)
				}},
				{name: "agents", fn: func() (map[string]string, error) {
					return compiler.CompileAgents([]CanonicalAgent{{Name: "reviewer"}}, provider)
				}},
			}
			for _, call := range calls {
				files, err := call.fn()
				if err != nil || len(files) != 0 {
					t.Errorf("%s files=%v err=%v", call.name, files, err)
				}
			}
			if _, err := compiler.CompileEntrypoint(nil, nil); err == nil {
				t.Fatal("nil provider")
			}
		})
	}
}

func TestMCPMerge(t *testing.T) {
	generated := `[mcp_servers."hindsight-project"]
url = "http://127.0.0.1:8888/mcp/demo-app/"

[mcp_servers."hindsight-user"]
url = "http://127.0.0.1:8888/mcp/ada/"
`
	existing := "# keep me\nsandbox_mode = \"workspace-write\"\n\n[mcp_servers.\"hindsight-project\"]\nurl = \"http://old/mcp/stale/\"\n\n[mcp_servers.\"hindsight-project\".oauth]\nclient_id = \"drop-me\"\n\n[sandbox]\nmode = \"on\"\n"
	merged, err := mergeMCPDocument("config.toml", existing, generated)
	if err != nil {
		t.Fatal(err)
	}
	want := "# keep me\nsandbox_mode = \"workspace-write\"\n\n[sandbox]\nmode = \"on\"\n\n" + generated
	if merged != want {
		t.Fatalf("toml merge =\n%s\nwant =\n%s", merged, want)
	}
	again, err := mergeMCPDocument("config.toml", merged, generated)
	if err != nil || again != merged {
		t.Fatalf("toml merge not idempotent: %v\n%s", err, again)
	}
	only, err := mergeMCPDocument(".codex/config.toml", generated, generated)
	if err != nil || only != generated {
		t.Fatalf("identical toml changed: %q %v", only, err)
	}

	fresh, err := mergeMCPDocument(".mcp.json", "", `{"mcpServers":{}}`+"\n")
	if err != nil || fresh != `{"mcpServers":{}}`+"\n" {
		t.Fatalf("empty json = %q %v", fresh, err)
	}
	withOther := "{\n  \"theme\": \"dark\",\n  \"mcpServers\": {\n    \"other\": {\"type\": \"http\", \"url\": \"https://example.com/mcp\"},\n    \"hindsight-project\": {\"type\": \"http\", \"url\": \"http://old\"}\n  }\n}\n"
	jsonMerged, err := mergeJSONMCP(withOther, generatedJSON())
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{`"theme": "dark"`, "https://example.com/mcp", "http://127.0.0.1:8888/mcp/demo-app/", "hindsight-user"} {
		if !strings.Contains(jsonMerged, needle) {
			t.Errorf("json merge missing %q\n%s", needle, jsonMerged)
		}
	}
	if strings.Contains(jsonMerged, "http://old") {
		t.Fatalf("stale url kept:\n%s", jsonMerged)
	}
	same, err := mergeJSONMCP(jsonMerged, generatedJSON())
	if err != nil || same != jsonMerged {
		t.Fatal("json merge not idempotent")
	}
	if _, err := mergeJSONMCP("{", generatedJSON()); err == nil {
		t.Fatal("invalid json")
	}
	if _, err := mergeJSONMCP(`{"mcpServers":[]}`, generatedJSON()); err == nil {
		t.Fatal("mcpServers array")
	}
	if _, err := mergeJSONMCP("null", generatedJSON()); err != nil {
		t.Fatal(err)
	}
	if _, err := mergeMCPDocument("mcp.yaml", "x: 1\n", generated); err == nil {
		t.Fatal("yaml config")
	}
	replaced, err := mergeMCPDocument("config.toml", "[mcp_servers.\"hindsight-project\"]\nurl = \"http://old\"\n", generated)
	if err != nil || replaced != generated {
		t.Fatalf("owned-only toml = %q %v", replaced, err)
	}
	blank, err := mergeMCPDocument("config.toml", " \n", generated)
	if err != nil || blank != generated {
		t.Fatalf("blank toml = %q %v", blank, err)
	}
	noNewline := mergeTOMLMCP("keep = 1\n", "url = 1")
	if noNewline != "keep = 1\n\nurl = 1\n" {
		t.Fatalf("toml without newline = %q", noNewline)
	}
	if err := writeMergedMCP(".mcp.json", "  ", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	abs, err := resolveCompiledPath("/tmp/aicockpit-mcp.toml", t.TempDir())
	if err != nil || abs != "/tmp/aicockpit-mcp.toml" {
		t.Fatalf("absolute path = %q %v", abs, err)
	}
}

func TestDeployHindsightMCP(t *testing.T) {
	servers := testHindsightServers(t)
	for _, name := range []string{"codex", "claude", "gemini", "github-copilot", "antigravity", "devin", "goose"} {
		t.Run(name, func(t *testing.T) {
			disabled := deployHindsightFixture(t, name, nil)
			alsoDisabled := deployHindsightFixture(t, name, mustDisabledServers(t))
			if !sameFiles(disabled, alsoDisabled) {
				t.Fatalf("disabled servers changed %s output", name)
			}
			assertGoldenTree(t, filepath.Join("disabled", name), disabled)

			enabled := deployHindsightFixture(t, name, servers)
			for path, content := range disabled {
				if enabled[path] != content {
					t.Fatalf("enabled deploy changed %s", path)
				}
			}
			switch name {
			case "claude", "codex", "gemini", "github-copilot":
				if len(enabled) != len(disabled)+1 {
					t.Fatalf("enabled files = %v", fileKeys(enabled))
				}
			default:
				if len(enabled) != len(disabled) {
					t.Fatalf("provider without mcp gained files: %v", fileKeys(enabled))
				}
			}
		})
	}
}

func TestDeployMCPMergePreservesExisting(t *testing.T) {
	servers := testHindsightServers(t)
	root, project, home := hindsightRoots(t)
	cockpit := filepath.Join(root, "cockpit")
	writeHindsightCanonical(t, cockpit)
	t.Setenv("HOME", home)

	existing := "{\n  \"mcpServers\": {\n    \"other\": {\"type\": \"http\", \"url\": \"https://example.com/mcp\"}\n  }\n}\n"
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".mcp.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}
	codexCfg := "# keep me\nsandbox_mode = \"workspace-write\"\n"
	if err := os.MkdirAll(filepath.Join(project, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".codex", "config.toml"), []byte(codexCfg), 0o644); err != nil {
		t.Fatal(err)
	}

	pm := NewProviderManager(testMCPProviderConfig())
	pm.SetHindsightMCP(servers)
	if err := pm.Deploy("claude", cockpit, project); err != nil {
		t.Fatal(err)
	}
	if err := pm.Deploy("codex", cockpit, project); err != nil {
		t.Fatal(err)
	}
	mergedJSON, err := os.ReadFile(filepath.Join(project, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(mergedJSON), "https://example.com/mcp") || !strings.Contains(string(mergedJSON), "/mcp/demo-app/") {
		t.Fatalf("claude merge =\n%s", mergedJSON)
	}
	mergedTOML, err := os.ReadFile(filepath.Join(project, ".codex", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(mergedTOML), codexCfg) || !strings.Contains(string(mergedTOML), `/mcp/ada/`) {
		t.Fatalf("codex merge =\n%s", mergedTOML)
	}

	info, err := os.Stat(filepath.Join(project, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := pm.Deploy("claude", cockpit, project); err != nil {
		t.Fatal(err)
	}
	again, err := os.Stat(filepath.Join(project, ".mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !again.ModTime().Equal(info.ModTime()) {
		t.Fatal("second deploy rewrote an unchanged mcp file")
	}
	body, err := os.ReadFile(filepath.Join(project, ".mcp.json"))
	if err != nil || string(body) != string(mergedJSON) {
		t.Fatal("second deploy changed mcp bytes")
	}
}

func TestDeployMCPReadError(t *testing.T) {
	project := t.TempDir()
	if err := os.Mkdir(filepath.Join(project, ".mcp.json"), 0o755); err != nil {
		t.Fatal(err)
	}
	pm := NewProviderManager(testMCPProviderConfig())
	pm.SetHindsightMCP(testHindsightServers(t))
	err := pm.Deploy("claude", t.TempDir(), project)
	if err == nil || !strings.Contains(err.Error(), "mcp config") {
		t.Fatalf("error = %v", err)
	}
}

func TestDeployMCPCompilerError(t *testing.T) {
	pm := NewProviderManager(&ProvidersConfig{Providers: map[string]*Provider{
		"mock": {Name: "mock", Features: map[string]*FeatureConfig{}},
	}})
	pm.Register(&mcpErrCompiler{mockCompiler: mockCompiler{name: "mock"}})
	pm.SetHindsightMCP(testHindsightServers(t))
	err := pm.Deploy("mock", t.TempDir(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "mcp broke") {
		t.Fatalf("error = %v", err)
	}
}

func TestUnsupportedProvidersDoNotImplementMCP(t *testing.T) {
	pm := NewProviderManager(&ProvidersConfig{})
	for _, name := range []string{"antigravity", "devin", "goose"} {
		if _, ok := pm.compilers[name].(MCPCompiler); ok {
			t.Errorf("%s implements MCPCompiler", name)
		}
	}
	for _, name := range []string{"claude", "codex", "gemini", "github-copilot"} {
		compiler := pm.compilers[name]
		if compiler == nil {
			t.Fatalf("missing compiler %s", name)
		}
		if _, ok := compiler.(MCPCompiler); !ok {
			t.Errorf("%s does not implement MCPCompiler", name)
		}
	}
}

func TestSetHindsightMCPCopies(t *testing.T) {
	pm := NewProviderManager(&ProvidersConfig{})
	servers := testHindsightServers(t)
	pm.SetHindsightMCP(servers)
	servers[0].URL = "changed"
	if pm.hindsightMCP[0].URL == "changed" {
		t.Fatal("manager kept the caller slice")
	}
	pm.SetHindsightMCP(nil)
	if pm.hindsightMCP != nil {
		t.Fatal("nil servers were stored")
	}
}

type mcpErrCompiler struct {
	mockCompiler
}

func (c *mcpErrCompiler) CompileMCP([]MCPServer, *Provider) (map[string]string, error) {
	return nil, errors.New("mcp broke")
}

func testHindsightServers(t *testing.T) []MCPServer {
	t.Helper()
	servers, err := HindsightServers(hindsight.Settings{
		Enabled: true,
		APIURL:  "http://127.0.0.1:8888",
		Banks:   hindsight.BanksConfig{Project: "demo-app", User: "ada"},
	}, memoryhindsight.Identity{})
	if err != nil {
		t.Fatal(err)
	}
	return servers
}

func mustDisabledServers(t *testing.T) []MCPServer {
	t.Helper()
	servers, err := HindsightServers(hindsight.Settings{
		Banks: hindsight.BanksConfig{Project: "demo-app", User: "ada"},
	}, memoryhindsight.Identity{})
	if err != nil || len(servers) != 0 {
		t.Fatalf("disabled servers = %#v err=%v", servers, err)
	}
	return servers
}

func generatedJSON() string {
	files, err := NewClaudeCompiler().CompileMCP([]MCPServer{
		{Name: hindsightProjectServer, Bank: "demo-app", URL: "http://127.0.0.1:8888/mcp/demo-app/"},
		{Name: hindsightUserServer, Bank: "ada", URL: "http://127.0.0.1:8888/mcp/ada/"},
	}, &Provider{})
	if err != nil {
		panic(err)
	}
	return files[".mcp.json"]
}

func testMCPProviders() map[string]*Provider {
	cfg := testMCPProviderConfig()
	out := map[string]*Provider{}
	for name, provider := range cfg.Providers {
		switch name {
		case "github-copilot":
			out["copilot"] = provider
		default:
			out[name] = provider
		}
	}
	return out
}

func testMCPProviderConfig() *ProvidersConfig {
	feature := func(path string) *FeatureConfig {
		return &FeatureConfig{Enabled: true, Path: path}
	}
	return &ProvidersConfig{Providers: map[string]*Provider{
		"codex": {
			Name: "Codex", Workspace: "~/.codex", Binary: "codex",
			Features: map[string]*FeatureConfig{
				"rules":       feature("AGENTS.md"),
				"skills":      feature(".agents/skills"),
				"workflows":   feature(".agents/skills"),
				"permissions": feature(".codex/rules/aicockpit.rules"),
				"agents":      feature(".codex/agents"),
				"mcp":         feature(".codex/config.toml"),
			},
		},
		"claude": {
			Name: "Claude Code", Workspace: "~/.claude", Binary: "claude",
			Features: map[string]*FeatureConfig{"mcp": feature(".mcp.json")},
		},
		"gemini": {
			Name: "Gemini CLI", Workspace: "~/.gemini", Binary: "gemini",
			Features: map[string]*FeatureConfig{"mcp": feature(".gemini/settings.json")},
		},
		"github-copilot": {
			Name: "GitHub Copilot CLI", Workspace: "~/.copilot", Binary: "copilot",
			Features: map[string]*FeatureConfig{
				"rules":     feature("~/.copilot/copilot-instructions.md"),
				"skills":    feature("~/.copilot/skills"),
				"workflows": feature("~/.copilot/skills"),
				"agents":    feature("~/.copilot/agents"),
				"mcp":       feature("~/.copilot/mcp-config.json"),
			},
		},
		"antigravity": {
			Name: "Antigravity", Workspace: ".gemini",
			Features: map[string]*FeatureConfig{
				"rules":  feature(".gemini/GEMINI.md"),
				"skills": feature(".gemini/skills"),
			},
		},
		"devin": {
			Name: "Devin", Workspace: ".devin",
			Features: map[string]*FeatureConfig{
				"rules":  feature(".devin/AGENTS.md"),
				"skills": feature(".devin/skills"),
			},
		},
		"goose": {
			Name: "Goose", Workspace: ".goose",
			Features: map[string]*FeatureConfig{
				"rules":  feature(".goosehints"),
				"skills": feature(".goose/skills"),
			},
		},
	}}
}

func deployHindsightFixture(t *testing.T, providerName string, servers []MCPServer) map[string]string {
	t.Helper()
	root, project, home := hindsightRoots(t)
	cockpit := filepath.Join(root, "cockpit")
	writeHindsightCanonical(t, cockpit)
	t.Setenv("HOME", home)
	pm := NewProviderManager(testMCPProviderConfig())
	pm.SetHindsightMCP(servers)
	if err := pm.Deploy(providerName, cockpit, project); err != nil {
		t.Fatal(err)
	}
	return snapshotRoots(t, map[string]string{"project": project, "home": home})
}

func hindsightRoots(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	project := filepath.Join(root, "project")
	home := filepath.Join(root, "home")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, project, home
}

func writeHindsightCanonical(t *testing.T, dir string) {
	t.Helper()
	files := map[string]string{
		"COCKPIT.md":               "Project context.\n",
		"rules/core-gold-rules.md": "Use tests.\n",
		"rules/style.md":           "Prefer small diffs.\n",
		"skills/review/SKILL.md":   "---\nname: review\ndescription: Review changes\n---\nRead the diff.\n",
	}
	for rel, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func snapshotRoots(t *testing.T, roots map[string]string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for prefix, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			out[prefix+"/"+filepath.ToSlash(rel)] = string(data)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func sameFiles(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for path, content := range left {
		if right[path] != content {
			return false
		}
	}
	return true
}

func fileKeys(files map[string]string) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertGoldenFile(t *testing.T, rel, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		path := filepath.Join(rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(rel)
	if err != nil {
		t.Fatalf("read golden %s: %v", rel, err)
	}
	if string(data) != got {
		t.Fatalf("golden %s mismatch\n got:\n%s\nwant:\n%s", rel, got, data)
	}
}

func assertGoldenTree(t *testing.T, rel string, got map[string]string) {
	t.Helper()
	dir := filepath.Join("testdata", "hindsight", rel)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
		if len(got) == 0 {
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".empty"), nil, 0o644); err != nil {
				t.Fatal(err)
			}
			return
		}
		for path, content := range got {
			full := filepath.Join(dir, filepath.FromSlash(path))
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	want := readGoldenTree(t, dir)
	if !sameFiles(got, want) {
		t.Fatalf("golden tree %s mismatch\n got: %v\nwant: %v", rel, fileKeys(got), fileKeys(want))
	}
}

func readGoldenTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("missing golden %s: %v", dir, err)
	}
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if d.Name() == ".empty" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
