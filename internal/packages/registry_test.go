package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func createTestRegistry(t *testing.T, dir string) string {
	registryDir := filepath.Join(dir, "cache", "registries", "test-registry")
	if err := os.MkdirAll(registryDir, 0o755); err != nil {
		t.Fatalf("Failed to create registry directory: %v", err)
	}

	// Create package index
	index := &PackageIndex{
		Version:     "1.0",
		Name:        "Test Registry",
		Description: "Test package registry",
		URL:         "https://github.com/test/packages",
		Maintainer:  "Test",
		Email:       "test@example.com",
		Metadata: RegistryMetadata{
			TotalPackages: 2,
			Categories:    []string{"utilities", "documentation"},
		},
		Packages: []PackageIndexEntry{
			{
				Name:        "test-package-1",
				Version:     "1.0.0",
				Description: "Test package 1",
				Author:      "Test",
				License:     "MIT",
				Category:    "utilities",
				Tags:        []string{"test", "utility"},
				Path:        "test-package-1",
				Status:      "stable",
			},
			{
				Name:        "test-package-2",
				Version:     "1.0.0",
				Description: "Test package 2",
				Author:      "Test",
				License:     "MIT",
				Category:    "documentation",
				Tags:        []string{"test", "docs"},
				Path:        "test-package-2",
				Status:      "stable",
			},
		},
	}

	// Save index
	rm := NewRegistryManager(filepath.Join(dir))
	if err := rm.SavePackageIndex("test-registry", index); err != nil {
		t.Fatalf("Failed to save package index: %v", err)
	}

	return registryDir
}

func TestNewRegistryManager(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	expectedCacheDir := filepath.Join(tmpDir, "cache", "registries")
	if rm.GetCacheDir() != expectedCacheDir {
		t.Errorf("Expected cache dir %s, got %s", expectedCacheDir, rm.GetCacheDir())
	}
}

func TestSaveAndLoadPackageIndex(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	// Create index
	index := &PackageIndex{
		Version:     "1.0",
		Name:        "Test Registry",
		Description: "Test registry",
		URL:         "https://github.com/test/packages",
		Maintainer:  "Test",
		Email:       "test@example.com",
		Metadata: RegistryMetadata{
			TotalPackages: 1,
			Categories:    []string{"utilities"},
		},
		Packages: []PackageIndexEntry{
			{
				Name:        "test-package",
				Version:     "1.0.0",
				Description: "Test package",
				Author:      "Test",
				License:     "MIT",
				Category:    "utilities",
				Status:      "stable",
			},
		},
	}

	// Save index
	if err := rm.SavePackageIndex("test-registry", index); err != nil {
		t.Fatalf("SavePackageIndex failed: %v", err)
	}

	// Load index
	loaded, err := rm.LoadPackageIndex("test-registry")
	if err != nil {
		t.Fatalf("LoadPackageIndex failed: %v", err)
	}

	if loaded.Name != "Test Registry" {
		t.Errorf("Expected name 'Test Registry', got '%s'", loaded.Name)
	}

	if len(loaded.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(loaded.Packages))
	}
}

func TestSearchPackages(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
	}

	// Search for "test"
	results, err := rm.SearchPackages("test", registries)
	if err != nil {
		t.Fatalf("SearchPackages failed: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected to find packages")
	}
}

func TestGetPackage(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
	}

	// Get package
	pkg, registryName, err := rm.GetPackage("test-package-1", registries)
	if err != nil {
		t.Fatalf("GetPackage failed: %v", err)
	}

	if pkg.Name != "test-package-1" {
		t.Errorf("Expected package 'test-package-1', got '%s'", pkg.Name)
	}

	if registryName != "test-registry" {
		t.Errorf("Expected registry 'test-registry', got '%s'", registryName)
	}
}

func TestGetPackageNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
	}

	// Get nonexistent package
	_, _, err := rm.GetPackage("nonexistent-package", registries)
	if err == nil {
		t.Error("Expected error for nonexistent package")
	}
}

func TestGetPackageFromRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)

	// Get package from specific registry
	pkg, err := rm.GetPackageFromRegistry("test-package-1", "test-registry")
	if err != nil {
		t.Fatalf("GetPackageFromRegistry failed: %v", err)
	}

	if pkg.Name != "test-package-1" {
		t.Errorf("Expected package 'test-package-1', got '%s'", pkg.Name)
	}
}

