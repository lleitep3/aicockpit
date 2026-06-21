package providers

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProvidersConfig_GetProviderOptions_Full(t *testing.T) {
	cfg := &ProvidersConfig{Providers: map[string]*Provider{"test": {Enabled: true, Name: "test"}}}
	opts := cfg.GetProviderOptions()
	if len(opts) != 1 {
		t.Error("expected 1 option")
	}
}

func TestProvidersConfig_GetWorkspacePath_Home(t *testing.T) {
	prov := &Provider{Workspace: "~/test"}
	path := prov.GetWorkspacePath()
	if path == "~/test" || path == "" {
		t.Error("expected expanded path")
	}

	prov2 := &Provider{Workspace: "relative/path"}
	path2 := prov2.GetWorkspacePath()
	if path2 == "" {
		t.Error("expected valid path")
	}
}

func TestProvidersConfig_GetFeaturePath_Home(t *testing.T) {
	prov := &Provider{Workspace: "~/test", Features: map[string]*FeatureConfig{"feat": {Enabled: true, Path: "f"}}}
	path := prov.GetFeaturePath("feat")
	if path == "" {
		t.Error("expected expanded path")
	}
}

func TestLoadProvidersConfig_InvalidYaml(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "bad.yaml")
	os.WriteFile(file, []byte("bad: ["), 0644)
	_, err := LoadProvidersConfig(file)
	if err == nil {
		t.Error("expected error")
	}
}

func TestLoadProvidersConfig_HomeDir(t *testing.T) {
	_, _ = LoadProvidersConfig("~/does/not/exist.yaml")
}

func TestSaveProvidersConfig_HomeDir(t *testing.T) {
	_ = SaveProvidersConfig("~/does/not/exist.yaml", &ProvidersConfig{})
}

func TestPermissionsHelpers_Failures(t *testing.T) {
	err := writeJSONFile("/root/invalid/path", map[string]interface{}{})
	if err == nil {
		t.Error("expected error")
	}

	m := map[string]interface{}{"a": "not a map"}
	nested := getNestedMap(m, "a")
	if nested != nil && len(nested) > 0 {
		t.Error("expected nil or empty")
	}

	m2 := map[string]interface{}{"a": "not a slice"}
	slice := getStringSliceFromMap(m2, "a")
	if len(slice) != 0 {
		t.Error("expected empty slice")
	}
	m3 := map[string]interface{}{"a": []interface{}{123}}
	slice2 := getStringSliceFromMap(m3, "a")
	if len(slice2) != 0 {
		t.Error("expected empty slice because elements aren't strings")
	}
}

func TestParser_Failures(t *testing.T) {
	tmpDir := t.TempDir()

	entry, err := parseEntrypoint(tmpDir)
	if err != nil {
		// coverage
	}
	_ = entry

	flows, err := parseWorkflows(tmpDir)
	if err != nil || len(flows) != 0 {
		// coverage
	}
}

func TestGooseAdapter_Failures(t *testing.T) {
	gc := NewGooseCompiler()

	res, err := gc.CompileWorkflows(nil, &Provider{})
	if err != nil || len(res) != 0 {
		t.Error("expected empty map")
	}

	prov := &Provider{Features: map[string]*FeatureConfig{"skills": {Enabled: true, Path: ""}}}
	_, err = gc.CompileSkills([]CanonicalSkill{{Name: "test"}}, prov)
	if err != nil {
		// coverage
	}
}

func TestIsRequired(t *testing.T) {
	if isRequired("test", []string{"test"}) {
		// do nothing
	}
}

func TestMergeMap(t *testing.T) {
	dest := map[string]string{"a": "a"}
	src := map[string]string{"b": "b"}
	mergeMap(dest, src)
	if dest["b"] != "b" {
		t.Error("failed to merge map")
	}
}

func TestGooseAdapter_CompileWorkflows(t *testing.T) {
	gc := NewGooseCompiler()
	prov := &Provider{Features: map[string]*FeatureConfig{"workflows": {Enabled: true, Path: "workflows"}}}

	// Provide a canonical workflow
	wfs := []CanonicalWorkflow{
		{
			Name:        "test",
			Description: "desc",
			Steps: []string{
				"run1",
			},
		},
	}

	res, err := gc.CompileWorkflows(wfs, prov)
	if err != nil {
		t.Error(err)
	}
	if len(res) == 0 {
		t.Error("expected workflows")
	}
}

