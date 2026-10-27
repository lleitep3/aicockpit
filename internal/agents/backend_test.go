package agents

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeTask(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "work space")
	if err := mkdir(nested); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		task    Task
		wantErr error
		check   func(t *testing.T, got Task)
	}{
		{
			name: "prompt and model",
			task: Task{Prompt: "  hello \n world  ", Model: " gpt-5.4 ", WorkDir: nested},
			check: func(t *testing.T, got Task) {
				if got.Prompt != "hello \n world" {
					t.Fatalf("prompt = %q", got.Prompt)
				}
				if got.Model != "gpt-5.4" {
					t.Fatalf("model = %q", got.Model)
				}
				if !filepath.IsAbs(got.WorkDir) || !strings.Contains(got.WorkDir, "work space") {
					t.Fatalf("workdir = %q", got.WorkDir)
				}
			},
		},
		{
			name: "aliases",
			task: Task{Prompt: "hi", Model: "sonnet"},
			check: func(t *testing.T, got Task) {
				if got.Model != "sonnet" || got.WorkDir != "" {
					t.Fatalf("got %+v", got)
				}
			},
		},
		{name: "empty prompt", task: Task{Prompt: " \n "}, wantErr: ErrEmptyPrompt},
		{name: "prompt flag", task: Task{Prompt: "-rf"}, wantErr: ErrInvalidPrompt},
		{name: "prompt nul", task: Task{Prompt: "bad\x00"}, wantErr: ErrInvalidPrompt},
		{name: "model space", task: Task{Prompt: "hi", Model: "gpt 5"}, wantErr: ErrInvalidModel},
		{name: "model dash", task: Task{Prompt: "hi", Model: "-m"}, wantErr: ErrInvalidModel},
		{name: "model newline", task: Task{Prompt: "hi", Model: "gpt\n5"}, wantErr: ErrInvalidModel},
		{name: "missing dir", task: Task{Prompt: "hi", WorkDir: filepath.Join(dir, "missing")}, wantErr: ErrInvalidWorkDir},
		{name: "file dir", task: Task{Prompt: "hi", WorkDir: writeTempFile(t, dir)}, wantErr: ErrInvalidWorkDir},
		{name: "nul dir", task: Task{Prompt: "hi", WorkDir: "bad\x00"}, wantErr: ErrInvalidWorkDir},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeTask(tt.task)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, got)
			}
		})
	}
}

func TestBindContext(t *testing.T) {
	_, _, err := bindContext(nil, 0) //nolint:staticcheck // nil context is the condition under test
	if !errors.Is(err, ErrNilContext) {
		t.Fatalf("nil ctx: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = bindContext(ctx, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled: %v", err)
	}
	_, _, err = bindContext(context.Background(), -1)
	if !errors.Is(err, ErrInvalidTimeout) {
		t.Fatalf("timeout: %v", err)
	}
	child, cancelChild, err := bindContext(context.Background(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cancelChild()
	if child.Err() == nil {
		t.Fatal("expected child to cancel")
	}
	caller := context.Background()
	child, stop, err := bindContext(caller, 0)
	if err != nil || child == nil {
		t.Fatal(err)
	}
	stop()
	if child.Err() == nil {
		t.Fatal("zero timeout did not honor cancel")
	}
	if caller.Err() != nil {
		t.Fatal("child cancel stopped the caller")
	}
}

func TestEventClone(t *testing.T) {
	ev := Event{Raw: []byte(`{"a":1}`), Text: "x"}
	cloned := ev.clone()
	cloned.Raw[0] = ' '
	if ev.Raw[0] != '{' {
		t.Fatal("clone shares raw bytes")
	}
	empty := Event{}.clone()
	if len(empty.Raw) != 0 {
		t.Fatal("empty raw")
	}
}

func mkdir(path string) error {
	return writeDir(path)
}
