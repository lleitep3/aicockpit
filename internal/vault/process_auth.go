package vault

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ProcessAuth verifies the identity of the calling process
type ProcessAuth struct {
	authorizedPaths    []string
	requireInteraction bool
}

func NewProcessAuth() *ProcessAuth {
	return &ProcessAuth{
		authorizedPaths: []string{
			"/usr/local/bin/",
			os.Getenv("HOME") + "/.local/bin/",
		},
		requireInteraction: true, // Require human interaction for unlock
	}
}

// VerifyCaller verifies if the caller is authorized to perform vault operations
func (pa *ProcessAuth) VerifyCaller(operation string) error {
	// Get parent process (the one that called cockpit)
	parentPID := os.Getppid()

	// Get parent process info
	parentExe, err := getProcessPathForAuth(parentPID)
	if err != nil {
		return fmt.Errorf("failed to identify caller: %w", err)
	}

	// Check if it's an interactive terminal
	if !pa.isInteractiveTerminal() {
		return fmt.Errorf("operation must be run from interactive terminal")
	}

	// Check if the parent is an authorized executable
	if !pa.isAuthorizedExecutable(parentExe) {
		return fmt.Errorf("unauthorized caller: %s", parentExe)
	}

	// For sensitive operations (unlock), require human interaction
	if operation == "unlock" && pa.requireInteraction {
		if !pa.hasHumanInteraction() {
			return fmt.Errorf("unlock requires human interaction")
		}
	}

	return nil
}

// isInteractiveTerminal checks if running in an interactive terminal
func (pa *ProcessAuth) isInteractiveTerminal() bool {
	// Check if stdin is a terminal
	if fileInfo, _ := os.Stdin.Stat(); (fileInfo.Mode() & os.ModeCharDevice) != 0 {
		return true
	}

	// Check environment variables
	if os.Getenv("TERM") != "" {
		return true
	}

	return false
}

// isAuthorizedExecutable checks if the executable is in an authorized location
func (pa *ProcessAuth) isAuthorizedExecutable(exePath string) bool {
	// Allow cockpit itself
	if filepath.Base(exePath) == "cockpit" {
		return true
	}

	// Check if in authorized paths
	for _, authPath := range pa.authorizedPaths {
		if strings.HasPrefix(exePath, authPath) {
			return true
		}
	}

	// Dev mode bypass
	if os.Getenv("COCKPIT_DEV_MODE") == "true" {
		return true
	}

	return false
}

// hasHumanInteraction checks if there's evidence of human interaction
func (pa *ProcessAuth) hasHumanInteraction() bool {
	// This is a heuristic - in production, you'd want more sophisticated checks

	// Check if running from a shell with a TTY
	if os.Getenv("TERM") != "" {
		return true
	}

	// Check if parent process is a shell
	parentPID := os.Getppid()
	parentName, err := getProcessName(parentPID)
	if err == nil {
		shells := []string{"bash", "zsh", "fish", "sh", "cmd", "powershell"}
		for _, shell := range shells {
			if strings.Contains(parentName, shell) {
				return true
			}
		}
	}

	return false
}

// getProcessPathForAuth gets the executable path for a PID (for authentication)
func getProcessPathForAuth(pid int) (string, error) {
	if runtime.GOOS == "linux" {
		// Linux: read /proc/[PID]/exe
		procPath := fmt.Sprintf("/proc/%d/exe", pid)
		path, err := os.Readlink(procPath)
		if err == nil {
			return path, nil
		}
	}

	// Fallback: try using ps command
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get process path: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// getProcessName gets the process name for a PID
func getProcessName(pid int) (string, error) {
	cmd := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get process name: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
