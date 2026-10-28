// Package bot provides the multi-bot engine. This file holds the service that
// creates/manages bots, their per-bot folders, profiles, activities, routines
// and observers, then hands their conversational loop to the personalagent
// layer for AI turns.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

// Service is the entry point for bot management. It owns a store for bots and
// a base directory where each bot gets its own folder. It embeds
// personalagent.Service so the conversational loop can reuse the turn layer.
type Service struct {
	*personalagent.Service
	botStore *Store
	base     string
	mu       sync.Mutex
}

// botStoreAdapter adapts the bot Store to the personalagent botStore interface.
type botStoreAdapter struct {
	*Store
}

func (a *botStoreAdapter) Save(ctx context.Context, bot any) error {
	return a.Store.Save(ctx, bot.(*Bot))
}
func (a *botStoreAdapter) Load(ctx context.Context, id string) (any, error) {
	return a.Store.Load(ctx, id)
}
func (a *botStoreAdapter) List(ctx context.Context) ([]any, error) {
	items, err := a.Store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(items))
	for i, it := range items {
		out[i] = it
	}
	return out, nil
}
func (a *botStoreAdapter) Delete(ctx context.Context, id string) error {
	return a.Store.Delete(ctx, id)
}

// NewService creates a bot service rooted at base (usually ~/.cockpit/bots).
func NewService(pa *personalagent.Service, store *Store, base string) *Service {
	if base == "" {
		base = filepath.Join(cockpitDir(), "bots")
	}
	bs := &Service{Service: pa, botStore: store, base: base}
	// Register the adapter so the personalagent bot methods resolve against
	// this store.
	pa.SetBotStore(&botStoreAdapter{Store: store})
	return bs
}

func cockpitDir() string {
	if d := os.Getenv("COCKPIT_DATA_DIR"); d != "" {
		return d
	}
	return filepath.Join(os.Getenv("HOME"), ".cockpit")
}

// CreateBot persists a new bot, creates its folder and returns the rendered bot.
func (s *Service) CreateBot(ctx context.Context, profile *BotProfile) (*Bot, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}
	if profile.Name == "" {
		return nil, errors.New("profile name is required")
	}
	if profile.Handle == "" {
		return nil, errors.New("profile handle is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	bot := &Bot{
		ID:           profile.Handle,
		Name:         profile.Name,
		Handle:       profile.Handle,
		Role:         profile.Role,
		Objective:    profile.Objective,
		Persona:      profile.Persona,
		Skills:       profile.Skills,
		Instructions: profile.Instructions,
		Selection:    Selection{ProviderModel: ProviderModel{ProviderID: profile.ProviderID, ModelID: profile.ModelID}, ReasoningEffort: profile.ReasoningEffort},
		WorkspaceDir: profile.WorkspaceDir,
		Status:       BotStatusInactive,
		CreatedAt:    now,
		UpdatedAt:    now,
		Activities:   []BotTask{},
		Routines:     []BotRoutine{},
		Observers:    []BotObserver{},
		LastPrompt:   "",
		LastResponse: nil,
		LastError:    "",
	}

	if bot.WorkspaceDir == "" {
		dir, err := MakeBotFolder(bot.ID, s.base)
		if err != nil {
			return nil, err
		}
		bot.WorkspaceDir = dir
	}

	if err := s.botStore.Save(ctx, bot); err != nil {
		return nil, err
	}
	if err := s.SaveProfile(ctx, bot.ID, profile); err != nil {
		// Persist the bot without the profile so the CLI can retry.
		return bot, err
	}
	return bot, nil
}

// GetBot returns one bot by id.
func (s *Service) GetBot(ctx context.Context, id string) (*Bot, error) {
	return s.botStore.Load(ctx, id)
}

// ListBots returns every bot, most recently updated first.
func (s *Service) ListBots(ctx context.Context) ([]*Bot, error) {
	return s.botStore.List(ctx)
}

