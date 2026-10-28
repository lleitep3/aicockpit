package personalagent

import (
	"context"
	"encoding/json"
	"time"
)

const CurrentSchemaVersion = 3

const (
	RoutingModeManual    = "manual"
	RoutingModeAutomatic = "automatic"
)

type Provider struct {
	Quota        ProviderQuota `json:"quota"`
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Detected     bool          `json:"detected"`
	Supported    bool          `json:"supported"`
	Executable   string        `json:"executable,omitempty"`
	Version      string        `json:"version,omitempty"`
	Status       string        `json:"status"`
	Capabilities []string      `json:"capabilities"`
}

type Model struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	Default          bool     `json:"default"`
	Selectable       bool     `json:"selectable"`
	Hidden           bool     `json:"hidden"`
	ReasoningEfforts []string `json:"reasoning_efforts,omitempty"`
}

type Catalog struct {
	ProviderID   string    `json:"provider_id"`
	Source       string    `json:"source"`
	CLIVersion   string    `json:"cli_version,omitempty"`
	DiscoveredAt time.Time `json:"discovered_at"`
	Models       []Model   `json:"models"`
}

type Selection struct {
	ProviderID          string    `json:"provider_id"`
	ModelID             string    `json:"model_id"`
	ReasoningEffort     string    `json:"reasoning_effort,omitempty"`
	ApprovalPolicy      string    `json:"approval_policy"`
	RoutingMode         string    `json:"routing_mode"`
	CatalogSource       string    `json:"catalog_source"`
	CatalogDiscoveredAt time.Time `json:"catalog_discovered_at"`
}

type RouteRule struct {
	TaskClass       string `json:"task_class"`
	ProviderID      string `json:"provider_id"`
	ModelID         string `json:"model_id"`
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
}

type RoutingConfig struct {
	Rules     []RouteRule `json:"rules"`
	UpdatedAt time.Time   `json:"updated_at,omitempty"`
}

type Project struct {
	ID              string    `json:"id"`
	Slug            string    `json:"slug,omitempty"`
	Name            string    `json:"name"`
	Description     string    `json:"description,omitempty"`
	Workspace       string    `json:"workspace,omitempty"`
	Workspaces      []string  `json:"workspaces,omitempty"`
	Repositories    []string  `json:"repositories,omitempty"`
	KnowledgeBases  []string  `json:"knowledge_bases,omitempty"`
	Tags            []string  `json:"tags,omitempty"`
	TaskCount       int       `json:"task_count,omitempty"`
	Instructions    string    `json:"instructions,omitempty"`
	ProviderID      string    `json:"provider_id,omitempty"`
	ModelID         string    `json:"model_id,omitempty"`
	ReasoningEffort string    `json:"reasoning_effort,omitempty"`
	ApprovalPolicy  string    `json:"approval_policy,omitempty"`
	RoutingMode     string    `json:"routing_mode,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ProjectInput struct {
	Name            string   `json:"name"`
	Description     string   `json:"description,omitempty"`
	Workspace       string   `json:"workspace,omitempty"`
	Workspaces      []string `json:"workspaces,omitempty"`
	Repositories    []string `json:"repositories,omitempty"`
	KnowledgeBases  []string `json:"knowledge_bases,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Instructions    string   `json:"instructions,omitempty"`
	ProviderID      string   `json:"provider_id,omitempty"`
	ModelID         string   `json:"model_id,omitempty"`
	ReasoningEffort string   `json:"reasoning_effort,omitempty"`
	ApprovalPolicy  string   `json:"approval_policy,omitempty"`
	RoutingMode     string   `json:"routing_mode,omitempty"`
}

type Attachment struct {
	Name      string `json:"name"`
	MediaType string `json:"media_type,omitempty"`
	Content   string `json:"content"`
}

