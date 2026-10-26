package agents

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

func TestDoctorView(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		want   DoctorView
	}{
		{
			name:   "missing",
			status: Status{Name: "Claude CLI"},
			want:   DoctorView{Name: "Claude CLI", Status: "skipped", Message: "Claude CLI is not installed"},
		},
		{
			name:   "version",
			status: Status{Name: "Codex CLI", Installed: true, Version: "codex-cli 1.2.3"},
			want:   DoctorView{Name: "Codex CLI", Status: "ok", Message: "Codex CLI codex-cli 1.2.3"},
		},
		{
			name:   "unknown version",
			status: Status{Binary: "claude", Installed: true, Detail: "timeout"},
			want:   DoctorView{Name: "claude", Status: "ok", Message: "claude is installed (version unknown: timeout)"},
		},
		{
			name:   "unknown version without detail",
			status: Status{Name: "Claude CLI", Installed: true},
			want:   DoctorView{Name: "Claude CLI", Status: "ok", Message: "Claude CLI is installed (version unknown)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.status.Doctor()
			if got != tt.want || got.Fatal {
				t.Fatalf("got = %+v", got)
			}
		})
	}
}

func TestDetectInjected(t *testing.T) {
	t.Run("not installed checker", func(t *testing.T) {
		installed := false
		runner := &scriptedRunner{installed: &installed}
		got := Detect(context.Background(), runner)
		if len(got) != 2 || got[0].Installed || got[1].Installed || len(runner.specs) != 0 {
			t.Fatalf("got = %+v specs=%d", got, len(runner.specs))
		}
		if got[0].Doctor().Status != "skipped" || got[0].Doctor().Fatal {
			t.Fatalf("doctor = %+v", got[0].Doctor())
		}
	})

	t.Run("not installed start", func(t *testing.T) {
		runner := &startOnlyRunner{err: fmt.Errorf("%w: claude", ErrNotInstalled)}
		got := Detect(nil, runner) //nolint:staticcheck // nil context is the condition under test
		if len(got) != 2 || got[0].Installed || got[1].Installed {
			t.Fatalf("got = %+v", got)
		}
		if len(runner.specs) != 2 || runner.specs[0].Args[0] != "--version" {
			t.Fatalf("specs = %#v", runner.specs)
		}
	})

	t.Run("version on stderr", func(t *testing.T) {
		runner := &scriptedRunner{proc: &memProc{stderr: "\ncodex-cli 1.2.3\nextra"}}
		got := Detect(context.Background(), runner)
		for _, status := range got {
			if !status.Installed || status.Version != "codex-cli 1.2.3" {
				t.Fatalf("status = %+v", status)
			}
			if status.Doctor().Fatal || status.Doctor().Status != "ok" {
				t.Fatalf("doctor = %+v", status.Doctor())
			}
		}
	})

	t.Run("version failed", func(t *testing.T) {
		runner := &scriptedRunner{proc: &memProc{code: 2, wait: errors.New("exit 2")}}
		got := Detect(context.Background(), runner)
		view := got[0].Doctor()
		if !got[0].Installed || got[0].Version != "" || view.Status != "ok" || view.Fatal || !strings.Contains(view.Message, "version unknown") {
			t.Fatalf("status = %+v doctor = %+v", got[0], view)
		}
	})

	t.Run("version code only", func(t *testing.T) {
		runner := &scriptedRunner{proc: &memProc{code: 9}}
		got := Detect(context.Background(), runner)
		if got[0].Detail != "version command failed" {
			t.Fatalf("detail = %q", got[0].Detail)
		}
	})

	t.Run("start error", func(t *testing.T) {
		runner := &scriptedRunner{startErr: errors.New("boom")}
		got := Detect(context.Background(), runner)
		if !got[0].Installed || !strings.Contains(got[0].Detail, "boom") {
			t.Fatalf("status = %+v", got[0])
		}
	})

	t.Run("probe timeout", func(t *testing.T) {
		previous := probeTimeout
		probeTimeout = 20 * time.Millisecond
		t.Cleanup(func() { probeTimeout = previous })
		got := Detect(context.Background(), blockingRunner{})
		if !got[0].Installed || !strings.Contains(got[0].Detail, context.DeadlineExceeded.Error()) {
			t.Fatalf("status = %+v", got[0])
		}
		if got[0].Doctor().Fatal {
			t.Fatal("timeout was fatal")
		}
	})
}

func TestFirstLine(t *testing.T) {
	if firstLine(" \n ") != "" || firstLine("\r\nsecond") != "second" {
		t.Fatalf("blank = %q %q", firstLine(" \n "), firstLine("\r\nsecond"))
	}
	if firstLine("  one  \r\n two") != "one" {
		t.Fatalf("line = %q", firstLine("  one  \r\n two"))
	}
	long := strings.Repeat("q", 250)
	if len(firstLine(long)) != 200 {
		t.Fatalf("len = %d", len(firstLine(long)))
	}
}

type scriptedRunner struct {
	installed *bool
	startErr  error
	proc      Proc
	specs     []Spec
}

func (s *scriptedRunner) Installed(string) bool {
	if s.installed == nil {
		return true
	}
	return *s.installed
}

func (s *scriptedRunner) Start(ctx context.Context, spec Spec) (Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.specs = append(s.specs, spec)
	if s.startErr != nil {
		return nil, s.startErr
	}
	if s.proc == nil {
		s.proc = &memProc{}
	}
	return s.proc, nil
}

type startOnlyRunner struct {
	err   error
	specs []Spec
}

func (s *startOnlyRunner) Start(ctx context.Context, spec Spec) (Proc, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.specs = append(s.specs, spec)
	if s.err != nil {
		return nil, s.err
	}
	return &memProc{stdout: "1.0.0"}, nil
}

type blockingRunner struct{}

func (blockingRunner) Installed(string) bool { return true }

func (blockingRunner) Start(ctx context.Context, _ Spec) (Proc, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

type memProc struct {
	stdout string
	stderr string
	code   int
	wait   error
}

func (p *memProc) Stdout() io.Reader { return strings.NewReader(p.stdout) }

func (p *memProc) Stderr() io.Reader { return strings.NewReader(p.stderr) }

func (p *memProc) Wait() (int, error) { return p.code, p.wait }
