package bot

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

func newTestService(t *testing.T) (*Service, *Store) {
	t.Helper()
	root := t.TempDir()
	paStore, err := personalagent.NewStore(filepath.Join(root, "pa"))
	if err != nil {
		t.Fatalf("personalagent NewStore() error = %v", err)
	}
	pa, err := personalagent.NewService(paStore, personalagent.NewProviderRegistry(), root)
	if err != nil {
		t.Fatalf("personalagent NewService() error = %v", err)
	}
	store, err := NewStore(filepath.Join(root, "bots"))
	if err != nil {
		t.Fatalf("bot NewStore() error = %v", err)
	}
	svc := NewService(pa, store, root)
	t.Cleanup(func() {
		_ = os.RemoveAll(root)
	})
	return svc, store
}

func TestCreateAndGetBot(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	profile := &BotProfile{
		Name:            "Dev Bot",
		Handle:          "dev",
		Role:            "Development",
		Objective:       "Build and maintain Go applications.",
		Persona:         "Pragmatic engineer",
		Skills:          []string{"go", "test", "lint"},
		Instructions:    "Always run tests before committing.",
		ProviderID:      "codex",
		ModelID:         "codex-1",
		ReasoningEffort: "",
	}
	bot, err := svc.CreateBot(ctx, profile)
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}
	if bot.Name != "Dev Bot" {
		t.Errorf("Name = %q, want %q", bot.Name, "Dev Bot")
	}
	if bot.Handle != "dev" {
		t.Errorf("Handle = %q, want %q", bot.Handle, "dev")
	}
	if bot.WorkspaceDir == "" {
		t.Errorf("WorkspaceDir is empty")
	}
	if _, err := svc.GetBot(ctx, bot.ID); err != nil {
		t.Fatalf("GetBot() error = %v", err)
	}
}

func TestGetNonExistentBot(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.GetBot(ctx, "missing"); err == nil {
		t.Fatalf("GetBot() expected error, got nil")
	}
}

func TestCreateBotEmptyProfile(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()
	if _, err := svc.CreateBot(ctx, nil); err == nil {
		t.Fatalf("CreateBot(nil) expected error, got nil")
	}
}

func TestListBots(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	profile := &BotProfile{Name: "One", Handle: "one"}
	if _, err := svc.CreateBot(ctx, profile); err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}
	profile2 := &BotProfile{Name: "Two", Handle: "two"}
	if _, err := svc.CreateBot(ctx, profile2); err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	bots, err := svc.ListBots(ctx)
	if err != nil {
		t.Fatalf("ListBots() error = %v", err)
	}
	if len(bots) != 2 {
		t.Fatalf("ListBots() = %d, want 2", len(bots))
	}
}

func TestSaveAndLoadProfile(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "P", Handle: "p"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}
	profile := &BotProfile{
		Name:            "Updated",
		Handle:          "p",
		Role:            "Tester",
		Objective:       "Write tests",
		Instructions:    "Be thorough",
		ProviderID:      "codex",
		ModelID:         "codex-1",
		ReasoningEffort: "",
	}
	if err := svc.SaveProfile(ctx, bot.ID, profile); err != nil {
		t.Fatalf("SaveProfile() error = %v", err)
	}

	loaded, err := svc.LoadProfile(ctx, bot.ID)
	if err != nil {
		t.Fatalf("LoadProfile() error = %v", err)
	}
	if loaded.Name != "Updated" {
		t.Errorf("Name = %q, want %q", loaded.Name, "Updated")
	}
	if loaded.Role != "Tester" {
		t.Errorf("Role = %q, want %q", loaded.Role, "Tester")
	}
}

func TestActivities(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "T", Handle: "t"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	tasks, err := svc.Activities(ctx, bot.ID)
	if err != nil {
		t.Fatalf("Activities() error = %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("Activities() = %d, want 0", len(tasks))
	}
}

func TestDeleteBot(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "X", Handle: "x"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}
	if err := svc.DeleteBot(ctx, bot.ID); err != nil {
		t.Fatalf("DeleteBot() error = %v", err)
	}
	if _, err := svc.GetBot(ctx, bot.ID); err == nil {
		t.Fatalf("GetBot() after delete expected error, got nil")
	}
}