func TestListPackages(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
	}

	// List packages
	packages, err := rm.ListPackages(registries)
	if err != nil {
		t.Fatalf("ListPackages failed: %v", err)
	}

	if len(packages) != 2 {
		t.Errorf("Expected 2 packages, got %d", len(packages))
	}
}

func TestSearchByCategory(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
	}

	// Search by category
	results, err := rm.SearchByCategory("utilities", registries)
	if err != nil {
		t.Fatalf("SearchByCategory failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("Expected 1 package in utilities category, got %d", len(results))
	}

	if results[0].Name != "test-package-1" {
		t.Errorf("Expected 'test-package-1', got '%s'", results[0].Name)
	}
}

func TestSearchByTag(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  true,
			Priority: 1,
		},
	}

	// Search by tag
	results, err := rm.SearchByTag("test", registries)
	if err != nil {
		t.Fatalf("SearchByTag failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 packages with 'test' tag, got %d", len(results))
	}
}

func TestCreatePackageIndex(t *testing.T) {
	index := CreatePackageIndex(
		"Test Registry",
		"Test registry",
		"https://github.com/test/packages",
		"Test",
		"test@example.com",
	)

	if index.Name != "Test Registry" {
		t.Errorf("Expected name 'Test Registry', got '%s'", index.Name)
	}

	if len(index.Packages) != 0 {
		t.Errorf("Expected 0 packages, got %d", len(index.Packages))
	}
}

func TestAddPackageToIndex(t *testing.T) {
	index := CreatePackageIndex(
		"Test Registry",
		"Test registry",
		"https://github.com/test/packages",
		"Test",
		"test@example.com",
	)

	// Add package
	entry := PackageIndexEntry{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Author:      "Test",
		License:     "MIT",
		Category:    "utilities",
		Status:      "stable",
	}

	index.AddPackage(entry)

	if len(index.Packages) != 1 {
		t.Errorf("Expected 1 package, got %d", len(index.Packages))
	}

	if index.Metadata.TotalPackages != 1 {
		t.Errorf("Expected total_packages=1, got %d", index.Metadata.TotalPackages)
	}
}

func TestRemovePackageFromIndex(t *testing.T) {
	index := CreatePackageIndex(
		"Test Registry",
		"Test registry",
		"https://github.com/test/packages",
		"Test",
		"test@example.com",
	)

	// Add package
	entry := PackageIndexEntry{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Author:      "Test",
		License:     "MIT",
		Category:    "utilities",
		Status:      "stable",
	}

	index.AddPackage(entry)

	// Remove package
	removed := index.RemovePackage("test-package")

	if !removed {
		t.Error("Expected package to be removed")
	}

	if len(index.Packages) != 0 {
		t.Errorf("Expected 0 packages, got %d", len(index.Packages))
	}

	// Removing a missing package should return false.
	if index.RemovePackage("missing") {
		t.Error("Expected RemovePackage to return false for missing package")
	}
}

func TestGetPackageByName(t *testing.T) {
	index := CreatePackageIndex(
		"Test Registry",
		"Test registry",
		"https://github.com/test/packages",
		"Test",
		"test@example.com",
	)

	// Add package
	entry := PackageIndexEntry{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Author:      "Test",
		License:     "MIT",
		Category:    "utilities",
		Status:      "stable",
	}

	index.AddPackage(entry)

	// Get package
	pkg := index.GetPackageByName("test-package")

	if pkg == nil {
		t.Fatal("Expected to find package")
	}

	if pkg.Name != "test-package" {
		t.Errorf("Expected 'test-package', got '%s'", pkg.Name)
	}
}

func TestClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)

	// Verify cache exists
	cacheDir := rm.GetCacheDir()
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Error("Cache directory should exist")
	}

	// Clear cache
	if err := rm.ClearCache(); err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	// Verify cache is cleared
	if _, err := os.Stat(cacheDir); err == nil {
		t.Error("Cache directory should be removed")
	}
}

func TestDisabledRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  false, // Disabled
			Priority: 1,
		},
	}

	// Search should return empty results
	results, err := rm.SearchPackages("test", registries)
	if err != nil {
		t.Fatalf("SearchPackages failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results for disabled registry, got %d", len(results))
	}
}

