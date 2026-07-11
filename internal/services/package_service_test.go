package services

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/packages"
)

// MockPackageService is a test double that implements PackageService.
// It is exported so other packages can use it when testing code that depends
// on the service interface.
type MockPackageService struct {
	SearchPackagesFunc        func(string, []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	SearchByCategoryFunc      func(string, []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	SearchByTagFunc           func(string, []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	ListPackagesFunc          func([]packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	GetPackageFunc            func(string, []packages.RegistryConfig) (*packages.PackageIndexEntry, string, error)
	GetPackageFromCacheFunc   func(string, string) (string, error)
	InstallPackageFunc        func(string, map[string]interface{}) error
	UninstallPackageFunc      func(string) error
	UpgradePackageFunc        func(string, string) error
	PackageExistsFunc         func(string) bool
	GetInstalledPackageFunc   func(string) (*packages.Package, error)
	ListInstalledPackagesFunc func() ([]*packages.Package, error)
	GetPackageInstallPathFunc func(string) string
	RunPackageHooksFunc       func(string, []packages.Hook) error
	SyncPackageAssetsFunc     func(*packages.Package, string) error
	RemovePackageAssetsFunc   func(*packages.Package) error
	TriggerDeployFunc         func(string) error
}

func (m *MockPackageService) SearchPackages(query string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	if m.SearchPackagesFunc != nil {
		return m.SearchPackagesFunc(query, registries)
	}
	return nil, nil
}

func (m *MockPackageService) SearchByCategory(category string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	if m.SearchByCategoryFunc != nil {
		return m.SearchByCategoryFunc(category, registries)
	}
	return nil, nil
}

func (m *MockPackageService) SearchByTag(tag string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	if m.SearchByTagFunc != nil {
		return m.SearchByTagFunc(tag, registries)
	}
	return nil, nil
}

func (m *MockPackageService) ListPackages(registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	if m.ListPackagesFunc != nil {
		return m.ListPackagesFunc(registries)
	}
	return nil, nil
}

func (m *MockPackageService) GetPackage(name string, registries []packages.RegistryConfig) (*packages.PackageIndexEntry, string, error) {
	if m.GetPackageFunc != nil {
		return m.GetPackageFunc(name, registries)
	}
	return nil, "", nil
}

func (m *MockPackageService) GetPackageFromCache(registryName, packageName string) (string, error) {
	if m.GetPackageFromCacheFunc != nil {
		return m.GetPackageFromCacheFunc(registryName, packageName)
	}
	return "", nil
}

func (m *MockPackageService) InstallPackage(sourcePath string, cfg map[string]interface{}) error {
	if m.InstallPackageFunc != nil {
		return m.InstallPackageFunc(sourcePath, cfg)
	}
	return nil
}

func (m *MockPackageService) UninstallPackage(packageName string) error {
	if m.UninstallPackageFunc != nil {
		return m.UninstallPackageFunc(packageName)
	}
	return nil
}

func (m *MockPackageService) UpgradePackage(packageName, sourcePath string) error {
	if m.UpgradePackageFunc != nil {
		return m.UpgradePackageFunc(packageName, sourcePath)
	}
	return nil
}

func (m *MockPackageService) PackageExists(packageName string) bool {
	if m.PackageExistsFunc != nil {
		return m.PackageExistsFunc(packageName)
	}
	return false
}

func (m *MockPackageService) GetInstalledPackage(packageName string) (*packages.Package, error) {
	if m.GetInstalledPackageFunc != nil {
		return m.GetInstalledPackageFunc(packageName)
	}
	return nil, nil
}

func (m *MockPackageService) ListInstalledPackages() ([]*packages.Package, error) {
	if m.ListInstalledPackagesFunc != nil {
		return m.ListInstalledPackagesFunc()
	}
	return nil, nil
}

func (m *MockPackageService) GetPackageInstallPath(packageName string) string {
	if m.GetPackageInstallPathFunc != nil {
		return m.GetPackageInstallPathFunc(packageName)
	}
	return ""
}

func (m *MockPackageService) RunPackageHooks(packageDir string, hooks []packages.Hook) error {
	if m.RunPackageHooksFunc != nil {
		return m.RunPackageHooksFunc(packageDir, hooks)
	}
	return nil
}

func (m *MockPackageService) SyncPackageAssets(pkg *packages.Package, installPath string) error {
	if m.SyncPackageAssetsFunc != nil {
		return m.SyncPackageAssetsFunc(pkg, installPath)
	}
	return nil
}

func (m *MockPackageService) RemovePackageAssets(pkg *packages.Package) error {
	if m.RemovePackageAssetsFunc != nil {
		return m.RemovePackageAssetsFunc(pkg)
	}
	return nil
}

func (m *MockPackageService) TriggerDeploy(cockpitBin string) error {
	if m.TriggerDeployFunc != nil {
		return m.TriggerDeployFunc(cockpitBin)
	}
	return nil
}

func TestNewPackageService(t *testing.T) {
	svc := NewPackageService(t.TempDir())
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestDefaultPackageService_Delegation(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewPackageService(tmpDir)

	if svc.PackageExists("nonexistent") {
		t.Error("PackageExists(nonexistent) should be false on a fresh dir")
	}

	if path := svc.GetPackageInstallPath("foo"); path == "" {
		t.Error("GetPackageInstallPath(foo) should return a non-empty string")
	}

	installed, err := svc.ListInstalledPackages()
	if err != nil {
		t.Errorf("ListInstalledPackages() unexpected error: %v", err)
	}
	if len(installed) != 0 {
		t.Errorf("ListInstalledPackages() should be empty on fresh dir, got %d", len(installed))
	}

	if err := svc.InstallPackage(filepath.Join(tmpDir, "does-not-exist"), nil); err == nil {
		t.Error("InstallPackage(nonexistent) should return an error")
	}

	if err := svc.UninstallPackage("ghost"); err == nil {
		t.Error("UninstallPackage(ghost) should return an error")
	}

	if err := svc.UpgradePackage("ghost", "/tmp"); err == nil {
		t.Error("UpgradePackage(ghost) should return an error")
	}

	if _, err := svc.GetInstalledPackage("ghost"); err == nil {
		t.Error("GetInstalledPackage(ghost) should return an error")
	}

	results, err := svc.SearchPackages("query", nil)
	if err != nil {
		t.Errorf("SearchPackages(query, nil) unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("SearchPackages(query, nil) should be empty with nil registries, got %d", len(results))
	}

	categoryResults, err := svc.SearchByCategory("cat", nil)
	if err != nil {
		t.Errorf("SearchByCategory unexpected error: %v", err)
	}
	if len(categoryResults) != 0 {
		t.Errorf("SearchByCategory should be empty with nil registries, got %d", len(categoryResults))
	}

	tagResults, err := svc.SearchByTag("tag", nil)
	if err != nil {
		t.Errorf("SearchByTag unexpected error: %v", err)
	}
	if len(tagResults) != 0 {
		t.Errorf("SearchByTag should be empty with nil registries, got %d", len(tagResults))
	}

	listResults, err := svc.ListPackages(nil)
	if err != nil {
		t.Errorf("ListPackages(nil) unexpected error: %v", err)
	}
	if len(listResults) != 0 {
		t.Errorf("ListPackages(nil) should be empty with nil registries, got %d", len(listResults))
	}

	if _, _, err := svc.GetPackage("ghost", nil); err == nil {
		t.Error("GetPackage(ghost, nil) should return an error")
	}

	if _, err := svc.GetPackageFromCache("reg", "pkg"); err == nil {
		t.Error("GetPackageFromCache(reg, pkg) should return an error on a fresh cache")
	}

	if err := svc.RunPackageHooks(tmpDir, nil); err != nil {
		t.Errorf("RunPackageHooks(dir, nil) unexpected error: %v", err)
	}

	pkg := &packages.Package{Name: "empty-pkg", Features: packages.Features{}}
	if err := svc.SyncPackageAssets(pkg, tmpDir); err != nil {
		t.Errorf("SyncPackageAssets(empty pkg) unexpected error: %v", err)
	}
	if err := svc.RemovePackageAssets(pkg); err != nil {
		t.Errorf("RemovePackageAssets(empty pkg) unexpected error: %v", err)
	}

	t.Setenv("COCKPIT_SKIP_DEPLOY", "1")
	if err := svc.TriggerDeploy(""); err != nil {
		t.Errorf("TriggerDeploy(\"\") with COCKPIT_SKIP_DEPLOY=1 unexpected error: %v", err)
	}
}

func TestMockPackageService(t *testing.T) {
	called := false
	mock := &MockPackageService{
		SearchPackagesFunc: func(query string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
			called = true
			return []packages.PackageIndexEntry{{Name: "mock-pkg"}}, nil
		},
	}

	results, err := mock.SearchPackages("q", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected SearchPackagesFunc to be called")
	}
	if len(results) != 1 || results[0].Name != "mock-pkg" {
		t.Errorf("unexpected results: %+v", results)
	}

	// Ensure default methods do not panic.
	ctx := context.Background()
	_, _ = mock.ListPackages(nil)
	_, _, _ = mock.GetPackage("x", nil)
	_, _ = mock.GetPackageFromCache("r", "p")
	_ = mock.InstallPackage("src", nil)
	_ = mock.UninstallPackage("x")
	_ = mock.UpgradePackage("x", "src")
	_ = mock.PackageExists("x")
	_, _ = mock.GetInstalledPackage("x")
	_, _ = mock.ListInstalledPackages()
	_ = mock.GetPackageInstallPath("x")
	_ = mock.RunPackageHooks("dir", nil)
	_ = mock.SyncPackageAssets(&packages.Package{Name: "x"}, "dir")
	_ = mock.RemovePackageAssets(&packages.Package{Name: "x"})
	_ = mock.TriggerDeploy("")
	_ = ctx
}
