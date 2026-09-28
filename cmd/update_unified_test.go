package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/i18n"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/update"
	"github.com/spf13/cobra"
)

func TestUnifiedUpdateOutcomes(t *testing.T) {
	for _, kind := range []string{"current", "binary-failure", "binary-updated", "check", "handoff-failure", "package-failure", "checker-failure", "dev", "binary-only", "packages-only"} {
		t.Run(kind, func(t *testing.T) {
			options := unifiedOptions{all: true, yes: true}
			checker := &mockUpdateCheckerUpdate{}
			direct, handoff, installed := 0, 0, 0
			deps := unifiedDependencies{current: "1.0.0", checker: checker, install: func(string) error { installed++; return nil }, packages: func(check bool) update.Report {
				direct++
				status := "updated"
				if check {
					status = "available"
				}
				return update.Report{Outcomes: []update.Outcome{{Component: "package", Name: "test", Status: status}}}
			}, handoff: func() (update.Report, error) {
				handoff++
				return update.Report{Outcomes: []update.Outcome{{Component: "package", Name: "child", Status: "updated"}}}, nil
			}}
			switch kind {
			case "binary-failure":
				checker.version = "2.0.0"
				deps.install = func(string) error { return errors.New("binary failure") }
			case "binary-updated", "handoff-failure":
				checker.version = "2.0.0"
			case "check":
				checker.version = "2.0.0"
				options.check = true
			case "checker-failure":
				checker.err = errors.New("offline")
			case "package-failure":
				deps.packages = func(bool) update.Report { return update.Report{Outcomes: []update.Outcome{{Status: "failed"}}} }
			case "dev":
				deps.current = "dev"
			case "binary-only":
				options.all = false
				options.binaryOnly = true
			case "packages-only":
				options.all = false
				options.packagesOnly = true
				checker.err = errors.New("must not check binary")
			}
			if kind == "handoff-failure" {
				deps.handoff = func() (update.Report, error) { handoff++; return update.Report{}, errors.New("child failed") }
			}
			report := executeUnified(options, deps)
			wantFailure := kind == "binary-failure" || kind == "checker-failure" || kind == "handoff-failure" || kind == "package-failure"
			if report.Failed() != wantFailure {
				t.Fatalf("wrong status: %+v", report)
			}
			if kind == "binary-updated" || kind == "handoff-failure" {
				if handoff != 1 || direct != 0 {
					t.Fatal("new executable not used")
				}
			} else if handoff != 0 {
				t.Fatal("unexpected handoff")
			}
			if (kind == "current" || kind == "binary-failure" || kind == "checker-failure") && direct != 1 {
				t.Fatal("packages skipped")
			}
			if options.check && installed != 0 {
				t.Fatal("check installed binary")
			}
			if kind == "binary-only" && direct != 0 {
				t.Fatal("binary-only updated packages")
			}
		})
	}
}
func TestUnifiedCommandOptionsAndReport(t *testing.T) {
	for _, kind := range []string{"check", "apply", "missing-yes", "exclusive", "report-error", "partial"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("HOME", t.TempDir())
			root := config.GetCockpitDir()
			if kind == "partial" {
				if err := os.MkdirAll(filepath.Join(root, "packages", "broken"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			command := NewUpdateCommand(nil, &config.Config{}, i18n.New("en-us"))
			output := &bytes.Buffer{}
			command.SetOut(output)
			command.SetErr(output)
			reportPath := filepath.Join(t.TempDir(), "report.json")
			args := []string{"--packages-only", "--check", "--report", reportPath}
			switch kind {
			case "apply":
				args = []string{"--packages-only", "--yes", "--report", reportPath}
			case "missing-yes":
				args = []string{"--all"}
			case "exclusive":
				args = []string{"--all", "--binary-only", "--yes"}
			case "report-error":
				args = []string{"--packages-only", "--check", "--report", filepath.Join(t.TempDir(), "missing", "report.json")}
			}
			command.SetArgs(args)
			err := command.Execute()
			wantErr := kind != "check" && kind != "apply"
			if (err != nil) != wantErr {
				t.Fatalf("outcome: %v %s", err, output)
			}
			if kind == "check" || kind == "apply" || kind == "partial" {
				data, err := os.ReadFile(reportPath)
				if err != nil {
					t.Fatal(err)
				}
				var report update.Report
				if err := json.Unmarshal(data, &report); err != nil {
					t.Fatal(err)
				}
				if report.SchemaVersion != 1 {
					t.Fatal(report)
				}
			}
		})
	}
}

func TestUnifiedHandoffSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX subprocess fixture")
	}
	for _, kind := range []string{"success", "partial", "missing", "invalid", "schema", "unexpected-exit"} {
		t.Run(kind, func(t *testing.T) {
			report := update.Report{SchemaVersion: 1, Outcomes: []update.Outcome{{Component: "package", Name: "child", Status: "updated"}}}
			exitCode := 0
			if kind == "partial" {
				report.Outcomes[0].Status = "failed"
				exitCode = 1
			}
			if kind == "schema" {
				report.SchemaVersion = 2
			}
			if kind == "unexpected-exit" {
				exitCode = 1
			}
			data, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			if kind == "invalid" {
				data = []byte("invalid json")
			}
			script := "#!/bin/sh\n[ \"$1\" = update ] && [ \"$2\" = --packages-only ] && [ \"$3\" = --yes ] && [ \"$4\" = --report ] || exit 99\n"
			if kind != "missing" {
				script += fmt.Sprintf("printf '%%s' '%s' > \"$5\"\n", data)
			}
			script += fmt.Sprintf("exit %d\n", exitCode)
			executable := filepath.Join(t.TempDir(), "new-cockpit")
			if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			command := &cobra.Command{}
			command.SetContext(context.Background())
			command.SetOut(&bytes.Buffer{})
			command.SetErr(&bytes.Buffer{})
			got, err := handoffPackagesTo(command, executable)
			wantErr := kind != "success" && kind != "partial"
			if (err != nil) != wantErr {
				t.Fatalf("handoff: %+v %v", got, err)
			}
			if kind == "partial" && !got.Failed() {
				t.Fatal("child failure lost")
			}
		})
	}
}

