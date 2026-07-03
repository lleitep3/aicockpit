package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func setupSchedulerTest(t *testing.T) (*logging.Manager, *config.Config, *i18n.Translator, func()) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	cockpitDir := filepath.Join(tmpDir, ".cockpit")
	if err := os.MkdirAll(cockpitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	logMgr, err := logging.NewManager(filepath.Join(cockpitDir, "logs"))
	if err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{
		Version:    "1.0.0",
		Language:   "en-us",
		AIProvider: "antigravity",
	}
	translator := i18n.New("en-us")

	cleanup := func() {
		os.Setenv("HOME", oldHome)
		logMgr.Close()
	}

	return logMgr, cfg, translator, cleanup
}

func captureStdout(fn func() error) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	os.Stdout = w

	var buf bytes.Buffer
	var fnErr error

	done := make(chan struct{})
	go func() {
		_, _ = buf.ReadFrom(r)
		close(done)
	}()

	fnErr = fn()

	_ = w.Close()
	<-done
	os.Stdout = oldStdout

	return buf.String(), fnErr
}

func TestNewSchedulerCommand(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	cmd := NewSchedulerCommand(logMgr, cfg, translator)
	if cmd == nil {
		t.Fatal("expected scheduler command to be non-nil")
	}
	if cmd.Name() != "scheduler" {
		t.Errorf("expected command name 'scheduler', got %s", cmd.Name())
	}
}

func TestNewSchedulerAddCommand(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	cmd := NewSchedulerAddCommand(logMgr, cfg, translator)
	cmd.SetArgs([]string{})

	var errBuf bytes.Buffer
	cmd.SetOut(&errBuf)
	cmd.SetErr(&errBuf)

	out, err := captureStdout(cmd.Execute)
	if err == nil {
		t.Fatal("expected error for missing command flag")
	}
	_ = out
	if !strings.Contains(err.Error(), "--command") {
		t.Errorf("expected error about --command, got %v", err)
	}
}

func TestNewSchedulerAddCommandValid(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	cmd := NewSchedulerAddCommand(logMgr, cfg, translator)
	cmd.SetArgs([]string{"--command", "echo hello", "--cron", "0 9 * * *"})

	out, err := captureStdout(cmd.Execute)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.Contains(out, "Agendamento criado") {
		t.Errorf("expected success message, got %s", out)
	}
}

func TestNewSchedulerListCommand(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	addCmd := NewSchedulerAddCommand(logMgr, cfg, translator)
	addCmd.SetArgs([]string{"--command", "echo hello", "--cron", "0 9 * * *"})
	_, _ = captureStdout(addCmd.Execute)

	listCmd := NewSchedulerListCommand(logMgr, cfg, translator)
	out, err := captureStdout(listCmd.Execute)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.Contains(out, "echo hello") {
		t.Errorf("expected list to contain command, got %s", out)
	}
}

func TestNewSchedulerRemoveCommand(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	addCmd := NewSchedulerAddCommand(logMgr, cfg, translator)
	addCmd.SetArgs([]string{"--command", "echo hello", "--cron", "0 9 * * *"})
	_, _ = captureStdout(addCmd.Execute)

	listCmd := NewSchedulerListCommand(logMgr, cfg, translator)
	out, _ := captureStdout(listCmd.Execute)

	id := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "sched_") {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				id = fields[0]
				break
			}
		}
	}

	if id == "" {
		t.Fatal("expected to find a job id")
	}

	removeCmd := NewSchedulerRemoveCommand(logMgr, cfg, translator)
	removeCmd.SetArgs([]string{id})

	removeOut, err := captureStdout(removeCmd.Execute)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.Contains(removeOut, "removido") {
		t.Errorf("expected removal message, got %s", removeOut)
	}
}

func TestNewSchedulerRunCommandNoJobs(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	cmd := NewSchedulerRunCommand(logMgr, cfg, translator)
	out, err := captureStdout(cmd.Execute)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.Contains(out, "no jobs due") {
		t.Errorf("expected no jobs message, got %s", out)
	}
}

func TestNewSchedulerAddUbuntuSecurityCommand(t *testing.T) {
	logMgr, cfg, translator, cleanup := setupSchedulerTest(t)
	defer cleanup()

	cmd := NewSchedulerAddUbuntuSecurityCommand(logMgr, cfg, translator)
	cmd.SetArgs([]string{"--cron", "0 3 * * *"})

	out, err := captureStdout(cmd.Execute)
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if !strings.Contains(out, "ubuntu-security") {
		t.Errorf("expected ubuntu-security in output, got %s", out)
	}
	if !strings.Contains(out, "0 3 * * *") {
		t.Errorf("expected cron expression in output, got %s", out)
	}
}
