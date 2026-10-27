package personalagent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseDevinModels(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantIDs     []string
		wantErrPart string
	}{
		{
			name: "model variants",
			input: `{"families":[
				{"family_label":"Adaptive","variants":[{"model_uid":"adaptive","label":"Adaptive","description":"Automatic model selection"}]},
				{"family_label":"GPT","variants":[{"model_uid":"gpt-codex","label":"GPT Codex","description":null},{"model_uid":"gpt-empty-label","label":"","description":null},{"model_uid":"","label":"Missing id"}]},
				{"family_label":"Duplicate","variants":[{"model_uid":"adaptive","label":"Duplicate"}]}
			]}`,
			wantIDs: []string{"adaptive", "gpt-codex", "gpt-empty-label"},
		},
		{
			name:        "malformed JSON",
			input:       `{"families":[`,
			wantErrPart: "decode Devin model catalog",
		},
		{
			name:        "empty catalog",
			input:       `{"families":[]}`,
			wantErrPart: "contains no models",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			models, err := parseDevinModels([]byte(test.input))
			if test.wantErrPart != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
					t.Fatalf("parseDevinModels() error = %v, want it to contain %q", err, test.wantErrPart)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseDevinModels() error = %v", err)
			}
			if len(models) != len(test.wantIDs) {
				t.Fatalf("got %d models, want %d: %+v", len(models), len(test.wantIDs), models)
			}
			for i, wantID := range test.wantIDs {
				if models[i].ID != wantID {
					t.Errorf("models[%d].ID = %q, want %q", i, models[i].ID, wantID)
				}
				if !models[i].Selectable {
					t.Errorf("models[%d] should be selectable", i)
				}
				if models[i].Default {
					t.Errorf("models[%d] should not claim an unknown CLI default", i)
				}
			}
			if models[1].Name != "GPT Codex" || models[1].Description != "GPT" {
				t.Errorf("family metadata was not applied: %+v", models[1])
			}
			if models[2].Name != "gpt-empty-label" || models[2].Description != "GPT" {
				t.Errorf("model fallback metadata was not applied: %+v", models[2])
			}
		})
	}
}

func TestDevinAdapterCatalog(t *testing.T) {
	modelsJSON := `{"families":[{"family_label":"Adaptive","variants":[{"model_uid":"adaptive","label":"Adaptive","description":"Automatic model selection"}]}]}`
	adapter := NewDevinAdapter(writeFakeDevinCLI(t, `printf '%s\n' "$@" > "$DEVIN_ARGS_FILE"
printf '%s' "$DEVIN_RESPONSE"`))
	t.Setenv("DEVIN_RESPONSE", modelsJSON)
	argsFile := filepath.Join(t.TempDir(), "args.txt")
	t.Setenv("DEVIN_ARGS_FILE", argsFile)

	catalog, err := adapter.Catalog(context.Background())
	if err != nil {
		t.Fatalf("Catalog() error = %v", err)
	}
	if catalog.ProviderID != "devin" || catalog.Source != "devin-cli:models-list-json" {
		t.Fatalf("unexpected catalog metadata: %+v", catalog)
	}
	if catalog.CLIVersion != "devin 2.1.0" {
		t.Errorf("CLI version = %q, want devin 2.1.0", catalog.CLIVersion)
	}
	if len(catalog.Models) != 1 || catalog.Models[0].ID != "adaptive" {
		t.Fatalf("unexpected catalog models: %+v", catalog.Models)
	}
	gotArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read fake CLI arguments: %v", err)
	}
	if got, want := strings.Fields(string(gotArgs)), []string{"models", "list", "--format", "json"}; !reflect.DeepEqual(got, want) {
		t.Errorf("catalog arguments = %v, want %v", got, want)
	}
}

