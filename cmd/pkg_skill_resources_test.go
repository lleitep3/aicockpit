package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/packages"
	"github.com/lleitep3/aicockpit/internal/providers"
	"github.com/lleitep3/aicockpit/internal/services"
)

// Exercise the real install and provider pipeline without re-executing the test
// binary or touching the user's provider configuration.
type localSkillService struct {
	services.PackageService
	canonical, destination string
}

func (s localSkillService) TriggerDeploy(_ string) error {
	manager := providers.NewProviderManager(&providers.ProvidersConfig{Providers: map[string]*providers.Provider{
		"codex": {Enabled: true, Name: "Codex", Features: map[string]*providers.FeatureConfig{
			"skills": {Enabled: true, Path: ".agents/skills"},
		}},
	}})
	return manager.Deploy("codex", s.canonical, s.destination)
}

func writeLocalSkillFixture(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestInstallNestedLocalSkillToCodex(t *testing.T) {
	registry, canonical, destination := t.TempDir(), t.TempDir(), t.TempDir()
	// The package directory deliberately differs from its manifest name.
	packageRoot := filepath.Join(registry, "packages", "ux-bundle")
	manifest := `name: ux-fixture
version: 0.1.0
description: Local install regression
author: Test
license: MIT
requirements:
  cockpit: "0.2.0"
features:
  skills:
    - name: ux-fixture
      path: skills/ux-fixture
installation:
  supported_providers: [codex]
  provider_features:
    codex: [skills]
  method: copy
`
	writeLocalSkillFixture(t, packageRoot, "cockpit-package.yml", manifest)
	resources := map[string]string{
		"SKILL.md":             "---\nname: ux-fixture\ndescription: Local fixture\n---\nRead references/states.md.\n",
		"references/states.md": "Default, focus, error.\n",
		"agents/openai.yaml":   "interface:\n  display_name: UX fixture\n",
	}
	for name, content := range resources {
		writeLocalSkillFixture(t, packageRoot, "skills/ux-fixture/"+name, content)
	}
	writeLocalSkillFixture(t, registry, "package-index.yaml", `packages:
  - name: ux-fixture
    version: 0.1.0
    path: packages/ux-bundle
    url: https://github.com/user/placeholder
`)
	for _, args := range [][]string{
		{"init", "-b", "feature/local-test"},
		{"add", "."},
		{"-c", "user.name=Test", "-c", "user.email=test@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", "fixture"},
	} {
		command := exec.Command("git", append([]string{"-C", registry}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git fixture: %v: %s", err, output)
		}
	}
	svc := localSkillService{services.NewPackageService(canonical, nil), canonical, destination}
	cfg := &config.Config{PackageRegistries: []packages.RegistryConfig{
		{Name: "unselected", URL: "file:///nonexistent-unused-registry", Branch: "main", Enabled: true, Priority: 1},
		{Name: "local-test", URL: "file://" + registry, Branch: "feature/local-test", Enabled: true, Priority: 10},
	}}
	for _, cacheState := range []string{"cold", "warm"} {
		t.Run(cacheState, func(t *testing.T) {
			command := NewPkgInstallCommand(svc, cfg)
			args := []string{"ux-fixture", "--source", "local-test"}
			if cacheState == "warm" {
				args = append(args, "--force")
			}
			command.SetArgs(args)
			if err := command.Execute(); err != nil {
				t.Fatal(err)
			}
			for _, root := range []string{
				filepath.Join(canonical, "packages/ux-fixture/skills/ux-fixture"),
				filepath.Join(canonical, "skills/ux-fixture"),
				filepath.Join(destination, ".agents/skills/ux-fixture"),
			} {
				for name, want := range resources {
					got, err := os.ReadFile(filepath.Join(root, name))
					if err != nil {
						t.Errorf("missing %s: %v", filepath.Join(root, name), err)
						continue
					}
					if name == "SKILL.md" {
						if !strings.Contains(string(got), "Read references/states.md.") {
							t.Errorf("skill body missing at %s", root)
						}
					} else if string(got) != want {
						t.Errorf("resource %s changed at %s", name, root)
					}
				}
			}
		})
	}
}
