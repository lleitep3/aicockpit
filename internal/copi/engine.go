package copi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

// Config holds runtime configuration for the Copi Engine.
type Config struct {
	SandboxDir       string        `json:"sandbox_dir"`
	AutoScanInterval time.Duration `json:"auto_scan_interval"`
}

// DefaultConfig returns the default configuration with ~/.cockpit sandbox.
func DefaultConfig(cockpitDir string) Config {
	if cockpitDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cockpitDir = filepath.Join(home, ".cockpit")
		} else {
			cockpitDir = ".cockpit"
		}
	}

	return Config{
		SandboxDir:       cockpitDir,
		AutoScanInterval: 10 * time.Second,
	}
}

// ConversationSource defines the operations Copi needs from the underlying session store.
type ConversationSource interface {
	List(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error)
	Get(ctx context.Context, id string) (*personalagent.Conversation, error)
	SendMessage(ctx context.Context, convID string, message string) error
	DecideAction(ctx context.Context, convID string, actionID string, approved bool) error
}

// Engine is the isolated core service powering Copi (both for TUI and headless daemon).
type Engine struct {
	cfg        Config
	source     ConversationSource
	mu         sync.RWMutex
	snapshots  map[string]SessionSnapshot
	lastReport UpdateReport
	scanTicker *time.Ticker
	stopChan   chan struct{}
	onUpdate   func(UpdateReport)
}

// NewEngine creates an initialized Engine instance.
func NewEngine(cfg Config, source ConversationSource) (*Engine, error) {
	if err := os.MkdirAll(cfg.SandboxDir, 0755); err != nil {
		return nil, fmt.Errorf("ensure sandbox dir: %w", err)
	}

	return &Engine{
		cfg:       cfg,
		source:    source,
		snapshots: make(map[string]SessionSnapshot),
		stopChan:  make(chan struct{}),
	}, nil
}

// SetUpdateHook sets an optional callback triggered when a background scan detects changes.
func (e *Engine) SetUpdateHook(fn func(UpdateReport)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onUpdate = fn
}

// SandboxDir returns the configured sandbox path.
func (e *Engine) SandboxDir() string {
	return e.cfg.SandboxDir
}

// ScanUpdates fetches all conversations, computes diffs against stored snapshots, and returns an UpdateReport.
// This is 100% deterministic and consumes 0 tokens.
func (e *Engine) ScanUpdates(ctx context.Context) (UpdateReport, error) {
	if e.source == nil {
		return UpdateReport{}, fmt.Errorf("no conversation source configured")
	}

	summaries, err := e.source.List(ctx, false)
	if err != nil {
		return UpdateReport{}, fmt.Errorf("list conversations: %w", err)
	}

	currentSnapshots := make(map[string]SessionSnapshot)
	for _, summary := range summaries {
		conv, err := e.source.Get(ctx, summary.ID)
		if err != nil || conv == nil {
			continue
		}
		currentSnapshots[conv.ID] = SnapshotFromConversation(conv)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	report := BuildUpdateReport(e.snapshots, currentSnapshots)
	e.snapshots = currentSnapshots
	e.lastReport = report

	return report, nil
}

// LastReport returns the latest computed update report.
func (e *Engine) LastReport() UpdateReport {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.lastReport
}

// ListSnapshots returns all known session snapshots.
func (e *Engine) ListSnapshots() []SessionSnapshot {
	e.mu.RLock()
	defer e.mu.RUnlock()

	list := make([]SessionSnapshot, 0, len(e.snapshots))
	for _, s := range e.snapshots {
		list = append(list, s)
	}
	return list
}

// PendingActions returns all pending human-in-the-loop actions across all sessions.
func (e *Engine) PendingActions() []PendingAction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var actions []PendingAction
	for _, snap := range e.snapshots {
		actions = append(actions, snap.PendingActions...)
	}
	return actions
}

// ApproveAction resolves a pending action in a child session.
func (e *Engine) ApproveAction(ctx context.Context, convID string, actionID string) error {
	if e.source == nil {
		return fmt.Errorf("no conversation source configured")
	}
	return e.source.DecideAction(ctx, convID, actionID, true)
}

// RejectAction cancels a pending action in a child session.
func (e *Engine) RejectAction(ctx context.Context, convID string, actionID string) error {
	if e.source == nil {
		return fmt.Errorf("no conversation source configured")
	}
	return e.source.DecideAction(ctx, convID, actionID, false)
}

// SendMessage sends an instruction directly into a specific session.
func (e *Engine) SendMessage(ctx context.Context, convID string, message string) error {
	if e.source == nil {
		return fmt.Errorf("no conversation source configured")
	}
	return e.source.SendMessage(ctx, convID, message)
}

// StartAutoScan launches background periodic scans (for daemon mode).
func (e *Engine) StartAutoScan(ctx context.Context) {
	e.mu.Lock()
	if e.scanTicker != nil {
		e.mu.Unlock()
		return
	}
	e.scanTicker = time.NewTicker(e.cfg.AutoScanInterval)
	e.mu.Unlock()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-e.stopChan:
				return
			case <-e.scanTicker.C:
				report, err := e.ScanUpdates(ctx)
				if err == nil && report.HasChanges {
					e.mu.RLock()
					hook := e.onUpdate
					e.mu.RUnlock()
					if hook != nil {
						hook(report)
					}
				}
			}
		}
	}()
}

// Stop stops background polling.
func (e *Engine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.scanTicker != nil {
		e.scanTicker.Stop()
		close(e.stopChan)
		e.scanTicker = nil
	}
}
