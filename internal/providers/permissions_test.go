package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandHome(t *testing.T) {
	original := os.Getenv("HOME")
	defer os.Setenv("HOME", original)
	os.Setenv("HOME", "/fakehome")

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", "/fakehome/foo/bar"},
		{"/abs/path", "/abs/path"},
		{"relative/path", "relative/path"},
		{"~", "/fakehome"},
	}
	for _, tt := range tests {
		got, err := expandHome(tt.input)
		if err != nil {
			t.Errorf("expandHome(%q) error: %v", tt.input, err)
		}
		if got != tt.want {
			t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMergeStringSlice(t *testing.T) {
	tests := []struct {
		existing  []string
		additions []string
		wantLen   int
		wantFirst string
	}{
		{[]string{"a", "b"}, []string{"b", "c"}, 3, "a"},
		{[]string{}, []string{"x", "y"}, 2, "x"},
		{[]string{"a"}, []string{}, 1, "a"},
		{[]string{}, []string{}, 0, ""},
	}
	for _, tt := range tests {
		got := mergeStringSlice(tt.existing, tt.additions)
		if len(got) != tt.wantLen {
			t.Errorf("mergeStringSlice len = %d, want %d", len(got), tt.wantLen)
		}
		if tt.wantLen > 0 && got[0] != tt.wantFirst {
			t.Errorf("mergeStringSlice[0] = %q, want %q", got[0], tt.wantFirst)
		}
	}
}

func TestReadWriteJSONFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.json")

	// Non-existent file returns empty map
	m, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("readJSONFile on non-existent: %v", err)
	}
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}

	// Write and read back
	original := map[string]interface{}{
		"key": "value",
		"num": float64(42),
	}
	if err := writeJSONFile(path, original); err != nil {
		t.Fatalf("writeJSONFile: %v", err)
	}

	read, err := readJSONFile(path)
	if err != nil {
		t.Fatalf("readJSONFile after write: %v", err)
	}
	if read["key"] != "value" {
		t.Errorf("key = %v, want 'value'", read["key"])
	}
}

func TestReadJSONFile_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := readJSONFile(path)
	if err == nil {
		t.Error("expected error on invalid JSON")
	}
}

func TestGetNestedMap(t *testing.T) {
	m := map[string]interface{}{}

	// Creates nested maps if absent
	inner := getNestedMap(m, "a", "b", "c")
	inner["key"] = "val"

	// Navigate back
	a := m["a"].(map[string]interface{})
	b := a["b"].(map[string]interface{})
	c := b["c"].(map[string]interface{})
	if c["key"] != "val" {
		t.Errorf("nested map key = %v, want 'val'", c["key"])
	}
}

func TestGetSetStringSliceInMap(t *testing.T) {
	m := map[string]interface{}{}

	// Empty key
	got := getStringSliceFromMap(m, "missing")
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}

	// Set and get back
	setStringSliceInMap(m, "items", []string{"a", "b", "c"})
	got = getStringSliceFromMap(m, "items")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Errorf("getStringSliceFromMap = %v, want [a b c]", got)
	}

	// Wrong type in map
	m["bad"] = 42
	got = getStringSliceFromMap(m, "bad")
	if len(got) != 0 {
		t.Errorf("expected empty slice for wrong type, got %v", got)
	}
}

// --- Antigravity permissions tests ---

func TestAntigravityAdapter_ApplyPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	adapter := NewAntigravityAdapter()
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("applyPermissions: %v", err)
	}

	// Verify file was created and contains expected structure
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config.json: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to parse config.json: %v", err)
	}

	us, ok := result["userSettings"].(map[string]interface{})
	if !ok {
		t.Fatal("userSettings missing or wrong type")
	}
	grants, ok := us["globalPermissionGrants"].(map[string]interface{})
	if !ok {
		t.Fatal("globalPermissionGrants missing or wrong type")
	}
	allow := getStringSliceFromMap(grants, "allow")
	if len(allow) == 0 {
		t.Fatal("allow is empty")
	}

	// Check at least one cockpit permission is present
	found := false
	for _, p := range allow {
		if p == "command(cockpit)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'command(cockpit)' in allow, got %v", allow)
	}
}

