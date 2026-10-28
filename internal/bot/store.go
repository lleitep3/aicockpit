// Package bot persists bots and their per-bot folders on disk. It follows the
// personalagent model: one JSON file per entity (a bot), a directory of them.
package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Store keeps the bot registry in one directory. Each Bot is a separate JSON
// file named after its ID, so listing, reloading and deleting are cheap.
type Store struct {
	directory string
	mu        sync.RWMutex
}

func validBotID(id string) bool {
	if id == "" || len(id) > 64 {
		return false
	}
	for _, char := range id {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '-' || char == '_' {
			continue
		}
		return false
	}
	return true
}

// NewStore creates a bot store under directory (created if missing).
func NewStore(directory string) (*Store, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create bot store directory: %w", err)
	}
	if err := os.Chmod(directory, 0o700); err != nil {
		return nil, fmt.Errorf("restrict bot store permissions: %w", err)
	}
	return &Store{directory: directory}, nil
}

func (s *Store) directoryPath() string {
	return s.directory
}

// Save writes the bot to store/bot_<id>.json.
func (s *Store) Save(ctx context.Context, bot *Bot) error {
	if bot == nil {
		return errors.New("bot is nil")
	}
	if bot.ID == "" {
		return errors.New("bot id is empty")
	}
	if !validBotID(bot.ID) {
		return fmt.Errorf("invalid bot id %q", bot.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.filePath(bot.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create bot directory: %w", err)
	}
	data, err := json.MarshalIndent(bot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode bot %s: %w", bot.ID, err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write bot %s: %w", bot.ID, err)
	}
	return nil
}

// Load returns a bot by id, or ErrNotFound when it does not exist.
func (s *Store) Load(ctx context.Context, id string) (*Bot, error) {
	if !validBotID(id) {
		return nil, fmt.Errorf("invalid bot id %q", id)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	bot, err := s.loadInternal(id)
	if err != nil {
		return nil, err
	}
	if bot.ID != id {
		return nil, fmt.Errorf("bot %s has invalid identity", id)
	}
	return bot, nil
}

func (s *Store) loadInternal(id string) (*Bot, error) {
	path := s.filePath(id)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("bot %s not found", id)
		}
		return nil, fmt.Errorf("read bot %s: %w", id, err)
	}
	var bot Bot
	if err := json.Unmarshal(data, &bot); err != nil {
		return nil, fmt.Errorf("decode bot %s: %w", id, err)
	}
	if bot.ID != id {
		return nil, fmt.Errorf("bot %s has invalid identity", id)
	}
	return &bot, nil
}

// List returns every bot, most recently updated first.
func (s *Store) List(ctx context.Context) ([]*Bot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read bot store: %w", err)
	}
	var bots []*Bot
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !validBotID(id) {
			continue
		}
		bot, err := s.loadInternal(id)
		if err != nil {
			continue
		}
		bots = append(bots, bot)
	}
	sort.Slice(bots, func(i, j int) bool {
		return bots[i].UpdatedAt.After(bots[j].UpdatedAt)
	})
	return bots, nil
}

// Delete removes a bot file and returns errors as ErrNotFound when missing.
func (s *Store) Delete(ctx context.Context, id string) error {
	if !validBotID(id) {
		return fmt.Errorf("invalid bot id %q", id)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.filePath(id)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("bot %s not found", id)
		}
		return fmt.Errorf("delete bot %s: %w", id, err)
	}
	return nil
}

func (s *Store) filePath(id string) string {
	return filepath.Join(s.directory, id+".json")
}

// BotWorkspace returns the per-bot folder for a bot (idempotent).
func BotWorkspace(id string, baseDir string) string {
	return filepath.Join(baseDir, id)
}

// MakeBotFolder creates the per-bot directory (profile, state, logs) and returns
// the path. It is safe to call repeatedly.
func MakeBotFolder(id, baseDir string) (string, error) {
	if !validBotID(id) {
		return "", fmt.Errorf("invalid bot id %q", id)
	}
	dir := BotWorkspace(id, baseDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create bot folder %s: %w", dir, err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return "", fmt.Errorf("restrict bot folder permissions: %w", err)
	}
	return dir, nil
}

// ProfileFile returns the path of a bot's persisted profile.
func ProfileFile(botID, baseDir string) string {
	return filepath.Join(BotWorkspace(botID, baseDir), "profile.json")
}

// StateFile returns the path of a bot's runtime state file.
func StateFile(botID, baseDir string) string {
	return filepath.Join(BotWorkspace(botID, baseDir), "state.json")
}

// LogDir returns the per-bot log directory.
func LogDir(botID, baseDir string) string {
	return filepath.Join(BotWorkspace(botID, baseDir), "logs")
}

// ProfilePath returns the full path for a bot's profile, plus the directory we
// would create it in.
func ProfilePath(botID, baseDir string) (string, error) {
	dir, err := MakeBotFolder(botID, baseDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profile.json"), nil
}

// StatePath returns the full path for a bot's runtime state.
func StatePath(botID, baseDir string) (string, error) {
	dir, err := MakeBotFolder(botID, baseDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

// LogPath returns the path for a bot's log file for the given subsystem.
func LogPath(botID, baseDir, subsystem string) (string, error) {
	dir, err := MakeBotFolder(botID, baseDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o700); err != nil {
		return "", fmt.Errorf("create bot logs directory: %w", err)
	}
	name := subsystem
	if name == "" {
		name = "bot"
	}
	return filepath.Join(dir, "logs", name+".log"), nil
}
