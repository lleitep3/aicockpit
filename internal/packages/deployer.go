package packages

import (
	"fmt"
	"os"
	"os/exec"
)

// Deployer triggers redeployment of canonical assets to active providers.
type Deployer struct{}

// NewDeployer creates a new Deployer.
func NewDeployer() *Deployer {
	return &Deployer{}
}

// Trigger runs the cockpit deploy command to recompile all canonical assets
// to the active providers. cockpitBin is the path to the cockpit binary; if empty,
// the current process binary is used via os.Executable.
func (d *Deployer) Trigger(cockpitBin string) error {
	// Allow tests to skip deploy to avoid re-invoking the test binary
	if os.Getenv("COCKPIT_SKIP_DEPLOY") == "1" {
		return nil
	}

	if cockpitBin == "" {
		bin, err := os.Executable()
		if err != nil {
			return fmt.Errorf("failed to resolve cockpit binary: %w", err)
		}
		cockpitBin = bin
	}

	cmd := exec.Command(cockpitBin, "deploy") //nolint:gosec // path from os.Executable
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cockpit deploy failed: %w", err)
	}

	return nil
}
