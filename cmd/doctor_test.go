package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/agents"
	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/spf13/cobra"
)

func TestNewDoctorCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewDoctorCommand(log, cfg, tr)
	if cmd == nil {
		t.Fatal("NewDoctorCommand() returned nil")
	}
	if cmd.Use != "doctor" {
		t.Errorf("Use = %q, want %q", cmd.Use, "doctor")
	}
}

func TestRunDoctor_AllPresent(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// Redirect HOME so GetCockpitDir() points to our temp dir.
	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	t.Setenv("HOME", tmpDir)

	// Create all checked paths.
	for _, sub := range []string{"", "vault", "logs", "packages", "cache"} {
		if err := os.MkdirAll(filepath.Join(cockpitDir, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", sub, err)
		}
	}
	// Create the config file too.
	configPath := filepath.Join(cockpitDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("version: 0.1.0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	if err := runDoctor(&cobra.Command{}, log, cfg, tr, false); err != nil {
		t.Errorf("runDoctor() error = %v", err)
	}
}

func TestRunDoctor_MissingDirs(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// Use a completely empty HOME — no .cockpit dir at all.
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// runDoctor must NOT return an error even when checks fail —
	// it just prints failures and returns nil.
	if err := runDoctor(&cobra.Command{}, log, cfg, tr, false); err != nil {
		t.Errorf("runDoctor() with missing dirs should return nil, got %v", err)
	}
}

func TestRunDoctor_JSON(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	tmpDir := t.TempDir()
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	t.Setenv("HOME", tmpDir)

	for _, sub := range []string{"", "vault", "logs", "packages", "cache"} {
		if err := os.MkdirAll(filepath.Join(cockpitDir, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", sub, err)
		}
	}
	configPath := filepath.Join(cockpitDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("version: 0.1.0\n"), 0o644); err != nil {
		t.Fatalf("WriteFile config: %v", err)
	}

	if err := runDoctor(&cobra.Command{}, log, cfg, tr, true); err != nil {
		t.Errorf("runDoctor() json error = %v", err)
	}
}

func TestRunDoctor_HindsightOptional(t *testing.T) {
	prepareDoctorHome(t)
	log, cfg, tr := newTestDeps(t)
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runDoctor(cmd, log, cfg, tr, true); err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Passed bool `json:"passed"`
		Checks []struct {
			Name    string `json:"check_name"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatalf("json: %v\n%s", err, out.String())
	}
	if !payload.Passed {
		t.Fatal("unconfigured hindsight failed doctor")
	}
	var found bool
	for _, check := range payload.Checks {
		if check.Name == "Hindsight" {
			found = true
			if check.Status != "skipped" || !strings.Contains(check.Message, "not configured") {
				t.Fatalf("check = %+v", check)
			}
		}
	}
	if !found {
		t.Fatal("hindsight check missing")
	}
	if cmd.Annotations["telemetry_status"] == "error" {
		t.Fatal("skipped hindsight was recorded as an error")
	}
}

func TestRunDoctor_HindsightConfigured(t *testing.T) {
	tests := []struct {
		name       string
		report     hindsight.Report
		err        error
		wantPassed bool
		wantStatus string
	}{
		{
			name:       "healthy",
			report:     hindsight.Report{Configured: true, Running: true, Healthy: true, Message: "Hindsight API is healthy"},
			wantPassed: true,
			wantStatus: "ok",
		},
		{
			name:       "down",
			report:     hindsight.Report{Configured: true, Message: "Hindsight is configured but the container is not running"},
			wantPassed: false,
			wantStatus: "error",
		},
		{
			name:       "probe failed",
			err:        errors.New("hindsight status: docker unavailable"),
			wantPassed: false,
			wantStatus: "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDoctorHome(t)
			log, cfg, tr := newTestDeps(t)
			cfg.Memory.Hindsight.Enabled = true
			previous := newMemoryManager
			newMemoryManager = func(*config.Config) memoryService {
				return &stubMemory{report: tt.report, err: tt.err}
			}
			t.Cleanup(func() { newMemoryManager = previous })

			var out bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			if err := runDoctor(cmd, log, cfg, tr, false); err != nil {
				t.Fatal(err)
			}
			if tt.wantPassed && strings.Contains(out.String(), "Some checks failed") {
				t.Fatalf("output = %s", out.String())
			}
			if !tt.wantPassed && !strings.Contains(out.String(), "Some checks failed") {
				t.Fatalf("output = %s", out.String())
			}
			if !strings.Contains(out.String(), "Hindsight") {
				t.Fatalf("output = %s", out.String())
			}
			if tt.wantPassed && cmd.Annotations["telemetry_status"] == "error" {
				t.Fatal("healthy check set telemetry error")
			}
			if !tt.wantPassed && cmd.Annotations["telemetry_status"] != "error" {
				t.Fatal("failed check did not set telemetry error")
			}
		})
	}

	log, cfg, tr := newTestDeps(t)
	cfg.Memory.Hindsight.Enabled = false
	previous := newMemoryManager
	newMemoryManager = func(*config.Config) memoryService {
		return &stubMemory{err: errors.New("ignored while disabled")}
	}
	t.Cleanup(func() { newMemoryManager = previous })
	check, fatal := hindsightDoctorCheck(context.Background(), cfg)
	if fatal || check["status"] != "skipped" {
		t.Fatalf("disabled error was fatal: %+v %v", check, fatal)
	}
	_ = log
	_ = tr
}

func prepareDoctorHome(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	for _, sub := range []string{"", "vault", "logs", "packages", "cache"} {
		if err := os.MkdirAll(filepath.Join(cockpitDir, sub), 0o755); err != nil {
			t.Fatalf("MkdirAll %s: %v", sub, err)
		}
	}
	if err := os.WriteFile(filepath.Join(cockpitDir, "config.yaml"), []byte("version: 0.1.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunDoctor_AgentCLIsOptional(t *testing.T) {
	prepareDoctorHome(t)
	log, cfg, tr := newTestDeps(t)
	previous := detectAgentCLIs
	t.Cleanup(func() { detectAgentCLIs = previous })

	tests := []struct {
		name    string
		detect  func(context.Context) []agents.Status
		want    string
		missing string
	}{
		{
			name:   "not installed",
			detect: stubAgentCLIs,
			want:   "Claude CLI is not installed",
		},
		{
			name: "installed",
			detect: func(context.Context) []agents.Status {
				return []agents.Status{
					{ID: "claude", Name: "Claude CLI", Binary: "claude", Installed: true, Version: "9.9.9-test"},
					{ID: "codex", Name: "Codex CLI", Binary: "codex", Installed: true, Detail: "version command failed"},
				}
			},
			want: "Claude CLI 9.9.9-test",
		},
		{
			name:    "detector disabled",
			detect:  nil,
			missing: "Claude CLI",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			detectAgentCLIs = tt.detect
			var out bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			if err := runDoctor(cmd, log, cfg, tr, true); err != nil {
				t.Fatal(err)
			}
			var payload struct {
				Passed bool `json:"passed"`
				Checks []struct {
					Name    string `json:"check_name"`
					Status  string `json:"status"`
					Message string `json:"message"`
				} `json:"checks"`
			}
			if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
				t.Fatalf("json: %v\n%s", err, out.String())
			}
			if !payload.Passed {
				t.Fatal("agent CLI checks failed doctor")
			}
			if cmd.Annotations["telemetry_status"] == "error" {
				t.Fatal("optional CLI check set telemetry error")
			}
			var sawClaude bool
			for _, check := range payload.Checks {
				if check.Name == "Claude CLI" {
					sawClaude = true
					if tt.want != "" && check.Message != tt.want && !strings.Contains(check.Message, tt.want) {
						t.Fatalf("check = %+v", check)
					}
				}
			}
			if tt.missing != "" && sawClaude {
				t.Fatal("detector disabled still reported a CLI")
			}
			if tt.missing == "" && !sawClaude {
				t.Fatal("claude check missing")
			}
		})
	}

	detectAgentCLIs = func(context.Context) []agents.Status {
		return []agents.Status{{ID: "claude", Name: "Claude CLI", Installed: true, Version: "1.2.3"}}
	}
	var text bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&text)
	if err := runDoctor(cmd, log, cfg, tr, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text.String(), "Claude CLI 1.2.3") || !strings.Contains(text.String(), "All checks passed!") {
		t.Fatalf("output = %s", text.String())
	}
}

func TestRunDoctor_APITokensOptional(t *testing.T) {
	prepareDoctorHome(t)
	log, cfg, tr := newTestDeps(t)
	if cfg.Agents.APIs == nil {
		cfg.Agents.APIs = make(map[string]agents.APISettings)
	}
	cfg.Agents.APIs["openai"] = agents.APISettings{Provider: "openai"}
	cfg.Agents.API.Provider = "openai"
	previous := probeAPITokens
	t.Cleanup(func() { probeAPITokens = previous })
	secret := "sk-doctor-secret-value"
	probeAPITokens = func(context.Context, agents.APISettings) []agents.TokenStatus {
		return []agents.TokenStatus{
			{Provider: "anthropic", Name: "Anthropic API", Configured: true},
			{Provider: "openai", Name: "OpenAI API"},
			{Provider: "ollama", Name: "Ollama API", Locked: true},
		}
	}
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	if err := runDoctor(cmd, log, cfg, tr, true); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), secret) {
		t.Fatal("doctor printed a secret")
	}
	var payload struct {
		Passed bool `json:"passed"`
		Checks []struct {
			Name    string `json:"check_name"`
			Status  string `json:"status"`
			Message string `json:"message"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(out.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Passed {
		t.Fatal("token rows failed doctor")
	}
	if cmd.Annotations["telemetry_status"] == "error" {
		t.Fatal("token rows set telemetry error")
	}
	var sawConfigured, sawMissing bool
	for _, check := range payload.Checks {
		if check.Name == "Anthropic API" && check.Status == "ok" && strings.Contains(check.Message, "configured") {
			sawConfigured = true
		}
		if check.Name == "OpenAI API" && check.Status == "skipped" {
			sawMissing = true
		}
	}
	if !sawConfigured || !sawMissing {
		t.Fatalf("checks = %+v", payload.Checks)
	}

	probeAPITokens = nil
	out.Reset()
	if err := runDoctor(cmd, log, cfg, tr, false); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Anthropic API") {
		t.Fatal("nil probe still reported a provider")
	}
}

func TestNewDoctorCommand_Execute(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cmd := NewDoctorCommand(log, cfg, tr)
	// Execute via cobra — exercises the RunE lambda
	if err := cmd.Execute(); err != nil {
		t.Errorf("doctor Execute error = %v", err)
	}
}
