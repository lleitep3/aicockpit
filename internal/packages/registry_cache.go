package packages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/resilience"
	"gopkg.in/yaml.v3"
)

// RegistryCache manages local cache of package registries
type RegistryCache struct {
	cacheDir  string
	gitRunner *resilience.GitRunner
	rename    func(string, string) error
}

// NewRegistryCache creates a new registry cache manager
func NewRegistryCache(cockpitDir string) *RegistryCache {
	return &RegistryCache{
		cacheDir:  filepath.Join(cockpitDir, "cache", "registries"),
		gitRunner: resilience.DefaultGitRunner(),
		rename:    os.Rename,
	}
}

// gitContext returns a background context with a generous timeout for git operations.
func gitContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Minute)
}

// GetRegistryCachePath returns the cache path for a registry
func (rc *RegistryCache) GetRegistryCachePath(registryName string) string {
	return filepath.Join(rc.cacheDir, registryName)
}

// EnsureRegistry ensures the registry is cloned and up-to-date
func (rc *RegistryCache) EnsureRegistry(registry RegistryConfig) error {
	return rc.updateRegistry(rc.GetRegistryCachePath(registry.Name), registry)
}

// cloneRegistry clones a registry to cache
func (rc *RegistryCache) cloneRegistry(registry RegistryConfig, cachePath string) error {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	ctx, cancel := gitContext()
	defer cancel()

	_, err := rc.gitRunner.Run(ctx, "", "clone", "--depth", "1", "-b", registry.Branch, "--", registry.URL, cachePath)
	if err != nil {
		return fmt.Errorf("failed to clone registry: %w", err)
	}

	fmt.Printf("✓ Registry cloned successfully\n")
	return nil
}

// updateRegistry updates an existing registry clone
func (rc *RegistryCache) updateRegistry(cachePath string, registry RegistryConfig) error {
	return rc.refreshRegistry(cachePath, registry)
}

// GetPackageIndexPath returns the path to package-index.yaml in cache
func (rc *RegistryCache) GetPackageIndexPath(registryName string) string {
	return filepath.Join(rc.GetRegistryCachePath(registryName), "package-index.yaml")
}

// GetPackagePath returns the path to a package in cache
func (rc *RegistryCache) GetPackagePath(registryName, packagePath string) string {
	return filepath.Join(rc.GetRegistryCachePath(registryName), packagePath)
}

// LoadPackageIndexFromCache loads package index from local cache
func (rc *RegistryCache) LoadPackageIndexFromCache(registryName string) (*PackageIndex, error) {
	indexPath := rc.GetPackageIndexPath(registryName)

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package index from cache: %w", err)
	}

	var index PackageIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse package index: %w", err)
	}

	return &index, nil
}

// ListPackagesInCache lists all packages in a registry cache
func (rc *RegistryCache) ListPackagesInCache(registryName string) ([]string, error) {
	registryPath := rc.GetRegistryCachePath(registryName)

	entries, err := os.ReadDir(registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry cache: %w", err)
	}

	var packages []string
	for _, entry := range entries {
		// Skip non-directories and special files
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		// Check if directory has cockpit-package.yml
		manifestPath := filepath.Join(registryPath, entry.Name(), "cockpit-package.yml")
		if _, err := os.Stat(manifestPath); err == nil {
			packages = append(packages, entry.Name())
		}
	}

	return packages, nil
}

// GetPackageFromCache gets a package from cache
func (rc *RegistryCache) GetPackageFromCache(registryName, packagePath string) (string, error) {
	registryPath := rc.GetRegistryCachePath(registryName)
	cleanPath := filepath.Clean(packagePath)
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid package path: %s", packagePath)
	}

	cachedPath := rc.GetPackagePath(registryName, cleanPath)
	relativePath, err := filepath.Rel(registryPath, cachedPath)
	if err != nil || relativePath == ".." || strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid package path: %s", packagePath)
	}

	// Check if package exists
	if _, err := os.Stat(cachedPath); err != nil {
		return "", fmt.Errorf("package not found in cache: %s", packagePath)
	}

	return cachedPath, nil
}