// TestListPackages_DisabledRegistry ensures disabled registries are skipped by ListPackages.
func TestListPackages_DisabledRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:     "test-registry",
			URL:      "https://github.com/test/packages",
			Branch:   "main",
			Enabled:  false, // Disabled — must be skipped
			Priority: 1,
		},
	}

	packages, err := rm.ListPackages(registries)
	if err != nil {
		t.Fatalf("ListPackages failed: %v", err)
	}

	if len(packages) != 0 {
		t.Errorf("Expected 0 packages for disabled registry, got %d", len(packages))
	}
}

// TestListPackages_EmptyCache ensures ListPackages does not error when EnsureRegistry
// fails (e.g. no network) and the cache is empty — it simply returns an empty list.
func TestListPackages_EmptyCache(t *testing.T) {
	tmpDir := t.TempDir() // fresh dir — no cache, no git repo

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{
			Name:    "nonexistent-registry",
			URL:     "https://github.com/does-not-exist/repo",
			Branch:  "main",
			Enabled: true,
		},
	}

	// EnsureRegistry will fail (no network / no repo), LoadPackageIndexFromCache
	// will also fail — ListPackages must return empty list without error.
	packages, err := rm.ListPackages(registries)
	if err != nil {
		t.Errorf("ListPackages should not error on empty cache, got: %v", err)
	}

	if len(packages) != 0 {
		t.Errorf("Expected 0 packages for empty cache, got %d", len(packages))
	}
}

// ── GetRegistryCacheDir ───────────────────────────────────────────────────────

func TestGetRegistryCacheDir(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	got := rm.GetRegistryCacheDir("my-registry")
	want := filepath.Join(tmpDir, "cache", "registries", "my-registry")

	if got != want {
		t.Errorf("GetRegistryCacheDir: got %q, want %q", got, want)
	}
}

func TestGetRegistryCacheDir_DifferentRegistries(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	registries := []string{"official", "community", "private"}
	for _, reg := range registries {
		got := rm.GetRegistryCacheDir(reg)
		want := filepath.Join(tmpDir, "cache", "registries", reg)
		if got != want {
			t.Errorf("GetRegistryCacheDir(%q): got %q, want %q", reg, got, want)
		}
	}
}

// ── ClearRegistryCache ────────────────────────────────────────────────────────

func TestClearRegistryCache_ExistingRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)

	// Verify registry cache dir exists.
	regCacheDir := rm.GetRegistryCacheDir("test-registry")
	if _, err := os.Stat(regCacheDir); os.IsNotExist(err) {
		t.Fatal("registry cache dir should exist before clearing")
	}

	// Clear only the test-registry cache.
	if err := rm.ClearRegistryCache("test-registry"); err != nil {
		t.Fatalf("ClearRegistryCache failed: %v", err)
	}

	// The registry's cache dir should be gone.
	if _, err := os.Stat(regCacheDir); err == nil {
		t.Error("expected registry cache dir to be removed")
	}

	// Parent cache dir should still exist.
	if _, err := os.Stat(rm.GetCacheDir()); os.IsNotExist(err) {
		t.Error("parent cache dir should still exist")
	}
}

func TestClearRegistryCache_NonExistentRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	// Clearing a non-existent registry cache must not return an error
	// (os.RemoveAll is a no-op on missing paths).
	if err := rm.ClearRegistryCache("nonexistent-registry"); err != nil {
		t.Errorf("ClearRegistryCache on non-existent registry should not error: %v", err)
	}
}

