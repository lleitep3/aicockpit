package main

import (
	"fmt"
	"os"

	"github.com/lleite/aicockpit/cmd"
	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logger"
)

func main() {
	// Initialize logger
	log := logger.New()
	defer log.Close()

	// Initialize config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Initialize translator
	t := i18n.New(cfg.Language)

	// Execute CLI
	rootCmd := cmd.NewRootCommand(log, cfg, t)
	if err := rootCmd.Execute(); err != nil {
		log.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
