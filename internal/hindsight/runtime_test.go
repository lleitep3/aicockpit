package hindsight

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func nilCtx() context.Context { return nil }

func TestParseInspect(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		wantErr string
		running bool
		status  string
	}{
		{name: "running", in: "running|true|abc\n", running: true, status: "running"},
		{name: "exited", in: " exited|false|def ", status: "exited"},
		{name: "parts", in: "running|true", wantErr: "unexpected docker inspect output"},
		{name: "bool", in: "running|yes|abc", wantErr: "running flag"},
		{name: "empty id", in: "running|true|", wantErr: "unexpected docker inspect output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseInspect(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got.Running != tt.running || got.Status != tt.status || got.ID == "" {
				t.Fatalf("state = %+v", got)
			}
		})
	}
}

func TestDockerCLI(t *testing.T) {
	secret := "sk-docker-secret"
	ctx := context.Background()
	spec := Spec{
		Name: DefaultContainer, Image: DefaultImage, APIPort: 8888, UIPort: 9999,
		Volume: DefaultVolume, Mount: PG0Mount, ShmSize: DefaultShmSize, Restart: DefaultRestart,
		Env: []EnvVar{{Name: EnvLLMAPIKey, Value: secret, Secret: true}, {Name: EnvLLMProvider, Value: "openai"}},
	}

	t.Run("inspect running", func(t *testing.T) {
		cli := DockerCLI{exec: func(ctx context.Context, bin string, args []string) (string, string, error) {
			if bin != "docker" || args[0] != "inspect" {
				t.Fatalf("cmd = %s %v", bin, args)
			}
			if strings.Contains(strings.Join(args, " "), "Env") {
				t.Fatal("inspect requested the environment")
			}
			return "running|true|abc123", "", nil
		}}
		state, err := cli.Inspect(ctx, DefaultContainer)
		if err != nil {
			t.Fatalf("inspect: %v", err)
		}
		if !state.Running || state.ID != "abc123" || state.Name != DefaultContainer {
			t.Fatalf("state = %+v", state)
		}
	})

	t.Run("inspect not found and missing docker and bad output", func(t *testing.T) {
		cases := []struct {
			name    string
			stderr  string
			err     error
			stdout  string
			wantIs  error
			wantSub string
		}{
			{name: "missing object", stderr: "Error: No such object: x", err: errors.New("exit status 1"), wantIs: ErrNotFound},
			{name: "missing container", stderr: "no such container", err: errors.New("exit status 1"), wantIs: ErrNotFound},
			{name: "runtime", err: ErrRuntimeMissing, wantIs: ErrRuntimeMissing},
			{name: "other", stderr: "permission denied", err: errors.New("exit status 1"), wantSub: "permission denied"},
			{name: "bad stdout", stdout: "nope", wantSub: "unexpected docker inspect"},
		}
		for _, tt := range cases {
			t.Run(tt.name, func(t *testing.T) {
				cli := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) {
					return tt.stdout, tt.stderr, tt.err
				}}
				_, err := cli.Inspect(ctx, DefaultContainer)
				if tt.wantIs != nil && !errors.Is(err, tt.wantIs) {
					t.Fatalf("error = %v", err)
				}
				if tt.wantSub != "" && (err == nil || !strings.Contains(err.Error(), tt.wantSub)) {
					t.Fatalf("error = %v", err)
				}
			})
		}
	})

	t.Run("run redacts and does not use env file", func(t *testing.T) {
		cli := DockerCLI{bin: "docker", exec: func(ctx context.Context, bin string, args []string) (string, string, error) {
			joined := strings.Join(args, " ")
			if strings.Contains(joined, "--env-file") {
				t.Fatal("env file was used")
			}
			if !strings.Contains(joined, EnvLLMAPIKey+"="+secret) {
				t.Fatal("secret was not passed as -e")
			}
			return "", "failed to start " + secret, errors.New("exit status 1")
		}}
		err := cli.Run(ctx, spec)
		if err == nil || strings.Contains(err.Error(), secret) || !strings.Contains(err.Error(), "[redacted]") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("run success and runtime missing and bad spec", func(t *testing.T) {
		ok := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) { return "id", "", nil }}
		if err := ok.Run(ctx, spec); err != nil {
			t.Fatalf("run: %v", err)
		}
		missing := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) {
			return "", "", ErrRuntimeMissing
		}}
		if err := missing.Run(ctx, spec); !errors.Is(err, ErrRuntimeMissing) {
			t.Fatalf("error = %v", err)
		}
		if err := ok.Run(ctx, Spec{}); err == nil {
			t.Fatal("empty spec ran")
		}
	})

	t.Run("stop and remove", func(t *testing.T) {
		var got []string
		cli := DockerCLI{exec: func(ctx context.Context, bin string, args []string) (string, string, error) {
			got = append(got, strings.Join(args, " "))
			return "", "", nil
		}}
		if err := cli.Stop(ctx, DefaultContainer); err != nil {
			t.Fatal(err)
		}
		if err := cli.Remove(ctx, DefaultContainer); err != nil {
			t.Fatal(err)
		}
		if strings.Join(got, "|") != "stop "+DefaultContainer+"|rm "+DefaultContainer {
			t.Fatalf("calls = %v", got)
		}
		if strings.Contains(strings.Join(got, " "), "volume") {
			t.Fatal("volume was removed")
		}
		fail := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) {
			return "", strings.Repeat("x", 400), errors.New("exit status 1")
		}}
		err := fail.Stop(ctx, DefaultContainer)
		if err == nil || len(err.Error()) > 400 {
			t.Fatalf("error = %v", err)
		}
		gone := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) {
			return "", "", ErrRuntimeMissing
		}}
		if err := gone.Remove(ctx, DefaultContainer); !errors.Is(err, ErrRuntimeMissing) {
			t.Fatalf("error = %v", err)
		}
		emptyErr := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) {
			return "", "", errors.New("exit status 1")
		}}
		if err := emptyErr.Stop(ctx, "name"); err == nil || !strings.Contains(err.Error(), "exit status 1") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("nil context", func(t *testing.T) {
		cli := DockerCLI{exec: func(context.Context, string, []string) (string, string, error) {
			t.Fatal("exec called")
			return "", "", nil
		}}
		if _, err := cli.Inspect(nilCtx(), "n"); !errors.Is(err, ErrNilContext) {
			t.Fatalf("inspect = %v", err)
		}
		if err := cli.Run(nilCtx(), spec); !errors.Is(err, ErrNilContext) {
			t.Fatalf("run = %v", err)
		}
		if err := cli.Stop(nilCtx(), "n"); !errors.Is(err, ErrNilContext) {
			t.Fatalf("stop = %v", err)
		}
	})
}

