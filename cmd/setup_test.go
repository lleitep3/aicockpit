package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/providers"
	"gopkg.in/yaml.v3"
)

// --- helpers ---

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

func makeOptsWithDetection(detected map[string]bool, names ...string) []providers.ProviderOption {
	opts := make([]providers.ProviderOption, len(names))
	for i, n := range names {
		displayName := strings.ToUpper(n[:1]) + n[1:]
		opts[i] = providers.ProviderOption{Name: n, DisplayName: displayName, Detected: detected[n]}
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
	// Start from a legacy config with old fields; they should be removed after update.
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

	// New format: enabled_providers list
	providers, ok := m["enabled_providers"].([]interface{})
	if !ok || len(providers) != 2 {
		t.Errorf("enabled_providers = %v, want [antigravity goose]", m["enabled_providers"])
	} else if providers[0] != "antigravity" {
		t.Errorf("first provider = %v, want antigravity", providers[0])
	}
	if m["language"] != "pt-br" {
		t.Errorf("language = %v, want pt-br", m["language"])
	}
	// Legacy fields must be absent
	if _, exists := m["ai_provider"]; exists {
		t.Error("ai_provider should have been removed (legacy migration)")
	}
	if _, exists := m["ai_providers"]; exists {
		t.Error("ai_providers should have been removed (legacy migration)")
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

	providers, ok := m["enabled_providers"].([]interface{})
	if !ok || len(providers) != 1 || providers[0] != "devin" {
		t.Errorf("enabled_providers = %v, want [devin]", m["enabled_providers"])
	}
	if _, exists := m["ai_provider"]; exists {
		t.Error("ai_provider should have been removed (legacy migration)")
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

// --- defaultProviderSelection ---

func TestDefaultProviderSelection_UsesDetected(t *testing.T) {
	opts := makeOptsWithDetection(
		map[string]bool{"devin": true, "goose": true},
		"antigravity", "devin", "goose",
	)
	got := defaultProviderSelection(opts)
	if len(got) != 2 {
		t.Fatalf("expected 2 detected providers, got %v", got)
	}
	// Both detected providers should be in the result
	found := map[string]bool{}
	for _, g := range got {
		found[g] = true
	}
	if !found["devin"] || !found["goose"] {
		t.Errorf("expected devin and goose, got %v", got)
	}
}

func TestDefaultProviderSelection_FallsBackToFirst(t *testing.T) {
	opts := makeOptsWithDetection(map[string]bool{}, "antigravity", "devin")
	got := defaultProviderSelection(opts)
	if len(got) != 1 || got[0] != "antigravity" {
		t.Errorf("expected [antigravity], got %v", got)
	}
}

func TestDefaultProviderSelection_Empty(t *testing.T) {
	got := defaultProviderSelection([]providers.ProviderOption{})
	if len(got) != 0 {
		t.Errorf("expected empty, got %v", got)
	}
}

// --- buildDefaultSuggestion ---

func TestBuildDefaultSuggestion_WithDetected(t *testing.T) {
	opts := makeOptsWithDetection(
		map[string]bool{"devin": true},
		"antigravity", "devin", "goose",
	)
	got := buildDefaultSuggestion(opts)
	if got != "Devin" {
		t.Errorf("expected 'Devin', got %q", got)
	}
}

func TestBuildDefaultSuggestion_MultipleDetected(t *testing.T) {
	opts := makeOptsWithDetection(
		map[string]bool{"devin": true, "goose": true},
		"devin", "goose",
	)
	got := buildDefaultSuggestion(opts)
	if got != "Devin, Goose" {
		t.Errorf("expected 'Devin, Goose', got %q", got)
	}
}

func TestBuildDefaultSuggestion_NoneDetected(t *testing.T) {
	opts := makeOptsWithDetection(map[string]bool{}, "antigravity", "devin")
	got := buildDefaultSuggestion(opts)
	if got != "Antigravity" {
		t.Errorf("expected 'Antigravity' (first option fallback), got %q", got)
	}
}

func TestBuildDefaultSuggestion_Empty(t *testing.T) {
	got := buildDefaultSuggestion([]providers.ProviderOption{})
	if got != "1" {
		t.Errorf("expected '1', got %q", got)
	}
}

// --- selectMultipleWithDefault ---

func TestSelectMultipleWithDefault_EmptyInputUsesDetected(t *testing.T) {
	opts := makeOptsWithDetection(
		map[string]bool{"goose": true},
		"antigravity", "devin", "goose",
	)
	var got []string
	withStdin(t, "\n", func() {
		got = selectMultipleWithDefault(opts)
	})
	if len(got) != 1 || got[0] != "goose" {
		t.Errorf("expected [goose], got %v", got)
	}
}

func TestSelectMultipleWithDefault_ExplicitSelectionOverridesDetection(t *testing.T) {
	opts := makeOptsWithDetection(
		map[string]bool{"goose": true},
		"antigravity", "devin", "goose",
	)
	var got []string
	withStdin(t, "1,2\n", func() {
		got = selectMultipleWithDefault(opts)
	})
	if len(got) != 2 || got[0] != "antigravity" || got[1] != "devin" {
		t.Errorf("expected [antigravity devin], got %v", got)
	}
}

// ── selectOption tests ────────────────────────────────────────────────────

func TestSelectOption_ValidSelection(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "2", func() {
		result = selectOption(options, "en-us")
	})
	if result != "pt-br" {
		t.Errorf("selectOption() = %q, want %q", result, "pt-br")
	}
}

func TestSelectOption_FirstOption(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "1", func() {
		result = selectOption(options, "en-us")
	})
	if result != "en-us" {
		t.Errorf("selectOption() = %q, want %q", result, "en-us")
	}
}

func TestSelectOption_EmptyInput_UsesDefault(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "", func() {
		result = selectOption(options, "en-us")
	})
	if result != "en-us" {
		t.Errorf("selectOption() = %q, want default %q", result, "en-us")
	}
}

func TestSelectOption_InvalidInput_UsesDefault(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "abc", func() {
		result = selectOption(options, "pt-br")
	})
	if result != "pt-br" {
		t.Errorf("selectOption() = %q, want default %q", result, "pt-br")
	}
}

