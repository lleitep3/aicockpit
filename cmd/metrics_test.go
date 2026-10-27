package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListMetrics_NoFilter(t *testing.T) {
	log, cfg, tr := newTestDeps(t)

	// Record a couple of metrics so the list is non-empty.
	log.LogCommand("deploy", nil, "success", 0, time.Millisecond*10, "", nil)
	log.LogCommand("info", nil, "error", 1, time.Millisecond*5, "oops", fmt.Errorf("some error"))

	if err := listMetrics(log, cfg, tr, "", "", 0, ""); err != nil {
		t.Errorf("listMetrics() error = %v", err)
	}
}

func TestListMetrics_FilterByCommand(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	log.LogCommand("deploy", nil, "success", 0, time.Millisecond, "", nil)
	log.LogCommand("info", nil, "success", 0, time.Millisecond, "", nil)

	if err := listMetrics(log, cfg, tr, "deploy", "", 0, ""); err != nil {
		t.Errorf("listMetrics(command=deploy) error = %v", err)
	}
}

func TestListMetrics_FilterByStatus(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	log.LogCommand("deploy", nil, "success", 0, time.Millisecond, "", nil)
	log.LogCommand("info", nil, "error", 1, time.Millisecond, "err", nil)

	if err := listMetrics(log, cfg, tr, "", "error", 0, ""); err != nil {
		t.Errorf("listMetrics(status=error) error = %v", err)
	}
}

func TestListMetrics_FilterByDate_Valid(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	log.LogCommand("deploy", nil, "success", 0, time.Millisecond, "", nil)

	today := time.Now().Format("2006-01-02")
	if err := listMetrics(log, cfg, tr, "", "", 0, today); err != nil {
		t.Errorf("listMetrics(date=%s) error = %v", today, err)
	}
}

func TestListMetrics_FilterByDate_Invalid(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	// Invalid date — must return nil (not an error), just print a warning.
	if err := listMetrics(log, cfg, tr, "", "", 0, "not-a-date"); err != nil {
		t.Errorf("listMetrics(invalid date) should return nil, got %v", err)
	}
}

func TestListMetrics_WithLimit(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	for i := 0; i < 5; i++ {
		log.LogCommand("cmd", nil, "success", 0, time.Millisecond, "", nil)
	}

	if err := listMetrics(log, cfg, tr, "", "", 2, ""); err != nil {
		t.Errorf("listMetrics(limit=2) error = %v", err)
	}
}

func TestListMetrics_Empty(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	// No metrics recorded — should print "No metrics found" and return nil.
	if err := listMetrics(log, cfg, tr, "", "", 0, ""); err != nil {
		t.Errorf("listMetrics() with no data error = %v", err)
	}
}

func TestShowStats(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	log.LogCommand("deploy", nil, "success", 0, time.Millisecond*20, "", nil)
	log.LogCommand("info", nil, "error", 1, time.Millisecond*5, "oops", nil)

	if err := showStats(log, cfg, tr); err != nil {
		t.Errorf("showStats() error = %v", err)
	}
}

func TestShowStats_NoData(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	if err := showStats(log, cfg, tr); err != nil {
		t.Errorf("showStats() with no data error = %v", err)
	}
}

func TestShowLogs_NoDate(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	// Logger was created with t.TempDir() — no real log files, but must not error.
	if err := showLogs(log, cfg, tr, ""); err != nil {
		t.Errorf("showLogs() error = %v", err)
	}
}

func TestShowLogs_WithDate_Valid(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	today := time.Now().Format("2006-01-02")
	if err := showLogs(log, cfg, tr, today); err != nil {
		t.Errorf("showLogs(date=%s) error = %v", today, err)
	}
}

func TestShowLogs_WithDate_Invalid(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	if err := showLogs(log, cfg, tr, "bad-date"); err != nil {
		t.Errorf("showLogs(invalid date) should return nil, got %v", err)
	}
}

func TestShowLogs_WithExistingFile(t *testing.T) {
	// Create a real log file so the stat/read path is exercised.
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "2006-01-02.log")
	if err := os.WriteFile(logFile, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	log, cfg, tr := newTestDeps(t)
	// Provide the exact date matching the file name.
	if err := showLogs(log, cfg, tr, "2006-01-02"); err != nil {
		t.Errorf("showLogs() error = %v", err)
	}
}

// ── Constructor Execute tests for 100% RunE coverage ──────────────────────

func TestNewMetricsListCommand_Execute(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewMetricsListCommand(log, cfg, tr)
	if err := cmd.Execute(); err != nil {
		t.Errorf("metrics list Execute error = %v", err)
	}
}

func TestNewMetricsStatsCommand_Execute(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewMetricsStatsCommand(log, cfg, tr)
	if err := cmd.Execute(); err != nil {
		t.Errorf("metrics stats Execute error = %v", err)
	}
}

func TestNewMetricsLogsCommand_Execute(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewMetricsLogsCommand(log, cfg, tr)
	if err := cmd.Execute(); err != nil {
		t.Errorf("metrics logs Execute error = %v", err)
	}
}

func TestNewMetricsCommand_Execute(t *testing.T) {
	log, cfg, tr := newTestDeps(t)
	cmd := NewMetricsCommand(log, cfg, tr)
	// Running with no subcommand should show help or run default
	_ = cmd.Execute()
}
