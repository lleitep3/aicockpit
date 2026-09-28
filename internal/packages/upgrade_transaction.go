package packages

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var upgradeName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`)

func safeUpgradeName(name string) bool { return upgradeName.MatchString(name) && len(name) <= 128 }

type upgradeSnapshot struct {
	target, backup string
	existed        bool
}
type upgradeTransaction struct {
	dir, stage string
	snapshots  []upgradeSnapshot
}

func relativeUpgradePath(path string) bool {
	clean := filepath.Clean(path)
	return path != "" && !filepath.IsAbs(path) && filepath.VolumeName(path) == "" && clean != "." && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator)) && !strings.Contains(path, "\\")
}

// checkUpgradeTree refuses links and special files instead of dereferencing a
// package-controlled path while copying or restoring managed assets.
func checkUpgradeTree(root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&os.ModeSymlink != 0 || (!d.IsDir() && !d.Type().IsRegular()) {
			return fmt.Errorf("unsupported link or special file in upgrade: %s", path)
		}
		return nil
	})
}

func validateUpgradePackage(pkg *Package, root string) error {
	if err := checkUpgradeTree(root); err != nil {
		return err
	}
	groups := [][]Feature{pkg.Features.Modules, pkg.Features.Skills, pkg.Features.Agents, pkg.Features.Rules, pkg.Features.Workflows}
	for _, group := range groups {
		for _, f := range group {
			if !safeUpgradeName(f.Name) || !relativeUpgradePath(f.Path) {
				return fmt.Errorf("invalid feature name or path")
			}
		}
	}
	for _, f := range pkg.Features.KB {
		if !relativeUpgradePath(f.Path) {
			return fmt.Errorf("invalid KB path")
		}
	}
	if err := pkg.Validate(root); err != nil {
		return fmt.Errorf("candidate validation failed: %w", err)
	}
	return nil
}

func upgradeAssetPaths(root string, pkgs ...*Package) ([]string, error) {
	paths := map[string]bool{filepath.Join(root, "COCKPIT.md"): true}
	for _, pkg := range pkgs {
		groups := map[string][]Feature{"skills": pkg.Features.Skills, "rules": pkg.Features.Rules, "agents": pkg.Features.Agents, "workflows": pkg.Features.Workflows}
		for dir, features := range groups {
			for _, f := range features {
				if !safeUpgradeName(f.Name) {
					return nil, fmt.Errorf("invalid canonical asset name")
				}
				paths[filepath.Join(root, dir, f.Name)] = true
			}
		}
		if len(pkg.Features.KB) > 0 {
			paths[filepath.Join(root, "kb", "packages", pkg.Name)] = true
		}
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result, nil
}

// checkUpgradeParents prevents a canonical parent symlink from redirecting a
// snapshot/restore outside the Cockpit root. It also rejects non-directory parents.
func checkUpgradeParents(root, target string) error {
	for path := filepath.Dir(target); ; path = filepath.Dir(path) {
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			return fmt.Errorf("invalid upgrade parent: %s", path)
		}
		if path == root {
			return nil
		}
		if path == filepath.Dir(path) {
			return fmt.Errorf("target outside Cockpit directory")
		}
	}
}

func snapshotUpgradePath(target, backup string) (upgradeSnapshot, error) {
	item := upgradeSnapshot{target: target, backup: backup}
	info, err := os.Lstat(target)
	if os.IsNotExist(err) {
		return item, nil
	}
	if err != nil {
		return item, err
	}
	if err := checkUpgradeTree(target); err != nil {
		return item, err
	}
	item.existed = true
	if info.IsDir() {
		if err := os.MkdirAll(backup, 0700); err != nil {
			return item, err
		}
		return item, copyDir(target, backup)
	}
	return item, copyFile(target, backup)
}

func prepareUpgrade(root, installed, source string, oldPkg, newPkg *Package) (*upgradeTransaction, error) {
	paths, err := upgradeAssetPaths(root, oldPkg, newPkg)
	if err != nil {
		return nil, err
	}
	paths = append([]string{installed}, paths...)
	for _, path := range paths {
		if err := checkUpgradeParents(root, path); err != nil {
			return nil, err
		}
	}
	backups := filepath.Join(root, "backups")
	if err := checkUpgradeParents(root, filepath.Join(backups, "placeholder")); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(backups, 0700); err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp(backups, "upgrade-"+newPkg.Name+"-")
	if err != nil {
		return nil, err
	}
	tx := &upgradeTransaction{dir: dir, stage: filepath.Join(dir, "candidate")}
	if err := os.Mkdir(tx.stage, 0700); err != nil {
		return nil, err
	}
	if err := copyDir(source, tx.stage); err != nil {
		return nil, fmt.Errorf("failed to stage candidate: %w", err)
	}
	for i, path := range paths {
		item, err := snapshotUpgradePath(path, filepath.Join(dir, fmt.Sprintf("snapshot-%d", i)))
		if err != nil {
			return nil, fmt.Errorf("failed to snapshot managed path: %w", err)
		}
		tx.snapshots = append(tx.snapshots, item)
	}
	// Paths, not secret contents, are recorded for manual crash recovery.
	var journal strings.Builder
	for _, item := range tx.snapshots {
		fmt.Fprintf(&journal, "%t\t%s\t%s\n", item.existed, item.backup, item.target)
	}
	if err := os.WriteFile(filepath.Join(dir, "recovery.tsv"), []byte(journal.String()), 0600); err != nil {
		return nil, err
	}
	return tx, nil
}

func (tx *upgradeTransaction) restore() error {
	var failures []error
	for _, item := range tx.snapshots {
		// Re-check parents: a hook might have replaced a parent with a symlink.
		root := filepath.Dir(filepath.Dir(tx.dir))
		if err := checkUpgradeParents(root, item.target); err != nil {
			failures = append(failures, err)
			continue
		}
		var info os.FileInfo
		if item.existed {
			if err := checkUpgradeTree(item.backup); err != nil {
				failures = append(failures, err)
				continue
			}
			var err error
			info, err = os.Stat(item.backup)
			if err != nil {
				failures = append(failures, err)
				continue
			}
		}
		if err := os.RemoveAll(item.target); err != nil {
			failures = append(failures, err)
			continue
		}
		if !item.existed {
			continue
		}
		var err error
		if info.IsDir() {
			err = os.MkdirAll(item.target, 0700)
			if err == nil {
				err = copyDir(item.backup, item.target)
			}
		} else {
			err = os.MkdirAll(filepath.Dir(item.target), 0700)
			if err == nil {
				err = copyFile(item.backup, item.target)
			}
		}
		if err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}
