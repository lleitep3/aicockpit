package intent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi/llm"
	"github.com/lleitep3/aicockpit/internal/copi/router"
)

// ActiveIntent represents a goal session currently tracked by Copi.
type ActiveIntent struct {
	ID        string                `json:"id"`
	Category  router.IntentCategory `json:"category"`
	Goal      string                `json:"goal"`
	Target    string                `json:"target,omitempty"`
	StartedAt time.Time             `json:"started_at"`
	UpdatedAt time.Time             `json:"updated_at"`
	Messages  []llm.ChatTurn        `json:"messages,omitempty"`
}

// Manager tracks and persists the active intent file (~/.cockpit/current_intent.json).
type Manager struct {
	mu       sync.RWMutex
	filePath string
	active   *ActiveIntent
}

// NewManager creates a Manager instance using the given sandbox directory.
func NewManager(sandboxDir string) *Manager {
	filePath := filepath.Join(sandboxDir, "current_intent.json")
	m := &Manager{
		filePath: filePath,
	}
	_ = m.load()
	return m
}

// Get returns the current active intent, or nil if none is set.
func (m *Manager) Get() *ActiveIntent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.active
}

// SetOrUpdate sets a new intent or updates the existing one.
func (m *Manager) SetOrUpdate(category router.IntentCategory, goal, target string) *ActiveIntent {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UTC()
	if m.active == nil || m.active.Category != category {
		m.active = &ActiveIntent{
			ID:        fmt.Sprintf("intent_%d", now.UnixNano()),
			Category:  category,
			Goal:      goal,
			Target:    target,
			StartedAt: now,
			UpdatedAt: now,
		}
	} else {
		m.active.Goal = goal
		if target != "" {
			m.active.Target = target
		}
		m.active.UpdatedAt = now
	}

	_ = m.save()
	return m.active
}

// RecordTurn appends user & assistant messages to the active intent context.
func (m *Manager) RecordTurn(userMessage, assistantResponse string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.active == nil {
		m.active = &ActiveIntent{
			ID:        fmt.Sprintf("intent_%d", time.Now().Unix()),
			Category:  router.IntentComplexChat,
			Goal:      userMessage,
			StartedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
	}

	m.active.Messages = append(m.active.Messages,
		llm.ChatTurn{Role: "user", Content: userMessage},
		llm.ChatTurn{Role: "assistant", Content: assistantResponse},
	)

	// Keep window bounded to last 6 turns of this intent
	if len(m.active.Messages) > 6 {
		m.active.Messages = m.active.Messages[len(m.active.Messages)-6:]
	}

	m.active.UpdatedAt = time.Now().UTC()
	_ = m.save()
}

// History returns messages recorded specifically for this active intent.
func (m *Manager) History() []llm.ChatTurn {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.active == nil {
		return nil
	}
	out := make([]llm.ChatTurn, len(m.active.Messages))
	copy(out, m.active.Messages)
	return out
}

// Clear archives or clears the active intent.
func (m *Manager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.active = nil
	_ = os.Remove(m.filePath)
}

func (m *Manager) load() error {
	data, err := os.ReadFile(m.filePath)
	if err != nil {
		return err
	}
	var intent ActiveIntent
	if err := json.Unmarshal(data, &intent); err != nil {
		return err
	}
	m.active = &intent
	return nil
}

func (m *Manager) save() error {
	if m.active == nil {
		_ = os.Remove(m.filePath)
		return nil
	}
	data, err := json.MarshalIndent(m.active, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.filePath, data, 0644)
}