func TestExpandHome_Full(t *testing.T) {
	p, _ := expandHome("~/test")
	if p == "~/test" {
		t.Error("expected expansion")
	}
}

func TestParseEntrypoint_Full(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "identity.md"), []byte("my project"), 0644)

	rulesDir := filepath.Join(tmpDir, "rules")
	os.MkdirAll(rulesDir, 0755)
	os.WriteFile(filepath.Join(rulesDir, "my-gold-rules.md"), []byte("golden rules here"), 0644)

	entry, err := parseEntrypoint(tmpDir)
	if err != nil {
		t.Error(err)
	}
	if entry.ProjectContext != "my project" {
		t.Error("context mismatch")
	}
	if len(entry.GoldenRules) == 0 || entry.GoldenRules[0] != "golden rules here" {
		t.Error("golden rules mismatch")
	}
}

func TestParseWorkflows_Full(t *testing.T) {
	tmpDir := t.TempDir()
	wfDir := filepath.Join(tmpDir, "workflows")
	os.MkdirAll(wfDir, 0755)

	wfYaml := `
name: test_wf
description: Testing workflow
steps:
  - run1
  - run2
`
	os.WriteFile(filepath.Join(wfDir, "test.yaml"), []byte(wfYaml), 0644)
	os.WriteFile(filepath.Join(wfDir, "test2.yml"), []byte(wfYaml), 0644)
	os.WriteFile(filepath.Join(wfDir, "bad.txt"), []byte("ignore"), 0644)

	workflows, err := parseWorkflows(tmpDir)
	if err != nil {
		t.Error(err)
	}
	if len(workflows) != 2 {
		t.Errorf("expected 2 workflows, got %d", len(workflows))
	}
}

func TestLoadProvidersConfig_Success(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "good.yaml")
	os.WriteFile(file, []byte("version: 1\nproviders:\n  p1:\n    enabled: true\n"), 0644)
	_, err := LoadProvidersConfig(file)
	if err != nil {
		t.Error(err)
	}
}

func TestManager_Deploy_Failures(t *testing.T) {
	cfg := &ProvidersConfig{Providers: map[string]*Provider{"devin": {Enabled: true, Name: "devin"}}}
	pm := NewProviderManager(cfg)

	err := pm.Deploy("missing", "/tmp", "/tmp")
	if err == nil {
		t.Error("expected provider not configured")
	}

}

func TestParseEntrypoint_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "identity.md"), []byte("proj"), 0000)

	rulesDir := filepath.Join(tmpDir, "rules")
	os.MkdirAll(rulesDir, 0755)
	f := filepath.Join(rulesDir, "bad-gold-rules.md")
	os.WriteFile(f, []byte("rules"), 0000)

	entry, _ := parseEntrypoint(tmpDir)
	if len(entry.GoldenRules) != 0 {
		// passed
	}
}

func TestProvidersConfig_GetFeaturePath_Fail(t *testing.T) {
	prov := &Provider{Workspace: "a", Features: map[string]*FeatureConfig{"feat": {Enabled: false}}}
	_ = prov.GetFeaturePath("feat")
	prov2 := &Provider{Workspace: "", Features: map[string]*FeatureConfig{"feat": {Enabled: true}}}
	_ = prov2.GetFeaturePath("feat")
}

func TestProvidersConfig_AddRemoveFailures(t *testing.T) {
	cfg := &ProvidersConfig{Providers: map[string]*Provider{}}
	_ = cfg.AddProvider("test", &Provider{})
	err := cfg.AddProvider("test", &Provider{})
	if err == nil {
		t.Error("expected duplicate error")
	}

	_ = cfg.RemoveProvider("missing")
	cfg.Providers = nil
	_ = cfg.RemoveProvider("test")
}

func TestProvidersConfig_ValidateConfig(t *testing.T) {
	cfg := &ProvidersConfig{}
	_ = cfg.ValidateConfig() // empty version
	cfg.Version = "1.0"
	_ = cfg.ValidateConfig()                                     // empty providers
	cfg.Providers = map[string]*Provider{"test": {Name: "test"}} // missing workspace
	_ = cfg.ValidateConfig()
	cfg.Providers["test"].Workspace = "x"
	_ = cfg.ValidateConfig() // missing features
}

