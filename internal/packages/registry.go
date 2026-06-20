package packages

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"gopkg.in/yaml.v3"
)

// RegistryConfig represents a package registry configuration.
type RegistryConfig struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Branch   string `yaml:"branch"`
	Enabled  bool   `yaml:"enabled"`
	Priority int    `yaml:"priority"`
}

// PackageIndex represents the package-index.yaml file in a registry.
type PackageIndex struct {
	Version     string                 `yaml:"version"`
	Name        string                 `yaml:"name"`
	Description string                 `yaml:"description"`
	URL         string                 `yaml:"url"`
	Maintainer  string                 `yaml:"maintainer"`
	Email       string                 `yaml:"maintainer_email"`
	UpdatedAt   string                 `yaml:"updated_at"`
	Metadata    RegistryMetadata       `yaml:"metadata"`
	Packages    []PackageIndexEntry    `yaml:"packages"`
}

// RegistryMetadata represents registry metadata.
type RegistryMetadata struct {
	TotalPackages int      `yaml:"total_packages"`
	Categories    []string `yaml:"categories"`
}

// PackageIndexEntry represents a package entry in the registry index.
type PackageIndexEntry struct {
	Name                  string   `yaml:"name"`
	Version               string   `yaml:"version"`
	Description           string   `yaml:"description"`
	Author                string   `yaml:"author"`
	License               string   `yaml:"license"`
	Category              string   `yaml:"category"`
	Tags                  []string `yaml:"tags"`
	Path                  string   `yaml:"path"`
	URL                   string   `yaml:"url"`
	Homepage              string   `yaml:"homepage"`
	Repository            string   `yaml:"repository"`
	SupportedProviders    []string `yaml:"supported_providers"`
	Features              []string `yaml:"features"`
	Requirements          Requirements `yaml:"requirements"`
	Dependencies          []Dependency `yaml:"dependencies"`
	InstallationMethod    string   `yaml:"installation_method"`
	Checksum              string   `yaml:"checksum"`
	SizeBytes             int64    `yaml:"size_bytes"`
	Status                string   `yaml:"status"`
	ReleasedAt            string   `yaml:"released_at"`
}

// RegistryManager manages package registries.
type RegistryManager struct {
	cockpitDir string
	cacheDir   string
}

// NewRegistryManager creates a new registry manager.
func NewRegistryManager(cockpitDir string) *RegistryManager {
	return &RegistryManager{
		cockpitDir: cockpitDir,
		cacheDir:   filepath.Join(cockpitDir, "cache", "registries"),
	}
}

// LoadPackageIndex loads a package index from a registry.
func (rm *RegistryManager) LoadPackageIndex(registryName string) (*PackageIndex, error) {
	indexPath := filepath.Join(rm.cacheDir, registryName, "package-index.yaml")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package index: %w", err)
	}

	var index PackageIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse package index: %w", err)
	}

	return &index, nil
}

// SavePackageIndex saves a package index to cache.
func (rm *RegistryManager) SavePackageIndex(registryName string, index *PackageIndex) error {
	cacheDir := filepath.Join(rm.cacheDir, registryName)

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	indexPath := filepath.Join(cacheDir, "package-index.yaml")

	data, err := yaml.Marshal(index)
	if err != nil {
		return fmt.Errorf("failed to marshal package index: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write package index: %w", err)
	}

	return nil
}

// SearchPackages searches for packages in registries.
func (rm *RegistryManager) SearchPackages(query string, registries []RegistryConfig) ([]PackageIndexEntry, error) {
	var results []PackageIndexEntry

	// Sort registries by priority
	sortedRegistries := make([]RegistryConfig, len(registries))
	copy(sortedRegistries, registries)
	sort.Slice(sortedRegistries, func(i, j int) bool {
		return sortedRegistries[i].Priority < sortedRegistries[j].Priority
	})

	// Search in each registry
	for _, registry := range sortedRegistries {
		if !registry.Enabled {
			continue
		}

		index, err := rm.LoadPackageIndex(registry.Name)
		if err != nil {
			// Skip registries that can't be loaded
			continue
		}

		// Search in packages
		for _, pkg := range index.Packages {
			if rm.matchesQuery(pkg, query) {
				results = append(results, pkg)
			}
		}
	}

	return results, nil
}

// GetPackage gets a specific package from registries.
func (rm *RegistryManager) GetPackage(packageName string, registries []RegistryConfig) (*PackageIndexEntry, string, error) {
	// Sort registries by priority
	sortedRegistries := make([]RegistryConfig, len(registries))
	copy(sortedRegistries, registries)
	sort.Slice(sortedRegistries, func(i, j int) bool {
		return sortedRegistries[i].Priority < sortedRegistries[j].Priority
	})

	// Search in each registry
	for _, registry := range sortedRegistries {
		if !registry.Enabled {
			continue
		}

		index, err := rm.LoadPackageIndex(registry.Name)
		if err != nil {
			continue
		}

		// Find package
		for _, pkg := range index.Packages {
			if pkg.Name == packageName {
				return &pkg, registry.Name, nil
			}
		}
	}

	return nil, "", fmt.Errorf("package not found: %s", packageName)
}

