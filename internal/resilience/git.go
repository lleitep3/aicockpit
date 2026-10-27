package resilience

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// GitRunner runs git commands with retry on transient failures.
type GitRunner struct {
	config Config
}

// NewGitRunner creates a new GitRunner with the provided retry configuration.
func NewGitRunner(config Config) *GitRunner {
	return &GitRunner{config: config}
}

// DefaultGitRunner returns a GitRunner with the default retry configuration.
func DefaultGitRunner() *GitRunner {
	return NewGitRunner(DefaultConfig())
}

// Run executes a git command with retries. The command name must be "git";
// args are the git arguments. It returns CombinedOutput-style error messages.
func (gr *GitRunner) Run(ctx context.Context, dir string, args ...string) ([]byte, error) {
	var output []byte
	op := func() error {
		cmd := exec.CommandContext(ctx, gitBinary(), args...)
		if dir != "" {
			cmd.Dir = dir
		}

		var err error
		output, err = cmd.CombinedOutput()
		if err != nil {
			return AsRetryable(fmt.Errorf("git %s failed: %w\n%s", args[0], err, string(output)))
		}
		return nil
	}

	if err := Retry(ctx, gr.config, op); err != nil {
		return output, err
	}
	return output, nil
}

// AllowTestOverride permits tests to swap the underlying git binary path
// via the COCKPIT_TEST_GIT environment variable.
func gitBinary() string {
	if bin := os.Getenv("COCKPIT_TEST_GIT"); bin != "" {
		return bin
	}
	return "git"
}
