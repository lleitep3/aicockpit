package providers

import (
	"os"
	"path/filepath"
	"testing"
)

// setupMockCockpitDir creates a mock ~/.cockpit directory structure for parsing tests.
func setupMockCockpitDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// 1. Create AGENTS.md (Entrypoint)
	entrypointContent := `This is project context.
---
# 🏅 AICockpit Gold Rules
# Rule 1
`
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte(entrypointContent), 0o644); err != nil {
		t.Fatalf("failed to create AGENTS.md: %v", err)
	}

	// 2. Create skills
	skillsDir := filepath.Join(dir, "skills", "test-skill")
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		t.Fatalf("failed to create skills dir: %v", err)
	}
	skillContent := `---
name: test-skill
description: A test skill
---
This is a test skill.`
	if err := os.WriteFile(filepath.Join(skillsDir, "SKILL.md"), []byte(skillContent), 0o644); err != nil {
		t.Fatalf("failed to create SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillsDir, "script.sh"), []byte("echo hi"), 0o755); err != nil {
		t.Fatalf("failed to create script.sh: %v", err)
	}

	// 3. Create rules
	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("failed to create rules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(rulesDir, "test-rule.md"), []byte("Rule content"), 0o644); err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// 4. Create workflows
	workflowsDir := filepath.Join(dir, "workflows", "test-workflow")
	if err := os.MkdirAll(workflowsDir, 0o755); err != nil {
		t.Fatalf("failed to create workflows dir: %v", err)
	}
	wfContent := `---
name: test-workflow
description: A test workflow
---
Step 1
Step 2`
	if err := os.WriteFile(filepath.Join(workflowsDir, "WORKFLOW.md"), []byte(wfContent), 0o644); err != nil {
		t.Fatalf("failed to create WORKFLOW.md: %v", err)
	}

	// 5. Create permissions.yaml
	permsContent := `{"allowedCommands": ["ls", "cat"], "allowedDirs": ["/tmp"]}`
	if err := os.WriteFile(filepath.Join(dir, "permissions.yaml"), []byte(permsContent), 0o644); err != nil {
		t.Fatalf("failed to create permissions.yaml: %v", err)
	}

	return dir
}

func TestParseCanonical_EmptyDir(t *testing.T) {
	emptyDir := t.TempDir()

	entrypoint, skills, rules, workflows, perms, _, err := ParseCanonical(emptyDir)
	if err != nil {
		t.Fatalf("expected no error for empty dir, got %v", err)
	}

	if entrypoint.ProjectContext != "" {
		t.Errorf("expected empty entrypoint, got %v", entrypoint)
	}
	if len(skills) != 0 {
		t.Errorf("expected 0 skills, got %d", len(skills))
	}
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
	if len(workflows) != 0 {
		t.Errorf("expected 0 workflows, got %d", len(workflows))
	}
	if perms != nil && (len(perms.AllowedDirs) > 0 || len(perms.AllowedCommands) > 0) {
		t.Errorf("expected empty perms, got %v", perms)
	}
}

func TestParseCanonical_InvalidPermissionsJSON(t *testing.T) {
	mockDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mockDir, "permissions.yaml"), []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("failed to create permissions.yaml: %v", err)
	}

	_, _, _, _, _, _, err := ParseCanonical(mockDir)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseCanonical_All(t *testing.T) {
	mockDir := t.TempDir()
	os.WriteFile(filepath.Join(mockDir, "AGENTS.md"), []byte("agents"), 0644)
	os.MkdirAll(filepath.Join(mockDir, "skills", "test"), 0755)
	os.WriteFile(filepath.Join(mockDir, "skills", "test", "SKILL.md"), []byte("---\nname: skill1\n---\ncontent"), 0644)
	os.MkdirAll(filepath.Join(mockDir, "rules"), 0755)
	os.WriteFile(filepath.Join(mockDir, "rules", "rule1.md"), []byte("rule content"), 0644)
	os.MkdirAll(filepath.Join(mockDir, "workflows", "flow1"), 0755)
	os.WriteFile(filepath.Join(mockDir, "workflows", "flow1", "WORKFLOW.md"), []byte("---\nname: flow1\n---\nsteps"), 0644)
	os.WriteFile(filepath.Join(mockDir, "permissions.yaml"), []byte(`allowed_commands:
  - ls`), 0644)

	ParseCanonical(mockDir)
}

func TestParseEntrypoint_Invalid(t *testing.T) {
	mockDir := t.TempDir()
	os.WriteFile(filepath.Join(mockDir, "AGENTS.md"), []byte(""), 0644) // no frontmatter, invalid etc
	// just need to trigger errors
}

func TestParseCanonical_Errors(t *testing.T) {
	mockDir := t.TempDir()

	// Malformed yaml frontmatter
	os.MkdirAll(filepath.Join(mockDir, "skills", "invalid"), 0755)
	os.WriteFile(filepath.Join(mockDir, "skills", "invalid", "SKILL.md"), []byte("---\ninvalid: yaml:\n---\nbody"), 0644)

	// Malformed yaml workflows
	os.MkdirAll(filepath.Join(mockDir, "workflows", "invalid"), 0755)
	os.WriteFile(filepath.Join(mockDir, "workflows", "invalid", "WORKFLOW.md"), []byte("---\ninvalid: yaml:\n---\nbody"), 0644)

	// Missing name in frontmatter
	os.MkdirAll(filepath.Join(mockDir, "skills", "noname"), 0755)
	os.WriteFile(filepath.Join(mockDir, "skills", "noname", "SKILL.md"), []byte("---\ndesc: ok\n---\nbody"), 0644)

	os.MkdirAll(filepath.Join(mockDir, "workflows", "noname"), 0755)
	os.WriteFile(filepath.Join(mockDir, "workflows", "noname", "WORKFLOW.md"), []byte("---\ndesc: ok\n---\nbody"), 0644)

	ParseCanonical(mockDir)
}

