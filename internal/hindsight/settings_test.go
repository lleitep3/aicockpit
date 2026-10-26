package hindsight

import (
	"errors"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		in      Settings
		wantErr string
		check   func(t *testing.T, got Resolved)
	}{
		{
			name:    "disabled",
			wantErr: ErrNotConfigured.Error(),
		},
		{
			name: "defaults",
			in:   Settings{Enabled: true},
			check: func(t *testing.T, got Resolved) {
				t.Helper()
				if got.Image != DefaultImage || got.APIPort != DefaultAPIPort || got.UIPort != DefaultUIPort {
					t.Fatalf("ports or image = %+v", got)
				}
				if got.ContainerName != DefaultContainer || got.Volume != DefaultVolume {
					t.Fatalf("name or volume = %+v", got)
				}
				if got.LLMProvider != DefaultProvider || got.VaultNamespace != DefaultVaultNamespace {
					t.Fatalf("provider or namespace = %+v", got)
				}
				if got.Secrets[EnvLLMAPIKey] != DefaultAPIKeyName {
					t.Fatalf("api key ref = %q", got.Secrets[EnvLLMAPIKey])
				}
				if _, ok := got.Secrets[EnvDatabaseURL]; ok {
					t.Fatal("database url was accepted")
				}
				if got.HealthURL() != "http://127.0.0.1:8888/health" || got.APIURL() != "http://127.0.0.1:8888" || got.UIURL() != "http://127.0.0.1:9999" {
					t.Fatalf("urls = %s %s %s", got.HealthURL(), got.APIURL(), got.UIURL())
				}
			},
		},
		{
			name: "custom and extra secret",
			in: Settings{
				Enabled: true, Image: " ghcr.io/vectorize-io/hindsight:latest ",
				APIPort: 18088, UIPort: 18099, LLMProvider: "groq", LLMModel: "openai/gpt-oss-20b",
				ContainerName: "mem-1", Volume: "mem_data", VaultNamespace: "hindsight",
				APIKeyName: "llm_api_key",
				Secrets:    map[string]string{"HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY": "embed_key"},
			},
			check: func(t *testing.T, got Resolved) {
				t.Helper()
				if got.APIPort != 18088 || got.UIPort != 18099 || got.LLMProvider != "groq" || got.LLMModel != "openai/gpt-oss-20b" {
					t.Fatalf("resolved = %+v", got)
				}
				if got.Secrets[EnvLLMAPIKey] != "llm_api_key" || got.Secrets["HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY"] != "embed_key" {
					t.Fatalf("secrets = %+v", got.Secrets)
				}
			},
		},
		{
			name: "optional key",
			in:   Settings{Enabled: true, LLMProvider: "ollama", APIKeyOptional: true},
			check: func(t *testing.T, got Resolved) {
				t.Helper()
				if _, ok := got.Secrets[EnvLLMAPIKey]; ok {
					t.Fatalf("optional key was required: %+v", got.Secrets)
				}
			},
		},
		{name: "bad api port", in: Settings{Enabled: true, APIPort: 70000}, wantErr: "invalid api port"},
		{name: "bad ui port", in: Settings{Enabled: true, UIPort: -1}, wantErr: "invalid ui port"},
		{name: "same port", in: Settings{Enabled: true, APIPort: 8888, UIPort: 8888}, wantErr: "must differ"},
		{name: "bad image", in: Settings{Enabled: true, Image: "Bad Image"}, wantErr: "invalid hindsight image"},
		{name: "bad container", in: Settings{Enabled: true, ContainerName: "has space"}, wantErr: "invalid container name"},
		{name: "host path volume", in: Settings{Enabled: true, Volume: "/tmp/hindsight"}, wantErr: "named volume"},
		{name: "bad provider", in: Settings{Enabled: true, LLMProvider: "OpenAI"}, wantErr: "invalid llm provider"},
		{name: "bad model", in: Settings{Enabled: true, LLMModel: "gpt 4"}, wantErr: "invalid llm model"},
		{name: "bad namespace", in: Settings{Enabled: true, VaultNamespace: "Bad"}, wantErr: "invalid vault namespace"},
		{name: "bad api key name", in: Settings{Enabled: true, APIKeyName: "LLM"}, wantErr: "invalid api key name"},
		{name: "database url", in: Settings{Enabled: true, Secrets: map[string]string{EnvDatabaseURL: "db_url"}}, wantErr: "embedded pg0"},
		{name: "bad env", in: Settings{Enabled: true, Secrets: map[string]string{"PATH": "path_key"}}, wantErr: "refusing environment"},
		{name: "bad vault key", in: Settings{Enabled: true, Secrets: map[string]string{"HINDSIGHT_API_LLM_MODEL": "Bad Key"}}, wantErr: "invalid vault key"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.in.Resolve()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want substring %q", err, tt.wantErr)
				}
				if tt.name == "disabled" && !errors.Is(err, ErrNotConfigured) {
					t.Fatal("disabled settings were not ErrNotConfigured")
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			tt.check(t, got)
		})
	}
}