type Message struct {
	ID          string       `json:"id"`
	TurnID      string       `json:"turn_id"`
	Role        string       `json:"role"`
	Content     string       `json:"content"`
	Attachments []Attachment `json:"attachments,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

type ExecutionEvent struct {
	Type       string    `json:"type"`
	ItemID     string    `json:"item_id,omitempty"`
	Decision   string    `json:"decision,omitempty"`
	OccurredAt time.Time `json:"occurred_at"`
}

type Turn struct {
	ID                      string           `json:"id"`
	UserMessageID           string           `json:"user_message_id"`
	AssistantMessageID      string           `json:"assistant_message_id"`
	ProviderID              string           `json:"provider_id"`
	ModelID                 string           `json:"model_id"`
	ReasoningEffort         string           `json:"reasoning_effort,omitempty"`
	RoutingMode             string           `json:"routing_mode"`
	TaskClass               string           `json:"task_class,omitempty"`
	RouteReason             string           `json:"route_reason,omitempty"`
	CatalogSource           string           `json:"catalog_source"`
	CatalogDiscoveredAt     time.Time        `json:"catalog_discovered_at"`
	RuntimeVersion          string           `json:"runtime_version"`
	AdapterVersion          string           `json:"adapter_version"`
	CLIVersion              string           `json:"cli_version,omitempty"`
	NativeSessionID         string           `json:"native_session_id,omitempty"`
	NativeTurnID            string           `json:"native_turn_id,omitempty"`
	Workspace               string           `json:"workspace"`
	Sandbox                 string           `json:"sandbox"`
	ApprovalPolicy          string           `json:"approval_policy"`
	ContextTransferMode     string           `json:"context_transfer_mode"`
	ContextSourceMessageIDs []string         `json:"context_source_message_ids,omitempty"`
	StartedAt               time.Time        `json:"started_at"`
	CompletedAt             *time.Time       `json:"completed_at,omitempty"`
	DurationMS              int64            `json:"duration_ms,omitempty"`
	Status                  string           `json:"status"`
	ExitCode                *int             `json:"exit_code,omitempty"`
	Error                   string           `json:"error,omitempty"`
	Events                  []ExecutionEvent `json:"events,omitempty"`
	Usage                   *Usage           `json:"usage,omitempty"`
}

type Usage struct {
	InputTokens  *int64   `json:"input_tokens,omitempty"`
	OutputTokens *int64   `json:"output_tokens,omitempty"`
	Cost         *float64 `json:"cost,omitempty"`
}

type Conversation struct {
	SchemaVersion        int                   `json:"schema_version"`
	ID                   string                `json:"id"`
	AssistantMode        bool                  `json:"assistant_mode,omitempty"`
	BotID                string                `json:"bot_id,omitempty"`
	Archived             bool                  `json:"archived,omitempty"`
	ProjectID            string                `json:"project_id,omitempty"`
	ParentConversationID string                `json:"parent_conversation_id,omitempty"`
	RootConversationID   string                `json:"root_conversation_id,omitempty"`
	Title                string                `json:"title"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
	Workspace            string                `json:"workspace"`
	Selection            Selection             `json:"selection"`
	Messages             []Message             `json:"messages"`
	Turns                []Turn                `json:"turns"`
	NativeSessions       map[string]string     `json:"native_sessions,omitempty"`
	AssistantActions     []AssistantAction     `json:"assistant_actions,omitempty"`
	AssistantAudit       []AssistantAuditEvent `json:"assistant_audit,omitempty"`
}

type ConversationSummary struct {
	ID                   string    `json:"id"`
	AssistantMode        bool      `json:"assistant_mode,omitempty"`
	Archived             bool      `json:"archived,omitempty"`
	ProjectID            string    `json:"project_id,omitempty"`
	ParentConversationID string    `json:"parent_conversation_id,omitempty"`
	RootConversationID   string    `json:"root_conversation_id,omitempty"`
	Title                string    `json:"title"`
	UpdatedAt            time.Time `json:"updated_at"`
	MessageCount         int       `json:"message_count"`
	LatestTurnStatus     string    `json:"latest_turn_status,omitempty"`
}

type AssistantAction struct {
	ID                   string    `json:"id"`
	TurnID               string    `json:"turn_id"`
	ConversationID       string    `json:"conversation_id"`
	DestinationConfig    Selection `json:"destination_config"`
	DestinationSelection Selection `json:"destination_selection"`
	Content              string    `json:"content"`
	PayloadHash          string    `json:"payload_hash"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	ExpiresAt            time.Time `json:"expires_at"`
	DecidedAt            time.Time `json:"decided_at,omitempty"`
	DestinationTurnID    string    `json:"destination_turn_id,omitempty"`
	Error                string    `json:"error,omitempty"`
}

type AssistantAuditEvent struct {
	TurnID            string    `json:"turn_id"`
	Tool              string    `json:"tool"`
	ConversationID    string    `json:"conversation_id,omitempty"`
	ActionID          string    `json:"action_id,omitempty"`
	PayloadHash       string    `json:"payload_hash,omitempty"`
	MessageIDs        []string  `json:"message_ids,omitempty"`
	DestinationTurnID string    `json:"destination_turn_id,omitempty"`
	DurationMS        int64     `json:"duration_ms,omitempty"`
	Error             string    `json:"error,omitempty"`
	Status            string    `json:"status"`
	OccurredAt        time.Time `json:"occurred_at"`
}

type TurnEvent struct {
	ID         uint64         `json:"id"`
	Type       string         `json:"type"`
	Data       map[string]any `json:"data,omitempty"`
	OccurredAt time.Time      `json:"occurred_at"`
}

type ApprovalRequest struct {
	RequestID string         `json:"request_id"`
	Method    string         `json:"method"`
	Params    map[string]any `json:"params"`
}

type RunRequest struct {
	TurnID          string
	ProviderID      string
	ModelID         string
	ReasoningEffort string
	ApprovalPolicy  string
	Sandbox         string
	Workspace       string
	NativeSessionID string
	Prompt          string
	Attachments     []Attachment
	Tools           []map[string]any
	OnToolCall      func(context.Context, string, json.RawMessage) (string, error) `json:"-"`
	OnStatus        func(string)                                                   `json:"-"`
}

type RunResult struct {
	NativeSessionID string
	NativeTurnID    string
	CLIVersion      string
	Events          []ExecutionEvent
}