// GetPackageFromRegistry gets a package from a specific registry.
func (rm *RegistryManager) GetPackageFromRegistry(packageName string, registryName string) (*PackageIndexEntry, error) {
	index, err := rm.LoadPackageIndex(registryName)
	if err != nil {
		return nil, err
	}

	for _, pkg := range index.Packages {
		if pkg.Name == packageName {
			return &pkg, nil
		}
	}

	return nil, fmt.Errorf("package not found in registry: %s", packageName)
}

// ListPackages lists all packages in registries.
func (rm *RegistryManager) ListPackages(registries []RegistryConfig) ([]PackageIndexEntry, error) {
	var results []PackageIndexEntry

	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}

		index, err := rm.LoadPackageIndex(registry.Name)
		if err != nil {
			continue
		}

		results = append(results, index.Packages...)
	}

	return results, nil
}

// SearchByCategory searches packages by category.
func (rm *RegistryManager) SearchByCategory(category string, registries []RegistryConfig) ([]PackageIndexEntry, error) {
	var results []PackageIndexEntry

	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}

		index, err := rm.LoadPackageIndex(registry.Name)
		if err != nil {
			continue
		}

		for _, pkg := range index.Packages {
			if pkg.Category == category {
				results = append(results, pkg)
			}
		}
	}

	return results, nil
}

// SearchByTag searches packages by tag.
func (rm *RegistryManager) SearchByTag(tag string, registries []RegistryConfig) ([]PackageIndexEntry, error) {
	var results []PackageIndexEntry

	for _, registry := range registries {
		if !registry.Enabled {
			continue
		}

		index, err := rm.LoadPackageIndex(registry.Name)
		if err != nil {
			continue
		}

		for _, pkg := range index.Packages {
			for _, t := range pkg.Tags {
				if t == tag {
					results = append(results, pkg)
					break
				}
			}
		}
	}

	return results, nil
}

// matchesQuery checks if a package matches the search query.
func (rm *RegistryManager) matchesQuery(pkg PackageIndexEntry, query string) bool {
	// Match name
	if contains(pkg.Name, query) {
		return true
	}

	// Match description
	if contains(pkg.Description, query) {
		return true
	}

	// Match tags
	for _, tag := range pkg.Tags {
		if contains(tag, query) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	// Simple substring check
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) > len(substr))
}

// GetCacheDir returns the cache directory.
func (rm *RegistryManager) GetCacheDir() string {
	return rm.cacheDir
}

// GetRegistryCacheDir returns the cache directory for a specific registry.
func (rm *RegistryManager) GetRegistryCacheDir(registryName string) string {
	return filepath.Join(rm.cacheDir, registryName)
}

// ClearCache clears the registry cache.
func (rm *RegistryManager) ClearCache() error {
	return os.RemoveAll(rm.cacheDir)
}

// ClearRegistryCache clears the cache for a specific registry.
func (rm *RegistryManager) ClearRegistryCache(registryName string) error {
	return os.RemoveAll(rm.GetRegistryCacheDir(registryName))
}

// CreatePackageIndex creates a new package index.
func CreatePackageIndex(name, description, url, maintainer, email string) *PackageIndex {
	return &PackageIndex{
		Version:     "1.0",
		Name:        name,
		Description: description,
		URL:         url,
		Maintainer:  maintainer,
		Email:       email,
		UpdatedAt:   time.Now().Format(time.RFC3339),
		Metadata: RegistryMetadata{
			TotalPackages: 0,
			Categories:    []string{},
		},
		Packages: []PackageIndexEntry{},
	}
}

// AddPackageToIndex adds a package to the index.
func (pi *PackageIndex) AddPackage(entry PackageIndexEntry) {
	pi.Packages = append(pi.Packages, entry)
	pi.Metadata.TotalPackages = len(pi.Packages)
	pi.UpdatedAt = time.Now().Format(time.RFC3339)

	// Update categories
	categoryMap := make(map[string]bool)
	for _, pkg := range pi.Packages {
		if pkg.Category != "" {
			categoryMap[pkg.Category] = true
		}
	}

	pi.Metadata.Categories = []string{}
	for category := range categoryMap {
		pi.Metadata.Categories = append(pi.Metadata.Categories, category)
	}
	sort.Strings(pi.Metadata.Categories)
}

// RemovePackageFromIndex removes a package from the index.
func (pi *PackageIndex) RemovePackage(packageName string) bool {
	for i, pkg := range pi.Packages {
		if pkg.Name == packageName {
			pi.Packages = append(pi.Packages[:i], pi.Packages[i+1:]...)
			pi.Metadata.TotalPackages = len(pi.Packages)
			pi.UpdatedAt = time.Now().Format(time.RFC3339)
			return true
		}
	}
	return false
}

// GetPackageByName gets a package by name from the index.
func (pi *PackageIndex) GetPackageByName(packageName string) *PackageIndexEntry {
	for i, pkg := range pi.Packages {
		if pkg.Name == packageName {
			return &pi.Packages[i]
		}
	}
	return nil
}
