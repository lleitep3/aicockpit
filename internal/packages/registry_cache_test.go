package packages

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// ── GetPackagePath ────────────────────────────────────────────────────────────

func TestGetPackagePath(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	got := rc.GetPackagePath("my-registry", "my-package")
	want := filepath.Join(tmpDir, "cache", "registries", "my-registry", "my-package")

	if got != want {
		t.Errorf("GetPackagePath: got %q, want %q", got, want)
	}
}

func TestGetPackagePath_DifferentInputs(t *testing.T) {
	tests := []struct {
		registry string
		pkg      string
	}{
		{"reg-a", "pkg-1"},
		{"reg-b", "pkg-2"},
		{"official", "cockpit-tools"},
	}

	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	for _, tc := range tests {
		got := rc.GetPackagePath(tc.registry, tc.pkg)
		want := filepath.Join(tmpDir, "cache", "registries", tc.registry, tc.pkg)
		if got != want {
			t.Errorf("GetPackagePath(%q, %q): got %q, want %q", tc.registry, tc.pkg, got, want)
		}
	}
}

// ── updateRegistry ────────────────────────────────────────────────────────────

// updateRegistry calls `git fetch` and `git pull` in the given path.
// We test the error path by supplying a non-git directory so git fails immediately
// (without any network I/O — git exits quickly when the directory is not a repo).
func TestUpdateRegistry_ErrorPath(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Non-git directory with an obviously-invalid local "URL" — git fetch
	// should fail instantly with a local error (no network required).
	registry := RegistryConfig{
		Name:   "fake-registry",
		URL:    "file:///nonexistent-local-repo",
		Branch: "main",
	}

	err := rc.updateRegistry(tmpDir, registry)
	if err == nil {
		t.Error("expected error when running git fetch in a non-git directory")
	}
}

func TestUpdateRegistry_NonExistentPath(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	registry := RegistryConfig{
		Name:   "fake-registry",
		URL:    "file:///nonexistent-local-repo",
		Branch: "main",
	}

	// Completely non-existent path — git should fail immediately (no network).
	err := rc.updateRegistry("/nonexistent/path/to/registry", registry)
	if err == nil {
		t.Error("expected error for nonexistent git path")
	}
}

// ── ListPackagesInCache ───────────────────────────────────────────────────────

func TestListPackagesInCache_NonExistentRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Registry directory doesn't exist — should return an error.
	_, err := rc.ListPackagesInCache("nonexistent-registry")
	if err == nil {
		t.Error("expected error when listing packages in a non-existent registry cache")
	}
}

func TestListPackagesInCache_EmptyRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Create an empty registry directory (no package sub-dirs).
	registryPath := filepath.Join(tmpDir, "cache", "registries", "empty-registry")
	if err := os.MkdirAll(registryPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	pkgs, err := rc.ListPackagesInCache("empty-registry")
	if err != nil {
		t.Fatalf("ListPackagesInCache failed: %v", err)
	}

	if len(pkgs) != 0 {
		t.Errorf("expected 0 packages, got %d", len(pkgs))
	}
}

