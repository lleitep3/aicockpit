package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillsImportCommandImportsLocalSkill(t *testing.T) {
	tempHome := t.TempDir()
	setSkillsTestHome(t, tempHome)
	source := filepath.Join(tempHome, "copilot-skills", "review")
	writeSkillsCommandFile(t, source, "SKILL.md", "---\nname: review\ndescription: Review changes\n---\nReview the diff.\n")
	writeSkillsCommandFile(t, source, "references/checklist.md", "Check tests and docs.\n")

	command := NewSkillsCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"import", filepath.Dir(source)})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	imported := filepath.Join(tempHome, ".cockpit", "skills", "review", "references", "checklist.md")
	content, err := os.ReadFile(imported)
	if err != nil {
		t.Fatalf("read imported resource: %v", err)
	}
	if string(content) != "Check tests and docs.\n" {
		t.Errorf("imported resource = %q", content)
	}
	if !strings.Contains(output.String(), "Imported: review") {
		t.Errorf("command output = %q, want import summary", output.String())
	}
}

func TestSkillsImportCommandDryRunJSON(t *testing.T) {
	tempHome := t.TempDir()
	setSkillsTestHome(t, tempHome)
	source := filepath.Join(tempHome, "one-skill")
	writeSkillsCommandFile(t, source, "SKILL.md", "---\nname: one-skill\ndescription: One skill\n---\nDo one thing.\n")

	command := NewSkillsCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"import", source, "--dry-run", "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output.String(), `"status": "would-import"`) {
		t.Errorf("command output = %q, want planned import in JSON", output.String())
	}
	if _, err := os.Stat(filepath.Join(tempHome, ".cockpit", "skills")); !os.IsNotExist(err) {
		t.Fatalf("dry-run created canonical skills directory; stat error = %v", err)
	}
}

func TestSkillsImportCommandSavesConflictsForReview(t *testing.T) {
	tempHome := t.TempDir()
	setSkillsTestHome(t, tempHome)
	source := filepath.Join(tempHome, "copilot-skills", "review")
	writeSkillsCommandFile(t, source, "SKILL.md", "---\nname: review\ndescription: Local version\n---\nLocal body.\n")
	writeSkillsCommandFile(t, filepath.Join(tempHome, ".cockpit", "skills", "review"), "SKILL.md", "---\nname: review\ndescription: Canonical version\n---\nCanonical body.\n")
	review := filepath.Join(tempHome, ".cockpit", "skill-reviews", "copilot-local")

	command := NewSkillsCommand()
	var output bytes.Buffer
	command.SetOut(&output)
	command.SetErr(&output)
	command.SetArgs([]string{"import", filepath.Dir(source), "--save-conflicts", review, "--json"})
	if err := command.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(output.String(), `"status": "saved-conflict"`) {
		t.Errorf("command output = %q, want saved conflict in JSON", output.String())
	}
	saved := filepath.Join(review, "review", "SKILL.md")
	content, err := os.ReadFile(saved)
	if err != nil {
		t.Fatalf("read saved conflict: %v", err)
	}
	if !strings.Contains(string(content), "Local body.") {
		t.Errorf("saved conflict = %q, want local version", content)
	}
}

func setSkillsTestHome(t *testing.T, home string) {
	t.Helper()
	previous, wasSet := os.LookupEnv("HOME")
	if err := os.Setenv("HOME", home); err != nil {
		t.Fatalf("set HOME: %v", err)
	}
	t.Cleanup(func() {
		if wasSet {
			_ = os.Setenv("HOME", previous)
		} else {
			_ = os.Unsetenv("HOME")
		}
	})
}

func writeSkillsCommandFile(t *testing.T, directory, relativePath, content string) {
	t.Helper()
	path := filepath.Join(directory, relativePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