func TestDevinAdapterCatalogErrors(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		wantErrPart string
	}{
		{
			name:        "CLI failure",
			script:      `printf 'sensitive diagnostic' >&2; exit 9`,
			wantErrPart: "check Devin CLI authentication",
		},
		{
			name:        "invalid JSON",
			script:      `printf 'not json'`,
			wantErrPart: "decode Devin model catalog",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter := NewDevinAdapter(writeFakeDevinCLI(t, test.script))
			_, err := adapter.Catalog(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("Catalog() error = %v, want it to contain %q", err, test.wantErrPart)
			}
			if strings.Contains(err.Error(), "sensitive diagnostic") {
				t.Fatalf("Catalog() leaked CLI stderr: %v", err)
			}
		})
	}

	t.Run("version failure", func(t *testing.T) {
		adapter := NewDevinAdapter(writeFakeDevinCLIScript(t, `#!/bin/sh
if [ "$1" = "--version" ]; then exit 8; fi
exit 0`))
		_, err := adapter.Catalog(context.Background())
		if err == nil || !strings.Contains(err.Error(), "read Devin CLI version") {
			t.Fatalf("Catalog() error = %v, want a Devin version failure", err)
		}
	})

	t.Run("catalog context canceled", func(t *testing.T) {
		adapter := NewDevinAdapter(writeFakeDevinCLI(t, `exit 0`))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err := adapter.Catalog(ctx)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Catalog() error = %v, want context.Canceled", err)
		}
	})

	t.Run("model query context canceled", func(t *testing.T) {
		adapter := NewDevinAdapter(writeFakeDevinCLIScript(t, `#!/bin/sh
if [ "$1" = "--version" ]; then
  printf '%s\n' 'devin 2.1.0'
  exit 0
fi
exec sleep 1`))
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		_, err := adapter.Catalog(ctx)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("Catalog() error = %v, want context.DeadlineExceeded", err)
		}
	})
}

func TestDevinAdapterRunTurn(t *testing.T) {
	tests := []struct {
		name               string
		approvalPolicy     string
		wantPermissionMode string
		modelID            string
		nativeSessionID    string
		workspace          bool
	}{
		{
			name:               "on request",
			approvalPolicy:     "on-request",
			wantPermissionMode: "auto",
			modelID:            "adaptive",
			nativeSessionID:    "session-123",
			workspace:          true,
		},
		{
			name:               "explicitly bypass approvals",
			approvalPolicy:     "never",
			wantPermissionMode: "dangerous",
			modelID:            "adaptive",
			nativeSessionID:    "session-123",
			workspace:          true,
		},
		{
			name:               "fresh session defaults",
			wantPermissionMode: "auto",
			modelID:            "adaptive",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			argsFile := filepath.Join(t.TempDir(), "args.txt")
			t.Setenv("DEVIN_ARGS_FILE", argsFile)
			t.Setenv("DEVIN_RESPONSE", "Resposta do Devin")
			adapter := NewDevinAdapter(writeFakeDevinCLI(t, `printf '%s\n' "$@" > "$DEVIN_ARGS_FILE"
printf '%s' "$DEVIN_RESPONSE"`))

			var deltas []string
			var status string
			workspace := ""
			if test.workspace {
				workspace = t.TempDir()
			}
			result, err := adapter.RunTurn(
				context.Background(),
				RunRequest{
					ProviderID:      "devin",
					ModelID:         test.modelID,
					ApprovalPolicy:  test.approvalPolicy,
					Workspace:       workspace,
					NativeSessionID: test.nativeSessionID,
					Prompt:          "-- explique o projeto",
					OnStatus:        func(value string) { status = value },
				},
				func(delta string) { deltas = append(deltas, delta) },
				nil,
			)
			if err != nil {
				t.Fatalf("RunTurn() error = %v", err)
			}
			if result.CLIVersion != "devin 2.1.0" {
				t.Errorf("CLI version = %q, want devin 2.1.0", result.CLIVersion)
			}
			if result.NativeSessionID != test.nativeSessionID {
				t.Errorf("native session ID = %q, want %q", result.NativeSessionID, test.nativeSessionID)
			}
			if got := strings.Join(deltas, ""); got != "Resposta do Devin" {
				t.Errorf("response = %q, want %q", got, "Resposta do Devin")
			}
			if want := "Devin CLI · " + test.modelID; status != want {
				t.Errorf("status = %q, want %q", status, want)
			}

			gotArgsData, err := os.ReadFile(argsFile)
			if err != nil {
				t.Fatalf("read fake CLI arguments: %v", err)
			}
			gotArgs := strings.Split(strings.TrimSuffix(string(gotArgsData), "\n"), "\n")
			wantArgs := make([]string, 0, 10)
			if test.modelID != "" {
				wantArgs = append(wantArgs, "--model", test.modelID)
			}
			wantArgs = append(wantArgs, "--permission-mode", test.wantPermissionMode)
			if test.nativeSessionID != "" {
				wantArgs = append(wantArgs, "--resume", test.nativeSessionID)
			}
			wantArgs = append(wantArgs, "--print", "--", "-- explique o projeto")
			if !reflect.DeepEqual(gotArgs, wantArgs) {
				t.Errorf("CLI arguments = %q, want %q", gotArgs, wantArgs)
			}
		})
	}
}

