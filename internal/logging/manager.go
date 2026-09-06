package logging

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/lleitep3/aicockpit/internal/env"
	runtimepaths "github.com/lleitep3/aicockpit/internal/runtime"
)

// resolveLogDir selects a writable log directory without making logging a
// prerequisite for running a command. An explicit path wins; otherwise use
// the operating system's temporary directory.
func resolveLogDir(cockpitDir string) (string, error) {
	candidates := []string{}
	if configured := os.Getenv(env.CockpitLogDir.String()); configured != "" {
		candidates = append(candidates, configured)
	}
	candidates = append(candidates, filepath.Join(cockpitDir, "logs"))
	tempRoot := os.TempDir()
	if runtime.GOOS == "windows" {
		candidates = append(candidates, filepath.Join(tempRoot, "AICockpit", "logs"))
	} else {
		candidates = append(candidates, filepath.Join(tempRoot, "aicockpit", "logs"))
	}

	var lastErr error
	for _, candidate := range candidates {
		if err := os.MkdirAll(candidate, 0o755); err != nil {
			lastErr = err
			continue
		}
		probe, err := os.CreateTemp(candidate, ".write-test-*")
		if err != nil {
			lastErr = err
			continue
		}
		probeName := probe.Name()
		_ = probe.Close()
		_ = os.Remove(probeName)
		return candidate, nil
	}
	return "", fmt.Errorf("no writable log directory found: %w", lastErr)
}

// Manager manages all logging operations
type Manager struct {
	fileLogger *FileLogger
	metrics    *MetricsCollector
	cockpitDir string
}

// NewManager creates a new logging manager
func NewManager(cockpitDir string) (*Manager, error) {
	dataDir, err := runtimepaths.DataDir(cockpitDir)
	if err != nil {
		return &Manager{metrics: NewMetricsCollector(cockpitDir), cockpitDir: cockpitDir}, err
	}
	logsDir := filepath.Join(dataDir, "logs")
	if configured := os.Getenv(env.CockpitLogDir.String()); configured != "" {
		logsDir = configured
	}

	// Create file logger (JSON format)
	fileLogger, err := NewFileLogger(logsDir, true)
	if err != nil {
		return nil, err
	}

	// Create metrics collector
	metricsCollector := NewMetricsCollector(dataDir)

	manager := &Manager{
		fileLogger: fileLogger,
		metrics:    metricsCollector,
		cockpitDir: cockpitDir,
	}

	return manager, nil
}

// LogCommand logs a command execution
func (m *Manager) LogCommand(command string, args []string, status string, exitCode int, duration time.Duration, output string, err error) error {
	if m.fileLogger == nil {
		return nil
	}
	// Get current user
	currentUser, _ := user.Current()
	username := "unknown"
	if currentUser != nil {
		username = currentUser.Username
	}

	// Get version (from environment or default)
	version := os.Getenv(env.CockpitVersion.String())
	if version == "" {
		version = "0.1.0"
	}

	// Get language (from environment or default)
	language := os.Getenv(env.CockpitLanguage.String())
	if language == "" {
		language = "en-us"
	}

	// Prepare error information
	errorMsg := ""
	errorType := ""
	if err != nil {
		errorMsg = err.Error()
		errorType = fmt.Sprintf("%T", err)
	}

	// Create metric
	metric := ExecutionMetric{
		Timestamp:   time.Now(),
		Command:     command,
		Args:        args,
		Status:      status,
		ExitCode:    exitCode,
		Duration:    float64(duration.Milliseconds()),
		User:        username,
		Version:     version,
		Language:    language,
		Output:      output,
		Error:       errorMsg,
		ErrorType:   errorType,
		Environment: map[string]string{},
	}

	// Record metric
	if err := m.metrics.RecordExecution(metric); err != nil {
		// Log error but don't fail
		m.fileLogger.Log("error", "failed to record metric", map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Log to file
	context := map[string]interface{}{
		"command":     command,
		"args":        args,
		"status":      status,
		"exit_code":   exitCode,
		"duration_ms": duration.Milliseconds(),
		"user":        username,
	}

	if errorMsg != "" {
		context["error"] = errorMsg
		context["error_type"] = errorType
	}

	level := "INFO"
	if status == "error" {
		level = "ERROR"
	}

	return m.fileLogger.Log(level, fmt.Sprintf("Command executed: %s", command), context)
}

// LogInfo logs an info message
func (m *Manager) LogInfo(message string, context map[string]interface{}) error {
	if m.fileLogger == nil {
		return nil
	}
	return m.fileLogger.Log("INFO", message, context)
}

// LogWarn logs a warning message
func (m *Manager) LogWarn(message string, context map[string]interface{}) error {
	if m.fileLogger == nil {
		return nil
	}
	return m.fileLogger.Log("WARN", message, context)
}

// LogError logs an error message
func (m *Manager) LogError(message string, context map[string]interface{}) error {
	if m.fileLogger == nil {
		return nil
	}
	return m.fileLogger.Log("ERROR", message, context)
}

// GetMetrics returns the metrics collector
func (m *Manager) GetMetrics() *MetricsCollector {
	return m.metrics
}

// GetFileLogger returns the file logger
func (m *Manager) GetFileLogger() *FileLogger {
	return m.fileLogger
}

// Close closes all resources
func (m *Manager) Close() error {
	if m.fileLogger == nil {
		return nil
	}
	return m.fileLogger.Close()
}
