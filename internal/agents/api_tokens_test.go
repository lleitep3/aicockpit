package agents

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/vault"
	"github.com/zalando/go-keyring"
)

func TestTokenStatusDoctor(t *testing.T) {
	secret := testToken
	tests := []struct {
		name   string
		status TokenStatus
		want   string
		ok     bool
	}{
		{name: "configured", status: TokenStatus{Name: "Anthropic API", Configured: true}, want: "Anthropic API token is configured", ok: true},
		{name: "missing", status: TokenStatus{Name: "OpenAI API"}, want: "OpenAI API token is not configured"},
		{name: "locked", status: TokenStatus{Name: "Ollama API", Locked: true}, want: "vault is locked"},
		{name: "unavailable", status: TokenStatus{Name: "LM Studio API", Unavailable: true}, want: "token status unavailable"},
		{name: "invalid", status: TokenStatus{Name: "API provider", Invalid: true}, want: "configuration is invalid"},
		{name: "blank name", status: TokenStatus{Configured: true}, want: "API provider token is configured", ok: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			view := tt.status.Doctor()
			if view.Fatal {
				t.Fatal("token row is fatal")
			}
			if tt.ok && view.Status != "ok" {
				t.Fatalf("status = %s", view.Status)
			}
			if !tt.ok && view.Status != "skipped" {
				t.Fatalf("status = %s", view.Status)
			}
			if !strings.Contains(view.Message, tt.want) || strings.Contains(view.Message, secret) {
				t.Fatalf("message = %q", view.Message)
			}
		})
	}
}

func TestProbeTokens(t *testing.T) {
	secret := testToken
	probe := probeMap{
		"team/prod_key": true,
	}
	var missing context.Context
	got := ProbeTokens(missing, APISettings{Provider: "openai", VaultNamespace: "team", VaultKey: "prod_key"}, probe)
	if len(got) != len(KnownAPIProviders()) {
		t.Fatalf("rows = %d", len(got))
	}
	for _, status := range got {
		view := status.Doctor()
		if strings.Contains(view.Message, secret) {
			t.Fatal("secret leaked into doctor row")
		}
		switch status.Provider {
		case ProviderAnthropic:
			if status.Configured || status.Key != "anthropic_api_key" {
				t.Fatalf("anthropic = %+v", status)
			}
		case ProviderOpenAI:
			if !status.Configured || status.Namespace != "team" || status.Key != "prod_key" {
				t.Fatalf("openai = %+v", status)
			}
		}
	}

	invalid := ProbeTokens(context.Background(), APISettings{Provider: secret, VaultNamespace: secret}, probe)
	if len(invalid) == 0 || !invalid[0].Invalid {
		t.Fatalf("invalid = %+v", invalid)
	}
	if invalid[0].Key != "" || strings.Contains(invalid[0].Doctor().Message, secret) {
		t.Fatal("invalid provider echoed the value")
	}

	badRef := ProbeTokens(context.Background(), APISettings{Provider: "openai", VaultKey: "Bad-" + secret}, probe)
	var openai TokenStatus
	for _, status := range badRef {
		if status.Provider == ProviderOpenAI {
			openai = status
		}
	}
	if !openai.Invalid || openai.Key != "" || strings.Contains(openai.Doctor().Message, secret) {
		t.Fatalf("bad key = %+v", openai)
	}

	locked := ProbeTokens(context.Background(), APISettings{}, lockedProbe{err: errors.New(secret)})
	if !locked[0].Locked || strings.Contains(locked[0].Doctor().Message, secret) {
		t.Fatalf("locked = %+v msg %q", locked[0], locked[0].Doctor().Message)
	}
	down := ProbeTokens(context.Background(), APISettings{}, lockedProbe{err: errors.New("index " + secret), other: true})
	if !down[0].Unavailable || strings.Contains(down[0].Doctor().Message, secret) {
		t.Fatal("unavailable row leaked")
	}
}

