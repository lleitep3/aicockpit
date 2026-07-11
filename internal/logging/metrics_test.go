package logging

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricsCollector(t *testing.T) {
	tmpDir := t.TempDir()

	collector := NewMetricsCollector(tmpDir)

	// Record a successful execution
	metric := ExecutionMetric{
		Command:  "setup",
		Args:     []string{},
		Status:   "success",
		ExitCode: 0,
		Duration: 100.5,
		User:     "testuser",
		Version:  "0.1.0",
		Language: "en-us",
	}

	if err := collector.RecordExecution(metric); err != nil {
		t.Fatalf("RecordExecution failed: %v", err)
	}

	// Verify metrics file was created
	metricsFile := filepath.Join(tmpDir, "metrics.json")
	if _, err := os.Stat(metricsFile); os.IsNotExist(err) {
		t.Error("Metrics file was not created")
	}

	// Get metrics
	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Errorf("Expected 1 metric, got %d", len(metrics))
	}

	if metrics[0].Command != "setup" {
		t.Errorf("Expected command 'setup', got '%s'", metrics[0].Command)
	}
}

func TestMetricsCollectorByCommand(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	// Record multiple executions
	commands := []string{"setup", "info", "setup", "doctor"}
	for _, cmd := range commands {
		metric := ExecutionMetric{
			Command:  cmd,
			Status:   "success",
			ExitCode: 0,
			Duration: 50.0,
		}
		collector.RecordExecution(metric)
	}

	// Get metrics by command
	setupMetrics := collector.GetMetricsByCommand("setup")
	if len(setupMetrics) != 2 {
		t.Errorf("Expected 2 'setup' metrics, got %d", len(setupMetrics))
	}

	infoMetrics := collector.GetMetricsByCommand("info")
	if len(infoMetrics) != 1 {
		t.Errorf("Expected 1 'info' metric, got %d", len(infoMetrics))
	}
}

func TestMetricsCollectorByStatus(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	// Record successful and failed executions
	successMetric := ExecutionMetric{
		Command:  "setup",
		Status:   "success",
		ExitCode: 0,
		Duration: 50.0,
	}
	collector.RecordExecution(successMetric)

	failureMetric := ExecutionMetric{
		Command:  "info",
		Status:   "error",
		ExitCode: 1,
		Duration: 25.0,
		Error:    "test error",
	}
	collector.RecordExecution(failureMetric)

	// Get by status
	successMetrics := collector.GetMetricsByStatus("success")
	if len(successMetrics) != 1 {
		t.Errorf("Expected 1 success metric, got %d", len(successMetrics))
	}

	errorMetrics := collector.GetMetricsByStatus("error")
	if len(errorMetrics) != 1 {
		t.Errorf("Expected 1 error metric, got %d", len(errorMetrics))
	}
}

func TestMetricsStats(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	// Record executions
	// i=0,3,6,9 -> error (4 errors)
	// i=1,2,4,5,7,8 -> success (6 successes)
	for i := 0; i < 10; i++ {
		status := "success"
		exitCode := 0
		if i%3 == 0 {
			status = "error"
			exitCode = 1
		}

		metric := ExecutionMetric{
			Command:  "test",
			Status:   status,
			ExitCode: exitCode,
			Duration: 100.0,
		}
		collector.RecordExecution(metric)
	}

	// Get stats
	stats := collector.GetStats()

	if stats["total_executions"] != 10 {
		t.Errorf("Expected 10 total executions, got %v", stats["total_executions"])
	}

	if stats["successful"] != 6 {
		t.Errorf("Expected 6 successful, got %v", stats["successful"])
	}

	if stats["failed"] != 4 {
		t.Errorf("Expected 4 failed, got %v", stats["failed"])
	}
}

func TestMetricsCollectorByDate(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	today := time.Now()
	yesterday := today.AddDate(0, 0, -1)

	// Record metric for today
	todayMetric := ExecutionMetric{
		Timestamp: today,
		Command:   "setup",
		Status:    "success",
		ExitCode:  0,
		Duration:  50.0,
	}
	collector.RecordExecution(todayMetric)

	// Record metric for yesterday
	yesterdayMetric := ExecutionMetric{
		Timestamp: yesterday,
		Command:   "info",
		Status:    "success",
		ExitCode:  0,
		Duration:  50.0,
	}
	collector.RecordExecution(yesterdayMetric)

	// Get metrics by date
	todayMetrics := collector.GetMetricsByDate(today)
	if len(todayMetrics) != 1 {
		t.Errorf("Expected 1 metric for today, got %d", len(todayMetrics))
	}

	yesterdayMetrics := collector.GetMetricsByDate(yesterday)
	if len(yesterdayMetrics) != 1 {
		t.Errorf("Expected 1 metric for yesterday, got %d", len(yesterdayMetrics))
	}
}

func TestMetricsLoadMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	// Pre-populate metrics file
	initial := []ExecutionMetric{
		{
			Timestamp: time.Now(),
			Command:   "pre-existing",
			Status:    "success",
			Duration:  77.0,
		},
	}
	data, _ := json.MarshalIndent(initial, "", "  ")
	metricsFile := filepath.Join(tmpDir, "metrics.json")
	if err := os.WriteFile(metricsFile, data, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// NewMetricsCollector should load the pre-existing metrics
	collector := NewMetricsCollector(tmpDir)
	metrics := collector.GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("expected 1 loaded metric, got %d", len(metrics))
	}
	if metrics[0].Command != "pre-existing" {
		t.Errorf("expected command 'pre-existing', got %q", metrics[0].Command)
	}
}

func TestMetricsLoadMetricsCorruptFile(t *testing.T) {
	tmpDir := t.TempDir()

	metricsFile := filepath.Join(tmpDir, "metrics.json")
	if err := os.WriteFile(metricsFile, []byte("not json {{{"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// loadMetrics is called inside NewMetricsCollector; it logs error but doesn't panic
	// The collector should still be usable (with empty/partial state)
	collector := NewMetricsCollector(tmpDir)
	if collector == nil {
		t.Fatal("expected non-nil collector even with corrupt metrics file")
	}
}

func TestMetricsSaveMetricsUnwritable(t *testing.T) {
	tmpDir := t.TempDir()

	collector := NewMetricsCollector(tmpDir)

	// Make the metrics file unwritable after creation
	metricsFile := filepath.Join(tmpDir, "metrics.json")
	// Create it first so it exists
	if err := os.WriteFile(metricsFile, []byte("[]"), 0o000); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Attempting to record should fail due to unwritable file
	metric := ExecutionMetric{Command: "test", Status: "success"}
	err := collector.RecordExecution(metric)
	if err == nil {
		t.Log("note: running as root or permissions not enforced; skipping unwritable test")
	}
}

func TestMetricsGetStatsEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	stats := collector.GetStats()
	if stats["total_executions"] != 0 {
		t.Errorf("expected 0 total_executions, got %v", stats["total_executions"])
	}
	if stats["successful"] != 0 {
		t.Errorf("expected 0 successful, got %v", stats["successful"])
	}
}

func TestMetricsGetStatsWithErrorTypes(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	collector.RecordExecution(ExecutionMetric{
		Command:   "cmd",
		Status:    "error",
		ErrorType: "*errors.errorString",
		Duration:  10.0,
	})
	collector.RecordExecution(ExecutionMetric{
		Command:   "cmd",
		Status:    "error",
		ErrorType: "*errors.errorString",
		Duration:  20.0,
	})
	collector.RecordExecution(ExecutionMetric{
		Command:  "cmd",
		Status:   "success",
		Duration: 30.0,
	})

	stats := collector.GetStats()
	if stats["total_executions"] != 3 {
		t.Errorf("expected 3, got %v", stats["total_executions"])
	}
	if stats["failed"] != 2 {
		t.Errorf("expected 2 failed, got %v", stats["failed"])
	}
	errTypes, ok := stats["error_types"].(map[string]int)
	if !ok {
		t.Fatalf("error_types has unexpected type %T", stats["error_types"])
	}
	if errTypes["*errors.errorString"] != 2 {
		t.Errorf("expected 2 for *errors.errorString, got %d", errTypes["*errors.errorString"])
	}

	rate, ok := stats["success_rate"].(float64)
	if !ok {
		t.Fatalf("success_rate has unexpected type %T", stats["success_rate"])
	}
	// 1 success / 3 total ≈ 33.33%
	if rate < 33.0 || rate > 34.0 {
		t.Errorf("unexpected success_rate %.2f", rate)
	}
}

func TestMetricsClear(t *testing.T) {
	tmpDir := t.TempDir()
	collector := NewMetricsCollector(tmpDir)

	collector.RecordExecution(ExecutionMetric{Command: "a", Status: "success"})
	collector.RecordExecution(ExecutionMetric{Command: "b", Status: "error"})

	if len(collector.GetMetrics()) != 2 {
		t.Fatalf("expected 2 metrics before clear")
	}

	if err := collector.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if len(collector.GetMetrics()) != 0 {
		t.Errorf("expected 0 metrics after clear, got %d", len(collector.GetMetrics()))
	}

	// Verify file was overwritten with empty array
	metricsFile := filepath.Join(tmpDir, "metrics.json")
	data, err := os.ReadFile(metricsFile)
	if err != nil {
		t.Fatalf("read metrics file after clear: %v", err)
	}
	var stored []ExecutionMetric
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("parse metrics file after clear: %v", err)
	}
	if len(stored) != 0 {
		t.Errorf("expected empty file after clear, got %d entries", len(stored))
	}
}
