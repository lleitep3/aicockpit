package runtime

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/env"
)

// DataDir returns one writable root shared by logs, tracking, and metrics.
func DataDir(defaultDir string) (string, error) {
	candidates := []string{}
	if configured := os.Getenv(env.CockpitDataDir.String()); configured != "" {
		candidates = append(candidates, configured)
	}
	candidates = append(candidates, defaultDir, filepath.Join(os.TempDir(), "aicockpit"))
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
		name := probe.Name()
		_ = probe.Close()
		_ = os.Remove(name)
		return candidate, nil
	}
	return "", fmt.Errorf("no writable Cockpit data directory: %w", lastErr)
}
