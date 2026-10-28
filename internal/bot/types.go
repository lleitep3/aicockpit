// Package bot is the multi-bot engine of Cockpit: it owns bot identities,
// personas, persistent profiles, per-bot workspace folders, activities, routines
// (crons) and observers (email / PR / web), plus a conversational loop that
// reuses the personalagent layer for AI turns.
package bot

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// ProviderModel is the minimal provider/model selection a bot can be built on.
// It is intentionally small so a bot carries only the data it needs.
type ProviderModel struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
}

// Selection groups a provider/model with the routing knobs a bot uses.
type Selection struct {
	ProviderModel
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

// Bot is a personal specialist agent. It owns a folder in the host filesystem,
// a persona, an objective and a live loop that acts on events and messages.
type Bot struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Handle       string        `json:"handle"`
	Role         string        `json:"role"`
	Objective    string        `json:"objective"`
	Persona      string        `json:"persona"`
	Skills       []string      `json:"skills"`
	Instructions string        `json:"instructions"`
	Selection    Selection     `json:"selection"`
	WorkspaceDir string        `json:"workspace_dir"`
	Status       BotStatus     `json:"status"`
	LastRunAt    *time.Time    `json:"last_run_at,omitempty"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Activities   []BotTask     `json:"activities,omitempty"`
	Routines     []BotRoutine  `json:"routines,omitempty"`
	Observers    []BotObserver `json:"observers,omitempty"`
	LastPrompt   string        `json:"last_prompt,omitempty"`
	LastResponse *BotResponse  `json:"last_response,omitempty"`
	LastError    string        `json:"last_error,omitempty"`
}

// BotStatus defines the lifecycle states of a bot.
type BotStatus string

const (
	BotStatusInactive BotStatus = "inactive"
	BotStatusActive   BotStatus = "active"
	BotStatusPaused   BotStatus = "paused"
	BotStatusError    BotStatus = "error"
)

// BotProfile is the persisted, user-editable description of a bot.
type BotProfile struct {
	Name            string    `json:"name"`
	Handle          string    `json:"handle"`
	Role            string    `json:"role"`
	Objective       string    `json:"objective"`
	Persona         string    `json:"persona"`
	Skills          []string  `json:"skills"`
	Instructions    string    `json:"instructions"`
	ProviderID      string    `json:"provider_id,omitempty"`
	ModelID         string    `json:"model_id,omitempty"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
	WorkspaceDir    string    `json:"workspace_dir"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BotTask is a unit of work owned by a bot inside its own folder.
type BotTask struct {
	ID        string        `json:"id"`
	Title     string        `json:"title"`
	Status    BotTaskStatus `json:"status"`
	Owner     string        `json:"owner,omitempty"`
	DueAt     *time.Time    `json:"due_at,omitempty"`
	Notes     string        `json:"notes,omitempty"`
	BotID     string        `json:"bot_id"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// BotTaskStatus is the lifecycle of a bot activity.
type BotTaskStatus string

const (
	BotTaskStatusOpen       BotTaskStatus = "open"
	BotTaskStatusInProgress BotTaskStatus = "in_progress"
	BotTaskStatusDone       BotTaskStatus = "done"
	BotTaskStatusBlocked    BotTaskStatus = "blocked"
)

// BotRoutine is a time/event-triggered prompt the bot runs on its own.
type BotRoutine struct {
	ID          string           `json:"id"`
	BotID       string           `json:"bot_id"`
	Name        string           `json:"name"`
	Schedule    string           `json:"schedule,omitempty"`
	EventFilter string           `json:"event_filter,omitempty"`
	Prompt      string           `json:"prompt"`
	Status      BotRoutineStatus `json:"status"`
	LastRunAt   *time.Time       `json:"last_run_at,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// BotRoutineStatus is the lifecycle of a routine.
type BotRoutineStatus string

const (
	BotRoutineStatusActive   BotRoutineStatus = "active"
	BotRoutineStatusPaused   BotRoutineStatus = "paused"
	BotRoutineStatusDisabled BotRoutineStatus = "disabled"
)

// BotObserver watches an external source and turns matches into bot events.
type BotObserver struct {
	ID        string            `json:"id"`
	BotID     string            `json:"bot_id"`
	Kind      BotObserverKind   `json:"kind"`
	Config    ObserverConfig    `json:"config"`
	Status    BotObserverStatus `json:"status"`
	LastError string            `json:"last_error,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// BotObserverKind is the kind of source the observer watches.
type BotObserverKind string

const (
	BotObserverKindEmail BotObserverKind = "email"
	BotObserverKindPr    BotObserverKind = "pr"
	BotObserverKindWeb   BotObserverKind = "web"
)

// BotObserverStatus is the lifecycle state of an observer.
type BotObserverStatus string

const (
	BotObserverStatusActive   BotObserverStatus = "active"
	BotObserverStatusInactive BotObserverStatus = "inactive"
	BotObserverStatusPaused   BotObserverStatus = "paused"
	BotObserverStatusDisabled BotObserverStatus = "disabled"
	BotObserverStatusError    BotObserverStatus = "error"
)

// ObserverConfig is a JSON blob holding the arbitrary config each observer kind needs.
type ObserverConfig struct {
	// JSON is serialized as-is; the hosting service interprets it per kind.
}

// BotEvent is a discrete occurrence the bot can react to.
type BotEvent struct {
	ID         string         `json:"id"`
	BotID      string         `json:"bot_id"`
	Source     BotEventSource `json:"source"`
	Kind       string         `json:"kind"`
	Payload    string         `json:"payload,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
	HandledAt  *time.Time     `json:"handled_at,omitempty"`
	Done       bool           `json:"done"`
}

// BotEventSource identifies where a bot event came from.
type BotEventSource string

const (
	BotEventSourceEmail   BotEventSource = "email"
	BotEventSourcePr      BotEventSource = "pr"
	BotEventSourceRoutine BotEventSource = "routine"
	BotEventSourceInput   BotEventSource = "input"
)

// BotResponse is the last AI turn the bot ran.
type BotResponse struct {
	Text       string    `json:"text"`
	ModelID    string    `json:"model_id"`
	ProviderID string    `json:"provider_id"`
	Usage      *Usage    `json:"usage,omitempty"`
	Error      string    `json:"error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// Usage mirrors the token/usage shape used by the provider adapters.
type Usage struct {
	InputTokens  int64   `json:"input_tokens,omitempty"`
	OutputTokens int64   `json:"output_tokens,omitempty"`
	Cost         float64 `json:"cost,omitempty"`
}

// NewID generates a bot identifier.
func NewID() (string, error) {
	b, err := newIDBytes()
	if err != nil {
		return "", err
	}
	return "bot_" + hex.EncodeToString(b[:]), nil
}

// newIDBytes returns a 16-byte random slice for store keys.
func newIDBytes() ([16]byte, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return value, err
	}
	return value, nil
}
