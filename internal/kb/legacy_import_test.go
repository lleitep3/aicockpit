package kb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetadataLegacyDates(t *testing.T) {
	for _, value := range []string{"2026-06-20", `"2026-06-20"`, `"2026-06-20T00:00:00Z"`} {
		doc, err := ParseDocument("legacy", "legacy.md", "---\ncreated: "+value+"\nmodified: "+value+"\ncustom: retained\n---\nBody")
		if err != nil {
			t.Fatal(err)
		}
		want := time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC)
		if !doc.Metadata.Created.Equal(want) || !doc.Metadata.Modified.Equal(want) || doc.Metadata.Extra["custom"] != "retained" {
			t.Fatalf("metadata: %+v", doc.Metadata)
		}
	}
	if _, err := ParseDocument("bad", "bad.md", "---\ncreated: yesterday\n---\nBody"); err == nil {
		t.Fatal("invalid date accepted")
	}
}

func TestIndexFindsLegacyDocumentsSkipsGeneratedAssets(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"docs", ".git", ".agents", ".codex"} {
		if err := os.Mkdir(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, dir, "lesson.md"), []byte("---\ntitle: Legacy lesson\ncreated: \"2026-06-20\"\n---\nlegacydateverification"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	manager := NewManager([]string{root}, filepath.Join(t.TempDir(), "index.json"))
	if err := manager.RebuildIndex(); err != nil {
		t.Fatal(err)
	}
	result, err := manager.Search("legacydateverification")
	if err != nil || result.Total != 1 || result.Results[0].Path != filepath.Join("docs", "lesson.md") {
		t.Fatalf("search: %+v %v", result, err)
	}
}