func TestDefaultExec(t *testing.T) {
	stdout, stderr, err := defaultExec(context.Background(), "sh", []string{"-c", "printf out; printf err >&2"})
	if err != nil || stdout != "out" || stderr != "err" {
		t.Fatalf("stdout=%q stderr=%q err=%v", stdout, stderr, err)
	}
	if _, _, err := defaultExec(nilCtx(), "sh", []string{"-c", "true"}); !errors.Is(err, ErrNilContext) {
		t.Fatalf("nil ctx = %v", err)
	}
	if _, _, err := defaultExec(context.Background(), "cockpit-missing-binary-xyz", nil); !errors.Is(err, ErrRuntimeMissing) {
		t.Fatalf("missing = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := defaultExec(ctx, "sleep", []string{"30"}); err == nil {
		t.Fatal("canceled command succeeded")
	}
	cli := DockerCLI{}
	if cli.binary() != "docker" {
		t.Fatal(cli.binary())
	}
	if _, _, err := cli.runner()(context.Background(), "sh", []string{"-c", "exit 0"}); err != nil {
		t.Fatal(err)
	}
	if NewDockerCLI().binary() != "docker" {
		t.Fatal("NewDockerCLI")
	}
	if isNotFound("no such object", nil) {
		t.Fatal("nil error was treated as not found")
	}
}

func TestRedact(t *testing.T) {
	if redact("", []string{"a"}) != "" {
		t.Fatal("empty text")
	}
	if got := redact("a secret a", []string{"", "secret"}); got != "a [redacted] a" {
		t.Fatalf("redact = %q", got)
	}
	if err := redactError(nil, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	plain := errors.New("plain")
	if !errors.Is(redactError(plain, nil), plain) {
		t.Fatal("unchanged error was replaced")
	}
	if err := wrapOp("up", nil, nil); err != nil {
		t.Fatal(err)
	}
	wrapped := wrapOp("up", errors.New("nope"), nil)
	if !errors.Is(wrapped, errors.Unwrap(wrapped)) && !strings.Contains(wrapped.Error(), "nope") {
		t.Fatal(wrapped)
	}
	leaked := wrapOp("up", errors.New("saw sk-secret"), []string{"sk-secret"})
	if strings.Contains(leaked.Error(), "sk-secret") || errors.Unwrap(leaked) != nil {
		t.Fatalf("leaked = %v unwrap=%v", leaked, errors.Unwrap(leaked))
	}
}