func TestListPackagesInCache_WithPackages(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	registryPath := filepath.Join(tmpDir, "cache", "registries", "test-registry")

	// pkg-a — valid package (has cockpit-package.yml)
	pkgADir := filepath.Join(registryPath, "pkg-a")
	if err := os.MkdirAll(pkgADir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgADir, "cockpit-package.yml"), []byte("name: pkg-a\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// pkg-b — valid package
	pkgBDir := filepath.Join(registryPath, "pkg-b")
	if err := os.MkdirAll(pkgBDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pkgBDir, "cockpit-package.yml"), []byte("name: pkg-b\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// no-manifest — directory without manifest, should be skipped
	noManifestDir := filepath.Join(registryPath, "no-manifest")
	if err := os.MkdirAll(noManifestDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// regular-file.txt — plain file, should be skipped
	if err := os.WriteFile(filepath.Join(registryPath, "regular-file.txt"), []byte("txt"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// .git — dot-prefixed directory, should be skipped
	gitDir := filepath.Join(registryPath, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	pkgs, err := rc.ListPackagesInCache("test-registry")
	if err != nil {
		t.Fatalf("ListPackagesInCache failed: %v", err)
	}

	if len(pkgs) != 2 {
		t.Errorf("expected 2 packages, got %d: %v", len(pkgs), pkgs)
	}

	found := map[string]bool{}
	for _, p := range pkgs {
		found[p] = true
	}
	if !found["pkg-a"] {
		t.Error("expected pkg-a in results")
	}
	if !found["pkg-b"] {
		t.Error("expected pkg-b in results")
	}
}

func TestListPackagesInCache_SkipsDotDirs(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	registryPath := filepath.Join(tmpDir, "cache", "registries", "dot-registry")
	if err := os.MkdirAll(registryPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	// Dot-prefixed dir with manifest — must be skipped.
	hiddenDir := filepath.Join(registryPath, ".hidden-pkg")
	if err := os.MkdirAll(hiddenDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hiddenDir, "cockpit-package.yml"), []byte("name: hidden\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	pkgs, err := rc.ListPackagesInCache("dot-registry")
	if err != nil {
		t.Fatalf("ListPackagesInCache failed: %v", err)
	}

	if len(pkgs) != 0 {
		t.Errorf("expected dot-prefixed dirs to be skipped, got: %v", pkgs)
	}
}

// ── GetPackageFromCache ───────────────────────────────────────────────────────

func TestGetPackageFromCache_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	_, err := rc.GetPackageFromCache("my-registry", "nonexistent-package")
	if err == nil {
		t.Error("expected error when package does not exist in cache")
	}
}

func TestGetPackageFromCache_Found(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Create the package directory in cache.
	pkgPath := filepath.Join(tmpDir, "cache", "registries", "my-registry", "my-package")
	if err := os.MkdirAll(pkgPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	got, err := rc.GetPackageFromCache("my-registry", "my-package")
	if err != nil {
		t.Fatalf("GetPackageFromCache failed: %v", err)
	}

	if got != pkgPath {
		t.Errorf("got path %q, want %q", got, pkgPath)
	}
}

func TestGetPackageFromCache_RegistryExistsButPackageMissing(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Create registry dir but not the package sub-dir.
	registryPath := filepath.Join(tmpDir, "cache", "registries", "my-registry")
	if err := os.MkdirAll(registryPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := rc.GetPackageFromCache("my-registry", "missing-package")
	if err == nil {
		t.Error("expected error when package sub-dir does not exist")
	}
}

func TestLoadPackageIndexFromCache_Corrupted(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	registryPath := filepath.Join(tmpDir, "cache", "registries", "broken")
	if err := os.MkdirAll(registryPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(registryPath, "package-index.yaml"), []byte(": invalid"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := rc.LoadPackageIndexFromCache("broken")
	if err == nil {
		t.Error("expected error for corrupted package index")
	}
}

func TestEnsureRegistry_ClonesLocalRepo(t *testing.T) {
	// Create a local git repository to act as the registry remote.
	remoteDir := t.TempDir()
	if err := exec.Command("git", "init", "-b", "main", remoteDir).Run(); err != nil {
		t.Skip("git not available:", err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "package-index.yaml"), []byte("version: 1.0\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := exec.Command("git", "-C", remoteDir, "add", ".").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := exec.Command("git", "-C", remoteDir, "-c", "commit.gpgsign=false", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)
	registry := RegistryConfig{
		Name:   "local-registry",
		URL:    remoteDir,
		Branch: "main",
	}

	if err := rc.EnsureRegistry(registry); err != nil {
		t.Fatalf("EnsureRegistry failed: %v", err)
	}

	cachePath := rc.GetRegistryCachePath(registry.Name)
	if _, err := os.Stat(filepath.Join(cachePath, ".git")); err != nil {
		t.Errorf("expected cloned .git directory: %v", err)
	}
}

func TestEnsureRegistry_UpdatesExistingClone(t *testing.T) {
	remoteDir := t.TempDir()
	if err := exec.Command("git", "init", "-b", "main", remoteDir).Run(); err != nil {
		t.Skip("git not available:", err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "package-index.yaml"), []byte("version: 1.0\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := exec.Command("git", "-C", remoteDir, "add", ".").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := exec.Command("git", "-C", remoteDir, "-c", "commit.gpgsign=false", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)
	registry := RegistryConfig{
		Name:   "local-registry",
		URL:    remoteDir,
		Branch: "main",
	}

	if err := rc.EnsureRegistry(registry); err != nil {
		t.Fatalf("first EnsureRegistry failed: %v", err)
	}
	if err := rc.EnsureRegistry(registry); err != nil {
		t.Fatalf("second EnsureRegistry (update) failed: %v", err)
	}
}

func TestCloneRegistry_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Create a file where the cache directory parent should be.
	blocker := filepath.Join(tmpDir, "cache")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	registry := RegistryConfig{Name: "blocked", URL: "file:///nonexistent", Branch: "main"}
	err := rc.cloneRegistry(registry, filepath.Join(tmpDir, "cache", "registries", "blocked"))
	if err == nil {
		t.Error("Expected error when cache directory cannot be created")
	}
}

func TestUpdateRegistry_PullError(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Valid git repo but no remote, so pull fails.
	repoDir := filepath.Join(tmpDir, "repo")
	if err := exec.Command("git", "init", "-b", "main", repoDir).Run(); err != nil {
		t.Skip("git not available:", err)
	}
	if err := exec.Command("git", "-C", repoDir, "-c", "commit.gpgsign=false", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "--allow-empty", "-m", "init").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	registry := RegistryConfig{Name: "no-remote", URL: "file:///nonexistent", Branch: "main"}
	err := rc.updateRegistry(repoDir, registry)
	if err == nil {
		t.Error("Expected error when git pull has no remote")
	}
}

func TestUpdateRegistry_FetchError(t *testing.T) {
	tmpDir := t.TempDir()
	rc := NewRegistryCache(tmpDir)

	// Repository with invalid git metadata so fetch fails immediately.
	repoDir := filepath.Join(tmpDir, "bad-git")
	if err := os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	registry := RegistryConfig{Name: "bad-git", URL: "file:///nonexistent", Branch: "main"}
	err := rc.updateRegistry(repoDir, registry)
	if err == nil {
		t.Error("Expected error when git fetch fails")
	}
}

func TestUpdateRegistry_Success(t *testing.T) {
	remoteDir := t.TempDir()
	if err := exec.Command("git", "init", "-b", "main", remoteDir).Run(); err != nil {
		t.Skip("git not available:", err)
	}
	if err := os.WriteFile(filepath.Join(remoteDir, "package-index.yaml"), []byte("version: 1.0\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := exec.Command("git", "-C", remoteDir, "add", ".").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := exec.Command("git", "-C", remoteDir, "-c", "commit.gpgsign=false", "-c", "user.email=test@test.com", "-c", "user.name=Test", "commit", "-m", "init").Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	tmpDir := t.TempDir()
	cloneDir := filepath.Join(tmpDir, "clone")
	if err := exec.Command("git", "clone", remoteDir, cloneDir).Run(); err != nil {
		t.Fatalf("setup: %v", err)
	}

	rc := NewRegistryCache(tmpDir)
	registry := RegistryConfig{Name: "local-registry", URL: remoteDir, Branch: "main"}
	if err := rc.updateRegistry(cloneDir, registry); err != nil {
		t.Fatalf("updateRegistry failed: %v", err)
	}
}
