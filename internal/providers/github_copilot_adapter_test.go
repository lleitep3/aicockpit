package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitHubCopilotCompiler_Name(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	if got := compiler.Name(); got != githubCopilotProviderName {
		t.Fatalf("Name() = %q, want %q", got, githubCopilotProviderName)
	}
}

func TestGitHubCopilotCompiler_CompileEntrypointAndRules(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{"rules": {Enabled: true, Path: "~/.copilot/AGENTS.md"}}}

	files, err := compiler.CompileEntrypoint(&CanonicalEntrypoint{ProjectContext: "ctx", GoldenRules: []string{"rule 1", "", "rule 2"}}, provider)
	if err != nil {
		t.Fatalf("CompileEntrypoint() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("CompileEntrypoint() returned %d files, want 1", len(files))
	}
	got := files["~/.copilot/AGENTS.md"]
	if !strings.Contains(got, "ctx") || !strings.Contains(got, "rule 1") || !strings.Contains(got, "rule 2") {
		t.Fatalf("CompileEntrypoint() content = %q, want project context and rules", got)
	}

	ruleFiles, err := compiler.CompileRules([]CanonicalRule{{Name: "b", Content: "beta"}, {Name: "a", Content: "alpha"}}, provider)
	if err != nil {
		t.Fatalf("CompileRules() error = %v", err)
	}
	gotRule := ruleFiles["~/.copilot/AGENTS.md"]
	if !strings.Contains(gotRule, "alpha") || !strings.Contains(gotRule, "beta") {
		t.Fatalf("CompileRules() content = %q", gotRule)
	}
}

func TestGitHubCopilotCompiler_CompileSkillsWorkflowsAgents(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{
		"skills":    {Enabled: true, Path: "~/.copilot/skills"},
		"workflows": {Enabled: true, Path: "~/.copilot/workflows"},
		"agents":    {Enabled: true, Path: "~/.copilot/agents"},
	}}

	skillFiles, err := compiler.CompileSkills([]CanonicalSkill{{Name: "deploy", Description: "Deploy", Content: "Ship it."}}, provider)
	if err != nil {
		t.Fatalf("CompileSkills() error = %v", err)
	}
	if _, ok := skillFiles[filepath.Join("~/.copilot/skills", "deploy", "SKILL.md")]; !ok {
		t.Fatalf("missing skill file: %#v", skillFiles)
	}

	workflowFiles, err := compiler.CompileWorkflows([]CanonicalWorkflow{{Name: "release", Description: "Release", Steps: []string{"build", "ship"}}}, provider)
	if err != nil {
		t.Fatalf("CompileWorkflows() error = %v", err)
	}
	if _, ok := workflowFiles[filepath.Join("~/.copilot/workflows", "release.md")]; !ok {
		t.Fatalf("missing workflow file: %#v", workflowFiles)
	}

	agentFiles, err := compiler.CompileAgents([]CanonicalAgent{{Name: "builder", Description: "Builds things", Content: "Be helpful."}}, provider)
	if err != nil {
		t.Fatalf("CompileAgents() error = %v", err)
	}
	if _, ok := agentFiles[filepath.Join("~/.copilot/agents", "builder", "AGENT.md")]; !ok {
		t.Fatalf("missing agent file: %#v", agentFiles)
	}
}

func TestGitHubCopilotCompiler_CompilePermissions(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{"permissions": {Enabled: true, Path: "~/.copilot/permissions.json"}}}
	files, err := compiler.CompilePermissions(&CanonicalPermissions{AllowedCommands: []string{"git status"}, DeniedCommands: []string{"rm -rf /"}, Ask: []string{"Write(**)"}}, provider)
	if err != nil {
		t.Fatalf("CompilePermissions() error = %v", err)
	}
	got := files["~/.copilot/permissions.json"]
	if !strings.Contains(got, "git status") || !strings.Contains(got, "rm -rf /") || !strings.Contains(got, "Write(**)") {
		t.Fatalf("CompilePermissions() output = %q", got)
	}
}

func TestGitHubCopilotCompiler_DeployToGlobalHome(t *testing.T) {
	tempHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	if err := os.Setenv("HOME", tempHome); err != nil {
		t.Fatalf("Setenv(HOME) failed: %v", err)
	}
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	cockpitHome := filepath.Join(tempHome, ".cockpit")
	if err := os.MkdirAll(filepath.Join(cockpitHome, "skills", "deploy"), 0o755); err != nil {
		t.Fatalf("mkdir skill dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cockpitHome, "rules"), 0o755); err != nil {
		t.Fatalf("mkdir rules dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cockpitHome, "workflows"), 0o755); err != nil {
		t.Fatalf("mkdir workflows dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(cockpitHome, "agents", "builder"), 0o755); err != nil {
		t.Fatalf("mkdir agents dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "COCKPIT.md"), []byte("Project context"), 0o644); err != nil {
		t.Fatalf("write COCKPIT.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "skills", "deploy", "SKILL.md"), []byte("---\nname: \"deploy\"\ndescription: \"Deploy\"\n---\nDeploy the app.\n"), 0o644); err != nil {
		t.Fatalf("write skill: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "rules", "team.md"), []byte("Keep code clean."), 0o644); err != nil {
		t.Fatalf("write rules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "workflows", "release.yaml"), []byte("name: release\ndescription: Release\nsteps:\n  - build\n  - ship\n"), 0o644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "agents", "builder", "AGENT.md"), []byte("---\nname: \"builder\"\ndescription: \"Builds with care\"\n---\nShip safely.\n"), 0o644); err != nil {
		t.Fatalf("write agent: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cockpitHome, "permissions.yaml"), []byte("allowed_commands:\n  - git status\n"), 0o644); err != nil {
		t.Fatalf("write permissions: %v", err)
	}

	config := &ProvidersConfig{Providers: map[string]*Provider{
		githubCopilotProviderName: {
			Name:      githubCopilotProviderName,
			Enabled:   true,
			Workspace: "~/.copilot",
			Features: map[string]*FeatureConfig{
				"rules":       {Enabled: true, Path: "~/.copilot/AGENTS.md"},
				"skills":      {Enabled: true, Path: "~/.copilot/skills"},
				"workflows":   {Enabled: true, Path: "~/.copilot/workflows"},
				"permissions": {Enabled: true, Path: "~/.copilot/permissions.json"},
				"agents":      {Enabled: true, Path: "~/.copilot/agents"},
			},
		},
	}}

	pm := NewProviderManager(config)
	projectDir := filepath.Join(tempHome, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}

	if err := pm.Deploy(githubCopilotProviderName, cockpitHome, projectDir); err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	for _, rel := range []string{
		".copilot/AGENTS.md",
		".copilot/skills/deploy/SKILL.md",
		".copilot/workflows/release.md",
		".copilot/agents/builder/AGENT.md",
		".copilot/permissions.json",
	} {
		path := filepath.Join(tempHome, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected deployed file %s to exist: %v", rel, err)
		}
	}
}
