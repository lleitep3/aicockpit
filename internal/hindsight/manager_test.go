package hindsight

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type spySecrets struct {
	values map[string]string
	err    error
	calls  []string
}

func (s *spySecrets) Get(ctx context.Context, namespace, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.calls = append(s.calls, namespace+"/"+key)
	if s.err != nil {
		return s.values[key], s.err
	}
	return s.values[key], nil
}

type stubProbe struct {
	err  error
	urls []string
}

func (p *stubProbe) Probe(ctx context.Context, url string) error {
	p.urls = append(p.urls, url)
	if err := ctx.Err(); err != nil {
		return err
	}
	return p.err
}

func enabled() Settings {
	return Settings{Enabled: true, LLMProvider: "openai"}
}

func TestManagerLifecycle(t *testing.T) {
	secret := "sk-vault-secret"
	ctx := context.Background()

	tests := []struct {
		name      string
		settings  Settings
		prepare   func(*FakeRuntime)
		secrets   *spySecrets
		probe     Prober
		nilRun    bool
		op        string
		wantErr   string
		wantState string
		wantOK    bool
		noSecret  bool
		noGet     bool
	}{
		{name: "status disabled", op: "status", wantState: "not_configured"},
		{name: "up disabled", op: "up", wantErr: "not configured"},
		{name: "down disabled", op: "down", wantState: "not_configured"},
		{name: "status absent", settings: enabled(), op: "status", wantState: "absent"},
		{name: "status healthy", settings: enabled(), prepare: running, op: "status", wantState: "running", wantOK: true},
		{name: "status unhealthy", settings: enabled(), prepare: running, probe: &stubProbe{err: errors.New("HTTP 503")}, op: "status", wantState: "running"},
		{name: "status inspect error", settings: enabled(), prepare: func(f *FakeRuntime) {
			f.InspectErr = errors.New("daemon down")
			f.InspectErrOn = 1
		}, op: "status", wantErr: "daemon down"},
		{name: "status nil runtime", settings: enabled(), nilRun: true, op: "status", wantErr: "docker not found"},
		{name: "status bad config", settings: Settings{Enabled: true, APIPort: 99, UIPort: 99}, op: "status", wantErr: "must differ"},
		{name: "up starts", settings: enabled(), op: "up", wantState: "running", wantOK: true, noSecret: true},
		{name: "up already running skips vault", settings: enabled(), prepare: running, op: "up", wantOK: true, noGet: true},
		{name: "up recreates stopped", settings: enabled(), prepare: func(f *FakeRuntime) {
			f.Containers = map[string]ContainerState{DefaultContainer: {Status: "exited"}}
		}, op: "up", wantOK: true},
		{name: "up secret error", settings: enabled(), secrets: &spySecrets{err: errors.New("locked")}, op: "up", wantErr: "locked"},
		{name: "up run redacts", settings: enabled(), prepare: func(f *FakeRuntime) { f.RunErr = errors.New("engine rejected " + secret) }, op: "up", wantErr: "[redacted]", noSecret: true},
		{name: "up nil runtime", settings: enabled(), nilRun: true, op: "up", wantErr: "docker not found"},
		{name: "up remove stopped fails", settings: enabled(), prepare: func(f *FakeRuntime) {
			f.Containers = map[string]ContainerState{DefaultContainer: {Status: "exited"}}
			f.RemoveErr = errors.New("busy")
		}, op: "up", wantErr: "busy", noGet: true},
		{name: "up inspect after start fails", settings: enabled(), prepare: func(f *FakeRuntime) {
			f.InspectErr = errors.New("inspect broke")
			f.InspectErrOn = 2
		}, op: "up", wantErr: "inspect broke"},
		{name: "down running keeps volume", settings: enabled(), prepare: running, op: "down", wantState: "absent"},
		{name: "down absent", settings: enabled(), op: "down", wantState: "absent"},
		{name: "down stop error", settings: enabled(), prepare: func(f *FakeRuntime) {
			running(f)
			f.StopErr = errors.New("stop failed")
		}, op: "down", wantErr: "stop failed"},
		{name: "down remove error", settings: enabled(), prepare: func(f *FakeRuntime) {
			running(f)
			f.RemoveErr = errors.New("rm failed")
		}, op: "down", wantErr: "rm failed"},
		{name: "down nil runtime", settings: enabled(), nilRun: true, op: "down", wantErr: "docker not found"},
		{name: "down bad config", settings: Settings{Enabled: true, Image: "Bad"}, op: "down", wantErr: "invalid hindsight image"},
		{name: "probe nil", settings: enabled(), prepare: running, probe: nil, op: "status", wantState: "running"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &FakeRuntime{}
			if tt.prepare != nil {
				tt.prepare(fake)
			}
			secrets := tt.secrets
			if secrets == nil {
				secrets = &spySecrets{values: map[string]string{DefaultAPIKeyName: secret}}
			}
			probe := tt.probe
			if probe == nil && tt.name != "probe nil" {
				probe = &stubProbe{}
			}
			var runtime Runtime = fake
			if tt.nilRun {
				runtime = nil
			}
			manager := NewManager(tt.settings, runtime, probe, secrets)
			report, err := callOp(ctx, manager, tt.op)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
			} else if err != nil {
				t.Fatalf("op: %v", err)
			}
			if tt.wantState != "" && report.State != tt.wantState {
				t.Fatalf("state = %s message=%s", report.State, report.Message)
			}
			if tt.wantOK && !report.Healthy {
				t.Fatalf("health = %s", report.Health)
			}
			raw, marshalErr := json.Marshal(report)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if strings.Contains(string(raw), secret) || strings.Contains(report.FormatText(), secret) {
				t.Fatalf("report leaked secret: %s", raw)
			}
			if tt.noSecret && err != nil && strings.Contains(err.Error(), secret) {
				t.Fatalf("error leaked secret: %v", err)
			}
			if tt.noGet && len(secrets.calls) != 0 {
				t.Fatalf("vault reads = %v", secrets.calls)
			}
			for _, call := range fake.Calls {
				if strings.Contains(call, "volume") {
					t.Fatalf("volume call %q", call)
				}
			}
			if tt.name == "up starts" {
				if len(fake.Specs) != 1 {
					t.Fatalf("specs = %d", len(fake.Specs))
				}
				found := false
				for _, env := range fake.Specs[0].Env {
					if env.Name == EnvDatabaseURL {
						t.Fatal("database url was set")
					}
					if env.Name == EnvLLMAPIKey && env.Secret && env.Value == secret {
						found = true
					}
				}
				if !found || fake.Specs[0].Mount != PG0Mount || fake.Specs[0].Volume != DefaultVolume {
					t.Fatalf("spec = %+v", fake.Specs[0])
				}
				if len(secrets.calls) != 1 || secrets.calls[0] != DefaultVaultNamespace+"/"+DefaultAPIKeyName {
					t.Fatalf("vault reads = %v", secrets.calls)
				}
			}
			if tt.name == "up recreates stopped" {
				joined := strings.Join(fake.Calls, ",")
				if !strings.Contains(joined, "remove ") || !strings.Contains(joined, "run ") {
					t.Fatalf("calls = %s", joined)
				}
			}
			view := report.Doctor()
			if !tt.settings.Enabled && view.Fatal {
				t.Fatal("unconfigured doctor check was fatal")
			}
			if tt.wantOK && (view.Status != "ok" || view.Fatal) {
				t.Fatalf("doctor = %+v", view)
			}
			if tt.name == "status unhealthy" && (!view.Fatal || view.Status != "error") {
				t.Fatalf("doctor = %+v", view)
			}
			if tt.name == "status absent" && view.Fix != "cockpit memory up" {
				t.Fatalf("doctor = %+v", view)
			}
		})
	}
}

