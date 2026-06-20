package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExecutionMetric represents a single command execution
type ExecutionMetric struct {
	Timestamp   time.Time         `json:"timestamp"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Status      string            `json:"status"` // "success" or "error"
	ExitCode    int               `json:"exit_code"`
	Duration    float64           `json:"duration_ms"`
	User        string            `json:"user"`
	Version     string            `json:"version"`
	Language    string            `json:"language"`
	Output      string            `json:"output,omitempty"`
	Error       string            `json:"error,omitempty"`
	ErrorType   string            `json:"error_type,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// MetricsCollector collects and stores execution metrics
type MetricsCollector struct {
	metricsFile string
	mu          sync.Mutex
	metrics     []ExecutionMetric
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(cockpitDir string) *MetricsCollector {
	metricsFile := filepath.Join(cockpitDir, "metrics.json")

	collector := &MetricsCollector{
		metricsFile: metricsFile,
		metrics:     []ExecutionMetric{},
	}

	// Load existing metrics
	collector.loadMetrics()

	return collector
}

// RecordExecution records a command execution
func (mc *MetricsCollector) RecordExecution(metric ExecutionMetric) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Set timestamp if not already set
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	mc.metrics = append(mc.metrics, metric)

	// Save to file
	return mc.saveMetrics()
}

// loadMetrics loads metrics from file
func (mc *MetricsCollector) loadMetrics() error {
	if _, err := os.Stat(mc.metricsFile); os.IsNotExist(err) {
		return nil // File doesn't exist yet, that's ok
	}

	data, err := os.ReadFile(mc.metricsFile)
	if err != nil {
		return fmt.Errorf("failed to read metrics file: %w", err)
	}

	if err := json.Unmarshal(data, &mc.metrics); err != nil {
		return fmt.Errorf("failed to parse metrics: %w", err)
	}

	return nil
}

// saveMetrics saves metrics to file
func (mc *MetricsCollector) saveMetrics() error {
	data, err := json.MarshalIndent(mc.metrics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	if err := os.WriteFile(mc.metricsFile, data, 0o644); err != nil {
		return fmt.Errorf("failed to write metrics file: %w", err)
	}

	return nil
}

// GetMetrics returns all metrics
func (mc *MetricsCollector) GetMetrics() []ExecutionMetric {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	// Return a copy
	result := make([]ExecutionMetric, len(mc.metrics))
	copy(result, mc.metrics)
	return result
}

// GetMetricsByCommand returns metrics for a specific command
func (mc *MetricsCollector) GetMetricsByCommand(command string) []ExecutionMetric {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var result []ExecutionMetric
	for _, m := range mc.metrics {
		if m.Command == command {
			result = append(result, m)
		}
	}
	return result
}

// GetMetricsByDate returns metrics for a specific date
func (mc *MetricsCollector) GetMetricsByDate(date time.Time) []ExecutionMetric {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var result []ExecutionMetric
	for _, m := range mc.metrics {
		if m.Timestamp.Format("2006-01-02") == date.Format("2006-01-02") {
			result = append(result, m)
		}
	}
	return result
}

// GetMetricsByStatus returns metrics by status (success/error)
func (mc *MetricsCollector) GetMetricsByStatus(status string) []ExecutionMetric {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	var result []ExecutionMetric
	for _, m := range mc.metrics {
		if m.Status == status {
			result = append(result, m)
		}
	}
	return result
}

// GetStats returns statistics about executions
func (mc *MetricsCollector) GetStats() map[string]interface{} {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	stats := map[string]interface{}{
		"total_executions":  len(mc.metrics),
		"successful":        0,
		"failed":            0,
		"commands":          map[string]int{},
		"total_duration_ms": 0.0,
		"avg_duration_ms":   0.0,
		"error_types":       map[string]int{},
	}

	if len(mc.metrics) == 0 {
		return stats
	}

	successCount := 0
	failureCount := 0
	totalDuration := 0.0
	commandCounts := make(map[string]int)
	errorTypes := make(map[string]int)

	for _, m := range mc.metrics {
		if m.Status == "success" {
			successCount++
		} else {
			failureCount++
			if m.ErrorType != "" {
				errorTypes[m.ErrorType]++
			}
		}

		totalDuration += m.Duration
		commandCounts[m.Command]++
	}

	stats["successful"] = successCount
	stats["failed"] = failureCount
	stats["success_rate"] = float64(successCount) / float64(len(mc.metrics)) * 100
	stats["total_duration_ms"] = totalDuration
	stats["avg_duration_ms"] = totalDuration / float64(len(mc.metrics))
	stats["commands"] = commandCounts
	stats["error_types"] = errorTypes

	return stats
}

// Clear clears all metrics (useful for testing)
func (mc *MetricsCollector) Clear() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.metrics = []ExecutionMetric{}
	return mc.saveMetrics()
}
