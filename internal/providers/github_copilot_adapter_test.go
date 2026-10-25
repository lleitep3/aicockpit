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
	provider := &Provider{Features: map[string]*FeatureConfig{"rules": {Enabled: true, Path: "~/.copilot/copilot-instructions.md"}}}

	files, err := compiler.CompileEntrypoint(&CanonicalEntrypoint{ProjectContext: "ctx", GoldenRules: []string{"rule 1", "", "rule 2"}}, provider)
	if err != nil {
		t.Fatalf("CompileEntrypoint() error = %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("CompileEntrypoint() returned %d files, want 1", len(files))
	}
	got := files["~/.copilot/copilot-instructions.md"]
	if !strings.Contains(got, "ctx") || !strings.Contains(got, "rule 1") || !strings.Contains(got, "rule 2") {
		t.Fatalf("CompileEntrypoint() content = %q, want project context and rules", got)
	}

	ruleFiles, err := compiler.CompileRules([]CanonicalRule{{Name: "b", Content: "beta"}, {Name: "a", Content: "alpha"}}, provider)
	if err != nil {
		t.Fatalf("CompileRules() error = %v", err)
	}
	gotRule := ruleFiles["~/.copilot/copilot-instructions.md"]
	if !strings.Contains(gotRule, "alpha") || !strings.Contains(gotRule, "beta") {
		t.Fatalf("CompileRules() content = %q", gotRule)
	}
}

func TestGitHubCopilotCompiler_CompileSkillsWorkflowsAgents(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{
		"skills":    {Enabled: true, Path: "~/.copilot/skills"},
		"workflows": {Enabled: true, Path: "~/.copilot/skills"},
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
	workflowPath := filepath.Join("~/.copilot/skills", "cockpit-workflow-release", "SKILL.md")
	workflowContent, ok := workflowFiles[workflowPath]
	if !ok {
		t.Fatalf("missing workflow file: %#v", workflowFiles)
	}
	for _, expected := range []string{"name: \"cockpit-workflow-release\"", "1. build", "2. ship"} {
		if !strings.Contains(workflowContent, expected) {
			t.Errorf("workflow skill content missing %q: %q", expected, workflowContent)
		}
	}

	agentFiles, err := compiler.CompileAgents([]CanonicalAgent{{Name: "builder", Description: "Builds things", AllowedTools: []string{"view", "bash", "view"}, Content: "Be helpful."}}, provider)
	if err != nil {
		t.Fatalf("CompileAgents() error = %v", err)
	}
	agentPath := filepath.Join("~/.copilot/agents", "builder.agent.md")
	agentContent, ok := agentFiles[agentPath]
	if !ok {
		t.Fatalf("missing agent file: %#v", agentFiles)
	}
	for _, expected := range []string{"tools:", `- "view"`, `- "bash"`, "include-custom-instructions: true"} {
		if !strings.Contains(agentContent, expected) {
			t.Errorf("agent content missing %q: %q", expected, agentContent)
		}
	}
}

func TestGitHubCopilotCompiler_CompilePermissions(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{"permissions": {Enabled: true, Path: "~/.copilot/unsupported-policy"}}}
	_, err := compiler.CompilePermissions(&CanonicalPermissions{AllowedCommands: []string{"git status"}, DeniedCommands: []string{"rm -rf /"}, Ask: []string{"Write(**)"}}, provider)
	if err == nil || !strings.Contains(err.Error(), "cannot compile canonical permissions") {
		t.Fatalf("CompilePermissions() error = %v, want an unsupported-policy error", err)
	}

	disabledProvider := &Provider{Features: map[string]*FeatureConfig{"permissions": {Enabled: false}}}
	files, err := compiler.CompilePermissions(&CanonicalPermissions{AllowedCommands: []string{"git status"}}, disabledProvider)
	if err != nil {
		t.Fatalf("CompilePermissions() with disabled feature error = %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("CompilePermissions() with disabled feature returned files: %#v", files)
	}
}

func TestGitHubCopilotCompiler_CompileAgentsRejectsCanonicalPermissions(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{"agents": {Enabled: true, Path: "~/.copilot/agents"}}}
	_, err := compiler.CompileAgents([]CanonicalAgent{{Name: "restricted", Description: "Restricted agent", Permissions: map[string]interface{}{"deny": []string{"shell"}}}}, provider)
	if err == nil || !strings.Contains(err.Error(), "canonical permissions that cannot be represented") {
		t.Fatalf("CompileAgents() error = %v, want an unsupported-permissions error", err)
	}
}

