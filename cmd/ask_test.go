package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/agents"
	"github.com/lleitep3/aicockpit/internal/ask"
	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/memory"
	"github.com/lleitep3/aicockpit/internal/memoryhindsight"
	"github.com/spf13/cobra"
)

type askWorker struct {
	id   string
	text string
	task agents.Task
	err  error
}

func (w *askWorker) ID() string {
	if w.id == "" {
		return "claude"
	}
	return w.id
}

func (w *askWorker) Run(_ context.Context, task agents.Task) (*agents.Result, error) {
	w.task = task
	if w.err != nil {
		return nil, w.err
	}
	if task.OnEvent != nil {
		task.OnEvent(agents.Event{Kind: agents.KindText, Text: w.text, Partial: true})
		task.OnEvent(agents.Event{Kind: agents.KindResult, Text: w.text, Final: true})
	}
	return &agents.Result{Backend: w.ID(), Text: w.text, ExitCode: 0}, nil
}

type askCatalog struct {
	worker *askWorker
	err    error
	opened string
}

func (c *askCatalog) Check(_ context.Context, ref ask.Ref) (ask.Candidate, error) {
	if c.err != nil && c.worker == nil {
		return ask.Candidate{}, c.err
	}
	return ask.Candidate{Ref: ref, Available: c.worker != nil, Reason: "claude is not installed"}, nil
}

func (c *askCatalog) Auto(context.Context) ([]ask.Candidate, error) {
	if c.worker == nil {
		return []ask.Candidate{{
			Ref:    ask.Ref{Kind: ask.KindCLI, ID: "claude", Raw: "claude"},
			Reason: "claude is not installed",
		}}, nil
	}
	return []ask.Candidate{{
		Ref:       ask.Ref{Kind: ask.KindCLI, ID: c.worker.ID(), Raw: c.worker.ID()},
		Available: true,
	}}, nil
}

func (c *askCatalog) Open(_ context.Context, ref ask.Ref) (agents.Backend, error) {
	c.opened = ref.Raw
	if c.err != nil {
		return nil, c.err
	}
	return c.worker, nil
}

func TestAskCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cfg.Agents.Backend = "claude"
	worker := &askWorker{id: "claude", text: "hello from ask"}
	mem := memory.NewFake()
	catalog := &askCatalog{worker: worker}
	restore := swapAskDeps(t, func(*config.Config) (memory.Memory, memory.Memory, error) {
		return mem, nil, nil
	}, func(*config.Config) ask.Catalog {
		return catalog
	})
	defer restore()

	tests := []struct {
		name    string
		args    []string
		wantErr string
		wantOut string
		backend string
		model   string
		dir     string
		timeout time.Duration
		memory  bool
	}{
		{name: "stream", args: []string{"check the lock"}, wantOut: "hello from ask\n", backend: "claude", memory: true},
		{name: "json", args: []string{"--json", "check the lock"}, wantOut: `"text": "hello from ask"`, backend: "claude", memory: true},
		{name: "flags", args: []string{"--backend", "codex", "--model", "gpt-5.4", "--workdir", "/tmp", "--timeout", "3s", "--no-memory", "ship it"}, wantOut: "hello from ask", backend: "codex", model: "gpt-5.4", dir: "/tmp", timeout: 3 * time.Second},
		{name: "no backend", args: []string{"--no-memory", "ship it"}, wantErr: "no agent backend available"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "no backend" {
				catalog.worker = nil
				cfg.Agents.Backend = ""
			} else {
				catalog.worker = worker
				cfg.Agents.Backend = "claude"
				worker.id = "claude"
			}
			if tt.backend == "codex" {
				worker.id = "codex"
			}
			mem.RetainCalls = nil
			var out, errOut bytes.Buffer
			cmd := NewAskCommand(log, cfg, tr)
			cmd.SetOut(&out)
			cmd.SetErr(&errOut)
			cmd.SetArgs(tt.args)
			err := cmd.Execute()
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tt.wantOut) {
				t.Fatalf("output = %q", out.String())
			}
			if catalog.opened != tt.backend {
				t.Fatalf("opened %q", catalog.opened)
			}
			if tt.model != "" && (worker.task.Model != tt.model || worker.task.WorkDir != tt.dir || worker.task.Timeout != tt.timeout) {
				t.Fatalf("task = %+v", worker.task)
			}
			if tt.memory && len(mem.RetainCalls) == 0 {
				t.Fatal("expected retain")
			}
			if !tt.memory && tt.wantErr == "" && len(mem.RetainCalls) != 0 {
				t.Fatal("retain was called")
			}
		})
	}

	cmd := NewAskCommand(log, cfg, tr)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(nil)
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected missing task")
	}
	help := NewAskCommand(log, cfg, tr)
	var usage bytes.Buffer
	help.SetOut(&usage)
	help.SetArgs([]string{"--help"})
	if err := help.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"--backend", "--model", "--workdir", "--timeout", "--no-memory", "--json"} {
		if !strings.Contains(usage.String(), flag) {
			t.Fatalf("help missing %s", flag)
		}
	}
}

