package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lleite/aicockpit/internal/config"
	"github.com/lleite/aicockpit/internal/i18n"
	"github.com/lleite/aicockpit/internal/logging"
	"github.com/spf13/cobra"
)

// NewMetricsCommand creates the metrics command
func NewMetricsCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "View execution metrics and statistics",
		Long:  "View and analyze execution metrics and statistics",
	}

	cmd.AddCommand(NewMetricsListCommand(log, cfg, t))
	cmd.AddCommand(NewMetricsStatsCommand(log, cfg, t))
	cmd.AddCommand(NewMetricsLogsCommand(log, cfg, t))

	return cmd
}

// NewMetricsListCommand creates the metrics list command
func NewMetricsListCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var (
		command string
		status  string
		limit   int
		date    string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List execution metrics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return listMetrics(log, cfg, t, command, status, limit, date)
		},
	}

	cmd.Flags().StringVar(&command, "command", "", "Filter by command name")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status (success/error)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Limit number of results")
	cmd.Flags().StringVar(&date, "date", "", "Filter by date (YYYY-MM-DD)")

	return cmd
}

// NewMetricsStatsCommand creates the metrics stats command
func NewMetricsStatsCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show execution statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showStats(log, cfg, t)
		},
	}
}

// NewMetricsLogsCommand creates the metrics logs command
func NewMetricsLogsCommand(log *logging.Manager, cfg *config.Config, t *i18n.Translator) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show log files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showLogs(log, cfg, t, date)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Show logs for specific date (YYYY-MM-DD)")

	return cmd
}

func listMetrics(log *logging.Manager, cfg *config.Config, t *i18n.Translator, command, status string, limit int, dateStr string) error {
	metrics := log.GetMetrics()

	// Filter metrics
	var filtered []logging.ExecutionMetric
	for _, m := range metrics.GetMetrics() {
		if command != "" && m.Command != command {
			continue
		}
		if status != "" && m.Status != status {
			continue
		}
		if dateStr != "" {
			targetDate, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				fmt.Printf("Invalid date format: %s\n", dateStr)
				return nil
			}
			if m.Timestamp.Format("2006-01-02") != targetDate.Format("2006-01-02") {
				continue
			}
		}
		filtered = append(filtered, m)
	}

	// Apply limit
	if limit > 0 && len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}

	// Display results
	fmt.Println("Execution Metrics")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	if len(filtered) == 0 {
		fmt.Println("No metrics found")
		return nil
	}

	for _, m := range filtered {
		fmt.Printf("Command: %s\n", m.Command)
		fmt.Printf("  Timestamp: %s\n", m.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("  Status: %s\n", m.Status)
		fmt.Printf("  Exit Code: %d\n", m.ExitCode)
		fmt.Printf("  Duration: %.2f ms\n", m.Duration)
		fmt.Printf("  User: %s\n", m.User)
		if m.Error != "" {
			fmt.Printf("  Error: %s\n", m.Error)
		}
		fmt.Println()
	}

	fmt.Printf("Total: %d metrics\n", len(filtered))

	return nil
}

func showStats(log *logging.Manager, cfg *config.Config, t *i18n.Translator) error {
	metrics := log.GetMetrics()
	stats := metrics.GetStats()

	fmt.Println("Execution Statistics")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	fmt.Printf("Total Executions: %v\n", stats["total_executions"])
	fmt.Printf("Successful: %v\n", stats["successful"])
	fmt.Printf("Failed: %v\n", stats["failed"])

	if successRate, ok := stats["success_rate"].(float64); ok {
		fmt.Printf("Success Rate: %.2f%%\n", successRate)
	}

	fmt.Printf("Total Duration: %.2f ms\n", stats["total_duration_ms"])
	fmt.Printf("Average Duration: %.2f ms\n", stats["avg_duration_ms"])

	fmt.Println()
	fmt.Println("Commands:")
	if commands, ok := stats["commands"].(map[string]int); ok {
		for cmd, count := range commands {
			fmt.Printf("  %s: %d\n", cmd, count)
		}
	}

	fmt.Println()
	fmt.Println("Error Types:")
	if errorTypes, ok := stats["error_types"].(map[string]int); ok {
		if len(errorTypes) == 0 {
			fmt.Println("  No errors")
		} else {
			for errType, count := range errorTypes {
				fmt.Printf("  %s: %d\n", errType, count)
			}
		}
	}

	return nil
}

func showLogs(log *logging.Manager, cfg *config.Config, t *i18n.Translator, dateStr string) error {
	fileLogger := log.GetFileLogger()

	var logs []string
	var err error

	if dateStr != "" {
		targetDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			fmt.Printf("Invalid date format: %s\n", dateStr)
			return nil
		}
		logPath := fileLogger.GetLogsForDate(targetDate)
		if _, err := os.Stat(logPath); err == nil {
			logs = []string{logPath}
		}
	} else {
		logs, err = fileLogger.GetAllLogs()
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil
		}
	}

	fmt.Println("Log Files")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	if len(logs) == 0 {
		fmt.Println("No log files found")
		return nil
	}

	for _, logPath := range logs {
		info, err := os.Stat(logPath)
		if err != nil {
			continue
		}

		fmt.Printf("File: %s\n", filepath.Base(logPath))
		fmt.Printf("  Size: %d bytes\n", info.Size())
		fmt.Printf("  Modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))

		// Count lines
		data, err := os.ReadFile(logPath)
		if err == nil {
			lines := strings.Count(string(data), "\n")
			fmt.Printf("  Lines: %d\n", lines)
		}

		fmt.Println()
	}

	return nil
}