func TestUnifiedCommandRealPackageLifecycle(t *testing.T) {
	root := setupLocalGitRegistry(t)
	service, cfg := testPkgArgs(t)
	if err := service.InstallPackage(filepath.Join(root, "registry-work", "hello-pkg"), nil); err != nil {
		t.Fatal(err)
	}
	installed := service.GetPackageInstallPath("hello-pkg")
	pkg, err := packages.LoadPackage(installed)
	if err != nil {
		t.Fatal(err)
	}
	pkg.Version = "1.0.0"
	if err := packages.SavePackage(installed, pkg); err != nil {
		t.Fatal(err)
	}
	for _, step := range []struct{ flag, status string }{{"--check", "available"}, {"--yes", "updated"}, {"--yes", "current"}} {
		command := NewUpdateCommand(nil, cfg, i18n.New("en-us"))
		command.SetOut(&bytes.Buffer{})
		command.SetErr(&bytes.Buffer{})
		path := filepath.Join(t.TempDir(), "outcomes.json")
		command.SetArgs([]string{"--packages-only", step.flag, "--report", path})
		if err := command.Execute(); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var report update.Report
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if len(report.Outcomes) == 0 || report.Outcomes[0].Status != step.status {
			t.Fatalf("unexpected lifecycle outcome: %+v", report)
		}
		pkg, err := packages.LoadPackage(installed)
		if err != nil {
			t.Fatal(err)
		}
		expected := "2.0.0"
		if step.flag == "--check" {
			expected = "1.0.0"
		}
		if pkg.Version != expected {
			t.Fatalf("installed %s, expected %s", pkg.Version, expected)
		}
	}
}