func TestDevinAdapterRejectsEmptyResponseAndFailedCommand(t *testing.T) {
	tests := []struct {
		name        string
		script      string
		response    string
		wantErrPart string
	}{
		{
			name:        "empty response",
			script:      `printf '%s' "$DEVIN_RESPONSE"`,
			wantErrPart: "empty response",
		},
		{
			name:        "failed command",
			script:      `printf 'sensitive diagnostic' >&2; exit 7`,
			wantErrPart: "status 7",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("DEVIN_RESPONSE", test.response)
			adapter := NewDevinAdapter(writeFakeDevinCLI(t, test.script))
			_, err := adapter.RunTurn(context.Background(), RunRequest{
				ApprovalPolicy: "on-request",
				Workspace:      t.TempDir(),
				Prompt:         "test",
			}, func(string) {}, nil)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("RunTurn() error = %v, want it to contain %q", err, test.wantErrPart)
			}
			if strings.Contains(err.Error(), "sensitive diagnostic") {
				t.Fatalf("RunTurn() leaked CLI stderr: %v", err)
			}
		})
	}
}

func TestDevinAdapterRejectsUnsupportedSettings(t *testing.T) {
	tests := []struct {
		name        string
		policy      string
		effort      string
		wantErrPart string
	}{
		{name: "unknown approval policy", policy: "ask-always", wantErrPart: "unsupported Devin approval policy"},
		{name: "reasoning effort", policy: "on-request", effort: "high", wantErrPart: "does not expose a separate reasoning effort"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter := NewDevinAdapter("not-installed")
			_, err := adapter.RunTurn(context.Background(), RunRequest{
				ApprovalPolicy:  test.policy,
				ReasoningEffort: test.effort,
			}, func(string) {}, nil)
			if err == nil || !strings.Contains(err.Error(), test.wantErrPart) {
				t.Fatalf("RunTurn() error = %v, want it to contain %q", err, test.wantErrPart)
			}
		})
	}
}

func TestProviderRegistryDetectsDevinCLI(t *testing.T) {
	binDir := t.TempDir()
	binary := filepath.Join(binDir, "devin")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake Devin CLI: %v", err)
	}
	t.Setenv("PATH", binDir)

	registry := NewProviderRegistry()
	providers := registry.List()
	for _, provider := range providers {
		if provider.ID != "devin" {
			continue
		}
		if !provider.Detected || !provider.Supported || provider.Executable != binary {
			t.Fatalf("unexpected Devin provider status: %+v", provider)
		}
		if !reflect.DeepEqual(provider.Capabilities, []string{"models"}) {
			t.Errorf("Devin capabilities = %v, want [models]", provider.Capabilities)
		}
		if _, ok := registry.Adapter("devin"); !ok {
			t.Fatal("Devin adapter was not registered")
		}
		return
	}
	t.Fatal("Devin was not listed as a provider")
}

func writeFakeDevinCLI(t *testing.T, body string) string {
	t.Helper()
	return writeFakeDevinCLIScript(t, "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  printf '%s\\n' 'devin 2.1.0'\n  exit 0\nfi\n"+body+"\n")
}

func writeFakeDevinCLIScript(t *testing.T, script string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "devin")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake Devin CLI: %v", err)
	}
	return path
}