func TestAntigravityAdapter_ApplyPermissions_Merges(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	// Pre-existing config with user permissions
	existing := map[string]interface{}{
		"userSettings": map[string]interface{}{
			"globalPermissionGrants": map[string]interface{}{
				"allow": []interface{}{"command(my-tool)", "read_file(/my/project)"},
			},
		},
	}
	data, _ := json.Marshal(existing)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewAntigravityAdapter()
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("applyPermissions: %v", err)
	}

	result, _ := readJSONFile(configPath)
	us := getNestedMap(result, "userSettings")
	grants := getNestedMap(us, "globalPermissionGrants")
	allow := getStringSliceFromMap(grants, "allow")

	// User permissions preserved
	hasMyTool := false
	hasCockpit := false
	for _, p := range allow {
		if p == "command(my-tool)" {
			hasMyTool = true
		}
		if p == "command(cockpit)" {
			hasCockpit = true
		}
	}
	if !hasMyTool {
		t.Error("user permission 'command(my-tool)' was removed")
	}
	if !hasCockpit {
		t.Error("cockpit permission 'command(cockpit)' was not added")
	}
}

func TestAntigravityAdapter_ApplyPermissions_NoDuplicates(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	adapter := NewAntigravityAdapter()

	// Apply twice
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("first apply: %v", err)
	}
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("second apply: %v", err)
	}

	result, _ := readJSONFile(configPath)
	us := getNestedMap(result, "userSettings")
	grants := getNestedMap(us, "globalPermissionGrants")
	allow := getStringSliceFromMap(grants, "allow")

	// Count command(cockpit) occurrences
	count := 0
	for _, p := range allow {
		if p == "command(cockpit)" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("'command(cockpit)' appears %d times, want exactly 1", count)
	}
}

// --- Devin permissions tests ---

func TestDevinAdapter_ApplyPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.local.json")

	adapter := NewDevinAdapter()
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("applyPermissions: %v", err)
	}

	result, err := readJSONFile(configPath)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}

	perms := getNestedMap(result, "permissions")
	allow := getStringSliceFromMap(perms, "allow")

	found := false
	for _, p := range allow {
		if p == "Exec(cockpit)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'Exec(cockpit)' in allow, got %v", allow)
	}
}

func TestDevinAdapter_ApplyPermissions_Merges(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.local.json")

	existing := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Exec(my-script)"},
		},
	}
	data, _ := json.Marshal(existing)
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewDevinAdapter()
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("applyPermissions: %v", err)
	}

	result, _ := readJSONFile(configPath)
	perms := getNestedMap(result, "permissions")
	allow := getStringSliceFromMap(perms, "allow")

	hasScript, hasCockpit := false, false
	for _, p := range allow {
		if p == "Exec(my-script)" {
			hasScript = true
		}
		if p == "Exec(cockpit)" {
			hasCockpit = true
		}
	}
	if !hasScript {
		t.Error("user permission 'Exec(my-script)' was removed")
	}
	if !hasCockpit {
		t.Error("cockpit permission 'Exec(cockpit)' was not added")
	}
}

// --- Goose permissions tests ---

func TestGooseAdapter_ApplyPermissions_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	// Non-existent path — should be a no-op
	adapter := NewGooseAdapter()
	err := adapter.applyPermissions(filepath.Join(tmpDir, "nonexistent.yaml"))
	if err != nil {
		t.Errorf("expected no error for non-existent Goose config, got: %v", err)
	}
}

func TestGooseAdapter_ApplyPermissions_EnablesExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write a Goose config with developer disabled
	content := `extensions:
  developer:
    enabled: false
    type: platform
    name: developer
    description: 'Write and edit files'
    bundled: true
    available_tools: []
  skills:
    enabled: false
    type: platform
    name: skills
    bundled: true
    available_tools: []
GOOSE_PROVIDER: gemini
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewGooseAdapter()
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("applyPermissions: %v", err)
	}

	result, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	out := string(result)

	// developer and skills should now be enabled
	if !containsSequence(out, "developer:", "enabled: true") {
		t.Error("developer extension not enabled")
	}
	if !containsSequence(out, "skills:", "enabled: true") {
		t.Error("skills extension not enabled")
	}
	// GOOSE_PROVIDER should still be present
	if !containsLine(out, "GOOSE_PROVIDER: gemini") {
		t.Error("GOOSE_PROVIDER was removed from config")
	}
}

func TestGooseAdapter_ApplyPermissions_AppendsIfMissing(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Config without summon extension
	content := `extensions:
  developer:
    enabled: true
    type: platform
    name: developer
    bundled: true
    available_tools: []
