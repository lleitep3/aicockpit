package providers

import (
	"strings"
	"testing"
)

func TestDevinCompiler(t *testing.T) {
	compiler := NewDevinCompiler()
	if compiler.Name() != "devin" {
		t.Errorf("expected devin, got %s", compiler.Name())
	}

	provider := &Provider{
		Features: map[string]*FeatureConfig{
			"gold_rules": {Enabled: true, Path: "global_rules.md"},
			"rules":      {Enabled: true, Path: "AGENTS.md"},
			"skills":     {Enabled: true, Path: "skills"},
			"workflows":  {Enabled: true, Path: "workflows"},
		},
	}

	// Test Entrypoint
	ep := &CanonicalEntrypoint{GoldenRules: []string{"# Rule 1"}}
	files, err := compiler.CompileEntrypoint(ep, provider)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["global_rules.md"], "Rule 1") {
		t.Errorf("Missing rule in gold rules: %s", files["global_rules.md"])
	}

	// Test Skills
	skills := []CanonicalSkill{{Name: "test", Description: "desc", Content: "content"}}
	files, err = compiler.CompileSkills(skills, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected skills files")
	}

	// Test Workflows
	wfs := []CanonicalWorkflow{{Name: "flow", Steps: []string{"step"}}}
	files, err = compiler.CompileWorkflows(wfs, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected workflows files")
	}

	// Test Rules
	rules := []CanonicalRule{{Name: "rule", Content: "content"}}
	files, err = compiler.CompileRules(rules, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected rules files")
	}
}

func TestAntigravityCompiler(t *testing.T) {
	compiler := NewAntigravityCompiler()
	if compiler.Name() != "antigravity" {
		t.Errorf("expected antigravity, got %s", compiler.Name())
	}

	provider := &Provider{
		Features: map[string]*FeatureConfig{
			"rules":     {Enabled: true, Path: "AGENTS.md"},
			"skills":    {Enabled: true, Path: "skills"},
			"workflows": {Enabled: true, Path: "workflows"},
		},
	}

	// Test Entrypoint
	ep := &CanonicalEntrypoint{ProjectContext: "ctx"}
	files, err := compiler.CompileEntrypoint(ep, provider)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["AGENTS.md"], "ctx") {
		t.Errorf("Missing context")
	}

	// Test Skills
	skills := []CanonicalSkill{{Name: "test", Description: "desc", Content: "content"}}
	files, err = compiler.CompileSkills(skills, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected skills files")
	}

	// Test Workflows
	wfs := []CanonicalWorkflow{{Name: "flow", Steps: []string{"step"}}}
	files, err = compiler.CompileWorkflows(wfs, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected workflows files")
	}

	// Test Rules
	rules := []CanonicalRule{{Name: "rule", Content: "content"}}
	files, err = compiler.CompileRules(rules, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected rules files")
	}
}

func TestGooseCompiler(t *testing.T) {
	compiler := NewGooseCompiler()
	if compiler.Name() != "goose" {
		t.Errorf("expected goose, got %s", compiler.Name())
	}

	provider := &Provider{
		Features: map[string]*FeatureConfig{
			"rules": {Enabled: true, Path: ".goosehints"},
		},
	}

	// Test Entrypoint
	ep := &CanonicalEntrypoint{ProjectContext: "ctx"}
	files, err := compiler.CompileEntrypoint(ep, provider)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files[".goosehints"], "ctx") {
		t.Errorf("Missing context")
	}

	// Test Skills
	skills := []CanonicalSkill{{Name: "test", Description: "desc", Content: "content"}}
	files, err = compiler.CompileSkills(skills, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 { // Goose doesn't support skills this way
		t.Errorf("expected 0 skills files for goose, got %d", len(files))
	}

	// Test Rules
	rules := []CanonicalRule{{Name: "rule", Content: "content"}}
	files, err = compiler.CompileRules(rules, provider)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Error("expected rules files")
	}
}