func TestManagerContextAndSecrets(t *testing.T) {
	manager := NewManager(enabled(), &FakeRuntime{}, &stubProbe{}, &spySecrets{values: map[string]string{DefaultAPIKeyName: "sek"}})
	if _, err := manager.Up(nilCtx()); !errors.Is(err, ErrNilContext) {
		t.Fatalf("nil = %v", err)
	}
	if _, err := manager.Down(nilCtx()); !errors.Is(err, ErrNilContext) {
		t.Fatalf("nil down = %v", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := manager.Status(canceled); err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled status = %v", err)
	}
	if _, err := manager.Up(canceled); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled up = %v", err)
	}

	newline := NewManager(enabled(), &FakeRuntime{}, &stubProbe{}, &spySecrets{values: map[string]string{DefaultAPIKeyName: "bad\nsecret"}})
	_, err := newline.Up(context.Background())
	if err == nil || strings.Contains(err.Error(), "bad") {
		t.Fatalf("newline error = %v", err)
	}
	empty := NewManager(enabled(), &FakeRuntime{}, &stubProbe{}, &spySecrets{})
	if _, err := empty.Up(context.Background()); err == nil {
		t.Fatal("empty secret was accepted")
	}
	if _, err := NewManager(enabled(), &FakeRuntime{}, &stubProbe{}, nil).Up(context.Background()); err == nil || !strings.Contains(err.Error(), "secret source") {
		t.Fatal("nil source was accepted")
	}

	optional := Settings{Enabled: true, LLMProvider: "ollama", APIKeyOptional: true}
	optRuntime := &FakeRuntime{}
	opt := NewManager(optional, optRuntime, &stubProbe{}, nil)
	report, err := opt.Up(context.Background())
	if err != nil || !report.Healthy {
		t.Fatalf("optional = %+v %v", report, err)
	}
	for _, env := range optRuntime.Specs[0].Env {
		if env.Secret {
			t.Fatal("optional provider received a secret")
		}
	}

	shared := enabled()
	shared.Secrets = map[string]string{
		"HINDSIGHT_API_EMBEDDINGS_OPENAI_API_KEY": "llm_api_key",
		EnvLLMAPIKey: "llm_api_key",
	}
	sharedRuntime := &FakeRuntime{}
	sharedSecrets := &spySecrets{values: map[string]string{"llm_api_key": secretValue()}}
	sharedManager := NewManager(shared, sharedRuntime, &stubProbe{}, sharedSecrets)
	if _, err := sharedManager.Up(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(sharedSecrets.calls) != 1 {
		t.Fatalf("shared key reads = %v", sharedSecrets.calls)
	}
	if len(sharedRuntime.Specs[0].Env) < 3 {
		t.Fatalf("env = %+v", sharedRuntime.Specs[0].Env)
	}

	leaky := &spySecrets{values: map[string]string{DefaultAPIKeyName: "hidden-value"}, err: errors.New("failed hidden-value")}
	if _, err := NewManager(enabled(), &FakeRuntime{}, &stubProbe{}, leaky).Up(context.Background()); err == nil || strings.Contains(err.Error(), "hidden-value") {
		t.Fatalf("error = %v", err)
	}

	text := notConfiguredReport().FormatText()
	if !strings.Contains(text, "configured: false") {
		t.Fatal(text)
	}
	blank := Report{}.Doctor()
	if blank.Status != "skipped" || blank.Fatal {
		t.Fatalf("blank doctor = %+v", blank)
	}
	runningReport := Report{Configured: true, Running: true, Healthy: false}
	if view := runningReport.Doctor(); view.Fix != "cockpit memory status" {
		t.Fatalf("view = %+v", view)
	}
	downReport := Report{Configured: true}
	if view := downReport.Doctor(); !strings.Contains(view.Message, "not running") {
		t.Fatalf("view = %+v", view)
	}
	exited, err := NewManager(enabled(), &FakeRuntime{Containers: map[string]ContainerState{DefaultContainer: {Status: ""}}}, &stubProbe{}, nil).Status(context.Background())
	if err != nil || exited.State == "" {
		t.Fatalf("exited = %+v %v", exited, err)
	}
}

func running(f *FakeRuntime) {
	f.Containers = map[string]ContainerState{DefaultContainer: {ID: "1", Status: "running", Running: true}}
}

func secretValue() string { return "shared-secret" }

func callOp(ctx context.Context, m *Manager, op string) (Report, error) {
	switch op {
	case "up":
		return m.Up(ctx)
	case "down":
		return m.Down(ctx)
	default:
		return m.Status(ctx)
	}
}