func TestAskMemoryWarningsAndContext(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	worker := &askWorker{text: "still ran"}
	mem := memory.NewFake()
	restore := swapAskDeps(t, func(*config.Config) (memory.Memory, memory.Memory, error) {
		return mem, memory.NewFake(), errors.New("hindsight unreachable")
	}, func(*config.Config) ask.Catalog {
		return &askCatalog{worker: worker}
	})
	defer restore()

	var out, errOut bytes.Buffer
	cmd := NewAskCommand(log, cfg, tr)
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"hello"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "hindsight memory skipped") || !strings.Contains(out.String(), "still ran") {
		t.Fatalf("out %q err %q", out.String(), errOut.String())
	}

	openAskMemory = func(*config.Config) (memory.Memory, memory.Memory, error) {
		return nil, nil, errors.New("root missing")
	}
	failed := NewAskCommand(nil, cfg, tr)
	failed.SetOut(&bytes.Buffer{})
	failed.SetErr(&bytes.Buffer{})
	failed.SetArgs([]string{"hello"})
	if err := failed.Execute(); err == nil || !strings.Contains(err.Error(), "root missing") {
		t.Fatalf("open error = %v", err)
	}

	bare := &cobra.Command{}
	err := runAskCommand(bare, nil, cfg, ask.Request{Task: "hello", NoMemory: true})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOpenAskMemory(t *testing.T) {
	if _, _, err := defaultOpenAskMemory(nil); err == nil {
		t.Fatal("nil config")
	}
	root := t.TempDir()
	cfg := &config.Config{KB: config.KBConfig{Roots: []string{root}}}
	local, extra, err := defaultOpenAskMemory(cfg)
	if err != nil || local == nil || extra != nil {
		t.Fatalf("local memory extra=%v err=%v", extra, err)
	}
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.KB.Roots = []string{file}
	if _, _, err := defaultOpenAskMemory(cfg); err == nil {
		t.Fatal("file root")
	}

	cfg = &config.Config{}
	local, extra, err = defaultOpenAskMemory(cfg)
	if err != nil || local == nil || extra != nil {
		t.Fatalf("default root extra=%v err=%v", extra, err)
	}

	cfg.Memory.Hindsight.Enabled = true
	cfg.Memory.Hindsight.Banks.Project = "project-ask"
	cfg.Memory.Hindsight.Banks.User = "user-ask"
	local, extra, err = defaultOpenAskMemory(cfg)
	if err != nil || local == nil || extra == nil {
		t.Fatalf("hindsight memory extra=%v err=%v", extra, err)
	}

	previousIntegration := newHindsightIntegration
	previousToken := readHindsightAPIToken
	t.Cleanup(func() {
		newHindsightIntegration = previousIntegration
		readHindsightAPIToken = previousToken
	})
	readHindsightAPIToken = func(string, string) (string, error) {
		return "", errors.New("vault locked")
	}
	cfg.Memory.Hindsight.APITokenName = "api_token"
	local, extra, err = defaultOpenAskMemory(cfg)
	if err == nil || local == nil || extra != nil || !strings.Contains(err.Error(), "vault locked") {
		t.Fatalf("token err=%v extra=%v", err, extra)
	}
	cfg.Memory.Hindsight.APITokenName = ""
	newHindsightIntegration = func(hindsight.Settings, memoryhindsight.Identity, string) (kb.Contributor, error) {
		return nil, errors.New("daemon down")
	}
	_, _, err = defaultOpenAskMemory(cfg)
	if err == nil || !strings.Contains(err.Error(), "daemon down") {
		t.Fatalf("integration = %v", err)
	}
	newHindsightIntegration = func(hindsight.Settings, memoryhindsight.Identity, string) (kb.Contributor, error) {
		return memoryhindsight.Failed(errors.New("skip")), nil
	}
	local, extra, err = defaultOpenAskMemory(cfg)
	if err != nil || local == nil || extra != nil {
		t.Fatalf("non integration extra=%v err=%v", extra, err)
	}
}

func TestDefaultAskCatalog(t *testing.T) {
	catalog := defaultAskCatalog(&config.Config{})
	if catalog == nil {
		t.Fatal("nil catalog")
	}
	candidates, err := catalog.Auto(context.Background())
	if err != nil || len(candidates) < 2 {
		t.Fatalf("candidates = %+v %v", candidates, err)
	}
	if defaultAskCatalog(nil) == nil {
		t.Fatal("nil config catalog")
	}
	rootLog, rootCfg, tr := newTestDeps(t)
	command := NewRootCommand(rootLog, rootCfg, tr)
	if command.Commands() == nil || findAskCommand(command, "ask") == nil {
		t.Fatal("root is missing ask")
	}
}

func findAskCommand(root *cobra.Command, name string) *cobra.Command {
	for _, cmd := range root.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func swapAskDeps(t *testing.T, memoryFn func(*config.Config) (memory.Memory, memory.Memory, error), catalogFn func(*config.Config) ask.Catalog) func() {
	t.Helper()
	prevMemory := openAskMemory
	prevCatalog := newAskCatalog
	openAskMemory = memoryFn
	newAskCatalog = catalogFn
	return func() {
		openAskMemory = prevMemory
		newAskCatalog = prevCatalog
	}
}
