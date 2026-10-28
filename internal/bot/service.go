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
	"strings"
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
	service *Service
}

func (a *botStoreAdapter) Save(ctx context.Context, value any) error {
	bot, ok := value.(*Bot)
	if !ok {
		return fmt.Errorf("unsupported bot store value %T", value)
	}
	return a.Store.Save(ctx, bot)
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
	if a.service != nil {
		return a.service.DeleteBot(ctx, id)
	}
	return a.Store.Delete(ctx, id)
}

func (a *botStoreAdapter) CreateBot(ctx context.Context, value any) (any, error) {
	profile, err := decodeBotPayload[BotProfile](value)
	if err != nil {
		return nil, err
	}
	return a.service.CreateBot(ctx, profile)
}

func (a *botStoreAdapter) UpdateBot(ctx context.Context, id string, value any) (any, error) {
	profile, err := decodeBotPayload[BotProfile](value)
	if err != nil {
		return nil, err
	}
	return a.service.UpdateBot(ctx, id, profile)
}

func (a *botStoreAdapter) SendMessage(ctx context.Context, botID, content string, attachments []personalagent.Attachment) (string, error) {
	if a.service == nil {
		return "", errors.New("bot service is not registered")
	}
	return a.service.SendMessage(ctx, botID, content, attachments)
}

func (a *botStoreAdapter) ListObservers(ctx context.Context, botID string) ([]any, error) {
	observers, err := a.service.ListObservers(ctx, botID)
	if err != nil {
		return nil, err
	}
	result := make([]any, len(observers))
	for i, observer := range observers {
		result[i] = observer
	}
	return result, nil
}

func (a *botStoreAdapter) CreateObserver(ctx context.Context, botID string, kind, config any) (any, error) {
	var request struct {
		Kind   BotObserverKind `json:"kind"`
		Config ObserverConfig  `json:"config"`
	}
	if value, ok := kind.(string); ok {
		request.Kind = BotObserverKind(value)
		if config != nil {
			decoded, err := decodeBotPayload[ObserverConfig](config)
			if err != nil {
				return nil, err
			}
			request.Config = *decoded
		}
	} else if err := decodeBotPayloadInto(kind, &request); err != nil {
		return nil, err
	}
	if request.Kind == "" {
		return nil, errors.New("observer kind is required")
	}
	return a.service.CreateObserver(ctx, botID, request.Kind, request.Config)
}

func (a *botStoreAdapter) UpdateObserver(ctx context.Context, botID, id string, value any) (any, error) {
	var request struct {
		Status BotObserverStatus `json:"status"`
	}
	if status, ok := value.(string); ok {
		request.Status = BotObserverStatus(status)
	} else if err := decodeBotPayloadInto(value, &request); err != nil {
		return nil, err
	}
	return a.service.UpdateObserver(ctx, botID, id, request.Status)
}

func (a *botStoreAdapter) DeleteObserver(ctx context.Context, botID, id string) error {
	return a.service.DeleteObserver(ctx, botID, id)
}

