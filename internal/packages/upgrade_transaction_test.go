package packages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeUpgradeFixture(t *testing.T, root, version, skill, hook string) string {
	t.Helper()
	dir := filepath.Join(root, "source-"+version)
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0700); err != nil {
		t.Fatal(err)
	}
	manifest := `name: safe-pkg
version: "` + version + `"
description: Transaction test
author: Test
license: MIT
requirements:
  cockpit: ">=0.1.0"
features:
  skills:
    - name: ` + skill + `
      path: skills/test.md
installation:
  supported_providers: [all]
  provider_features:
    all: [skills]
` + hook
	for path, body := range map[string]string{"cockpit-package.yml": manifest, "skills/test.md": version} {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestUpgradeTransactionRestoresCompletePackage(t *testing.T) {
	root := t.TempDir()
	pm := NewPackageManager(root)
	old := writeUpgradeFixture(t, root, "1.0.0", "old-skill", "")
	if err := pm.InstallPackage(old, nil); err != nil {
		t.Fatal(err)
	}
	installed := filepath.Join(root, "packages", "safe-pkg")
	original, err := os.ReadFile(filepath.Join(installed, "cockpit-package.yml"))
	if err != nil {
		t.Fatal(err)
	}
	canonical := filepath.Join(root, "skills", "old-skill")
	if err := os.MkdirAll(filepath.Dir(canonical), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(canonical, []byte("local customization"), 0600); err != nil {
		t.Fatal(err)
	}
	source := writeUpgradeFixture(t, root, "2.0.0", "new-skill", "  post_install:\n    - script: fail.sh\n")
	if err := os.WriteFile(filepath.Join(source, "fail.sh"), []byte("#!/bin/sh\nexit 7\n"), 0700); err != nil {
		t.Fatal(err)
	}
	err = pm.UpgradePackage("safe-pkg", source)
	if err == nil || !strings.Contains(err.Error(), "managed files restored") {
		t.Fatalf("expected rollback: %v", err)
	}
	for path, expected := range map[string]string{filepath.Join(installed, "cockpit-package.yml"): string(original), filepath.Join(installed, "skills", "test.md"): "1.0.0", canonical: "local customization"} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != expected {
			t.Fatalf("restore %s: %q %v", path, data, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "new-skill")); !os.IsNotExist(err) {
		t.Fatalf("new asset remains: %v", err)
	}
	snapshots, _ := filepath.Glob(filepath.Join(root, "backups", "upgrade-*", "snapshot-0", "cockpit-package.yml"))
	if len(snapshots) != 1 {
		t.Fatalf("complete manifest backup missing: %v", snapshots)
	}
}

func TestUpgradeTransactionSuccessReplacesAssets(t *testing.T) {
	root := t.TempDir()
	pm := NewPackageManager(root)
	if err := pm.InstallPackage(writeUpgradeFixture(t, root, "1.0.0", "old-skill", ""), nil); err != nil {
		t.Fatal(err)
	}
	if err := pm.UpgradePackage("safe-pkg", writeUpgradeFixture(t, root, "2.0.0", "new-skill", "")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "skills", "new-skill"))
	if err != nil || string(data) != "2.0.0" {
		t.Fatalf("new skill: %q %v", data, err)
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "old-skill")); !os.IsNotExist(err) {
		t.Fatalf("stale skill: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".package-upgrade.lock")); !os.IsNotExist(err) {
		t.Fatalf("lock remains: %v", err)
	}
}

func TestUpgradeTransactionRejectsUnsafeCandidate(t *testing.T) {
	for _, scenario := range []string{"missing-feature", "identity", "symlink", "busy"} {
		t.Run(scenario, func(t *testing.T) {
			root := t.TempDir()
			pm := NewPackageManager(root)
			if err := pm.InstallPackage(writeUpgradeFixture(t, root, "1.0.0", "old-skill", ""), nil); err != nil {
				t.Fatal(err)
			}
			source := writeUpgradeFixture(t, root, "2.0.0", "new-skill", "")
			switch scenario {
			case "missing-feature":
				if err := os.Rename(filepath.Join(source, "skills"), filepath.Join(source, "hidden")); err != nil {
					t.Fatal(err)
				}
			case "identity":
				p, err := LoadPackage(source)
				if err != nil {
					t.Fatal(err)
				}
				p.Name = "another"
				if err := SavePackage(source, p); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Symlink(root, filepath.Join(source, "link")); err != nil {
					t.Skip(err)
				}
			case "busy":
				if err := os.WriteFile(filepath.Join(root, ".package-upgrade.lock"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := pm.UpgradePackage("safe-pkg", source); err == nil {
				t.Fatal("unsafe candidate accepted")
			}
			pkg, err := pm.GetInstalledPackage("safe-pkg")
			if err != nil || pkg.Version != "1.0.0" {
				t.Fatalf("original changed: %v %v", pkg, err)
			}
		})
	}
}

func TestUpgradeRestorePreservesTargetWhenBackupMissing(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "asset")
	if err := os.WriteFile(target, []byte("survivor"), 0600); err != nil {
		t.Fatal(err)
	}
	tx := upgradeTransaction{dir: filepath.Join(root, "backups", "transaction"), snapshots: []upgradeSnapshot{{target: target, backup: filepath.Join(root, "missing"), existed: true}}}
	if err := tx.restore(); err == nil {
		t.Fatal("missing snapshot accepted")
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "survivor" {
		t.Fatalf("target lost: %q %v", data, err)
	}
}

func TestUpgradeRelativePaths(t *testing.T) {
	for _, path := range []string{"", ".", "..", "../outside", "/outside", `dir\outside`} {
		if relativeUpgradePath(path) {
			t.Errorf("accepted %q", path)
		}
	}
	if !relativeUpgradePath("skills/test.md") {
		t.Fatal("valid path rejected")
	}
}

func TestUpgradeSnapshotRestoresAllManagedKinds(t *testing.T) {
	root := t.TempDir()
	source := writeUpgradeFixture(t, root, "1.0.0", "old-skill", "")
	old, err := LoadPackage(source)
	if err != nil {
		t.Fatal(err)
	}
	candidate := writeUpgradeFixture(t, root, "2.0.0", "new-skill", "")
	next, err := LoadPackage(candidate)
	if err != nil {
		t.Fatal(err)
	}
	// Snapshot file assets, directories and assets absent before activation.
	assets := map[string]string{"skills/old-skill/custom.txt": "custom", "COCKPIT.md": "user rules", "kb/packages/safe-pkg/note.md": "knowledge"}
	old.Features.KB = append(old.Features.KB, KBFeature{Path: "note.md"})
	for relative, body := range assets {
		path := filepath.Join(root, relative)
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	installed := filepath.Join(root, "packages", "safe-pkg")
	if err := os.MkdirAll(installed, 0700); err != nil {
		t.Fatal(err)
	}
	if err := copyDir(source, installed); err != nil {
		t.Fatal(err)
	}
	tx, err := prepareUpgrade(root, installed, candidate, old, next)
	if err != nil {
		t.Fatal(err)
	}
	for relative := range assets {
		if err := os.WriteFile(filepath.Join(root, relative), []byte("changed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "skills", "new-skill"), []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.restore(); err != nil {
		t.Fatal(err)
	}
	for relative, body := range assets {
		got, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil || string(got) != body {
			t.Fatalf("%s: %q %v", relative, got, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "skills", "new-skill")); !os.IsNotExist(err) {
		t.Fatalf("new asset remains: %v", err)
	}
}

func TestUpgradeRejectsUnsafeParentsAndFeatures(t *testing.T) {
	root := t.TempDir()
	if err := checkUpgradeParents(root, filepath.Join(t.TempDir(), "asset")); err == nil {
		t.Fatal("outside root accepted")
	}
	blocker := filepath.Join(root, "skills")
	if err := os.WriteFile(blocker, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := checkUpgradeParents(root, filepath.Join(blocker, "skill")); err == nil {
		t.Fatal("file parent accepted")
	}
	source := writeUpgradeFixture(t, root, "1.0.0", "skill", "")
	pkg, err := LoadPackage(source)
	if err != nil {
		t.Fatal(err)
	}
	pkg.Features.Skills[0].Name = "../escape"
	if err := validateUpgradePackage(pkg, source); err == nil {
		t.Fatal("unsafe name accepted")
	}
	if _, err := upgradeAssetPaths(root, pkg); err == nil {
		t.Fatal("unsafe canonical name accepted")
	}
	pkg.Features.Skills[0].Name = "skill"
	pkg.Features.KB = append(pkg.Features.KB, KBFeature{Path: "../escape"})
	if err := validateUpgradePackage(pkg, source); err == nil {
		t.Fatal("unsafe KB accepted")
	}
}

func TestUpgradeFailedRecoveryRetainsLock(t *testing.T) {
	root := t.TempDir()
	pm := NewPackageManager(root)
	if err := pm.InstallPackage(writeUpgradeFixture(t, root, "1.0.0", "old-skill", ""), nil); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0700); err != nil {
		t.Fatal(err)
	}
	source := writeUpgradeFixture(t, root, "2.0.0", "new-skill", "  post_install:\n    - script: fail.sh\n")
	script := "#!/bin/sh\nmv ../../skills ../../saved-skills\nprintf blocked > ../../skills\nexit 7\n"
	if err := os.WriteFile(filepath.Join(source, "fail.sh"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	err := pm.UpgradePackage("safe-pkg", source)
	if err == nil || !strings.Contains(err.Error(), "rollback failed") {
		t.Fatalf("missing recovery failure: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".package-upgrade.lock")); err != nil {
		t.Fatalf("recovery lock missing: %v", err)
	}
	if err := pm.UpgradePackage("safe-pkg", source); err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("unsafe retry accepted: %v", err)
	}
}
