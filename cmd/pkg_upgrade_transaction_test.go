package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/services"
)

type upgradeTestService struct {
	services.PackageService
	path, candidate string
	deployErr       error
	upgraded        bool
}

func (s *upgradeTestService) GetPackage(name string, regs []packages.RegistryConfig) (*packages.PackageIndexEntry, string, error) {
	entry, reg, err := s.PackageService.GetPackage(name, regs)
	if err == nil {
		entry.Path = "packages/hello-pkg"
	}
	return entry, reg, err
}
func (s *upgradeTestService) GetPackageFromCache(reg, path string) (string, error) {
	s.path = path
	return s.candidate, nil
}
func (s *upgradeTestService) UpgradePackage(name, path string) error {
	s.upgraded = true
	return s.PackageService.UpgradePackage(name, path)
}
func (s *upgradeTestService) TriggerDeploy(string) error { return s.deployErr }

func TestUpgradeIndexPathAndDeployOutcome(t *testing.T) {
	for _, scenario := range []string{"nested-path", "deploy-failure", "index-mismatch"} {
		t.Run(scenario, func(t *testing.T) {
			root := setupLocalGitRegistry(t)
			base, cfg := testPkgArgs(t)
			source := filepath.Join(root, "registry-work", "hello-pkg")
			if err := base.InstallPackage(source, nil); err != nil {
				t.Fatal(err)
			}
			installed := base.GetPackageInstallPath("hello-pkg")
			pkg, err := packages.LoadPackage(installed)
			if err != nil {
				t.Fatal(err)
			}
			pkg.Version = "1.0.0"
			if err := packages.SavePackage(installed, pkg); err != nil {
				t.Fatal(err)
			}
			svc := &upgradeTestService{PackageService: base, candidate: source}
			if scenario == "deploy-failure" {
				svc.deployErr = errors.New("provider unavailable")
			}
			if scenario == "index-mismatch" {
				data, err := os.ReadFile(filepath.Join(source, "cockpit-package.yml"))
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(source, "cockpit-package.yml"), []byte(strings.Replace(string(data), "2.0.0", "3.0.0", 1)), 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := NewPkgUpgradeCommand(svc, cfg)
			cmd.SetArgs([]string{"hello-pkg"})
			err = cmd.Execute()
			if svc.path != "packages/hello-pkg" {
				t.Fatalf("wrong path %q", svc.path)
			}
			switch scenario {
			case "nested-path":
				if err != nil {
					t.Fatal(err)
				}
			case "deploy-failure":
				if err == nil || !strings.Contains(err.Error(), "retry cockpit deploy") {
					t.Fatalf("missing partial failure: %v", err)
				}
			case "index-mismatch":
				if err == nil || svc.upgraded {
					t.Fatalf("mismatched candidate activated: %v", err)
				}
			}
		})
	}
}
