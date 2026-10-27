package packages

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// defaultHookTimeout is the maximum time a hook script is allowed to run.
// It is a package-level variable so tests can override it.
var defaultHookTimeout = 5 * time.Minute

// HookRunner executes package lifecycle hooks safely.
type HookRunner struct{}

// NewHookRunner creates a new HookRunner.
func NewHookRunner() *HookRunner {
	return &HookRunner{}
}

// validateHookScript ensures the hook script path is relative, does not escape
// the package directory, and resolves to a location inside packageDir.
func validateHookScript(packageDir, script string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", fmt.Errorf("hook script path is empty")
	}
	if filepath.IsAbs(script) {
		return "", fmt.Errorf("hook script path must be relative")
	}

	cleanScript := filepath.Clean(script)
	if strings.HasPrefix(cleanScript, "..") {
		return "", fmt.Errorf("hook script path attempts to escape package directory")
	}

	resolved := filepath.Join(packageDir, cleanScript)
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("failed to resolve hook script %q: %w", script, err)
	}

	absPackageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve package directory: %w", err)
	}

	rel, err := filepath.Rel(absPackageDir, absResolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("hook script %q resolves outside package directory", script)
	}

	return absResolved, nil
}

// Run executes a list of hooks from the given package directory.
// Each hook's Script path is relative to packageDir and must stay inside it.
// Missing, linked or non-regular scripts fail instead of reporting false success.
func (h *HookRunner) Run(packageDir string, hooks []Hook) error {
	absPackageDir, err := filepath.Abs(packageDir)
	if err != nil {
		return fmt.Errorf("failed to resolve package directory: %w", err)
	}

	for _, hook := range hooks {
		scriptPath, err := validateHookFile(absPackageDir, hook.Script)
		if err != nil {
			return fmt.Errorf("invalid hook script %q: %w", hook.Script, err)
		}

		desc := hook.Description
		if desc == "" {
			desc = hook.Script
		}
		fmt.Printf("  → Running hook: %s\n", desc)

		// Make the script executable.
		if err := os.Chmod(scriptPath, 0o755); err != nil {
			return fmt.Errorf("failed to chmod hook script %s: %w", hook.Script, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultHookTimeout)
		cmd := exec.CommandContext(ctx, "sh", scriptPath) //nolint:gosec // path already validated and confined to package dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = absPackageDir

		err = cmd.Run()
		cancel()
		if err != nil {
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("hook script %s timed out: %w", hook.Script, err)
			}
			return fmt.Errorf("hook script %s failed: %w", hook.Script, err)
		}
	}
	return nil
}

// validateHookFile applies the same contract during planning and execution.
func validateHookFile(packageDir, script string) (string, error) {
	path, err := validateHookScript(packageDir, script)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(packageDir)
	if err != nil {
		return "", err
	}
	if err := checkUpgradeParents(root, path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("hook script unavailable %s: %w", script, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("hook script must be a regular file: %s", script)
	}
	return path, nil
}
