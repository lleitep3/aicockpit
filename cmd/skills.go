package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/skillimport"
	"github.com/spf13/cobra"
)

// NewSkillsCommand creates commands for managing interoperable agent skills.
func NewSkillsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Import local agent skills into the Cockpit library",
		Long:  "Import GitHub Copilot and other Agent Skills directories into ~/.cockpit/skills for deployment to configured providers.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(NewSkillsImportCommand())
	return cmd
}

// NewSkillsImportCommand creates the local skill import command.
func NewSkillsImportCommand() *cobra.Command {
	var dryRun bool
	var jsonOutput bool
	var saveConflicts string
	cmd := &cobra.Command{
		Use:   "import <skill-directory-or-library>",
		Short: "Import local Agent Skills without overwriting existing skills",
		Long: `Import a directory containing SKILL.md, or a library whose immediate child
directories contain SKILL.md. The complete directory tree is copied into
~/.cockpit/skills. Same-name skills are left untouched and reported as unchanged
or conflicts. Use --dry-run to preview the result. Pass --save-conflicts with a
new directory to preserve conflicting local versions there for review; this
directory stays outside the Cockpit library and is not deployed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cockpitDir := config.GetCockpitDir()
			report, err := skillimport.ImportLocalWithConflictArchive(args[0], filepath.Join(cockpitDir, "skills"), saveConflicts, dryRun)
			if err != nil {
				return err
			}
			if jsonOutput {
				return writeSkillsImportJSON(cmd.OutOrStdout(), report)
			}
			return writeSkillsImportSummary(cmd.OutOrStdout(), report)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show planned imports without writing files")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Write a machine-readable JSON report")
	cmd.Flags().StringVar(&saveConflicts, "save-conflicts", "", "Copy conflicting local skills into a new review directory")
	return cmd
}

func writeSkillsImportSummary(writer io.Writer, report skillimport.Report) error {
	if _, err := fmt.Fprintf(writer, "Source: %s\nTarget: %s\n", report.Source, report.Target); err != nil {
		return err
	}
	for _, item := range report.Items {
		label := item.Status
		switch item.Status {
		case "imported":
			label = "Imported"
		case "would-import":
			label = "Would import"
		case "unchanged":
			label = "Unchanged"
		case "conflict":
			label = "Conflict"
		case "would-save-conflict":
			label = "Would save for review"
		case "saved-conflict":
			label = "Saved for review"
		}
		if item.Reason != "" {
			if _, err := fmt.Fprintf(writer, "  %s: %s (%s)\n", label, item.Name, item.Reason); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(writer, "  %s: %s\n", label, item.Name); err != nil {
			return err
		}
	}
	if report.ReviewTarget != "" {
		if _, err := fmt.Fprintf(writer, "Review copy: %s\n", report.ReviewTarget); err != nil {
			return err
		}
	}
	if report.DryRun {
		_, err := fmt.Fprintln(writer, "Preview only; no files were changed.")
		return err
	}
	_, err := fmt.Fprintln(writer, "Existing skills were not overwritten.")
	return err
}

func writeSkillsImportJSON(writer io.Writer, report skillimport.Report) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}