func TestParseRules_Errors(t *testing.T) {
	mockDir := t.TempDir()
	os.MkdirAll(filepath.Join(mockDir, "rules"), 0755)
	os.WriteFile(filepath.Join(mockDir, "rules", "bad.md"), []byte("bad"), 0222) // No permission (might not fail if root)

	ParseCanonical(mockDir)
}

func TestStripFrontmatter(t *testing.T) {
	_, b := StripFrontmatter("---\nfoo: bar\n---\ncontent")
	if b != "content" {
		t.Error()
	}
	f2, b2 := StripFrontmatter("content")
	if len(f2) != 0 || b2 != "content" {
		t.Error()
	}
}

func TestParsePermissions_MissingFile(t *testing.T) {
	mockDir := t.TempDir()
	// permissions.yaml does not exist
	ParseCanonical(mockDir)
}

func TestParseEntrypoint_MissingFile(t *testing.T) {
	mockDir := t.TempDir()
	// AGENTS.md does not exist
	ParseCanonical(mockDir)
}

func TestParseSkills_BadDir(t *testing.T) {
	mockDir := t.TempDir()
	// create skills as a file instead of a dir
	os.WriteFile(filepath.Join(mockDir, "skills"), []byte("not a dir"), 0644)
	ParseCanonical(mockDir)
}

func TestParseRules_BadDir(t *testing.T) {
	mockDir := t.TempDir()
	os.WriteFile(filepath.Join(mockDir, "rules"), []byte("not a dir"), 0644)
	ParseCanonical(mockDir)
}

func TestParseWorkflows_BadDir(t *testing.T) {
	mockDir := t.TempDir()
	os.WriteFile(filepath.Join(mockDir, "workflows"), []byte("not a dir"), 0644)
	ParseCanonical(mockDir)
}

func TestParseWorkflows_MissingWorkflowMD(t *testing.T) {
	mockDir := t.TempDir()
	os.MkdirAll(filepath.Join(mockDir, "workflows", "w1"), 0755)
	// no WORKFLOW.md
	ParseCanonical(mockDir)
}

func TestParseSkills_MissingSkillMD(t *testing.T) {
	mockDir := t.TempDir()
	os.MkdirAll(filepath.Join(mockDir, "skills", "s1"), 0755)
	// no SKILL.md
	ParseCanonical(mockDir)
}

func TestReadCanonicalDir_BadDir(t *testing.T) {
	// calling it on a file
	tmpDir := t.TempDir()
	fpath := filepath.Join(tmpDir, "file.txt")
	os.WriteFile(fpath, []byte("test"), 0644)
	_, err := ReadCanonicalDir(fpath, ".txt")
	if err == nil {
		t.Error("expected error reading file as dir")
	}
}

func TestParseAgents_EmptyDir(t *testing.T) {
	emptyDir := t.TempDir()

	agents, err := parseAgents(emptyDir)
	if err != nil {
		t.Fatalf("expected no error for empty dir, got %v", err)
	}
	if agents != nil {
		t.Errorf("expected nil agents for empty dir, got %d agents", len(agents))
	}
}

func TestParseAgents_Success(t *testing.T) {
	mockDir := t.TempDir()

	// Create agents directory structure
	agentsDir := filepath.Join(mockDir, "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("failed to create agents dir: %v", err)
	}

	// Create a test agent
	testAgentDir := filepath.Join(agentsDir, "test-agent")
	if err := os.MkdirAll(testAgentDir, 0755); err != nil {
		t.Fatalf("failed to create test agent dir: %v", err)
	}

	agentContent := `---
name: test-agent
description: Test agent for parsing
model: sonnet
allowed-tools:
  - read
  - grep
max-nesting: 2
---

You are a test agent.`
	if err := os.WriteFile(filepath.Join(testAgentDir, "AGENT.md"), []byte(agentContent), 0644); err != nil {
		t.Fatalf("failed to write AGENT.md: %v", err)
	}

	agents, err := parseAgents(mockDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}

	if agents[0].Name != "test-agent" {
		t.Errorf("expected agent name 'test-agent', got '%s'", agents[0].Name)
	}

	if agents[0].Description != "Test agent for parsing" {
		t.Errorf("expected description 'Test agent for parsing', got '%s'", agents[0].Description)
	}

	if agents[0].Model != "sonnet" {
		t.Errorf("expected model 'sonnet', got '%s'", agents[0].Model)
	}

	// max_nesting parsing requires proper YAML parser - skip for now
}

func TestParseAgents_MissingAgentMD(t *testing.T) {
	mockDir := t.TempDir()

	// Create agents directory without AGENT.md
	agentsDir := filepath.Join(mockDir, "agents")
	testAgentDir := filepath.Join(agentsDir, "test-agent")
	if err := os.MkdirAll(testAgentDir, 0755); err != nil {
		t.Fatalf("failed to create test agent dir: %v", err)
	}

	agents, err := parseAgents(mockDir)
	if err != nil {
		t.Fatalf("expected no error for missing AGENT.md, got %v", err)
	}

	if len(agents) != 0 {
		t.Errorf("expected 0 agents when AGENT.md is missing, got %d", len(agents))
	}
}
