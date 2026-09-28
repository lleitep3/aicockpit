package update

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func binaryFixture(t *testing.T) (*BinaryUpdater, string) {
	t.Helper()
	target := filepath.Join(t.TempDir(), "cockpit")
	if err := os.WriteFile(target, []byte("old"), 0700); err != nil {
		t.Fatal(err)
	}
	u := NewBinaryUpdater()
	u.platform = "linux"
	u.executable = func() (string, error) { return target, nil }
	u.build = func(_ context.Context, _, candidate, _ string) error {
		return os.WriteFile(candidate, []byte("new"), 0700)
	}
	u.probe = func(context.Context, string, string) error { return nil }
	return u, target
}
func TestBinaryUpdateRecovery(t *testing.T) {
	for _, failure := range []string{"success", "build", "candidate", "backup", "activate", "installed", "restore", "busy"} {
		t.Run(failure, func(t *testing.T) {
			u, target := binaryFixture(t)
			t.Chdir(t.TempDir())
			if err := os.WriteFile("user-file", []byte("unchanged"), 0600); err != nil {
				t.Fatal(err)
			}
			if failure == "build" {
				u.build = func(context.Context, string, string, string) error { return errors.New("build failed") }
			}
			probes := 0
			u.probe = func(_ context.Context, path, _ string) error {
				probes++
				if failure == "candidate" || ((failure == "installed" || failure == "restore") && probes == 2) {
					return errors.New("probe failed")
				}
				return nil
			}
			if failure == "backup" {
				u.build = func(_ context.Context, stage, candidate, _ string) error {
					if err := os.WriteFile(filepath.Join(stage, "previous"), nil, 0600); err != nil {
						return err
					}
					return os.WriteFile(candidate, []byte("new"), 0700)
				}
			}
			calls := 0
			u.rename = func(from, to string) error {
				calls++
				if failure == "activate" || (failure == "restore" && calls == 2) {
					return errors.New("rename failed")
				}
				return os.Rename(from, to)
			}
			if failure == "busy" {
				if err := os.WriteFile(target+".update.lock", nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			backup, err := u.Update(context.Background(), "1.2.3")
			if (err == nil) != (failure == "success") {
				t.Fatalf("unexpected outcome %s: %v", failure, err)
			}
			data, readErr := os.ReadFile(target)
			if readErr != nil {
				t.Fatal(readErr)
			}
			expected := "old"
			if failure == "success" || failure == "restore" {
				expected = "new"
			}
			if string(data) != expected {
				t.Fatalf("installed %q, expected %q", data, expected)
			}
			if failure == "success" {

				info, statErr := os.Stat(target)
				if statErr != nil || info.Mode().Perm() != 0700 {
					t.Fatalf("permissions changed: %v %v", info, statErr)
				}
				old, err := os.ReadFile(backup)
				if err != nil || string(old) != "old" {
					t.Fatalf("backup lost: %q %v", old, err)
				}
			}
			if failure == "restore" {
				if _, err := os.Stat(target + ".update.lock"); err != nil {
					t.Fatal("lock lost on failed recovery")
				}
				if !strings.Contains(err.Error(), "recover from") {
					t.Fatal(err)
				}
			}
			unchanged, err := os.ReadFile("user-file")
			if err != nil || string(unchanged) != "unchanged" {
				t.Fatal("working directory changed")
			}
		})
	}
}
func TestBinaryUpdateInputFailures(t *testing.T) {
	for _, kind := range []string{"version", "windows", "executable", "missing", "directory"} {
		t.Run(kind, func(t *testing.T) {
			u, target := binaryFixture(t)
			v := "1.2.3"
			switch kind {
			case "version":
				v = "../escape"
			case "windows":
				u.platform = "windows"
			case "executable":
				u.executable = func() (string, error) { return "", errors.New("unknown executable") }
			case "missing":
				u.executable = func() (string, error) { return target + "missing", nil }
			case "directory":
				u.executable = func() (string, error) { return filepath.Dir(target), nil }
			}
			if _, err := u.Update(context.Background(), v); err == nil {
				t.Fatal("unsafe update accepted")
			}
		})
	}
}
func TestCopyExecutableErrors(t *testing.T) {
	dir := t.TempDir()
	if err := copyExecutable(filepath.Join(dir, "missing"), filepath.Join(dir, "out"), 0700); err == nil {
		t.Fatal("missing source accepted")
	}
	if err := copyExecutable(dir, filepath.Join(dir, "out"), 0700); err == nil {
		t.Fatal("directory source accepted")
	}
}
func TestProbeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	for _, body := range []string{"echo 'cockpit version 1.2.3'", "echo 'cockpit version 9.9.9'", "exit 1"} {
		t.Run(body, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "probe")
			if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0700); err != nil {
				t.Fatal(err)
			}
			err := probeBinary(context.Background(), path, "1.2.3")
			if (err == nil) != (strings.Contains(body, "1.2.3")) {
				t.Fatal(err)
			}
		})
	}
}
func TestBuildEnvironmentIsolation(t *testing.T) {
	t.Setenv("GIT_DIR", "/wrong")
	t.Setenv("GOFLAGS", "-overlay=/wrong")
	t.Setenv("GOWORK", "/wrong")
	joined := strings.Join(buildEnvironment(), "\n")
	if strings.Contains(joined, "/wrong") {
		t.Fatal("caller overrides leaked")
	}
	if !strings.Contains(joined, "GOWORK=off") || !strings.Contains(joined, "GOFLAGS=-mod=readonly") {
		t.Fatal("build controls missing")
	}
}
func TestBuildReleaseFromLocalTag(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX updater")
	}
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "internal", "version"), 0700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"go.mod": "module github.com/lleitep3/aicockpit\n\ngo 1.26\n", "internal/version/version.go": "package version\nvar Version=\"dev\"\n", "main.go": "package main\nimport (\"fmt\"; \"github.com/lleitep3/aicockpit/internal/version\")\nfunc main(){fmt.Println(\"cockpit version\",version.Version)}\n"}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(source, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init"}, {"add", "."}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "fixture"}, {"tag", "v1.2.3"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = source
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git: %v %s", err, out)
		}
	}
	stage := t.TempDir()
	candidate := filepath.Join(stage, "candidate")
	if err := buildRelease(context.Background(), stage, candidate, "1.2.3", source); err != nil {
		t.Fatal(err)
	}
	if err := probeBinary(context.Background(), candidate, "1.2.3"); err != nil {
		t.Fatal(err)
	}
	if err := buildRelease(context.Background(), t.TempDir(), candidate, "9.9.9", source); err == nil {
		t.Fatal("missing tag accepted")
	}

	for _, args := range [][]string{{"checkout", "-b", "v1.2.3"}, {"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "different branch commit"}} {
		command := exec.Command("git", args...)
		command.Dir = source
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git collision fixture: %v %s", err, out)
		}
	}
	if err := buildRelease(context.Background(), t.TempDir(), candidate, "1.2.3", source); err == nil || !strings.Contains(err.Error(), "checkout does not match") {
		t.Fatalf("branch/tag collision accepted: %v", err)
	}
}

func TestBinaryUpdatePreservesSymlink(t *testing.T) {
	u, target := binaryFixture(t)
	link := filepath.Join(t.TempDir(), "cockpit")
	if err := os.Symlink(target, link); err != nil {
		t.Skip(err)
	}
	u.executable = func() (string, error) { return link, nil }
	if _, err := u.Update(context.Background(), "1.2.3"); err != nil {
		t.Fatal(err)
	}
	resolved, err := os.Readlink(link)
	if err != nil || resolved != target {
		t.Fatalf("link changed: %s %v", resolved, err)
	}
}
