package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/packages"
)

type batchOperations struct {
	manager        *packages.PackageManager
	calls, deploys int
	failName       string
	deployErr      error
}

func (o *batchOperations) UpgradePackage(name, path string) error {
	o.calls++
	if name == o.failName {
		return errors.New("package hook failed")
	}
	return o.manager.UpgradePackage(name, path)
}
func (o *batchOperations) TriggerDeploy(string) error { o.deploys++; return o.deployErr }
func batchPackage(t *testing.T, dir, name, version string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "skills"), 0700); err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf("name: %s\nversion: %s\ndescription: test\nauthor: test\nlicense: MIT\nrequirements:\n  cockpit: '>=0.1.0'\nfeatures:\n  skills:\n    - name: %s-skill\n      path: skills/test.md\ninstallation:\n  supported_providers: [all]\n  provider_features:\n    all: [skills]\n", name, version, name)
	for path, content := range map[string]string{"cockpit-package.yml": body, "skills/test.md": version} {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
func batchGit(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
}
func batchFixture(t *testing.T) (*PackageBatch, *batchOperations, string) {
	t.Helper()
	root, remote := t.TempDir(), t.TempDir()
	manager := packages.NewPackageManager(root)
	for _, name := range []string{"alpha", "beta"} {
		source := filepath.Join(t.TempDir(), name)
		batchPackage(t, source, name, "1.0.0")
		if err := manager.InstallPackage(source, nil); err != nil {
			t.Fatal(err)
		}
		batchPackage(t, filepath.Join(remote, "packages", name), name, "2.0.0")
	}
	index := "packages:\n  - {name: alpha, version: 2.0.0, path: packages/alpha}\n  - {name: beta, version: 2.0.0, path: packages/beta}\n"
	if err := os.WriteFile(filepath.Join(remote, "package-index.yaml"), []byte(index), 0600); err != nil {
		t.Fatal(err)
	}
	batchGit(t, remote, "init", "-b", "main")
	batchGit(t, remote, "add", ".")
	batchGit(t, remote, "commit", "-m", "registry")
	batch := NewPackageBatch(root, []packages.RegistryConfig{{Name: "registry", URL: remote, Branch: "main", Enabled: true}})
	operations := &batchOperations{manager: manager}
	batch.operations = operations
	return batch, operations, remote
}
func TestPackageBatchCheckApplyAndRepeat(t *testing.T) {
	batch, ops, _ := batchFixture(t)
	check := batch.Run(true)
	if check.Failed() || len(check.Outcomes) != 2 || ops.calls != 0 || ops.deploys != 0 {
		t.Fatalf("check mutated or failed: %+v", check)
	}
	for _, outcome := range check.Outcomes {
		if outcome.Status != "available" {
			t.Fatal(outcome)
		}
	}
	result := batch.Run(false)
	if result.Failed() || ops.calls != 2 || ops.deploys != 1 {
		t.Fatalf("apply: %+v", result)
	}
	again := batch.Run(false)
	if again.Failed() || ops.calls != 2 || ops.deploys != 1 {
		t.Fatalf("repeat changed installed state: %+v", again)
	}
	for _, outcome := range again.Outcomes {
		if outcome.Status != "current" {
			t.Fatal(outcome)
		}
	}
	snapshots, _ := filepath.Glob(filepath.Join(batch.root, "cache", "registries", ".refresh-*"))
	if len(snapshots) != 3 {
		t.Fatalf("expected one refresh per run: %v", snapshots)
	}
}
func TestPackageBatchPartialFailures(t *testing.T) {
	for _, kind := range []string{"package", "deploy", "invalid-manifest", "identity", "downgrade", "bad-version", "disabled", "missing-registry", "missing-package", "candidate-mismatch", "candidate-invalid", "candidate-missing"} {
		t.Run(kind, func(t *testing.T) {
			batch, ops, remote := batchFixture(t)
			installed := filepath.Join(batch.root, "packages", "alpha")
			switch kind {
			case "package":
				ops.failName = "alpha"
			case "deploy":
				ops.deployErr = errors.New("provider unavailable")
			case "invalid-manifest":
				if err := os.WriteFile(filepath.Join(installed, "cockpit-package.yml"), []byte("name: [broken"), 0600); err != nil {
					t.Fatal(err)
				}
			case "identity":
				batchPackage(t, installed, "other", "1.0.0")
			case "downgrade":
				batchPackage(t, installed, "alpha", "3.0.0")
			case "bad-version":
				batchPackage(t, installed, "alpha", "invalid")
			case "disabled":
				batch.registries[0].Enabled = false
			case "missing-registry":
				batch.registries[0].URL = filepath.Join(t.TempDir(), "absent")
			case "missing-package":
				batchPackage(t, filepath.Join(batch.root, "packages", "local"), "local", "1.0.0")
			case "candidate-mismatch":
				batchPackage(t, filepath.Join(remote, "packages", "alpha"), "alpha", "3.0.0")
			case "candidate-invalid":
				if err := os.Rename(filepath.Join(remote, "packages", "alpha", "skills"), filepath.Join(remote, "packages", "alpha", "hidden")); err != nil {
					t.Fatal(err)
				}
			case "candidate-missing":
				if err := os.Rename(filepath.Join(remote, "packages", "alpha", "cockpit-package.yml"), filepath.Join(remote, "packages", "alpha", "hidden.yml")); err != nil {
					t.Fatal(err)
				}
			}
			if kind == "candidate-mismatch" || kind == "candidate-invalid" || kind == "candidate-missing" {
				batchGit(t, remote, "add", ".")
				batchGit(t, remote, "commit", "-m", "candidate change")
			}
			report := batch.Run(false)
			if !report.Failed() {
				t.Fatalf("failure hidden: %+v", report)
			}
			if kind != "disabled" && kind != "missing-registry" {
				found := false
				for _, outcome := range report.Outcomes {
					if outcome.Name == "beta" && outcome.Status == "updated" {
						found = true
					}
				}
				if !found {
					t.Fatalf("independent package not updated: %+v", report)
				}
			}
		})
	}
}
func TestPackageBatchInventoryFailure(t *testing.T) {
	root := t.TempDir()
	batch := NewPackageBatch(root, nil)
	if report := batch.Run(true); report.Failed() || len(report.Outcomes) != 0 {
		t.Fatal(report)
	}
	if err := os.WriteFile(filepath.Join(root, "packages"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if report := batch.Run(true); !report.Failed() {
		t.Fatal("inventory failure hidden")
	}
}

func TestPackageBatchDoesNotIgnoreSymbolicInstall(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "packages"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(root, "packages", "linked")); err != nil {
		t.Skip(err)
	}
	report := NewPackageBatch(root, nil).Run(true)
	if !report.Failed() {
		t.Fatal("symbolic install silently omitted")
	}
}
