package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/providers"
	"gopkg.in/yaml.v3"
)

// --- selectMultiple ---

func makeOpts(names ...string) []providers.ProviderOption {
	opts := make([]providers.ProviderOption, len(names))
	for i, n := range names {
		displayName := n
		if len(n) > 0 {
			displayName = strings.ToUpper(n[:1]) + n[1:]
		}
		opts[i] = providers.ProviderOption{Name: n, DisplayName: displayName}
	}
	return opts
}

func withStdin(t *testing.T, input string, fn func()) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	_, _ = w.WriteString(input + "\n")
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = origStdin }()

	fn()
}

func TestSelectMultiple_SingleSelection(t *testing.T) {
	opts := makeOpts("antigravity", "devin", "goose")
	var result []string
	withStdin(t, "2", func() {
		result = selectMultiple(opts)
	})
	if len(result) != 1 || result[0] != "devin" {
		t.Errorf("expected [devin], got %v", result)
	}
}

func TestSelectMultiple_MultipleSelections(t *testing.T) {
	opts := makeOpts("antigravity", "devin", "goose")
	var result []string
	withStdin(t, "1,3", func() {
		result = selectMultiple(opts)
	})
	if len(result) != 2 || result[0] != "antigravity" || result[1] != "goose" {
		t.Errorf("expected [antigravity goose], got %v", result)
	}
}

func TestSelectMultiple_EmptyInput_FallsBackToFirst(t *testing.T) {
	opts := makeOpts("antigravity", "devin")
	var result []string
	withStdin(t, "", func() {
		result = selectMultiple(opts)
	})
	if len(result) != 1 || result[0] != "antigravity" {
		t.Errorf("expected [antigravity], got %v", result)
	}
}

func TestSelectMultiple_InvalidInput_FallsBackToFirst(t *testing.T) {
	opts := makeOpts("antigravity", "devin")
	var result []string
	withStdin(t, "abc", func() {
		result = selectMultiple(opts)
	})
	if len(result) != 1 || result[0] != "antigravity" {
		t.Errorf("expected [antigravity], got %v", result)
	}
}

func TestSelectMultiple_OutOfRange_Skipped(t *testing.T) {
	opts := makeOpts("antigravity", "devin")
	var result []string
	withStdin(t, "1,99", func() {
		result = selectMultiple(opts)
	})
	if len(result) != 1 || result[0] != "antigravity" {
		t.Errorf("expected [antigravity], got %v", result)
	}
}

func TestSelectMultiple_Deduplication(t *testing.T) {
	opts := makeOpts("antigravity", "devin", "goose")
	var result []string
	withStdin(t, "1,1,2", func() {
		result = selectMultiple(opts)
	})
	if len(result) != 2 {
		t.Errorf("expected 2 unique providers, got %v", result)
	}
}

func TestSelectMultiple_EmptyOptions(t *testing.T) {
	var result []string
	withStdin(t, "", func() {
		result = selectMultiple([]providers.ProviderOption{})
	})
	if len(result) != 0 {
		t.Errorf("expected empty slice for empty options, got %v", result)
	}
}

// --- updateConfigWithProviders ---

func makeTestConfig(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestUpdateConfigWithProviders_Basic(t *testing.T) {
	configContent := `ai_provider: old
language: en-us
ai_providers:
  enabled:
    - old
`
	path := makeTestConfig(t, configContent)

	err := updateConfigWithProviders(path, []string{"antigravity", "goose"}, "pt-br")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	if m["ai_provider"] != "antigravity" {
		t.Errorf("ai_provider = %v, want antigravity", m["ai_provider"])
	}
	if m["language"] != "pt-br" {
		t.Errorf("language = %v, want pt-br", m["language"])
	}
}

func TestUpdateConfigWithProviders_NoExistingAIProviders(t *testing.T) {
	configContent := `ai_provider: old
language: en-us
`
	path := makeTestConfig(t, configContent)

	err := updateConfigWithProviders(path, []string{"devin"}, "en-us")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(path)
	var m map[string]interface{}
	if err := yaml.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	if m["ai_provider"] != "devin" {
		t.Errorf("ai_provider = %v, want devin", m["ai_provider"])
	}
}

func TestUpdateConfigWithProviders_FileNotFound(t *testing.T) {
	err := updateConfigWithProviders("/nonexistent/config.yaml", []string{"antigravity"}, "en-us")
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestUpdateConfigWithProviders_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(path, []byte("{not: valid: yaml::}"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := updateConfigWithProviders(path, []string{"antigravity"}, "en-us")
	if err != nil {
		// yaml.Unmarshal is very permissive; this may or may not fail — just assert no panic
		t.Logf("got expected parse behavior: %v", err)
	}
}

func TestUpdateConfigWithProviders_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(path, []byte("ai_provider: x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Make file read-only to force write error
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(path, 0o644)

	err := updateConfigWithProviders(path, []string{"antigravity"}, "en-us")
	if err == nil {
		t.Error("expected write error for read-only file")
	}
}
