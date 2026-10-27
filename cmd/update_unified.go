package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/update"
	"github.com/lleitep3/aicockpit/internal/version"
	"github.com/spf13/cobra"
)

type unifiedOptions struct {
	all, binaryOnly, packagesOnly, yes, check bool
	report                                    string
}
type unifiedDependencies struct {
	current  string
	checker  updateChecker
	install  func(string) error
	packages func(bool) update.Report
	handoff  func() (update.Report, error)
}

func executeUnified(options unifiedOptions, deps unifiedDependencies) update.Report {
	report := update.Report{SchemaVersion: 1, Check: options.check, Outcomes: []update.Outcome{}}
	binaryChanged := false
	if !options.packagesOnly {
		outcome := update.Outcome{Component: "binary", Name: "cockpit", From: deps.current, Status: "current"}
		latest, _, err := deps.checker.CheckForUpdates()
		switch {
		case err != nil:
			outcome.Status = "failed"
			outcome.Message = err.Error()
		case deps.current == "dev":
			outcome.Status = "skipped"
			outcome.Message = "development build has no release version"
		case latest != "":
			outcome.To = latest
			outcome.Status = "available"
			if !options.check {
				if err := deps.install(latest); err != nil {
					outcome.Status = "failed"
					outcome.Message = err.Error()
				} else {
					outcome.Status = "updated"
					binaryChanged = true
				}
			}
		}
		report.Outcomes = append(report.Outcomes, outcome)
	}
	if options.all || options.packagesOnly {
		if binaryChanged {
			child, err := deps.handoff()
			report.Outcomes = append(report.Outcomes, child.Outcomes...)
			if err != nil {
				report.Outcomes = append(report.Outcomes, update.Outcome{Component: "packages", Name: "handoff", Status: "failed", Message: err.Error()})
			}
		} else {
			report.Outcomes = append(report.Outcomes, deps.packages(options.check).Outcomes...)
		}
	}
	return report
}
func runUnifiedUpdate(command *cobra.Command, cfg *config.Config, options unifiedOptions) error {
	if !options.check && !options.yes {
		return fmt.Errorf("scoped update requires --yes to apply; use --check to inspect first")
	}
	command.Println("Checking selected components...")
	report := executeUnified(options, unifiedDependencies{
		current: version.GetVersion(), checker: update.NewUpdateService(), install: performUpdateFunc,
		packages: update.NewPackageBatch(config.GetCockpitDir(), cfg.PackageRegistries).Run,
		handoff:  func() (update.Report, error) { return handoffPackages(command) },
	})
	for _, outcome := range report.Outcomes {
		command.Printf("%s %s: %s", outcome.Component, outcome.Name, outcome.Status)
		if outcome.To != "" {
			command.Printf(" (%s -> %s)", outcome.From, outcome.To)
		}
		if outcome.Message != "" {
			command.Printf(" — %s", outcome.Message)
		}
		command.Println()
	}
	if options.report != "" {
		if err := writeUpdateReport(options.report, report); err != nil {
			return fmt.Errorf("update finished but report could not be written: %w", err)
		}
	}
	if report.Failed() {
		return fmt.Errorf("update completed with failures; inspect component outcomes")
	}
	return nil
}
func writeUpdateReport(path string, report update.Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0600)
}
func handoffPackages(command *cobra.Command) (update.Report, error) {
	executable, err := os.Executable()
	if err != nil {
		return update.Report{}, err
	}
	return handoffPackagesTo(command, executable)
}

func handoffPackagesTo(command *cobra.Command, executable string) (update.Report, error) {
	dir, err := os.MkdirTemp("", "cockpit-update-report-")
	if err != nil {
		return update.Report{}, err
	}
	path := filepath.Join(dir, "report.json")
	child := exec.CommandContext(command.Context(), executable, "update", "--packages-only", "--yes", "--report", path)
	child.Stdout = command.OutOrStdout()
	child.Stderr = command.ErrOrStderr()
	runErr := child.Run()
	data, err := os.ReadFile(path)
	if err != nil {
		return update.Report{}, fmt.Errorf("new executable did not return report at %s: %w", path, err)
	}
	var report update.Report
	if err := json.Unmarshal(data, &report); err != nil {
		return update.Report{}, err
	}
	if report.SchemaVersion != 1 {
		return update.Report{}, fmt.Errorf("unsupported child report schema")
	}
	if runErr != nil && !report.Failed() {
		return report, fmt.Errorf("new executable failed: %w", runErr)
	}
	return report, nil
}
