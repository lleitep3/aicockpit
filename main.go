package main

import (
	"fmt"
	"os"

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
		fmt.Fprintf(os.Stderr, "Error: Failed to initialize logging: %v\n", err)
		os.Exit(1)
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
		os.Exit(1)
	}
}
