package packages

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// refreshRegistry replaces a cache from its configured origin, never merging
// local history. The previous directory remains available for manual recovery.
func (rc *RegistryCache) refreshRegistry(cachePath string, registry RegistryConfig) error {
	if !safeUpgradeName(registry.Name) {
		return fmt.Errorf("invalid registry name")
	}
	parent := filepath.Dir(cachePath)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return err
	}
	lockPath := cachePath + ".refresh.lock"
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("registry refresh unavailable: %w", err)
	}
	keepLock := false
	defer func() {
		if !keepLock {
			_ = os.Remove(lockPath)
		}
	}()
	if err := lock.Close(); err != nil {
		return err
	}
	info, err := os.Lstat(cachePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
		return fmt.Errorf("registry cache must be a real directory")
	}
	existed := err == nil
	stage, err := os.MkdirTemp(parent, ".refresh-"+registry.Name+"-")
	if err != nil {
		return err
	}
	candidate := filepath.Join(stage, "candidate")
	if err := rc.cloneRegistry(registry, candidate); err != nil {
		return err
	}
	if err := validateRegistrySnapshot(candidate); err != nil {
		return err
	}
	previous := filepath.Join(stage, "previous")
	if existed {
		if err := rc.rename(cachePath, previous); err != nil {
			return fmt.Errorf("preserve previous registry: %w", err)
		}
	}
	if err := rc.rename(candidate, cachePath); err != nil {
		if existed {
			if restoreErr := rc.rename(previous, cachePath); restoreErr != nil {
				keepLock = true
				return fmt.Errorf("activate registry: %w; restore failed: %w; recovery: %s", err, restoreErr, previous)
			}
		}
		return fmt.Errorf("activate registry: %w", err)
	}
	if existed {
		fmt.Printf("Previous registry preserved: %s\n", previous)
	}
	return nil
}

func validateRegistrySnapshot(root string) error {
	indexPath := filepath.Join(root, "package-index.yaml")
	info, err := os.Lstat(indexPath)
	if err != nil {
		return fmt.Errorf("registry index missing: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("registry index must be a regular file")
	}
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("registry index missing: %w", err)
	}
	var index PackageIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return fmt.Errorf("invalid registry index: %w", err)
	}
	if len(data) == 0 || (index.Version == "" && index.Packages == nil) {
		return fmt.Errorf("empty registry index")
	}
	seen := map[string]bool{}
	for _, entry := range index.Packages {
		path := entry.Path
		if path == "" {
			path = entry.Name
		}
		if !safeUpgradeName(entry.Name) || entry.Version == "" || !relativeUpgradePath(path) || seen[entry.Name] {
			return fmt.Errorf("invalid or duplicate registry package: %s", entry.Name)
		}
		seen[entry.Name] = true
	}
	return nil
}