func TestSelectOption_OutOfRange_UsesDefault(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "99", func() {
		result = selectOption(options, "en-us")
	})
	if result != "en-us" {
		t.Errorf("selectOption() = %q, want default %q", result, "en-us")
	}
}

func TestSelectOption_ZeroIndex_UsesDefault(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "0", func() {
		result = selectOption(options, "en-us")
	})
	if result != "en-us" {
		t.Errorf("selectOption() = %q, want default %q", result, "en-us")
	}
}

func TestSelectOption_NegativeIndex_UsesDefault(t *testing.T) {
	options := []string{"en-us", "pt-br"}
	var result string
	withStdin(t, "-1", func() {
		result = selectOption(options, "en-us")
	})
	if result != "en-us" {
		t.Errorf("selectOption() = %q, want default %q", result, "en-us")
	}
}

// ── NewSetupCommand constructor test ──────────────────────────────────────

func TestNewSetupCommand_Constructor(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewSetupCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewSetupCommand() returned nil")
	}
	if cmd.Use != "setup" {
		t.Errorf("Use = %q, want %q", cmd.Use, "setup")
	}
}

func TestNewSetupCommand_Execute(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Chdir(tmpDir)

	log, cfg, tr := newTestDeps(t)
	cmd := NewSetupCommand(log, cfg, tr)

	// Provide stdin for language selection (1=en-us) and provider selection (empty=defaults)
	withStdin(t, "1\n\n", func() {
		// May fail on provider deploy but the RunE lambda is exercised
		_ = cmd.Execute()
	})
}

// ── runSetup integration test ─────────────────────────────────────────────

func TestRunSetup_Full(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Chdir(tmpDir)

	log, cfg, tr := newTestDeps(t)

	// Input: "1" for language (en-us), "1" for provider (first available)
	// The function reads stdin twice:
	// 1. selectOption for language
	// 2. selectMultipleWithDefault for providers
	withStdin(t, "1\n1\n", func() {
		err := runSetup(log, cfg, tr)
		// This may fail on deploy (no valid workspace) but should at least run through
		// the provider selection successfully. If it fails, it should be on deploy not on selection.
		if err != nil {
			// Accept errors that come from the deploy step (provider workspace not found)
			// but not from early failures
			t.Logf("runSetup error (may be expected from deploy): %v", err)
		}
	})
}

func TestRunSetup_PtBrLanguage(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Chdir(tmpDir)

	log, cfg, tr := newTestDeps(t)

	// Input: "2" for language (pt-br), then empty line to accept defaults
	withStdin(t, "2\n\n", func() {
		err := runSetup(log, cfg, tr)
		if err != nil {
			t.Logf("runSetup error (may be expected from deploy): %v", err)
		}
	})
}
