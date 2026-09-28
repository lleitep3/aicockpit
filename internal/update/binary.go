package update

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const officialSource = "https://github.com/lleitep3/aicockpit.git"

// BinaryUpdater builds in isolation and activates only a verified executable.
// File operations are ports so recovery failures can be exercised deterministically.
type BinaryUpdater struct {
	executable func() (string, error)
	build      func(context.Context, string, string, string) error
	probe      func(context.Context, string, string) error
	rename     func(string, string) error
	platform   string
}

func NewBinaryUpdater() *BinaryUpdater {
	return &BinaryUpdater{executable: os.Executable, build: buildOfficialBinary, probe: probeBinary, rename: os.Rename, platform: runtime.GOOS}
}

func (u *BinaryUpdater) Update(ctx context.Context, version string) (string, error) {
	if !semver.IsValid("v"+version) || semver.Canonical("v"+version) != "v"+version {
		return "", fmt.Errorf("invalid release version: %s", version)
	}
	if u.platform == "windows" {
		return "", fmt.Errorf("automatic binary replacement on Windows requires a helper; install the release manually")
	}
	executable, err := u.executable()
	if err != nil {
		return "", err
	}
	target, err := filepath.EvalSymlinks(executable)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(target)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("executable is not a regular file")
	}
	lockPath := target + ".update.lock"
	lock, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", fmt.Errorf("cannot lock binary update; verify permissions and reconcile %s if no updater is running: %w", lockPath, err)
	}
	keepLock := false
	defer func() {
		if !keepLock {
			_ = os.Remove(lockPath)
		}
	}()
	if err := lock.Close(); err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(filepath.Dir(target), ".cockpit-update-")
	if err != nil {
		return "", err
	}
	candidate := filepath.Join(stage, "candidate")
	if err := u.build(ctx, stage, candidate, version); err != nil {
		return stage, fmt.Errorf("prepare binary: %w", err)
	}
	if err := os.Chmod(candidate, info.Mode().Perm()); err != nil {
		return stage, fmt.Errorf("preserve executable permissions: %w", err)
	}
	if err := u.probe(ctx, candidate, version); err != nil {
		return stage, fmt.Errorf("candidate validation: %w", err)
	}
	backup := filepath.Join(stage, "previous")
	if err := copyExecutable(target, backup, info.Mode()); err != nil {
		return stage, fmt.Errorf("backup executable: %w", err)
	}
	if err := u.rename(candidate, target); err != nil {
		return stage, fmt.Errorf("activate executable: %w", err)
	}
	if err := u.probe(ctx, target, version); err != nil {
		if restoreErr := u.rename(backup, target); restoreErr != nil {
			keepLock = true
			return stage, fmt.Errorf("installed binary validation: %w; restore failed: %w; recover from %s", err, restoreErr, backup)
		}
		return stage, fmt.Errorf("installed binary validation: %w; previous executable restored", err)
	}
	return backup, nil
}

func copyExecutable(source, dest string, mode os.FileMode) (result error) {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	defer func() {
		if err := out.Close(); result == nil {
			result = err
		}
	}()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func probeBinary(ctx context.Context, path, version string) error {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "--version").Output()
	if err != nil {
		return fmt.Errorf("executable probe: %w", err)
	}
	if strings.TrimSpace(string(out)) != "cockpit version "+version {
		return fmt.Errorf("executable version does not match %s", version)
	}
	return nil
}

// buildEnvironment avoids caller Git/Go settings redirecting the isolated build.
func buildEnvironment() []string {
	env := []string{}
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "GIT_") || key == "GOFLAGS" || key == "GOWORK" || key == "GOENV" || key == "GOOS" || key == "GOARCH" || key == "CGO_ENABLED" {
			continue
		}
		env = append(env, entry)
	}
	return append(env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0", "GOFLAGS=-mod=readonly", "GOWORK=off", "GOENV=off", "GOOS="+runtime.GOOS, "GOARCH="+runtime.GOARCH, "CGO_ENABLED=0")
}

func buildOfficialBinary(ctx context.Context, stage, candidate, version string) error {
	return buildRelease(ctx, stage, candidate, version, officialSource)
}

func buildRelease(ctx context.Context, stage, candidate, version, sourceURL string) error {
	source := filepath.Join(stage, "source")
	run := func(dir, name string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		cmd.Env = buildEnvironment()
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%s failed: %w: %s", name, err, output)
		}
		return strings.TrimSpace(string(output)), nil
	}
	if _, err := run(stage, "git", "clone", "--depth", "1", "--branch", "v"+version, "--", sourceURL, source); err != nil {
		return err
	}
	head, err := run(source, "git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	tag, err := run(source, "git", "rev-parse", "refs/tags/v"+version+"^{commit}")
	if err != nil {
		return err
	}
	if head != tag {
		return fmt.Errorf("checkout does not match requested release tag")
	}
	_, err = run(source, "go", "build", "-trimpath", "-ldflags=-X github.com/lleitep3/aicockpit/internal/version.Version="+version, "-o", candidate, ".")
	return err
}
