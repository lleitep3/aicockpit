package cmd

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/spf13/cobra"
)

type stubMemory struct {
	report hindsight.Report
	err    error
	calls  []string
}

func (s *stubMemory) Up(context.Context) (hindsight.Report, error) {
	s.calls = append(s.calls, "up")
	return s.report, s.err
}

func (s *stubMemory) Down(context.Context) (hindsight.Report, error) {
	s.calls = append(s.calls, "down")
	return s.report, s.err
}

func (s *stubMemory) Status(context.Context) (hindsight.Report, error) {
	s.calls = append(s.calls, "status")
	return s.report, s.err
}

func TestMemoryCommands(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	stub := &stubMemory{report: hindsight.Report{
		Configured: true, Backend: "hindsight", State: "running", Health: "ok",
		Container: hindsight.DefaultContainer, Message: "Hindsight API is healthy",
		APIURL: "http://127.0.0.1:8888", UIURL: "http://127.0.0.1:9999",
	}}
	previous := newMemoryManager
	newMemoryManager = func(*config.Config) memoryService { return stub }
	t.Cleanup(func() { newMemoryManager = previous })

	tests := []struct {
		name    string
		args    []string
		wantErr string
		wantOut string
		call    string
	}{
		{name: "status text", args: []string{"status"}, wantOut: "Hindsight API is healthy", call: "status"},
		{name: "status json", args: []string{"status", "--json"}, wantOut: `"state": "running"`, call: "status"},
		{name: "up", args: []string{"up"}, wantOut: "configured: true", call: "up"},
		{name: "down", args: []string{"down"}, wantOut: "aicockpit-hindsight", call: "down"},
		{name: "json parent flag", args: []string{"--json", "status"}, wantOut: `"health": "ok"`, call: "status"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub.calls = nil
			cmd := NewMemoryCommand(log, cfg, tr)
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&out)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("execute: %v", err)
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("output = %s", out.String())
			}
			if len(stub.calls) != 1 || stub.calls[0] != tt.call {
				t.Fatalf("calls = %v", stub.calls)
			}
		})
	}
}

func TestMemoryCommandErrorAndHelp(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	previous := newMemoryManager
	newMemoryManager = func(*config.Config) memoryService {
		return &stubMemory{err: errors.New("runtime unavailable")}
	}
	t.Cleanup(func() { newMemoryManager = previous })

	cmd := NewMemoryCommand(log, cfg, tr)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"up"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "runtime unavailable") {
		t.Fatalf("error = %v", err)
	}

	help := NewMemoryCommand(log, cfg, tr)
	help.SetOut(&out)
	help.SetErr(&out)
	help.SetArgs([]string{"--help"})
	if err := help.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "vault") || !strings.Contains(out.String(), "up") {
		t.Fatalf("help = %s", out.String())
	}

	rootLog, rootCfg, rootTr := newTestDeps(t)
	root := NewRootCommand(rootLog, rootCfg, rootTr)
	if root.Commands() == nil || findCommand(root, "memory") == nil {
		t.Fatal("memory command was not registered")
	}
}

func TestMemoryUpDoesNotPersistSecret(t *testing.T) {
	secret := "sk-live-memory-secret"
	logDir := t.TempDir()
	t.Setenv("COCKPIT_LOG_DIR", logDir)
	t.Setenv("COCKPIT_DATA_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	log, cfg, tr := newTestDeps(t)
	cfg.AutoUpdateCheck = false
	cfg.Memory.Hindsight = hindsight.Settings{Enabled: true, LLMProvider: "openai"}
	runtime := &hindsight.FakeRuntime{RunErr: errors.New("engine rejected " + secret)}
	previous := newMemoryManager
	newMemoryManager = func(cfg *config.Config) memoryService {
		return hindsight.NewManager(cfg.Memory.Hindsight, runtime, hindsight.HTTPProber{}, hindsight.SecretFunc(func(context.Context, string, string) (string, error) {
			return secret, nil
		}))
	}
	t.Cleanup(func() { newMemoryManager = previous })

	cmd := NewRootCommand(log, cfg, tr)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"memory", "up"})
	err := cmd.Execute()
	if err == nil || strings.Contains(err.Error(), secret) || strings.Contains(out.String(), secret) {
		t.Fatalf("err=%v out=%s", err, out.String())
	}
	if strings.Contains(err.Error(), "[redacted]") == false {
		t.Fatalf("error was not redacted: %v", err)
	}
	assertTreeHasNoSecret(t, logDir, secret)
	if len(runtime.Specs) != 1 || !specHasSecret(runtime.Specs[0], secret) {
		t.Fatal("secret was not passed to the runtime")
	}
	if specHasDatabaseURL(runtime.Specs[0]) || runtime.Specs[0].Mount != hindsight.PG0Mount {
		t.Fatalf("spec = %+v", runtime.Specs[0])
	}
}

