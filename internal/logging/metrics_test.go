package logging

import (
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