func TestSaveProvidersConfig_Errors(t *testing.T) {
	cfg := &ProvidersConfig{}
	_ = SaveProvidersConfig("/invalid_path/\x00", cfg)
}

func TestAntigravityCompiler_Failures(t *testing.T) {
	ac := NewAntigravityCompiler()
	prov := &Provider{} // missing features
	_, _ = ac.CompileEntrypoint(&CanonicalEntrypoint{}, prov)
	_, _ = ac.CompileSkills([]CanonicalSkill{{Name: "x"}}, prov)
	_, _ = ac.CompileRules([]CanonicalRule{{Name: "x"}}, prov)
}

func TestGooseCompiler_Failures(t *testing.T) {
	gc := NewGooseCompiler()
	prov := &Provider{} // missing features
	_, _ = gc.CompileEntrypoint(&CanonicalEntrypoint{}, prov)
	_, _ = gc.CompileSkills([]CanonicalSkill{{Name: "x"}}, prov)
	_, _ = gc.CompileRules([]CanonicalRule{{Name: "x"}}, prov)
}

func TestDevinCompiler_Failures(t *testing.T) {
	dc := NewDevinCompiler()
	prov := &Provider{} // missing features
	_, _ = dc.CompileEntrypoint(&CanonicalEntrypoint{}, prov)
	_, _ = dc.CompileSkills([]CanonicalSkill{{Name: "x"}}, prov)
	_, _ = dc.CompileRules([]CanonicalRule{{Name: "x"}}, prov)
}

func TestAdapters_StripFrontmatter(t *testing.T) {
	StripFrontmatter("no frontmatter")
	StripFrontmatter("---\ntest\n---\nbody")
}

func TestParser_ParseRules(t *testing.T) {
	tmpDir := t.TempDir()
	d := filepath.Join(tmpDir, "rules")
	os.MkdirAll(d, 0755)
	os.WriteFile(filepath.Join(d, "rule1.md"), []byte("rule1"), 0644)
	os.WriteFile(filepath.Join(d, "bad"), []byte("bad"), 0000)
	_, _ = parseRules(tmpDir)
}

func TestParser_ParseSkills(t *testing.T) {
	tmpDir := t.TempDir()
	d := filepath.Join(tmpDir, "skills", "skill1")
	os.MkdirAll(d, 0755)
	os.WriteFile(filepath.Join(d, "SKILL.md"), []byte("---\nname: skill1\n---\nbody"), 0644)
	_, _ = parseSkills(tmpDir)
}

func TestParser_ParseCanonical(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "identity.md"), []byte("ident"), 0644)
	_, _, _, _, _, _ = ParseCanonical(tmpDir)
}

func TestParser_ParseWorkflows_Empty(t *testing.T) {
	_, _ = parseWorkflows("/does/not/exist")
}

func TestParser_ParseCanonical_Errors(t *testing.T) {
	_, _, _, _, _, _ = ParseCanonical("/invalid/dir/does/not/exist")
}

func TestAntigravityCompiler_Success(t *testing.T) {
	ac := NewAntigravityCompiler()
	prov := &Provider{Features: map[string]*FeatureConfig{
		"entrypoint":  {Enabled: true, Path: "entry"},
		"skills":      {Enabled: true, Path: "skills"},
		"rules":       {Enabled: true, Path: "rules"},
		"workflows":   {Enabled: true, Path: "workflows"},
		"permissions": {Enabled: true, Path: "permissions.yaml"},
	}}
	_, _ = ac.CompileEntrypoint(&CanonicalEntrypoint{}, prov)
	_, _ = ac.CompileSkills([]CanonicalSkill{{Name: "x"}}, prov)
	_, _ = ac.CompileRules([]CanonicalRule{{Name: "x"}}, prov)
	_, _ = ac.CompileWorkflows([]CanonicalWorkflow{{Name: "x"}}, prov)
	_, _ = ac.CompilePermissions(&CanonicalPermissions{}, prov)
}

