package tracking

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/lleitep3/aicockpit/internal/env"
	runtimepaths "github.com/lleitep3/aicockpit/internal/runtime"
)

// baseDir is the default tracking directory used when no runtime root is available.
var baseDir = filepath.Join(".cockpit", "tracking")

// Event represents a generic tracking event.
type Event struct {
	Type      string         `json:"type"`
	Timestamp string         `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

// ensureDayDir ensures a directory for the current UTC day exists and returns its path.
func ensureDayDir() (string, error) {
	day := time.Now().UTC().Format("2006-01-02")
	root, err := runtimepaths.DataDir(baseDir)
	if err != nil {
		return "", err
	}
	if legacy := os.Getenv(env.TrackingDir.String()); legacy != "" {
		root = legacy
	}
	dir := filepath.Join(root, "tracking", day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("tracking: cannot create day dir %s: %w", dir, err)
	}
	return dir, nil
}

// Record writes the given event to a JSON‑L file under the day directory.
func Record(e Event) error {
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	dir, err := ensureDayDir()
	if err != nil {
		return err
	}
	filename := fmt.Sprintf("%s_%s.log", e.Type, e.Timestamp)
	path := filepath.Join(dir, filename)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("tracking: cannot open log file %s: %w", path, err)
	}
	defer f.Close()
	line, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("tracking: cannot marshal event: %w", err)
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("tracking: cannot write event: %w", err)
	}
	return nil
}