GOOSE_PROVIDER: gemini
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := NewGooseAdapter()
	if err := adapter.applyPermissions(configPath); err != nil {
		t.Fatalf("applyPermissions: %v", err)
	}

	result, _ := os.ReadFile(configPath)
	out := string(result)

	// summon should have been appended
	if !containsLine(out, "summon:") {
		t.Error("summon extension block was not appended")
	}
}

func TestEnsureGooseExtensionsEnabled(t *testing.T) {
	// Already enabled — no change expected
	input := `extensions:
  developer:
    enabled: true
    bundled: true
`
	out := ensureGooseExtensionsEnabled(input, []string{"developer"})
	if !containsLine(out, "enabled: true") {
		t.Error("enabled: true should be in output")
	}
}

func TestIsRequired(t *testing.T) {
	if !isRequired("developer", []string{"developer", "skills"}) {
		t.Error("expected developer to be required")
	}
	if isRequired("other", []string{"developer", "skills"}) {
		t.Error("expected other to not be required")
	}
}

func TestWriteJSONFile_MkdirError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a file where a directory is needed
	parentFile := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(parentFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := writeJSONFile(filepath.Join(parentFile, "nested.json"), map[string]interface{}{"k": "v"})
	if err == nil {
		t.Error("expected error when parent is a file")
	}
}

func TestAntigravityAdapter_ApplyPermissions_ReadError(t *testing.T) {
	// Directory masquerading as JSON file causes read error
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "config.json")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := NewAntigravityAdapter()
	if err := adapter.applyPermissions(dirPath); err == nil {
		t.Error("expected error reading directory as JSON")
	}
}

func TestDevinAdapter_ApplyPermissions_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "config.local.json")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := NewDevinAdapter()
	if err := adapter.applyPermissions(dirPath); err == nil {
		t.Error("expected error reading directory as JSON")
	}
}

func TestGooseAdapter_ApplyPermissions_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	dirPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := NewGooseAdapter()
	if err := adapter.applyPermissions(dirPath); err == nil {
		t.Error("expected error reading directory as yaml")
	}
}

func TestAntigravityAdapter_Compile_WithPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	provider := &Provider{
		Enabled:   true,
		Name:      "Antigravity",
		Workspace: tmpDir,
		Features: map[string]*FeatureConfig{
			"permissions": {Enabled: true, Path: configPath},
		},
	}

	adapter := NewAntigravityAdapter()
	if _, err := adapter.Compile(tmpDir, provider); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Error("expected config.json to be created")
	}
}

func TestDevinAdapter_Compile_WithPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.local.json")

	provider := &Provider{
		Enabled:   true,
		Name:      "Devin",
		Workspace: tmpDir,
		Features: map[string]*FeatureConfig{
			"permissions": {Enabled: true, Path: configPath},
		},
	}

	adapter := NewDevinAdapter()
	if _, err := adapter.Compile(tmpDir, provider); err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Error("expected config.local.json to be created")
	}
}

func TestGooseAdapter_Compile_WithPermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "goose_config.yaml")

	content := "extensions:\n  developer:\n    enabled: false\n    bundled: true\n    available_tools: []\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	provider := &Provider{
		Enabled:   true,
		Name:      "Goose",
		Workspace: tmpDir,
		Features: map[string]*FeatureConfig{
			"permissions": {Enabled: true, Path: configPath},
		},
	}

	adapter := NewGooseAdapter()
	if _, err := adapter.Compile(tmpDir, provider); err != nil {
		t.Fatalf("Compile: %v", err)
	}

	result, _ := os.ReadFile(configPath)
	if !containsLine(string(result), "enabled: true") {
		t.Error("expected developer extension to be enabled after Compile")
	}
}

// --- Helpers ---

// containsLine checks if text contains needle as a substring.
func containsLine(text, needle string) bool {
	return strings.Contains(text, needle)
}

// containsSequence checks that needle2 appears somewhere in the lines
// that follow needle1 in the same YAML block (before the next peer key).
func containsSequence(text, needle1, needle2 string) bool {
	lines := strings.Split(text, "\n")
	inBlock := false
	for _, line := range lines {
		if strings.Contains(line, needle1) {
			inBlock = true
			continue
		}
		if inBlock {
			// A peer key (same 2-space indent, ends with :) ends this block
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") &&
				strings.HasSuffix(trimmed, ":") && trimmed != needle1+":" {
				break
			}
			if strings.Contains(line, needle2) {
				return true
			}
		}
	}
	return false
}

func splitLines(s string) []string {
	return strings.Split(s, "\n")
}
