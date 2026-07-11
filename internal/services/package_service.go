// Package services provides a thin service layer between the CLI commands
// and the internal business logic packages, enabling dependency injection
// and easier unit testing of cmd/ files.
package services

import (
	"github.com/lleitep3/aicockpit/internal/packages"
)

// PackageService abstracts all package lifecycle and registry operations.
// cmd/ files depend on this interface rather than on concrete internal types.
type PackageService interface {
	// Registry / discovery
	SearchPackages(query string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	SearchByCategory(category string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	SearchByTag(tag string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	ListPackages(registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error)
	GetPackage(name string, registries []packages.RegistryConfig) (*packages.PackageIndexEntry, string, error)

	// Cache
	GetPackageFromCache(registryName, packageName string) (string, error)

	// Lifecycle
	InstallPackage(sourcePath string, cfg map[string]interface{}) error
	UninstallPackage(packageName string) error
	UpgradePackage(packageName, sourcePath string) error

	// Query
	PackageExists(packageName string) bool
	GetInstalledPackage(packageName string) (*packages.Package, error)
	ListInstalledPackages() ([]*packages.Package, error)
	GetPackageInstallPath(packageName string) string

	// Hooks / assets / deploy
	RunPackageHooks(packageDir string, hooks []packages.Hook) error
	SyncPackageAssets(pkg *packages.Package, installPath string) error
	RemovePackageAssets(pkg *packages.Package) error
	TriggerDeploy(cockpitBin string) error
}

// DefaultPackageService is the production implementation of PackageService.
// It delegates to PackageManager, RegistryManager, and RegistryCache.
type DefaultPackageService struct {
	pm    *packages.PackageManager
	rm    *packages.RegistryManager
	cache *packages.RegistryCache
}

// NewPackageService creates a DefaultPackageService backed by the given cockpit directory.
func NewPackageService(cockpitDir string) PackageService {
	return &DefaultPackageService{
		pm:    packages.NewPackageManager(cockpitDir),
		rm:    packages.NewRegistryManager(cockpitDir),
		cache: packages.NewRegistryCache(cockpitDir),
	}
}

func (s *DefaultPackageService) SearchPackages(query string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	return s.rm.SearchPackages(query, registries)
}

func (s *DefaultPackageService) SearchByCategory(category string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	return s.rm.SearchByCategory(category, registries)
}

func (s *DefaultPackageService) SearchByTag(tag string, registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	return s.rm.SearchByTag(tag, registries)
}

func (s *DefaultPackageService) ListPackages(registries []packages.RegistryConfig) ([]packages.PackageIndexEntry, error) {
	return s.rm.ListPackages(registries)
}

func (s *DefaultPackageService) GetPackage(name string, registries []packages.RegistryConfig) (*packages.PackageIndexEntry, string, error) {
	return s.rm.GetPackage(name, registries)
}

func (s *DefaultPackageService) GetPackageFromCache(registryName, packageName string) (string, error) {
	return s.cache.GetPackageFromCache(registryName, packageName)
}

func (s *DefaultPackageService) InstallPackage(sourcePath string, cfg map[string]interface{}) error {
	return s.pm.InstallPackage(sourcePath, cfg)
}

func (s *DefaultPackageService) UninstallPackage(packageName string) error {
	return s.pm.UninstallPackage(packageName)
}

func (s *DefaultPackageService) UpgradePackage(packageName, sourcePath string) error {
	return s.pm.UpgradePackage(packageName, sourcePath)
}

func (s *DefaultPackageService) PackageExists(packageName string) bool {
	return s.pm.PackageExists(packageName)
}

func (s *DefaultPackageService) GetInstalledPackage(packageName string) (*packages.Package, error) {
	return s.pm.GetInstalledPackage(packageName)
}

func (s *DefaultPackageService) ListInstalledPackages() ([]*packages.Package, error) {
	return s.pm.ListInstalledPackages()
}

func (s *DefaultPackageService) GetPackageInstallPath(packageName string) string {
	return s.pm.GetPackageInstallPath(packageName)
}

func (s *DefaultPackageService) RunPackageHooks(packageDir string, hooks []packages.Hook) error {
	return s.pm.RunPackageHooks(packageDir, hooks)
}

func (s *DefaultPackageService) SyncPackageAssets(pkg *packages.Package, installPath string) error {
	return s.pm.SyncPackageAssets(pkg, installPath)
}

func (s *DefaultPackageService) RemovePackageAssets(pkg *packages.Package) error {
	return s.pm.RemovePackageAssets(pkg)
}

func (s *DefaultPackageService) TriggerDeploy(cockpitBin string) error {
	return s.pm.TriggerDeploy(cockpitBin)
}