func TestVaultTokenSourceGuards(t *testing.T) {
	secret := testToken
	src := VaultTokenSource{
		Access: func() error { return errors.New(secret) },
		Read: func(string, string) (string, error) {
			t.Error("read called while locked")
			return secret, nil
		},
	}
	_, err := src.Get(context.Background(), "agents", "anthropic_api_key")
	if !errors.Is(err, ErrVaultLocked) || strings.Contains(err.Error(), secret) {
		t.Fatalf("err = %v", err)
	}
	var missing context.Context
	_, err = src.Get(missing, "agents", "anthropic_api_key")
	if !errors.Is(err, ErrNilContext) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = src.Get(ctx, "agents", "anthropic_api_key")
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_, err = src.Get(context.Background(), "Bad", secret)
	if !errors.Is(err, ErrInvalidVaultRef) || strings.Contains(err.Error(), secret) {
		t.Fatalf("err = %v", err)
	}

	src.Access = func() error { return nil }
	src.Read = func(string, string) (string, error) {
		return "", errors.New("boom " + secret)
	}
	_, err = src.Get(context.Background(), "agents", "anthropic_api_key")
	if !errors.Is(err, ErrTokenUnavailable) || strings.Contains(err.Error(), secret) {
		t.Fatalf("err = %v", err)
	}
	src.Read = func(string, string) (string, error) { return "bad\n" + secret, nil }
	_, err = src.Get(context.Background(), "agents", "anthropic_api_key")
	if !errors.Is(err, ErrTokenMissing) || strings.Contains(err.Error(), secret) {
		t.Fatalf("err = %v", err)
	}
	src.Read = func(string, string) (string, error) { return "   ", nil }
	_, err = src.Get(context.Background(), "agents", "anthropic_api_key")
	if !errors.Is(err, ErrTokenMissing) {
		t.Fatal(err)
	}

	src.List = func(string) ([]string, error) { return nil, errors.New(secret) }
	ok, err := src.Configured(context.Background(), "agents", "anthropic_api_key")
	if ok || !errors.Is(err, ErrTokenUnavailable) || strings.Contains(err.Error(), secret) {
		t.Fatalf("configured err = %v", err)
	}
	src.List = func(string) ([]string, error) { return []string{"other"}, nil }
	ok, err = src.Configured(context.Background(), "agents", "anthropic_api_key")
	if err != nil || ok {
		t.Fatalf("configured = %v %v", ok, err)
	}
	src.List = func(string) ([]string, error) { return []string{"anthropic_api_key"}, nil }
	ok, err = src.Configured(context.Background(), "agents", "anthropic_api_key")
	if err != nil || !ok {
		t.Fatalf("configured = %v %v", ok, err)
	}
}

func TestVaultTokenSourceKeyring(t *testing.T) {
	keyring.MockInit()
	t.Setenv("HOME", t.TempDir())
	secret := testToken
	ctx := context.Background()
	src := NewVaultTokenSource()

	if err := src.Access(); !errors.Is(err, ErrVaultLocked) {
		t.Fatalf("fresh vault access = %v", err)
	}
	statuses := ProbeTokens(ctx, APISettings{Provider: ProviderAnthropic}, nil)
	for _, status := range statuses {
		if !status.Locked || strings.Contains(status.Doctor().Message, secret) {
			t.Fatalf("status = %+v", status)
		}
	}

	lm := vault.NewLockManager("")
	if err := lm.Unlock("test"); err != nil {
		t.Fatal(err)
	}
	if err := vault.NewNamespacedVault(DefaultAPIVaultNamespace).Set("anthropic_api_key", secret); err != nil {
		t.Fatal(err)
	}
	if err := vault.NewNamespacedVault("team").Set("prod_key", secret); err != nil {
		t.Fatal(err)
	}
	got, err := src.Get(ctx, DefaultAPIVaultNamespace, "anthropic_api_key")
	if err != nil || got != secret {
		t.Fatal("vault get failed")
	}
	ok, err := src.Configured(ctx, DefaultAPIVaultNamespace, "anthropic_api_key")
	if err != nil || !ok {
		t.Fatal("index did not list the key")
	}
	statuses = ProbeTokens(ctx, APISettings{Provider: ProviderOpenAI, VaultNamespace: "team", VaultKey: "prod_key"}, nil)
	for _, status := range statuses {
		if strings.Contains(status.Doctor().Message, secret) || strings.Contains(status.Key, secret) {
			t.Fatal("probe leaked the secret")
		}
		if status.Provider == ProviderOpenAI && !status.Configured {
			t.Fatal("openai token was not reported")
		}
		if status.Provider == ProviderOllama && status.Configured {
			t.Fatal("ollama token was reported")
		}
	}

	if err := lm.Lock("test"); err != nil {
		t.Fatal(err)
	}
	_, err = src.Get(ctx, DefaultAPIVaultNamespace, "anthropic_api_key")
	if !errors.Is(err, ErrVaultLocked) || strings.Contains(err.Error(), secret) {
		t.Fatalf("locked get = %v", err)
	}
}

type probeMap map[string]bool

func (p probeMap) Configured(_ context.Context, namespace, key string) (bool, error) {
	return p[namespace+"/"+key], nil
}

type lockedProbe struct {
	err   error
	other bool
}

func (p lockedProbe) Configured(context.Context, string, string) (bool, error) {
	if p.other {
		return false, p.err
	}
	return false, ErrVaultLocked
}
