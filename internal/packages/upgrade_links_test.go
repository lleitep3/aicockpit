package packages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpgradeRuntimeLinksAndRollback(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "rollback"}[fail], func(t *testing.T) {
			root := t.TempDir()
			pm := NewPackageManager(root)
			if err := pm.InstallPackage(writeUpgradeFixture(t, root, "1.0.0", "skill", ""), nil); err != nil {
				t.Fatal(err)
			}
			installed := filepath.Join(root, "packages", "safe-pkg")
			runtimeDir := filepath.Join(installed, "node_modules", ".bin")
			if err := os.MkdirAll(runtimeDir, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(installed, "node_modules", "tool.js"), []byte("original"), 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink("../tool.js", filepath.Join(runtimeDir, "tool")); err != nil {
				t.Skip(err)
			}
			source := writeUpgradeFixture(t, root, "2.0.0", "skill", "  post_install:\n    - script: install.sh\n")
			script := "mkdir -p node_modules/.bin; printf replacement > node_modules/tool.js; ln -s ../tool.js node_modules/.bin/tool\n"
			if fail {
				script += "exit 7\n"
			}
			if err := os.WriteFile(filepath.Join(source, "install.sh"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			err := pm.UpgradePackage("safe-pkg", source)
			if (err != nil) != fail {
				t.Fatalf("unexpected result: %v", err)
			}
			link, err := os.Readlink(filepath.Join(runtimeDir, "tool"))
			if err != nil || link != "../tool.js" {
				t.Fatalf("link not preserved: %q %v", link, err)
			}
			expected := "replacement"
			if fail {
				expected = "original"
			}
			data, err := os.ReadFile(filepath.Join(runtimeDir, "tool"))
			if err != nil || string(data) != expected {
				t.Fatalf("runtime: %q %v", data, err)
			}
			if fail && !strings.Contains(pmVersion(t, pm), "1.0.0") {
				t.Fatal("old version lost")
			}
		})
	}
}

func pmVersion(t *testing.T, pm *PackageManager) string {
	t.Helper()
	pkg, err := pm.GetInstalledPackage("safe-pkg")
	if err != nil {
		t.Fatal(err)
	}
	return pkg.Version
}

func TestUpgradeLinkBoundaries(t *testing.T) {
	for _, kind := range []string{"internal", "external", "absolute", "dangling", "cycle", "exported", "asset-parent", "manifest"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			source := writeUpgradeFixture(t, root, "2.0.0", "skill", "")
			pkg, err := LoadPackage(source)
			if err != nil {
				t.Fatal(err)
			}
			target, link := "skills/test.md", filepath.Join(source, "link")
			switch kind {
			case "external":
				target = "../../outside"
			case "absolute":
				target = filepath.Join(source, "skills", "test.md")
			case "dangling":
				target = "missing"
			case "cycle":
				target = "link"
			case "exported":
				pkg.Features.Skills[0].Path = "link"
			case "asset-parent":
				target = "skills"
				pkg.Features.Skills[0].Path = "link/test.md"
			case "manifest":
				link = filepath.Join(source, "cockpit-package.yml")
				if err := os.Rename(link, filepath.Join(source, "manifest.yml")); err != nil {
					t.Fatal(err)
				}
				target = "manifest.yml"
			}
			if err := os.Symlink(target, link); err != nil {
				t.Skip(err)
			}
			err = validateUpgradePackage(pkg, source)
			if (err == nil) != (kind == "internal") {
				t.Fatalf("unexpected validation: %v", err)
			}
		})
	}
}

func TestHookFileValidation(t *testing.T) {
	for _, kind := range []string{"missing", "directory", "link", "parent-link", "valid"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			script := "hook.sh"
			if err := os.WriteFile(filepath.Join(root, "actual.sh"), []byte("exit 0\n"), 0600); err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "directory":
				if err := os.Mkdir(filepath.Join(root, script), 0700); err != nil {
					t.Fatal(err)
				}
			case "link":
				if err := os.Symlink("actual.sh", filepath.Join(root, script)); err != nil {
					t.Skip(err)
				}
			case "parent-link":
				outside := t.TempDir()
				if err := os.Symlink(outside, filepath.Join(root, "scripts")); err != nil {
					t.Skip(err)
				}
				script = "scripts/hook.sh"
			case "valid":
				script = "actual.sh"
			}
			_, err := validateHookFile(root, script)
			if (err == nil) != (kind == "valid") {
				t.Fatalf("unexpected validation: %v", err)
			}
		})
	}
}

func TestPackageRejectsMissingLifecycleHooks(t *testing.T) {
	for _, phase := range []string{"pre_install", "post_install", "pre_uninstall", "post_uninstall"} {
		t.Run(phase, func(t *testing.T) {
			source := writeUpgradeFixture(t, t.TempDir(), "1.0.0", "skill", "  "+phase+":\n    - script: missing.sh\n")
			pkg, err := LoadPackage(source)
			if err != nil {
				t.Fatal(err)
			}
			if err := pkg.Validate(source); err == nil {
				t.Fatal("missing hook accepted")
			}
		})
	}
}

func TestUpgradeLinkAndCopyErrors(t *testing.T) {
	root := t.TempDir()
	if err := validateInternalUpgradeLink(root, filepath.Join(root, "missing")); err == nil {
		t.Fatal("missing link accepted")
	}
	if err := copyUpgradeTree(filepath.Join(root, "missing"), filepath.Join(root, "dest")); err == nil {
		t.Fatal("missing tree accepted")
	}
	if err := os.Symlink(".", filepath.Join(root, "cycle-root")); err != nil {
		t.Skip(err)
	}
	if err := validateInternalUpgradeLink(root, filepath.Join(root, "cycle-root")); err == nil {
		t.Fatal("root link accepted")
	}
}