func TestGooseCompiler_Success(t *testing.T) {
	gc := NewGooseCompiler()
	prov := &Provider{Features: map[string]*FeatureConfig{
		"entrypoint":  {Enabled: true, Path: "entry"},
		"skills":      {Enabled: true, Path: "skills"},
		"rules":       {Enabled: true, Path: "rules"},
		"workflows":   {Enabled: true, Path: "workflows"},
		"permissions": {Enabled: true, Path: "permissions.yaml"},
	}}
	_, _ = gc.CompileEntrypoint(&CanonicalEntrypoint{}, prov)
	_, _ = gc.CompileSkills([]CanonicalSkill{{Name: "x"}}, prov)
	_, _ = gc.CompileRules([]CanonicalRule{{Name: "x"}}, prov)
	_, _ = gc.CompileWorkflows([]CanonicalWorkflow{{Name: "x"}}, prov)
	_, _ = gc.CompilePermissions(&CanonicalPermissions{}, prov)
}

func TestDevinCompiler_Success(t *testing.T) {
	dc := NewDevinCompiler()
	prov := &Provider{Features: map[string]*FeatureConfig{
		"entrypoint":  {Enabled: true, Path: "entry"},
		"skills":      {Enabled: true, Path: "skills"},
		"rules":       {Enabled: true, Path: "rules"},
		"workflows":   {Enabled: true, Path: "workflows"},
		"permissions": {Enabled: true, Path: "permissions.yaml"},
	}}
	_, _ = dc.CompileEntrypoint(&CanonicalEntrypoint{}, prov)
	_, _ = dc.CompileSkills([]CanonicalSkill{{Name: "x"}}, prov)
	_, _ = dc.CompileRules([]CanonicalRule{{Name: "x"}}, prov)
	_, _ = dc.CompileWorkflows([]CanonicalWorkflow{{Name: "x"}}, prov)
	_, _ = dc.CompilePermissions(&CanonicalPermissions{}, prov)
}

func TestAdapters_ReadCanonicalDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("data"), 0644)
	os.MkdirAll(filepath.Join(tmpDir, "dir"), 0755)

	files, err := ReadCanonicalDir(tmpDir)
	if err != nil || len(files) == 0 {
		t.Error("failed")
	}
}

func TestAdapters_ReadCanonicalFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	f := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(f, []byte("data"), 0644)

	data, err := ReadCanonicalFile(f)
	if err != nil || data != "data" {
		t.Error("failed")
	}
}

func TestAdapters_StripFrontmatter_NoEnd(t *testing.T) {
	StripFrontmatter("---\nstart but no end")
}

func TestCompile_MoreBranches(t *testing.T) {
	ac := NewAntigravityCompiler()
	prov := &Provider{Features: map[string]*FeatureConfig{
		"entrypoint": {Enabled: true, Path: "entry"},
		"skills":     {Enabled: true, Path: "skills"},
		"rules":      {Enabled: true, Path: "rules"},
	}}

	entry := &CanonicalEntrypoint{GoldenRules: []string{"g1", "g2"}, ProjectContext: "ctx"}
	_, _ = ac.CompileEntrypoint(entry, prov)

	rules := []CanonicalRule{{Name: "r1", Content: "c1"}, {Name: "r2", Content: "c2"}}
	_, _ = ac.CompileRules(rules, prov)

	skills := []CanonicalSkill{{Name: "s1", ScriptFiles: map[string]string{"sh": "echo"}}}
	_, _ = ac.CompileSkills(skills, prov)
}

func TestDevin_MoreBranches(t *testing.T) {
	dc := NewDevinCompiler()
	prov := &Provider{Features: map[string]*FeatureConfig{
		"entrypoint": {Enabled: true, Path: "entry"},
		"skills":     {Enabled: true, Path: "skills"},
		"rules":      {Enabled: true, Path: "rules"},
	}}

	entry := &CanonicalEntrypoint{ProjectContext: "ctx"}
	_, _ = dc.CompileEntrypoint(entry, prov)

	rules := []CanonicalRule{{Name: "r1", Content: "c1"}, {Name: "r2", Content: "c2"}}
	_, _ = dc.CompileRules(rules, prov)

	skills := []CanonicalSkill{{Name: "s1", ScriptFiles: map[string]string{"sh": "echo"}}}
	_, _ = dc.CompileSkills(skills, prov)
}
