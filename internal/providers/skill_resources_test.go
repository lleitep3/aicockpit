package providers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillResourcesReachCodex(t *testing.T) {
	canonical, destination := t.TempDir(), t.TempDir()
	root := filepath.Join(canonical, "skills", "ux-repro")
	writeCodexTestFile(t, filepath.Join(root, "SKILL.md"), "---\nname: ux-repro\ndescription: Test resources\n---\nRead references/nested/states.md.\n")
	resources := map[string]string{
		"references/nested/states.md": "States: default, focus, error.\n",
		"agents/openai.yaml":          "interface:\n  display_name: UX Test\n",
		"scripts/nested/check.sh":     "#!/bin/sh\nexit 0\n",
		"assets/sample.bin":           string([]byte{0, 255, 17, 128}),
	}
	for name, content := range resources {
		writeCodexTestFile(t, filepath.Join(root, name), content)
	}
	provider := &Provider{Enabled: true, Name: "Codex", Features: map[string]*FeatureConfig{
		"skills": {Enabled: true, Path: ".agents/skills"},
	}}
	manager := NewProviderManager(&ProvidersConfig{Providers: map[string]*Provider{"codex": provider}})
	for attempt := 0; attempt < 2; attempt++ {
		if err := manager.Deploy("codex", canonical, destination); err != nil {
			t.Fatal(err)
		}
		for name, want := range resources {
			got, err := os.ReadFile(filepath.Join(destination, ".agents/skills/ux-repro", name))
			if err != nil {
				t.Errorf("resource %s missing: %v", name, err)
				continue
			}
			if string(got) != want {
				t.Errorf("resource %s changed during deploy", name)
			}
		}
	}
}

func TestSkillResourcesRejectSymlinks(t *testing.T) {
	for _, targetDir := range []bool{false, true} {
		t.Run(map[bool]string{false: "file", true: "directory"}[targetDir], func(t *testing.T) {
			canonical := t.TempDir()
			root := filepath.Join(canonical, "skills", "unsafe")
			writeCodexTestFile(t, filepath.Join(root, "SKILL.md"), "---\nname: unsafe\ndescription: Test\n---\nTest\n")
			outside := t.TempDir()
			target := outside
			if !targetDir {
				target = filepath.Join(outside, "private.md")
				writeCodexTestFile(t, target, "outside content")
			}
			if err := os.MkdirAll(filepath.Join(root, "references"), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, filepath.Join(root, "references", "linked")); err != nil {
				t.Skipf("symlinks unavailable: %v", err)
			}
			_, err := parseSkills(canonical)
			if err == nil || !strings.Contains(err.Error(), "symlink") {
				t.Fatalf("expected explicit symlink error, got %v", err)
			}
		})
	}
}
