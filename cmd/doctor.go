package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/spf13/cobra"
)

// NewDoctorCommand creates the doctor command.
func NewDoctorCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: t.T("doctor.title"),
		Long:  t.T("doctor.title"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(log, cfg, t)
		},
	}
}

func runDoctor(log *logging.Manager, cfg *config.Config, t *i18n.Translator) error {
	startTime := time.Now()
	fmt.Println(t.T("doctor.title"))
	fmt.Println("=" + strings.Repeat("=", 49))
	fmt.Println()

	cockpitDir := config.GetCockpitDir()
	configPath := config.GetConfigPath()
	vaultPath := filepath.Join(cockpitDir, "vault")
	logsPath := filepath.Join(cockpitDir, "logs")
	packagesPath := filepath.Join(cockpitDir, "packages")
	cachePath := filepath.Join(cockpitDir, "cache")

	allOk := true

	// Check 1: Cockpit directory
	fmt.Printf(t.T("doctor.checking")+"\n", "Cockpit directory")
	if _, err := os.Stat(cockpitDir); err == nil {
		fmt.Printf(t.T("doctor.ok")+"\n", "Cockpit directory exists")
	} else {
		fmt.Printf(t.T("doctor.failed")+"\n", "Cockpit directory not found")
		allOk = false
	}

	// Check 2: Configuration file
	fmt.Printf(t.T("doctor.checking")+"\n", "Configuration file")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Printf(t.T("doctor.ok")+"\n", t.T("doctor.config_ok"))
	} else {
		fmt.Printf(t.T("doctor.failed")+"\n", t.T("doctor.config_bad"))
		allOk = false
	}

	// Check 3: Vault
	fmt.Printf(t.T("doctor.checking")+"\n", "Vault")
	if _, err := os.Stat(vaultPath); err == nil {
		fmt.Printf(t.T("doctor.ok")+"\n", t.T("doctor.vault_ok"))
	} else {
		fmt.Printf(t.T("doctor.failed")+"\n", t.T("doctor.vault_bad"))
		allOk = false
	}

	// Check 4: Logs directory
	fmt.Printf(t.T("doctor.checking")+"\n", "Logs directory")
	if _, err := os.Stat(logsPath); err == nil {
		fmt.Printf(t.T("doctor.ok")+"\n", "Logs directory exists")
	} else {
		fmt.Printf(t.T("doctor.failed")+"\n", "Logs directory not found")
		allOk = false
	}

	// Check 5: Packages directory
	fmt.Printf(t.T("doctor.checking")+"\n", "Packages directory")
	if _, err := os.Stat(packagesPath); err == nil {
		fmt.Printf(t.T("doctor.ok")+"\n", "Packages directory exists")
	} else {
		fmt.Printf(t.T("doctor.failed")+"\n", "Packages directory not found")
		allOk = false
	}

	// Check 6: Cache directory
	fmt.Printf(t.T("doctor.checking")+"\n", "Cache directory")
	if _, err := os.Stat(cachePath); err == nil {
		fmt.Printf(t.T("doctor.ok")+"\n", "Cache directory exists")
	} else {
		fmt.Printf(t.T("doctor.failed")+"\n", "Cache directory not found")
		allOk = false
	}

	fmt.Println()

	if allOk {
		fmt.Println(t.T("doctor.passed"))
	} else {
		fmt.Println(t.T("doctor.failed_msg"))
	}

	duration := time.Since(startTime)
	status := "success"
	if !allOk {
		status = "error"
	}
	log.LogCommand("doctor", []string{}, status, 0, duration, "", nil)

	return nil
}