func TestClearRegistryCache_IsolatesOtherRegistries(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	// Also create a second registry cache.
	secondRegPath := filepath.Join(tmpDir, "cache", "registries", "second-registry")
	if err := os.MkdirAll(secondRegPath, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(secondRegPath, "package-index.yaml"), []byte("version: 1.0\n"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	rm := NewRegistryManager(tmpDir)

	// Clear only test-registry.
	if err := rm.ClearRegistryCache("test-registry"); err != nil {
		t.Fatalf("ClearRegistryCache failed: %v", err)
	}

	// second-registry must still exist.
	if _, err := os.Stat(secondRegPath); os.IsNotExist(err) {
		t.Error("second-registry should not have been removed")
	}
}

func TestListPackages_SyncsCache(t *testing.T) {
	tmpDir := t.TempDir()

	// Manually write the cache where EnsureRegistry / LoadPackageIndexFromCache expects it.
	// This mirrors what the real EnsureRegistry does after cloning.
	cacheDir := filepath.Join(tmpDir, "cache", "registries", "my-registry")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("Failed to create cache dir: %v", err)
	}

	index := &PackageIndex{
		Version:     "1.0",
		Name:        "My Registry",
		Description: "Registry populated by EnsureRegistry",
		URL:         "https://github.com/test/packages",
		Maintainer:  "Test",
		Email:       "test@example.com",
		Metadata:    RegistryMetadata{TotalPackages: 1, Categories: []string{"tools"}},
		Packages: []PackageIndexEntry{
			{Name: "synced-pkg", Version: "2.0.0", Description: "Synced package",
				Author: "Test", License: "MIT", Category: "tools", Status: "stable"},
		},
	}

	rm := NewRegistryManager(tmpDir)
	if err := rm.SavePackageIndex("my-registry", index); err != nil {
		t.Fatalf("SavePackageIndex failed: %v", err)
	}

	registries := []RegistryConfig{
		{Name: "my-registry", URL: "https://github.com/test/packages",
			Branch: "main", Enabled: true, Priority: 1},
	}

	packages, err := rm.ListPackages(registries)
	if err != nil {
		t.Fatalf("ListPackages failed: %v", err)
	}

	if len(packages) != 1 {
		t.Errorf("Expected 1 package from cache, got %d", len(packages))
	}

	if packages[0].Name != "synced-pkg" {
		t.Errorf("Expected 'synced-pkg', got '%s'", packages[0].Name)
	}
}

func TestMatchesQuery(t *testing.T) {
	rm := NewRegistryManager(t.TempDir())
	pkg := PackageIndexEntry{
		Name:        "my-awesome-pkg",
		Description: "Does awesome things",
		Tags:        []string{"cli", "utility"},
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"my-awesome-pkg", true},
		{"awesome", true},
		{"cli", true},
		{"utility", true},
		{"missing", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("query=%q", tt.query), func(t *testing.T) {
			got := rm.matchesQuery(pkg, tt.query)
			if got != tt.want {
				t.Errorf("matchesQuery(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestLoadPackageIndex_Corrupted(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	cacheDir := filepath.Join(tmpDir, "cache", "registries", "broken")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "package-index.yaml"), []byte(": not yaml"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := rm.LoadPackageIndex("broken")
	if err == nil {
		t.Error("Expected error for corrupted package index")
	}
}

func TestSavePackageIndex_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	// Make cache dir path invalid by creating a file where a directory is needed.
	blocker := filepath.Join(tmpDir, "cache")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := rm.SavePackageIndex("registry", &PackageIndex{}); err == nil {
		t.Error("Expected error when cache directory cannot be created")
	}
}

func TestGetPackageFromRegistry_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	_, err := rm.GetPackageFromRegistry("does-not-exist", "test-registry")
	if err == nil {
		t.Error("Expected error when package is not in registry")
	}
}

func TestGetPackageFromRegistry_LoadIndexError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	_, err := rm.GetPackageFromRegistry("any", "missing-registry")
	if err == nil {
		t.Error("Expected error when registry index cannot be loaded")
	}
}

func TestGetPackage_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	createTestRegistry(t, tmpDir)

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{{
		Name:     "test-registry",
		URL:      "https://github.com/test/packages",
		Branch:   "main",
		Enabled:  true,
		Priority: 1,
	}}

	_, _, err := rm.GetPackage("missing", registries)
	if err == nil {
		t.Error("Expected error for missing package")
	}
}

func TestGetPackageByName_NotFound(t *testing.T) {
	index := CreatePackageIndex("Test Registry", "Test registry", "https://github.com/test/packages", "Test", "test@example.com")
	index.AddPackage(PackageIndexEntry{Name: "existing", Version: "1.0.0"})

	pkg := index.GetPackageByName("missing")
	if pkg != nil {
		t.Errorf("Expected nil for missing package, got %v", pkg)
	}
}

func TestLoadPackageIndex_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	_, err := rm.LoadPackageIndex("does-not-exist")
	if err == nil {
		t.Error("Expected error when index file is missing")
	}
}

