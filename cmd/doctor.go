package cmd

import (
	"encoding/json"
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
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: t.T("doctor.title"),
		Long:  t.T("doctor.title"),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDoctor(log, cfg, t, jsonOutput)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output doctor results as JSON")
	return cmd
}

func runDoctor(log *logging.Manager, cfg *config.Config, t *i18n.Translator, jsonOutput bool) error {
	startTime := time.Now()

	cockpitDir := config.GetCockpitDir()
	configPath := config.GetConfigPath()
	vaultPath := filepath.Join(cockpitDir, "vault")
	logsPath := filepath.Join(cockpitDir, "logs")
	packagesPath := filepath.Join(cockpitDir, "packages")
	cachePath := filepath.Join(cockpitDir, "cache")

	checks := []map[string]interface{}{
		{"name": "Cockpit directory", "path": cockpitDir},
		{"name": "Configuration file", "path": configPath},
		{"name": "Vault", "path": vaultPath},
		{"name": "Logs directory", "path": logsPath},
		{"name": "Packages directory", "path": packagesPath},
		{"name": "Cache directory", "path": cachePath},
	}

	results := make([]map[string]interface{}, 0, len(checks))
	allOk := true

	for _, check := range checks {
		name := check["name"].(string)
		path := check["path"].(string)
		_, err := os.Stat(path)
		ok := err == nil
		if !ok {
			allOk = false
		}
		results = append(results, map[string]interface{}{
			"check_name":  name,
			"status":      map[bool]string{true: "ok", false: "error"}[ok],
			"message":     map[bool]string{true: name + " exists", false: name + " not found"}[ok],
			"fixable":     false,
			"fix_command": "",
		})
	}

	if jsonOutput {
		output, err := json.Marshal(map[string]interface{}{
			"passed": allOk,
			"checks": results,
		})
		if err != nil {
			return fmt.Errorf("failed to marshal doctor output: %w", err)
		}
		fmt.Println(string(output))
	} else {
		fmt.Println(t.T("doctor.title"))
		fmt.Println("=" + strings.Repeat("=", 49))
		fmt.Println()
		for _, result := range results {
			fmt.Printf(t.T("doctor.checking")+"\n", result["check_name"])
			if result["status"] == "ok" {
				fmt.Printf(t.T("doctor.ok")+"\n", result["message"])
			} else {
				fmt.Printf(t.T("doctor.failed")+"\n", result["message"])
			}
		}
		fmt.Println()
		if allOk {
			fmt.Println(t.T("doctor.passed"))
		} else {
			fmt.Println(t.T("doctor.failed_msg"))
		}
	}

	duration := time.Since(startTime)
	status := "success"
	if !allOk {
		status = "error"
	}
	log.LogCommand("doctor", []string{}, status, 0, duration, "", nil)

	return nil
}
