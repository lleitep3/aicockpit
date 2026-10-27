package kb

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func importSource(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.md")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestImportLifecycle(t *testing.T) {
	root := filepath.Join(t.TempDir(), "kb")
	now := time.Date(2026, 9, 18, 15, 30, 0, 0, time.FixedZone("BRT", -3*3600))
	source := importSource(t, "---\ntitle: Vault diagnosis\ntags: [vault]\nauthor: User\ncustom: retained\n---\n# Vault diagnosis\nSandbox blocks keyring access.\n")
	opts := ImportOptions{Root: root, Context: "aicockpit"}
	doc, err := importFileAt(source, opts, now)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Path != filepath.Join("2026-09-18", "2026-09-18_153000-vault-diagnosis.md") {
		t.Fatal(doc.Path)
	}
	loaded, err := NewFileRepository(root).LoadDocument(doc.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Metadata.Created.Equal(now) || loaded.Metadata.Context != "aicockpit" || loaded.Metadata.Extra["custom"] != "retained" {
		t.Fatalf("metadata: %+v", loaded.Metadata)
	}
	// A second import at the same timestamp must never overwrite the first.
	collision, err := importFileAt(source, opts, now)
	if err != nil || collision.ID != doc.ID+"-2" {
		t.Fatalf("collision: %+v, %v", collision, err)
	}
	updatedSource := importSource(t, "# Revised diagnosis\nVerified in a separate process.\n")
	opts.UpdateID, opts.Context = doc.ID, ""
	later := now.AddDate(0, 0, 2)
	updated, err := importFileAt(updatedSource, opts, later)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Path != doc.Path || !updated.Metadata.Created.Equal(now) || !updated.Metadata.Modified.Equal(later) {
		t.Fatalf("identity changed: %+v", updated)
	}
	if updated.Metadata.Context != "aicockpit" || updated.Metadata.Author != "User" || updated.Metadata.Tags[0] != "vault" || updated.Metadata.Extra["custom"] != "retained" {
		t.Fatalf("metadata lost: %+v", updated.Metadata)
	}
	loaded, err = NewFileRepository(root).LoadDocument(doc.Path)
	if err != nil || !strings.Contains(loaded.Content, "separate process") {
		t.Fatalf("update not persisted: %v", err)
	}
	// Re-adding the managed file itself retains the original identity too.
	opts.UpdateID = ""
	same, err := importFileAt(filepath.Join(root, doc.Path), opts, later)
	if err != nil || same.Path != doc.Path {
		t.Fatalf("managed file duplicated: %+v %v", same, err)
	}
}

func TestImportValidation(t *testing.T) {
	for _, tc := range []struct{ name, body, context string }{
		{"empty", "  \n", "project"},
		{"metadata-only", "---\ntitle: Empty\n---\n", "project"},
		{"bad-yaml", "---\ntitle: [\n---\nBody", "project"},
		{"unclosed", "---\ntitle: Missing\nBody", "project"},
		{"binary", "# Test\n\x00", "project"},
		{"invalid-utf8", "# Test\n\xff", "project"},
		{"no-context", "# Test\nContent", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ImportFile(importSource(t, tc.body), ImportOptions{Root: t.TempDir(), Context: tc.context})
			if err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	for _, source := range []string{"missing.md", "file.txt", t.TempDir()} {
		if _, err := ImportFile(source, ImportOptions{Root: t.TempDir(), Context: "test"}); err == nil {
			t.Fatal("invalid source accepted")
		}
	}
}

func TestImportUpdateLookup(t *testing.T) {
	source := importSource(t, "# Update\nBody")
	for _, id := range []string{"missing", "../escape", `..\escape`} {
		if _, err := ImportFile(source, ImportOptions{Root: t.TempDir(), Context: "test", UpdateID: id}); err == nil {
			t.Fatalf("accepted %q", id)
		}
	}
	root := t.TempDir()
	for _, dir := range []string{"one", "two"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "same.md"), []byte("# Original"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := ImportFile(source, ImportOptions{Root: root, Context: "test", UpdateID: "same"}); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("ambiguous update: %v", err)
	}
}

func TestImportMetadataContextAndSlug(t *testing.T) {
	source := importSource(t, "---\ncontext: cockpit\n---\n# Configuração / paths\nBody")
	doc, err := ImportFile(source, ImportOptions{Root: t.TempDir()})
	if err != nil || !strings.HasSuffix(doc.Path, "configuração-paths.md") {
		t.Fatalf("import: %+v %v", doc, err)
	}
	for _, tc := range []struct{ input, want string }{{"../", "document"}, {"A / B", "a-b"}, {" Olá! ", "olá"}} {
		if got := documentSlug(tc.input); got != tc.want {
			t.Errorf("slug %q: %q", tc.input, got)
		}
	}
}

func TestImportFilesystemFailuresPreserveOriginal(t *testing.T) {
	source := importSource(t, "# New knowledge\nBody")
	if _, err := ImportFile(source, ImportOptions{}); err == nil {
		t.Fatal("empty root accepted")
	}
	if _, err := ImportFile(source, ImportOptions{Root: source, Context: "test"}); err == nil {
		t.Fatal("file destination accepted")
	}
	for _, tc := range []struct {
		name, body string
		mode       os.FileMode
	}{
		{"invalid-existing", "---\ntitle: [\n---\nOriginal", 0o755},
		{"read-only-directory", "# Original\nKeep this content", 0o555},
		{"missing-context", "# Original\nKeep this content", 0o755},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "existing.md")
			if err := os.WriteFile(path, []byte(tc.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(root, tc.mode); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
			context := "test"
			if tc.name == "missing-context" {
				context = ""
			}
			_, err := ImportFile(source, ImportOptions{Root: root, Context: context, UpdateID: "existing"})
			if err == nil {
				t.Fatal("expected failure")
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != tc.body {
				t.Fatalf("original changed: %q %v", got, err)
			}
		})
	}
	root := t.TempDir()
	if err := os.Symlink(source, filepath.Join(root, "external.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportFile(source, ImportOptions{Root: root, Context: "test", UpdateID: "external"}); err == nil {
		t.Fatal("updated external symlink")
	}
	now := time.Now()
	if err := os.WriteFile(filepath.Join(root, now.Format("2006-01-02")), []byte("obstruction"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := importFileAt(source, ImportOptions{Root: root, Context: "test"}, now); err == nil {
		t.Fatal("date path obstruction ignored")
	}
}

func TestImportWriteAndLegacyUpdate(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "closed")
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writeImport(file, "data"); err == nil {
		t.Fatal("closed write accepted")
	}
	root := t.TempDir()
	path := filepath.Join(root, "legacy.md")
	if err := os.WriteFile(path, []byte("# Legacy\nOriginal"), 0o600); err != nil {
		t.Fatal(err)
	}
	doc, err := ImportFile(importSource(t, "# Revised\nBody"), ImportOptions{Root: root, Context: "test", UpdateID: "legacy"})
	if err != nil || doc.Metadata.Created.IsZero() {
		t.Fatalf("legacy: %+v %v", doc, err)
	}
	if got := documentSlug(strings.Repeat("a", 200)); len(got) > 120 {
		t.Fatal("slug not bounded")
	}
}

func TestImportPermissionErrors(t *testing.T) {
	source := importSource(t, "# Permission test\nOriginal content")
	now := time.Now()
	for _, tc := range []struct {
		name              string
		rootMode, dayMode os.FileMode
	}{
		{"inaccessible-root", 0o000, 0o755},
		{"unwritable-day", 0o755, 0o555},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			day := filepath.Join(root, now.Format("2006-01-02"))
			if err := os.Mkdir(day, tc.dayMode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(root, tc.rootMode); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = os.Chmod(root, 0o755); _ = os.Chmod(day, 0o755) })
			if _, err := importFileAt(source, ImportOptions{Root: root, Context: "test"}, now); err == nil {
				t.Fatal("permission failure ignored")
			}
		})
	}
	root := t.TempDir()
	path := filepath.Join(root, "private.md")
	if err := os.WriteFile(path, []byte("# Keep me"), 0o000); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportFile(source, ImportOptions{Root: root, Context: "test", UpdateID: "private"}); err == nil {
		t.Fatal("unreadable original overwritten")
	}
}
