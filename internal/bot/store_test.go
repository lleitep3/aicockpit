package bot

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	t.Cleanup(func() {
		_ = removeAllEmpty(s.directoryPath())
	})
	return s
}

func timeNow() time.Time {
	return time.Now().UTC()
}

func removeAllEmpty(path string) error {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() {
			_ = removeAllEmpty(filepath.Join(path, e.Name()))
		} else {
			_ = os.Remove(filepath.Join(path, e.Name()))
		}
	}
	return os.Remove(path)
}

func TestStoreSaveAndLoad(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bot := &Bot{
		ID:        "test-bot",
		Name:      "Test Bot",
		Handle:    "test",
		Status:    BotStatusActive,
		CreatedAt: timeNow(),
		UpdatedAt: timeNow(),
	}

	if err := s.Save(ctx, bot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := s.Load(ctx, "test-bot")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Name != "Test Bot" {
		t.Errorf("Load() Name = %q, want %q", loaded.Name, "Test Bot")
	}
	if loaded.Status != BotStatusActive {
		t.Errorf("Load() Status = %v, want %v", loaded.Status, BotStatusActive)
	}
}

func TestStoreLoadNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	_, err := s.Load(ctx, "nope")
	if err == nil {
		t.Fatalf("Load() expected error, got nil")
	}
}

func TestStoreListEmpty(t *testing.T) {
	s := newTestStore(t)
	bots, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(bots) != 0 {
		t.Fatalf("List() = %d bots, want 0", len(bots))
	}
}

func TestStoreListWithBots(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bots := []*Bot{
		{ID: "a", Name: "A", CreatedAt: timeNow(), UpdatedAt: timeNow()},
		{ID: "b", Name: "B", CreatedAt: timeNow(), UpdatedAt: timeNow()},
	}
	for _, b := range bots {
		if err := s.Save(ctx, b); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
	}

	loaded, err := s.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("List() = %d bots, want 2", len(loaded))
	}
}

func TestStoreDelete(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()

	bot := &Bot{ID: "del-bot", Name: "Del"}
	if err := s.Save(ctx, bot); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := s.Delete(ctx, "del-bot"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := s.Load(ctx, "del-bot"); err == nil {
		t.Fatalf("Load() expected error after delete, got nil")
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	if err := s.Delete(ctx, "nope"); err == nil {
		t.Fatalf("Delete() expected error, got nil")
	}
}

func TestBotWorkspace(t *testing.T) {
	base := t.TempDir()
	id := "bot-123"
	dir := BotWorkspace(id, base)
	if dir != filepath.Join(base, id) {
		t.Errorf("BotWorkspace() = %q, want %q", dir, filepath.Join(base, id))
	}
}

func TestMakeBotFolder(t *testing.T) {
	base := t.TempDir()
	dir, err := MakeBotFolder("fbot", base)
	if err != nil {
		t.Fatalf("MakeBotFolder() error = %v", err)
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Fatalf("bot folder not created: %v", err)
	}
	// Safe to call again.
	dir2, err := MakeBotFolder("fbot", base)
	if err != nil {
		t.Fatalf("MakeBotFolder() second call error = %v", err)
	}
	if dir2 != dir {
		t.Errorf("second call returned different dir: %q vs %q", dir2, dir)
	}
}
