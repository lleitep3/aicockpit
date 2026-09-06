package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/lleitep3/aicockpit/cmd"
	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
)

func main() {
	// Initialize config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize logging manager
	cockpitDir := config.GetCockpitDir()
	log, err := logging.NewManager(cockpitDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: unable to record Cockpit logs: %v. Logging is important for understanding AI usage.\n", err)
	}
	defer log.Close()

	// Initialize translator
	t := i18n.New(cfg.Language)

	// Execute CLI
	rootCmd := cmd.NewRootCommand(log, cfg, t)
	if err := rootCmd.Execute(); err != nil {
		log.LogError("Command execution failed", map[string]interface{}{
			"error": err.Error(),
		})

		exitCode := 1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		os.Exit(exitCode)
	}
}