func TestMemoryUpHealthyOutputOmitsSecret(t *testing.T) {
	secret := "sk-healthy-memory-secret"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != hindsight.HealthPath {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	port, err := strconv.Atoi(strings.TrimPrefix(server.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatal(err)
	}
	logDir := t.TempDir()
	t.Setenv("COCKPIT_LOG_DIR", logDir)
	t.Setenv("HOME", t.TempDir())
	log, cfg, tr := newTestDeps(t)
	cfg.Memory.Hindsight = hindsight.Settings{Enabled: true, LLMProvider: "groq", APIPort: port, UIPort: port + 1}
	runtime := &hindsight.FakeRuntime{}
	previous := newMemoryManager
	newMemoryManager = func(cfg *config.Config) memoryService {
		return hindsight.NewManager(cfg.Memory.Hindsight, runtime, hindsight.HTTPProber{Client: server.Client()}, hindsight.SecretFunc(func(context.Context, string, string) (string, error) {
			return secret, nil
		}))
	}
	t.Cleanup(func() { newMemoryManager = previous })

	cmd := NewMemoryCommand(log, cfg, tr)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"up", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"healthy": true`) || strings.Contains(out.String(), secret) {
		t.Fatalf("output = %s", out.String())
	}
	assertTreeHasNoSecret(t, logDir, secret)
	if !specHasSecret(runtime.Specs[0], secret) {
		t.Fatal("runtime did not receive the vault secret")
	}
}

func TestVaultHindsightSecrets(t *testing.T) {
	secret := "vault-only-secret"
	ctx := context.Background()

	t.Run("locked vault does not read", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		previousAuth := authorizeHindsightVault
		previousOpen := openHindsightVault
		authorizeHindsightVault = func(string) error { return errors.New("vault access denied") }
		openHindsightVault = func(string) vaultGetter {
			t.Fatal("vault was opened after access was denied")
			return nil
		}
		t.Cleanup(func() {
			authorizeHindsightVault = previousAuth
			openHindsightVault = previousOpen
		})
		if _, err := (vaultHindsightSecrets{}).Get(ctx, "hindsight", "llm_api_key"); err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("error = %v", err)
		}
	})

	tests := []struct {
		name    string
		value   string
		err     error
		wantErr string
		want    string
	}{
		{name: "value", value: secret, want: secret},
		{name: "read error", err: errors.New("not found"), wantErr: "read vault key hindsight/llm_api_key"},
		{name: "empty", value: "  ", wantErr: "is empty"},
		{name: "newline", value: "bad\n" + secret, wantErr: "newline"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			previousAuth := authorizeHindsightVault
			previousOpen := openHindsightVault
			authorizeHindsightVault = func(namespace string) error {
				if namespace != "hindsight" {
					t.Fatalf("namespace = %s", namespace)
				}
				return nil
			}
			openHindsightVault = func(string) vaultGetter {
				return getterFunc(func(key string) (string, error) {
					if key != "llm_api_key" {
						t.Fatalf("key = %s", key)
					}
					return tt.value, tt.err
				})
			}
			t.Cleanup(func() {
				authorizeHindsightVault = previousAuth
				openHindsightVault = previousOpen
			})
			got, err := (vaultHindsightSecrets{}).Get(ctx, "hindsight", "llm_api_key")
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) || strings.Contains(err.Error(), secret) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Fatalf("got %q err %v", got, err)
			}
		})
	}

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := (vaultHindsightSecrets{}).Get(canceled, "hindsight", "llm_api_key"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled = %v", err)
	}

	t.Run("default authorize stays closed", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		t.Setenv("COCKPIT_DEV_MODE", "")
		previousOpen := openHindsightVault
		openHindsightVault = func(string) vaultGetter {
			t.Fatal("vault was opened while the lock denied access")
			return nil
		}
		t.Cleanup(func() { openHindsightVault = previousOpen })
		if _, err := (vaultHindsightSecrets{}).Get(ctx, "hindsight", "llm_api_key"); err == nil || strings.Contains(err.Error(), secret) {
			t.Fatalf("error = %v", err)
		}
	})
}

type getterFunc func(string) (string, error)

func (f getterFunc) Get(key string) (string, error) { return f(key) }

func findCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func specHasSecret(spec hindsight.Spec, secret string) bool {
	for _, env := range spec.Env {
		if env.Secret && env.Value == secret && env.Name == hindsight.EnvLLMAPIKey {
			return true
		}
	}
	return false
}

func specHasDatabaseURL(spec hindsight.Spec) bool {
	for _, env := range spec.Env {
		if env.Name == hindsight.EnvDatabaseURL {
			return true
		}
	}
	return false
}

func assertTreeHasNoSecret(t *testing.T, root, secret string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if strings.Contains(string(data), secret) {
			t.Fatalf("secret written to %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
