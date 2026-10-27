package packages

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func registryGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	args = append([]string{"-C", dir, "-c", "commit.gpgsign=false", "-c", "user.name=Test", "-c", "user.email=test@example.invalid"}, args...)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git: %v %s", err, out)
	}
}
func registryFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
func registryRemote(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	registryGit(t, dir, "init", "-b", "main")
	registryFile(t, dir, "package-index.yaml", "version: 1.0\npackages: []\n")
	registryGit(t, dir, "add", ".")
	registryGit(t, dir, "commit", "-m", "init")
	return dir
}
func TestRegistryRefreshPreservesDivergentCache(t *testing.T) {
	remote := registryRemote(t)
	rc := NewRegistryCache(t.TempDir())
	reg := RegistryConfig{Name: "test", URL: remote, Branch: "main"}
	if err := rc.EnsureRegistry(reg); err != nil {
		t.Fatal(err)
	}
	cache := rc.GetRegistryCachePath("test")
	registryFile(t, cache, "local.txt", "local commit")
	registryGit(t, cache, "add", ".")
	registryGit(t, cache, "commit", "-m", "local")
	registryFile(t, cache, "untracked.txt", "user file")
	registryFile(t, remote, "remote.txt", "new upstream")
	registryGit(t, remote, "add", ".")
	registryGit(t, remote, "commit", "-m", "remote")
	if err := rc.EnsureRegistry(reg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cache, "remote.txt")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"local.txt", "untracked.txt"} {
		backups, err := filepath.Glob(filepath.Join(filepath.Dir(cache), ".refresh-*", "previous", name))
		if err != nil || len(backups) != 1 {
			t.Fatalf("lost %s: %v %v", name, backups, err)
		}
	}
}
func TestRegistryRefreshUsesConfiguredOrigin(t *testing.T) {
	first, second := registryRemote(t), registryRemote(t)
	registryFile(t, second, "new-origin.txt", "new")
	registryGit(t, second, "add", ".")
	registryGit(t, second, "commit", "-m", "second")
	rc := NewRegistryCache(t.TempDir())
	reg := RegistryConfig{Name: "test", URL: first, Branch: "main"}
	if err := rc.EnsureRegistry(reg); err != nil {
		t.Fatal(err)
	}
	reg.URL = second
	if err := rc.EnsureRegistry(reg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(rc.GetRegistryCachePath("test"), "new-origin.txt")); err != nil {
		t.Fatal(err)
	}
}
func TestRegistryRefreshRejectsInvalidIndex(t *testing.T) {
	remote := registryRemote(t)
	rc := NewRegistryCache(t.TempDir())
	reg := RegistryConfig{Name: "test", URL: remote, Branch: "main"}
	if err := rc.EnsureRegistry(reg); err != nil {
		t.Fatal(err)
	}
	registryFile(t, remote, "package-index.yaml", "packages: [broken")
	registryGit(t, remote, "add", ".")
	registryGit(t, remote, "commit", "-m", "invalid")
	if err := rc.EnsureRegistry(reg); err == nil {
		t.Fatal("invalid index accepted")
	}
	index, err := rc.LoadPackageIndexFromCache("test")
	if err != nil || index.Version != "1.0" {
		t.Fatalf("old cache lost: %v %v", index, err)
	}
}
func TestRegistrySnapshotValidation(t *testing.T) {
	for _, body := range []string{"", "null", "   ", "packages: [broken", "packages: [{name: ../escape, version: 1.0.0}]", "packages: [{name: valid, version: 1.0.0, path: ../escape}]", "packages: [{name: duplicate, version: 1}, {name: duplicate, version: 2}]"} {
		t.Run(body, func(t *testing.T) {
			dir := t.TempDir()
			registryFile(t, dir, "package-index.yaml", body)
			if err := validateRegistrySnapshot(dir); err == nil {
				t.Fatal("accepted invalid index")
			}
		})
	}
}

func TestRegistryRefreshLocalFailures(t *testing.T) {
	for _, kind := range []string{"invalid-name", "busy", "file-cache", "parent-file", "missing-origin", "missing-index"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			rc := NewRegistryCache(root)
			remote := registryRemote(t)
			reg := RegistryConfig{Name: "test", URL: remote, Branch: "main"}
			if err := os.MkdirAll(rc.cacheDir, 0700); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "invalid-name":
				reg.Name = "../outside"
			case "busy":
				registryFile(t, rc.cacheDir, "test.refresh.lock", "held")
			case "file-cache":
				registryFile(t, rc.cacheDir, "test", "do not replace")
			case "parent-file":
				rc.cacheDir = filepath.Join(root, "blocker")
				registryFile(t, root, "blocker", "file")
			case "missing-origin":
				reg.URL = filepath.Join(root, "missing")
			case "missing-index":
				registryGit(t, remote, "mv", "package-index.yaml", "renamed.yaml")
				registryGit(t, remote, "commit", "-m", "missing index")
			}
			if err := rc.EnsureRegistry(reg); err == nil {
				t.Fatal("invalid refresh accepted")
			}
		})
	}
}

func TestRegistryRefreshActivationFailures(t *testing.T) {
	for _, failure := range []string{"preserve", "activate", "restore"} {
		t.Run(failure, func(t *testing.T) {
			remote := registryRemote(t)
			rc := NewRegistryCache(t.TempDir())
			reg := RegistryConfig{Name: "test", URL: remote, Branch: "main"}
			if err := rc.EnsureRegistry(reg); err != nil {
				t.Fatal(err)
			}
			cache := rc.GetRegistryCachePath("test")
			registryFile(t, cache, "user.txt", "preserve me")
			calls := 0
			rc.rename = func(from, to string) error {
				calls++
				if (failure == "preserve" && calls == 1) || (failure != "preserve" && calls == 2) || (failure == "restore" && calls == 3) {
					return errors.New("injected filesystem rename failure")
				}
				return os.Rename(from, to)
			}
			err := rc.EnsureRegistry(reg)
			if err == nil {
				t.Fatal("rename failure swallowed")
			}
			if failure == "restore" {
				if !strings.Contains(err.Error(), "recovery:") {
					t.Fatalf("no recovery path: %v", err)
				}
				if _, err := os.Stat(cache + ".refresh.lock"); err != nil {
					t.Fatal("lost recovery lock")
				}
				matches, _ := filepath.Glob(filepath.Join(rc.cacheDir, ".refresh-*", "previous", "user.txt"))
				if len(matches) != 1 {
					t.Fatal("lost recovery copy")
				}
			} else {
				data, err := os.ReadFile(filepath.Join(cache, "user.txt"))
				if err != nil || string(data) != "preserve me" {
					t.Fatalf("old cache lost: %q %v", data, err)
				}
			}
		})
	}
}
func TestRegistrySnapshotRejectsSymbolicIndex(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(target, []byte("packages: []"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, "package-index.yaml")); err != nil {
		t.Skip(err)
	}
	if err := validateRegistrySnapshot(dir); err == nil {
		t.Fatal("symbolic index accepted")
	}
}
