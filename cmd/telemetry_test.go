package cmd

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestDecorateCommandsLogsRunESuccess(t *testing.T) {
	log, _, _ := newTestDeps(t)
	root := &cobra.Command{Use: "cockpit"}
	group := &cobra.Command{Use: "pkg"}
	group.AddCommand(&cobra.Command{
		Use: "sample",
		RunE: func(_ *cobra.Command, _ []string) error {
			return nil
		},
	})
	root.AddCommand(group)

	decorateCommands(root, log)
	root.SetArgs([]string{"pkg", "sample", "first", "second"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	metrics := log.GetMetrics().GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("logged metrics = %d, want 1", len(metrics))
	}
	metric := metrics[0]
	if metric.Command != "pkg sample" {
		t.Errorf("Command = %q, want %q", metric.Command, "pkg sample")
	}
	if !reflect.DeepEqual(metric.Args, []string{"first", "second"}) {
		t.Errorf("Args = %v, want %v", metric.Args, []string{"first", "second"})
	}
	if metric.Status != "success" {
		t.Errorf("Status = %q, want %q", metric.Status, "success")
	}
	if metric.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", metric.ExitCode)
	}
	if metric.Error != "" {
		t.Errorf("Error = %q, want empty", metric.Error)
	}
}

func TestDecorateCommandsLogsRunEError(t *testing.T) {
	log, _, _ := newTestDeps(t)
	runErr := errors.New("command failed")
	root := &cobra.Command{Use: "cockpit"}
	root.AddCommand(&cobra.Command{
		Use: "failing",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runErr
		},
	})

	decorateCommands(root, log)
	root.SetArgs([]string{"failing"})

	if err := root.Execute(); !errors.Is(err, runErr) {
		t.Fatalf("Execute() error = %v, want %v", err, runErr)
	}

	metrics := log.GetMetrics().GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("logged metrics = %d, want 1", len(metrics))
	}
	metric := metrics[0]
	if metric.Status != "error" {
		t.Errorf("Status = %q, want %q", metric.Status, "error")
	}
	if metric.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", metric.ExitCode)
	}
	if metric.Error != runErr.Error() {
		t.Errorf("Error = %q, want %q", metric.Error, runErr.Error())
	}
}

func TestDecorateCommandsPreservesExitCode(t *testing.T) {
	if os.Getenv("GO_WANT_TELEMETRY_HELPER_PROCESS") == "1" {
		os.Exit(42)
	}

	runCmd := exec.Command(os.Args[0], "-test.run=TestDecorateCommandsPreservesExitCode")
	runCmd.Env = append(os.Environ(), "GO_WANT_TELEMETRY_HELPER_PROCESS=1")
	runErr := runCmd.Run()
	log, _, _ := newTestDeps(t)
	root := &cobra.Command{Use: "cockpit"}
	root.AddCommand(&cobra.Command{
		Use: "failing",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runErr
		},
	})

	decorateCommands(root, log)
	root.SetArgs([]string{"failing"})

	if err := root.Execute(); err == nil {
		t.Fatal("Execute() error = nil, want exit error")
	}

	metrics := log.GetMetrics().GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("logged metrics = %d, want 1", len(metrics))
	}
	if metrics[0].ExitCode != 42 {
		t.Errorf("ExitCode = %d, want 42", metrics[0].ExitCode)
	}
}

func TestDecorateCommandsLogsCustomStatusForRun(t *testing.T) {
	log, _, _ := newTestDeps(t)
	root := &cobra.Command{Use: "cockpit"}
	root.AddCommand(&cobra.Command{
		Use:         "warning",
		Annotations: map[string]string{"telemetry_status": "error"},
		Run: func(_ *cobra.Command, _ []string) {
		},
	})

	decorateCommands(root, log)
	root.SetArgs([]string{"warning"})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	metrics := log.GetMetrics().GetMetrics()
	if len(metrics) != 1 {
		t.Fatalf("logged metrics = %d, want 1", len(metrics))
	}
	metric := metrics[0]
	if metric.Status != "error" {
		t.Errorf("Status = %q, want %q", metric.Status, "error")
	}
	if metric.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", metric.ExitCode)
	}
}