func TestSavePackageIndex_WriteError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)

	parentDir := filepath.Join(tmpDir, "cache", "registries")
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	roDir := filepath.Join(parentDir, "readonly")
	if err := os.Mkdir(roDir, 0o555); err != nil {
		t.Fatalf("setup: %v", err)
	}
	defer os.Chmod(roDir, 0o755) //nolint:errcheck

	if err := rm.SavePackageIndex("readonly", &PackageIndex{}); err == nil {
		t.Error("Expected error when index file cannot be written")
	}
}

func TestSearchPackages_LoadIndexError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{{
		Name:     "empty-registry",
		URL:      "https://github.com/test/packages",
		Branch:   "main",
		Enabled:  true,
		Priority: 1,
	}}

	results, err := rm.SearchPackages("test", registries)
	if err != nil {
		t.Fatalf("SearchPackages should not error on load failure: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results when index cannot be loaded, got %d", len(results))
	}
}

func TestGetPackage_LoadIndexError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{{
		Name:     "empty-registry",
		URL:      "https://github.com/test/packages",
		Branch:   "main",
		Enabled:  true,
		Priority: 1,
	}}

	_, _, err := rm.GetPackage("any", registries)
	if err == nil {
		t.Error("Expected error when no registry index can be loaded")
	}
}

func TestGetPackage_FallbackRegistry(t *testing.T) {
	tmpDir := t.TempDir()

	// First registry has no cache.
	// Second registry has a valid index.
	cacheDir := filepath.Join(tmpDir, "cache", "registries", "second")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	index := &PackageIndex{
		Version:     "1.0",
		Name:        "Second Registry",
		Description: "Second",
		URL:         "https://github.com/test/packages",
		Maintainer:  "Test",
		Email:       "test@example.com",
		Metadata:    RegistryMetadata{TotalPackages: 1, Categories: []string{"tools"}},
		Packages: []PackageIndexEntry{
			{Name: "fallback-pkg", Version: "1.0.0", Description: "Fallback", Author: "Test", License: "MIT", Category: "tools", Status: "stable"},
		},
	}
	data, _ := yaml.Marshal(index)
	if err := os.WriteFile(filepath.Join(cacheDir, "package-index.yaml"), data, 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{
		{Name: "first", URL: "https://github.com/test/packages", Branch: "main", Enabled: true, Priority: 1},
		{Name: "second", URL: "https://github.com/test/packages", Branch: "main", Enabled: true, Priority: 2},
	}

	pkg, registryName, err := rm.GetPackage("fallback-pkg", registries)
	if err != nil {
		t.Fatalf("GetPackage failed: %v", err)
	}
	if pkg.Name != "fallback-pkg" {
		t.Errorf("Expected 'fallback-pkg', got '%s'", pkg.Name)
	}
	if registryName != "second" {
		t.Errorf("Expected registry 'second', got '%s'", registryName)
	}
}

func TestSearchByCategory_LoadIndexError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{{
		Name:     "empty-registry",
		URL:      "https://github.com/test/packages",
		Branch:   "main",
		Enabled:  true,
		Priority: 1,
	}}

	results, err := rm.SearchByCategory("utilities", registries)
	if err != nil {
		t.Fatalf("SearchByCategory should not error on load failure: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results when index cannot be loaded, got %d", len(results))
	}
}

func TestSearchByTag_LoadIndexError(t *testing.T) {
	tmpDir := t.TempDir()
	rm := NewRegistryManager(tmpDir)
	registries := []RegistryConfig{{
		Name:     "empty-registry",
		URL:      "https://github.com/test/packages",
		Branch:   "main",
		Enabled:  true,
		Priority: 1,
	}}

	results, err := rm.SearchByTag("test", registries)
	if err != nil {
		t.Fatalf("SearchByTag should not error on load failure: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results when index cannot be loaded, got %d", len(results))
	}
}

func TestMatchesQuery_DescriptionOnly(t *testing.T) {
	rm := NewRegistryManager(t.TempDir())
	pkg := PackageIndexEntry{
		Name:        "short",
		Description: "unique-description",
		Tags:        nil,
	}

	if !rm.matchesQuery(pkg, "unique") {
		t.Error("Expected query to match description")
	}
	if rm.matchesQuery(pkg, "shortname") {
		t.Error("Expected query not to match name substring not present")
	}
}
