package providers

import (
	"strings"
	"testing"
)

func newTestManifest() *AdapterManifest {
	return &AdapterManifest{
		Name:        "testprovider",
		Description: "Test provider",
		Version:     "1.0.0",
		Artifacts: map[string]ArtifactConfig{
			"entrypoint": {
				Enabled:  true,
				Path:     "entry.md",
				Template: "# Context\n{{ .ProjectContext }}\n## Rules\n{{ range .GoldenRules }}- {{ . }}\n{{ end }}",
			},
			"skills": {
				Enabled:  true,
				Path:     "skills.md",
				Template: "{{ range . }}{{ .Name }}: {{ .Description }}\n{{ end }}",
			},
			"rules": {
				Enabled:  true,
				Path:     "rules.md",
				Template: "{{ range . }}{{ .Name }}\n{{ .Content }}\n{{ end }}",
			},
			"workflows": {
				Enabled:  true,
				Path:     "workflows.md",
				Template: "{{ range . }}{{ .Name }}: {{ .Description }}\n{{ end }}",
			},
			"permissions": {
				Enabled:  true,
				Path:     "permissions.md",
				Template: "allowed: {{ range .AllowedCommands }}{{ . }} {{ end }}",
			},
			"agents": {
				Enabled:  true,
				Path:     "agents.md",
				Template: "{{ range . }}{{ .Name }}\n{{ end }}",
			},
		},
	}
}

func TestGenericYAMLCompiler_Name(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	if c.Name() != "testprovider" {
		t.Errorf("expected name testprovider, got %s", c.Name())
	}
}

func TestGenericYAMLCompiler_CompileRules_Enabled(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	rules := []CanonicalRule{
		{Name: "rule1", Content: "content1"},
		{Name: "rule2", Content: "content2"},
	}

	files, err := c.CompileRules(rules, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := files["rules.md"]
	if !ok {
		t.Fatalf("expected rules.md in output")
	}
	if !strings.Contains(content, "rule1") || !strings.Contains(content, "content2") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestGenericYAMLCompiler_CompileRules_Disabled(t *testing.T) {
	manifest := newTestManifest()
	cfg := manifest.Artifacts["rules"]
	cfg.Enabled = false
	manifest.Artifacts["rules"] = cfg

	c := NewGenericYAMLCompiler(manifest)
	files, err := c.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty map for disabled artifact, got %v", files)
	}
}

func TestGenericYAMLCompiler_CompileSkills_Empty(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	files, err := c.CompileSkills([]CanonicalSkill{}, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty map for empty skills, got %v", files)
	}
}

func TestGenericYAMLCompiler_CompileSkills_NonEmpty(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	skills := []CanonicalSkill{
		{Name: "skill1", Description: "desc1", Content: "content1"},
	}

	files, err := c.CompileSkills(skills, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := files["skills.md"]
	if !ok {
		t.Fatalf("expected skills.md in output")
	}
	if !strings.Contains(content, "skill1: desc1") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestGenericYAMLCompiler_CompileRules_Empty(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	files, err := c.CompileRules([]CanonicalRule{}, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty map for empty rules, got %v", files)
	}
}

func TestGenericYAMLCompiler_CompileEntrypoint(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	entrypoint := &CanonicalEntrypoint{
		ProjectContext: "project context",
		GoldenRules:    []string{"rule A", "rule B"},
	}

	files, err := c.CompileEntrypoint(entrypoint, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := files["entry.md"]
	if !ok {
		t.Fatalf("expected entry.md in output")
	}
	if !strings.Contains(content, "project context") {
		t.Errorf("expected project context in content: %s", content)
	}
	if !strings.Contains(content, "rule A") {
		t.Errorf("expected golden rule in content: %s", content)
	}
}

func TestGenericYAMLCompiler_CompileWorkflows(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	workflows := []CanonicalWorkflow{
		{Name: "wf1", Description: "desc1", Steps: []string{"step1"}},
	}

	files, err := c.CompileWorkflows(workflows, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := files["workflows.md"]
	if !ok {
		t.Fatalf("expected workflows.md in output")
	}
	if !strings.Contains(content, "wf1: desc1") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestGenericYAMLCompiler_CompilePermissions(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	perms := &CanonicalPermissions{
		AllowedCommands: []string{"git", "go"},
	}

	files, err := c.CompilePermissions(perms, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := files["permissions.md"]
	if !ok {
		t.Fatalf("expected permissions.md in output")
	}
	if !strings.Contains(content, "git") || !strings.Contains(content, "go") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestGenericYAMLCompiler_CompileAgents(t *testing.T) {
	c := NewGenericYAMLCompiler(newTestManifest())
	agents := []CanonicalAgent{
		{Name: "agent1", Description: "desc"},
	}

	files, err := c.CompileAgents(agents, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, ok := files["agents.md"]
	if !ok {
		t.Fatalf("expected agents.md in output")
	}
	if !strings.Contains(content, "agent1") {
		t.Errorf("unexpected content: %s", content)
	}
}

func TestGenericYAMLCompiler_CompileRules_InvalidTemplate(t *testing.T) {
	manifest := newTestManifest()
	cfg := manifest.Artifacts["rules"]
	cfg.Template = "{{ .Name"
	manifest.Artifacts["rules"] = cfg

	c := NewGenericYAMLCompiler(manifest)
	_, err := c.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	if err == nil {
		t.Fatal("expected error for invalid template")
	}
	if !strings.Contains(err.Error(), "failed to parse template") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenericYAMLCompiler_CompileRules_ExecutionError(t *testing.T) {
	manifest := newTestManifest()
	cfg := manifest.Artifacts["rules"]
	cfg.Template = "{{ .MissingField }}"
	manifest.Artifacts["rules"] = cfg

	c := NewGenericYAMLCompiler(manifest)
	_, err := c.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	if err == nil {
		t.Fatal("expected error for template execution failure")
	}
	if !strings.Contains(err.Error(), "failed to execute template") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestGenericYAMLCompiler_MissingArtifact(t *testing.T) {
	manifest := &AdapterManifest{
		Name:      "empty",
		Artifacts: map[string]ArtifactConfig{},
	}
	c := NewGenericYAMLCompiler(manifest)

	files, err := c.CompileEntrypoint(&CanonicalEntrypoint{}, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty map for missing artifact, got %v", files)
	}
}

func TestGenericYAMLCompiler_EnabledNoPath(t *testing.T) {
	manifest := &AdapterManifest{
		Name: "test",
		Artifacts: map[string]ArtifactConfig{
			"rules": {Enabled: true, Path: "", Template: "{{ .Name }}"},
		},
	}
	c := NewGenericYAMLCompiler(manifest)

	files, err := c.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty map when path is empty, got %v", files)
	}
}