func TestGitHubCopilotCompiler_CompileAgentsRejectsMaxNesting(t *testing.T) {
	compiler := NewGitHubCopilotCompiler()
	provider := &Provider{Features: map[string]*FeatureConfig{"agents": {Enabled: true, Path: "~/.copilot/agents"}}}
	_, err := compiler.CompileAgents([]CanonicalAgent{{Name: "nested", Description: "Nested agent", MaxNesting: 2}}, provider)
	if err == nil || !strings.Contains(err.Error(), "max_nesting value that cannot be represented") {
		t.Fatalf("CompileAgents() error = %v, want an unsupported max_nesting error", err)
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
	if err := os.MkdirAll(filepath.Join(cockpitHome, "skills", "deploy", "scripts", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested script dir: %v", err)
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
	deployScript := filepath.Join(cockpitHome, "skills", "deploy", "scripts", "nested", "run.sh")
	if err := os.WriteFile(deployScript, []byte("#!/bin/sh\necho deployed\n"), 0o755); err != nil {
		t.Fatalf("write skill script: %v", err)
	}
	if err := os.Chmod(deployScript, 0o755); err != nil {
		t.Fatalf("chmod skill script: %v", err)
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
				"rules":       {Enabled: true, Path: "~/.copilot/copilot-instructions.md"},
				"skills":      {Enabled: true, Path: "~/.copilot/skills"},
				"workflows":   {Enabled: true, Path: "~/.copilot/skills"},
				"permissions": {Enabled: false},
				"agents":      {Enabled: true, Path: "~/.copilot/agents"},
			},
		},
	}}

	pm := NewProviderManager(config)
	projectDir := filepath.Join(tempHome, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir project dir: %v", err)
	}
	globalInstructionsPath := filepath.Join(tempHome, ".copilot", "copilot-instructions.md")
	if err := os.MkdirAll(filepath.Dir(globalInstructionsPath), 0o755); err != nil {
		t.Fatalf("mkdir global Copilot directory: %v", err)
	}
	if err := os.WriteFile(globalInstructionsPath, []byte("Keep my manual Copilot guidance.\n"), 0o644); err != nil {
		t.Fatalf("write manual Copilot instructions: %v", err)
	}

	if err := pm.Deploy(githubCopilotProviderName, cockpitHome, projectDir); err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}

	for _, rel := range []string{
		".copilot/copilot-instructions.md",
		".copilot/skills/deploy/SKILL.md",
		".copilot/skills/deploy/scripts/nested/run.sh",
		".copilot/skills/cockpit-workflow-release/SKILL.md",
		".copilot/agents/builder.agent.md",
	} {
		path := filepath.Join(tempHome, rel)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected deployed file %s to exist: %v", rel, err)
		}
	}
	deployedScriptInfo, err := os.Stat(filepath.Join(tempHome, ".copilot", "skills", "deploy", "scripts", "nested", "run.sh"))
	if err != nil {
		t.Fatalf("stat deployed skill script: %v", err)
	}
	if got := deployedScriptInfo.Mode().Perm(); got != 0o755 {
		t.Errorf("deployed skill script mode = %o, want 755", got)
	}

	instructions, err := os.ReadFile(globalInstructionsPath)
	if err != nil {
		t.Fatalf("read deployed Copilot instructions: %v", err)
	}
	for _, expected := range []string{"Keep my manual Copilot guidance.", "Project context", "<!-- cockpit:global -->", "<!-- /cockpit:global -->"} {
		if !strings.Contains(string(instructions), expected) {
			t.Errorf("deployed instructions missing %q: %s", expected, instructions)
		}
	}
	if strings.Count(string(instructions), "<!-- cockpit:global -->") != 1 {
		t.Fatalf("expected one managed instruction block, got: %s", instructions)
	}

	if err := pm.Deploy(githubCopilotProviderName, cockpitHome, projectDir); err != nil {
		t.Fatalf("second Deploy() error = %v", err)
	}
	instructions, err = os.ReadFile(globalInstructionsPath)
	if err != nil {
		t.Fatalf("read Copilot instructions after second deploy: %v", err)
	}
	if strings.Count(string(instructions), "<!-- cockpit:global -->") != 1 {
		t.Fatalf("second deploy duplicated the managed instruction block: %s", instructions)
	}
	if _, err := os.Stat(filepath.Join(tempHome, ".copilot", "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("unexpected legacy AGENTS.md output: err = %v", err)
	}
}
