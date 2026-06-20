package packages

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RegistryCache manages local cache of package registries
type RegistryCache struct {
	cacheDir string
}

// NewRegistryCache creates a new registry cache manager
func NewRegistryCache(cockpitDir string) *RegistryCache {
	return &RegistryCache{
		cacheDir: filepath.Join(cockpitDir, "cache", "registries"),
	}
}

// GetRegistryCachePath returns the cache path for a registry
func (rc *RegistryCache) GetRegistryCachePath(registryName string) string {
	return filepath.Join(rc.cacheDir, registryName)
}

// EnsureRegistry ensures the registry is cloned and up-to-date
func (rc *RegistryCache) EnsureRegistry(registry RegistryConfig) error {
	cachePath := rc.GetRegistryCachePath(registry.Name)

	// Check if registry is already cloned
	if rc.isCloned(cachePath) {
		// Update existing clone
		fmt.Printf("Updating registry cache: %s\n", registry.Name)
		return rc.updateRegistry(cachePath, registry)
	}

	// Clone registry
	fmt.Printf("Cloning registry: %s\n", registry.Name)
	return rc.cloneRegistry(registry, cachePath)
}

// isCloned checks if a registry is already cloned
func (rc *RegistryCache) isCloned(cachePath string) bool {
	gitDir := filepath.Join(cachePath, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

// cloneRegistry clones a registry to cache
func (rc *RegistryCache) cloneRegistry(registry RegistryConfig, cachePath string) error {
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Clone the repository
	cmd := exec.Command("git", "clone", "--depth", "1", "-b", registry.Branch, registry.URL, cachePath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone registry: %w\n%s", err, string(output))
	}

	fmt.Printf("✓ Registry cloned successfully\n")
	return nil
}

// updateRegistry updates an existing registry clone
func (rc *RegistryCache) updateRegistry(cachePath string, registry RegistryConfig) error {
	// Fetch latest changes
	cmd := exec.Command("git", "-C", cachePath, "fetch", "origin", registry.Branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch registry: %w\n%s", err, string(output))
	}

	// Pull latest changes
	cmd = exec.Command("git", "-C", cachePath, "pull", "origin", registry.Branch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull registry: %w\n%s", err, string(output))
	}

	fmt.Printf("✓ Registry updated successfully\n")
	return nil
}

// GetPackageIndexPath returns the path to package-index.yaml in cache
func (rc *RegistryCache) GetPackageIndexPath(registryName string) string {
	return filepath.Join(rc.GetRegistryCachePath(registryName), "package-index.yaml")
}

// GetPackagePath returns the path to a package in cache
func (rc *RegistryCache) GetPackagePath(registryName, packageName string) string {
	return filepath.Join(rc.GetRegistryCachePath(registryName), packageName)
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
func (rc *RegistryCache) GetPackageFromCache(registryName, packageName string) (string, error) {
	packagePath := rc.GetPackagePath(registryName, packageName)

	// Check if package exists
	if _, err := os.Stat(packagePath); err != nil {
		return "", fmt.Errorf("package not found in cache: %s", packageName)
	}

	return packagePath, nil
}