// UpdateBot applies the supplied fields to a bot and persists the change.
func (s *Service) UpdateBot(ctx context.Context, id string, profile *BotProfile) (*Bot, error) {
	if profile == nil {
		return nil, errors.New("profile is nil")
	}
	existing, err := s.botStore.Load(ctx, id)
	if err != nil {
		return nil, err
	}

	if profile.Name != "" {
		existing.Name = profile.Name
	}
	if profile.Handle != "" {
		existing.Handle = profile.Handle
	}
	if profile.Role != "" {
		existing.Role = profile.Role
	}
	if profile.Objective != "" {
		existing.Objective = profile.Objective
	}
	if profile.Persona != "" {
		existing.Persona = profile.Persona
	}
	if profile.Skills != nil {
		existing.Skills = profile.Skills
	}
	if profile.Instructions != "" {
		existing.Instructions = profile.Instructions
	}
	if profile.ProviderID != "" {
		existing.Selection.ProviderID = profile.ProviderID
	}
	if profile.ModelID != "" {
		existing.Selection.ModelID = profile.ModelID
	}
	if profile.ReasoningEffort != "" {
		existing.Selection.ReasoningEffort = profile.ReasoningEffort
	}

	if profile.WorkspaceDir != "" {
		existing.WorkspaceDir = profile.WorkspaceDir
	}
	existing.LastRunAt = nil
	existing.LastError = ""
	existing.LastResponse = nil
	existing.LastPrompt = ""
	existing.UpdatedAt = time.Now().UTC()

	if err := s.botStore.Save(ctx, existing); err != nil {
		return nil, err
	}
	if err := s.SaveProfile(ctx, existing.ID, profile); err != nil {
		return existing, err
	}
	return existing, nil
}

// DeleteBot removes the bot record and its folder from disk.
func (s *Service) DeleteBot(ctx context.Context, id string) error {
	if err := s.botStore.Delete(ctx, id); err != nil {
		return err
	}
	dir := BotWorkspace(id, s.base)
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("remove bot folder %s: %w", dir, err)
	}
	return nil
}

// SaveProfile persists a bot's profile to its folder, creating the folder if needed.
func (s *Service) SaveProfile(ctx context.Context, botID string, profile *BotProfile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}
	if profile.Name == "" {
		return errors.New("profile name is required")
	}
	if err := os.MkdirAll(BotWorkspace(botID, s.base), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}
	path := ProfileFile(botID, s.base)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	return nil
}

// LoadProfile returns a bot's persisted profile, creating an empty one if missing.
func (s *Service) LoadProfile(ctx context.Context, botID string) (*BotProfile, error) {
	path := ProfileFile(botID, s.base)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &BotProfile{
				Name:         botID,
				Handle:       botID,
				CreatedAt:    time.Now().UTC(),
				UpdatedAt:    time.Now().UTC(),
				WorkspaceDir: BotWorkspace(botID, s.base),
			}, nil
		}
		return nil, fmt.Errorf("read profile: %w", err)
	}
	var profile BotProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, fmt.Errorf("decode profile: %w", err)
	}
	return &profile, nil
}

// Activity returns a bot's persisted activities, creating an empty slice if missing.
func (s *Service) Activities(ctx context.Context, botID string) ([]*BotTask, error) {
	path := filepath.Join(BotWorkspace(botID, s.base), "activities.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []*BotTask{}, nil
		}
		return nil, fmt.Errorf("read activities: %w", err)
	}
	if len(data) == 0 {
		return []*BotTask{}, nil
	}
	var tasks []*BotTask
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("decode activities: %w", err)
	}
	return tasks, nil
}

// SaveActivity persists a single bot activity.
func (s *Service) SaveActivity(ctx context.Context, botID string, task *BotTask) error {
	if task == nil {
		return errors.New("task is nil")
	}
	tasks, err := s.Activities(ctx, botID)
	if err != nil {
		return err
	}
	for i, t := range tasks {
		if t.ID == task.ID {
			tasks[i] = task
			break
		}
	}
	return s.saveSlice(ctx, botID, tasks)
}

func (s *Service) saveSlice(ctx context.Context, botID string, tasks []*BotTask) error {
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("encode activities: %w", err)
	}
	path := filepath.Join(BotWorkspace(botID, s.base), "activities.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write activities: %w", err)
	}
	return nil
}
