package copi

import "time"

// SessionStatus represents the execution state of a tracked conversation session.
type SessionStatus string

const (
	StatusIdle    SessionStatus = "idle"
	StatusRunning SessionStatus = "running"
	StatusPending SessionStatus = "pending_decision"
	StatusDone    SessionStatus = "completed"
	StatusFailed  SessionStatus = "failed"
)

// PendingAction represents a high-impact action waiting for human approval.
type PendingAction struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	TurnID         string    `json:"turn_id"`
	Description    string    `json:"description"`
	PayloadHash    string    `json:"payload_hash"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}

// SessionSnapshot contains the minimal, token-free state of a conversation.
type SessionSnapshot struct {
	ID             string          `json:"id"`
	Title          string          `json:"title"`
	ProjectID      string          `json:"project_id,omitempty"`
	Status         SessionStatus   `json:"status"`
	MessageCount   int             `json:"message_count"`
	TurnCount      int             `json:"turn_count"`
	LatestTurnID   string          `json:"latest_turn_id,omitempty"`
	InputTokens    int64           `json:"input_tokens"`
	OutputTokens   int64           `json:"output_tokens"`
	Cost           float64         `json:"cost"`
	PendingActions []PendingAction `json:"pending_actions,omitempty"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// SessionDiff describes what changed deterministically between two snapshots.
type SessionDiff struct {
	SessionID       string          `json:"session_id"`
	SessionTitle    string          `json:"session_title"`
	PreviousState   SessionStatus   `json:"previous_state"`
	CurrentState    SessionStatus   `json:"current_state"`
	NewMessages     int             `json:"new_messages"`
	NewTokens       int64           `json:"new_tokens"`
	NewPending      []PendingAction `json:"new_pending,omitempty"`
	ResolvedPending []string        `json:"resolved_pending,omitempty"`
	Summary         string          `json:"summary"`
}

// UpdateReport aggregates diffs across all monitored sessions (0 tokens).
type UpdateReport struct {
	Timestamp      time.Time     `json:"timestamp"`
	TotalSessions  int           `json:"total_sessions"`
	ActiveSessions int           `json:"active_sessions"`
	PendingCount   int           `json:"pending_count"`
	Diffs          []SessionDiff `json:"diffs"`
	HasChanges     bool          `json:"has_changes"`
	FormattedText  string        `json:"formatted_text"`
}
