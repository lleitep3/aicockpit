package packages

import (
	"os"
	"path/filepath"
	"testing"
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

// TestListPackages_SyncsCache verifies that ListPackages uses the synced cache
// (LoadPackageIndexFromCache) instead of the stale-only LoadPackageIndex path.
// It populates the cache in the exact location EnsureRegistry would write to
// and confirms ListPackages picks it up.
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
