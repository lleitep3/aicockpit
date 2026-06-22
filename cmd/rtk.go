package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/spf13/cobra"
)

// NewRtkCommand creates the rtk command.
func NewRtkCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	cmd := &cobra.Command{
		Use:       "rtk [on|off|status]",
		Short:     "Manage the rtk prefix global rule",
		Long:      "Activates or deactivates the RTK prefix rule globally across providers.",
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"on", "off", "status"},
		RunE: func(cmd *cobra.Command, args []string) error {
			action := args[0]
			cockpitDir := config.GetCockpitDir()
			rulePath := filepath.Join(cockpitDir, "rules", "rtk.md")

			switch action {
			case "on":
				return enableRtk(rulePath, cockpitDir, cfg, t)
			case "off":
				return disableRtk(rulePath, cockpitDir, cfg, t)
			case "status":
				return statusRtk(rulePath, t)
			default:
				return fmt.Errorf("%s", t.T("rtk.invalid", action))
			}
		},
	}
	return cmd
}

func enableRtk(rulePath, cockpitDir string, cfg *config.Config, t *i18n.Translator) error {
	// Validate if rtk exists in PATH
	if _, err := exec.LookPath("rtk"); err != nil {
		fmt.Println(t.T("rtk.not_found_warning"))
		// We just warn, but we proceed
	}

	// Create rules dir if not exists
	rulesDir := filepath.Dir(rulePath)
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return fmt.Errorf("failed to create rules directory: %w", err)
	}

	content := `<!-- cockpit:rtk -->
Always prefix terminal commands with ` + "`rtk`" + ` when available.
RTK transparently proxies every command and optimizes output tokens.
Examples: rtk git status, rtk go test ./..., rtk grep <pattern>
<!-- /cockpit:rtk -->`

	if err := os.WriteFile(rulePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write rtk rule: %w", err)
	}

	fmt.Println(t.T("rtk.enabled"))
	return runDeploy(cockpitDir, cfg, t)
}

func disableRtk(rulePath, cockpitDir string, cfg *config.Config, t *i18n.Translator) error {
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		fmt.Println(t.T("rtk.already_disabled"))
		return nil
	}

	if err := os.Remove(rulePath); err != nil {
		return fmt.Errorf("failed to remove rtk rule: %w", err)
	}

	fmt.Println(t.T("rtk.disabled"))
	return runDeploy(cockpitDir, cfg, t)
}

func statusRtk(rulePath string, t *i18n.Translator) error {
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		fmt.Println(t.T("rtk.off"))
	} else {
		fmt.Println(t.T("rtk.on"))
	}
	return nil
}
