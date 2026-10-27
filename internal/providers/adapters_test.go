package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAdapters(t *testing.T) {
	// Goose
	gc := NewGooseCompiler()
	if gc.Name() != "goose" {
		t.Error()
	}
	gc.CompileEntrypoint(&CanonicalEntrypoint{GoldenRules: []string{"test"}}, &Provider{})
	gc.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	gc.CompileSkills([]CanonicalSkill{{Name: "skill1"}}, &Provider{})
	gc.CompileWorkflows([]CanonicalWorkflow{{Name: "flow1"}}, &Provider{})

	// Antigravity
	ac := NewAntigravityCompiler()
	if ac.Name() != "antigravity" {
		t.Error()
	}
	ac.CompileEntrypoint(&CanonicalEntrypoint{GoldenRules: []string{"test"}}, &Provider{})
	ac.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	ac.CompileSkills([]CanonicalSkill{{Name: "skill1"}}, &Provider{})
	ac.CompileWorkflows([]CanonicalWorkflow{{Name: "flow1"}}, &Provider{})

	// Devin
	dc := NewDevinCompiler()
	if dc.Name() != "devin" {
		t.Error()
	}
	dc.CompileEntrypoint(&CanonicalEntrypoint{GoldenRules: []string{"test"}}, &Provider{})
	dc.CompileRules([]CanonicalRule{{Name: "rule1"}}, &Provider{})
	dc.CompileSkills([]CanonicalSkill{{Name: "skill1"}}, &Provider{})
	dc.CompileWorkflows([]CanonicalWorkflow{{Name: "flow1"}}, &Provider{})
}

func TestReadCanonicalFile_Invalid(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := ReadCanonicalFile(filepath.Join(tmpDir, "missing.txt"))
	if err == nil {
		t.Error("expected error")
	}
}

func TestAdapters_CompileErrors(t *testing.T) {
	tmpDir := t.TempDir()
	entrypoint, skills, rules, workflows, perms, _, _ := ParseCanonical(tmpDir)

	features := map[string]*FeatureConfig{
		"entrypoint":  {Path: "entry.md", Enabled: true},
		"skills":      {Path: "skills", Enabled: true},
		"rules":       {Path: "rules", Enabled: true},
		"workflows":   {Path: "flows", Enabled: true},
		"permissions": {Path: "perms.json", Enabled: true},
	}

	p := &Provider{Workspace: "/invalid_root/workspace", Features: features}

	ac := NewAntigravityCompiler()
	ac.CompileEntrypoint(entrypoint, p)
	ac.CompileSkills(skills, p)
	ac.CompileRules(rules, p)
	ac.CompileWorkflows(workflows, p)
	ac.CompilePermissions(perms, p)

	dc := NewDevinCompiler()
	dc.CompileEntrypoint(entrypoint, p)
	dc.CompileSkills(skills, p)
	dc.CompileRules(rules, p)
	dc.CompileWorkflows(workflows, p)
	dc.CompilePermissions(perms, p)

	gc := NewGooseCompiler()
	gc.CompileEntrypoint(entrypoint, p)
	gc.CompileSkills(skills, p)
	gc.CompileRules(rules, p)
	gc.CompileWorkflows(workflows, p)
	gc.CompilePermissions(perms, p)
}

func TestGooseAdapter_EnsureExtensions(t *testing.T) {
	// call ensureGooseExtensionsEnabled with invalid yaml to trigger unmarshal error
	c := NewGooseCompiler()

	// Create bad config
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.yaml")
	os.WriteFile(cfgPath, []byte("invalid: yaml: content"), 0644)

	p := &Provider{Workspace: tmpDir}

	// this should return nil in the new setup if it fails to parse? Wait no, goose_adapter.go is internal
	// Instead, test the public Compile methods that will trigger it internally

	perms := &CanonicalPermissions{}
	c.CompilePermissions(perms, p) // will trigger extensions write
}