func TestResolvedSpec(t *testing.T) {
	resolved, err := (Settings{Enabled: true, LLMModel: "gpt-4o-mini"}).Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	secret := "sk-spec-secret"
	spec, err := resolved.Spec(map[string]string{EnvLLMAPIKey: secret})
	if err != nil {
		t.Fatalf("spec: %v", err)
	}
	args, err := spec.args()
	if err != nil {
		t.Fatalf("args: %v", err)
	}
	joined := strings.Join(args, "\n")
	for _, unwanted := range []string{"--env-file", EnvDatabaseURL, "\n" + secret} {
		if strings.Contains(joined, unwanted) && unwanted != "\n"+secret {
			t.Fatalf("args contain %q", unwanted)
		}
	}
	if strings.Contains(joined, "--env-file") || strings.Contains(joined, EnvDatabaseURL) {
		t.Fatalf("args leaked a forbidden flag:\n%s", redact(joined, []string{secret}))
	}
	if !strings.Contains(joined, "-e\n"+EnvLLMAPIKey+"="+secret) {
		t.Fatal("api key was not passed with -e")
	}
	if !strings.Contains(joined, "-v\n"+DefaultVolume+":"+PG0Mount) {
		t.Fatal("pg0 volume mount missing")
	}
	if !strings.Contains(joined, "127.0.0.1:8888:8888") || !strings.Contains(joined, "127.0.0.1:9999:9999") {
		t.Fatal("published ports missing")
	}
	if args[0] != "run" || args[1] != "-d" {
		t.Fatalf("args start = %v", args[:2])
	}
	if len(spec.SecretValues()) != 1 || spec.SecretValues()[0] != secret {
		t.Fatalf("secret values = %v", spec.SecretValues())
	}

	if _, err := resolved.Spec(map[string]string{}); err == nil {
		t.Fatal("missing secret was accepted")
	}
	if _, err := resolved.Spec(map[string]string{EnvLLMAPIKey: "bad\nsecret"}); err == nil {
		t.Fatal("newline secret was accepted")
	}
	spec.Env = append(spec.Env, EnvVar{Name: EnvDatabaseURL, Value: "postgres"})
	if _, err := spec.args(); err == nil {
		t.Fatal("database url args were accepted")
	}
	spec.Env = []EnvVar{{Name: "PATH", Value: "x"}}
	if _, err := spec.args(); err == nil {
		t.Fatal("non-hindsight env was accepted")
	}
	empty := Spec{}
	if _, err := empty.args(); err == nil {
		t.Fatal("empty spec was accepted")
	}
	spec.Env = []EnvVar{{Name: EnvLLMAPIKey, Value: "bad\nsecret"}}
	spec.Mount = PG0Mount
	if _, err := spec.args(); err == nil {
		t.Fatal("newline env was accepted by args")
	}
	spec.Mount = "../data"
	spec.Env = nil
	spec.Name = "n"
	spec.Image = DefaultImage
	spec.Volume = DefaultVolume
	if _, err := spec.args(); err == nil {
		t.Fatal("parent mount was accepted")
	}
}
