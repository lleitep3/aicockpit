package packages

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Links remain relative when a package moves between staging, activation and backup.
// Reject external, dangling and cyclic targets; never follow links while copying.
func validateInternalUpgradeLink(root, path string) error {
	target, err := os.Readlink(path)
	if err != nil {
		return err
	}
	if filepath.IsAbs(target) {
		return fmt.Errorf("absolute upgrade link: %s", path)
	}
	lexical, err := filepath.Rel(root, filepath.Join(filepath.Dir(path), target))
	if err != nil || lexical == ".." || strings.HasPrefix(lexical, ".."+string(filepath.Separator)) {
		return fmt.Errorf("upgrade link escapes package: %s", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("unresolved upgrade link %s: %w", path, err)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("upgrade link escapes package: %s", path)
	}
	return nil
}

// Exported assets still use the strict, link-free contract of the asset syncer.
func validateUpgradeAsset(root, relative string) error {
	path := filepath.Join(root, relative)
	if err := checkUpgradeParents(root, path); err != nil {
		return err
	}
	return checkUpgradeTree(path)
}

func copyUpgradeTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0700)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported upgrade file: %s", path)
		}
		return copyFile(path, target)
	})
}