func decodeBotPayload[T any](value any) (*T, error) {
	var result T
	if err := decodeBotPayloadInto(value, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func decodeBotPayloadInto(value any, target any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode bot request: %w", err)
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode bot request: %w", err)
	}
	return nil
}

// NewService creates a bot service rooted at base (usually ~/.cockpit/bots).
func NewService(pa *personalagent.Service, store *Store, base string) *Service {
	if base == "" {
		base = filepath.Join(cockpitDir(), "bots")
	}
	bs := &Service{Service: pa, botStore: store, base: base}
	// Register the adapter so the personalagent bot methods resolve against
	// this store.
	pa.SetBotStore(&botStoreAdapter{Store: store, service: bs})
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
	if !validBotID(profile.Handle) {
		return nil, fmt.Errorf("invalid bot handle %q", profile.Handle)
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

	expectedWorkspace := filepath.Clean(BotWorkspace(bot.ID, s.base))
	if profile.WorkspaceDir != "" && filepath.Clean(profile.WorkspaceDir) != expectedWorkspace {
		return nil, errors.New("bot workspace must be managed by Cockpit")
	}
	dir, err := MakeBotFolder(bot.ID, s.base)
	if err != nil {
		return nil, err
	}
	bot.WorkspaceDir = dir

	if err := s.botStore.Save(ctx, bot); err != nil {
		return nil, err
	}
	savedProfile := *profile
	savedProfile.WorkspaceDir = bot.WorkspaceDir
	savedProfile.CreatedAt = now
	savedProfile.UpdatedAt = now
	if err := s.SaveProfile(ctx, bot.ID, &savedProfile); err != nil {
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
	if profile.Handle != "" && !validBotID(profile.Handle) {
		return nil, fmt.Errorf("invalid bot handle %q", profile.Handle)
	}
	existing, err := s.botStore.Load(ctx, id)
	if err != nil {
		return nil, err
	}
	if profile.WorkspaceDir != "" && filepath.Clean(profile.WorkspaceDir) != filepath.Clean(existing.WorkspaceDir) {
		return nil, errors.New("bot workspace cannot be changed")
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

	existing.LastRunAt = nil
	existing.LastError = ""
	existing.LastResponse = nil
	existing.LastPrompt = ""
	existing.UpdatedAt = time.Now().UTC()

	if err := s.botStore.Save(ctx, existing); err != nil {
		return nil, err
	}
	updatedProfile := &BotProfile{
		Name:            existing.Name,
		Handle:          existing.Handle,
		Role:            existing.Role,
		Objective:       existing.Objective,
		Persona:         existing.Persona,
		Skills:          existing.Skills,
		Instructions:    existing.Instructions,
		ProviderID:      existing.Selection.ProviderID,
		ModelID:         existing.Selection.ModelID,
		ReasoningEffort: existing.Selection.ReasoningEffort,
		WorkspaceDir:    existing.WorkspaceDir,
		CreatedAt:       existing.CreatedAt,
		UpdatedAt:       existing.UpdatedAt,
	}
	if err := s.SaveProfile(ctx, existing.ID, updatedProfile); err != nil {
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

func (s *Service) SendMessage(ctx context.Context, botID, content string, attachments []personalagent.Attachment) (string, error) {
	bot, err := s.GetBot(ctx, botID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(content) == "" && len(attachments) == 0 {
		return "", errors.New("message cannot be empty")
	}
	conversation, err := s.Service.BotConversation(botID)
	if err != nil {
		if !errors.Is(err, personalagent.ErrNotFound) {
			return "", err
		}
		selection := personalagent.Selection{
			ProviderID:      bot.Selection.ProviderID,
			ModelID:         bot.Selection.ModelID,
			ReasoningEffort: bot.Selection.ReasoningEffort,
			ApprovalPolicy:  "on-request",
			RoutingMode:     "manual",
		}
		created, err := personalagent.CreateConversationForBot(
			ctx, s.Service, bot.ID, bot.Handle, bot.WorkspaceDir, selection,
		)
		if err != nil {
			return "", fmt.Errorf("create bot conversation: %w", err)
		}
		conversation = *created
	}
	prompt := botPrompt(bot, content)
	turnID, err := s.Service.StartTurn(conversation.ID, prompt, attachments)
	if err != nil {
		return "", err
	}
	bot.LastPrompt = prompt
	bot.LastError = ""
	bot.UpdatedAt = time.Now().UTC()
	if err := s.botStore.Save(ctx, bot); err != nil {
		return "", fmt.Errorf("save bot turn state: %w", err)
	}
	return turnID, nil
}

func botPrompt(bot *Bot, content string) string {
	sections := make([]string, 0, 5)
	if bot.Name != "" {
		sections = append(sections, "Bot: "+bot.Name)
	}
	if bot.Role != "" {
		sections = append(sections, "Role: "+bot.Role)
	}
	if bot.Objective != "" {
		sections = append(sections, "Objective: "+bot.Objective)
	}
	if bot.Persona != "" {
		sections = append(sections, "Persona: "+bot.Persona)
	}
	if bot.Instructions != "" {
		sections = append(sections, "Instructions: "+bot.Instructions)
	}
	sections = append(sections, "User request:\n"+content)
	return strings.Join(sections, "\n\n")
}

// SaveProfile persists a bot's profile to its folder, creating the folder if needed.
func (s *Service) SaveProfile(ctx context.Context, botID string, profile *BotProfile) error {
	if profile == nil {
		return errors.New("profile is nil")
	}
	if !validBotID(botID) {
		return fmt.Errorf("invalid bot id %q", botID)
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
	if !validBotID(botID) {
		return nil, fmt.Errorf("invalid bot id %q", botID)
	}
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
