package copi_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConversationSource struct {
	mu           sync.Mutex
	summaries    []personalagent.ConversationSummary
	convs        map[string]*personalagent.Conversation
	sentMessages map[string][]string
	decisions    map[string]bool
	listErr      error
}

func (m *mockConversationSource) List(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.summaries, nil
}

func (m *mockConversationSource) Get(ctx context.Context, id string) (*personalagent.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conv, ok := m.convs[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return conv, nil
}

func (m *mockConversationSource) SendMessage(ctx context.Context, convID string, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sentMessages == nil {
		m.sentMessages = make(map[string][]string)
	}
	m.sentMessages[convID] = append(m.sentMessages[convID], message)
	return nil
}

func (m *mockConversationSource) DecideAction(ctx context.Context, convID string, actionID string, approved bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.decisions == nil {
		m.decisions = make(map[string]bool)
	}
	m.decisions[actionID] = approved
	return nil
}

func TestEngineConfigAndSandbox(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_test_sandbox")
	defer os.RemoveAll(tempDir)

	cfg := copi.DefaultConfig(tempDir)
	assert.Equal(t, tempDir, cfg.SandboxDir)
	assert.Equal(t, 10*time.Second, cfg.AutoScanInterval)

	defaultCfg := copi.DefaultConfig("")
	assert.Contains(t, defaultCfg.SandboxDir, ".cockpit")

	engine, err := copi.NewEngine(cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, tempDir, engine.SandboxDir())
	assert.DirExists(t, tempDir)
}

func TestEngineScanUpdatesAndDiff(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_test_scan")
	defer os.RemoveAll(tempDir)

	mockSource := &mockConversationSource{
		summaries: []personalagent.ConversationSummary{
			{ID: "c1", Title: "Task 1"},
		},
		convs: map[string]*personalagent.Conversation{
			"c1": {
				ID:    "c1",
				Title: "Task 1",
				Turns: []personalagent.Turn{
					{ID: "t1", Status: "running"},
				},
			},
		},
	}

	cfg := copi.DefaultConfig(tempDir)
	engine, err := copi.NewEngine(cfg, mockSource)
	require.NoError(t, err)

	ctx := context.Background()

	// Initial Scan
	report1, err := engine.ScanUpdates(ctx)
	require.NoError(t, err)
	assert.True(t, report1.HasChanges)
	assert.Equal(t, 1, report1.TotalSessions)
	assert.Equal(t, 1, report1.ActiveSessions)
	assert.Equal(t, report1, engine.LastReport())

	snapshots := engine.ListSnapshots()
	assert.Len(t, snapshots, 1)

	// Second Scan (no changes)
	report2, err := engine.ScanUpdates(ctx)
	require.NoError(t, err)
	assert.False(t, report2.HasChanges)

	// Update conversation state with pending action
	mockSource.convs["c1"] = &personalagent.Conversation{
		ID:    "c1",
		Title: "Task 1",
		Turns: []personalagent.Turn{
			{ID: "t1", Status: "running"},
		},
		AssistantActions: []personalagent.AssistantAction{
			{ID: "act_1", Status: "pending", Content: "Run db migration"},
		},
	}

	report3, err := engine.ScanUpdates(ctx)
	require.NoError(t, err)
	assert.True(t, report3.HasChanges)
	assert.Equal(t, 1, report3.PendingCount)

	pending := engine.PendingActions()
	require.Len(t, pending, 1)
	assert.Equal(t, "act_1", pending[0].ID)
}

func TestEngineActionsAndMessaging(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_test_actions")
	defer os.RemoveAll(tempDir)

	mockSource := &mockConversationSource{}
	engine, err := copi.NewEngine(copi.DefaultConfig(tempDir), mockSource)
	require.NoError(t, err)

	ctx := context.Background()

	// Approve
	err = engine.ApproveAction(ctx, "c1", "act_1")
	require.NoError(t, err)
	assert.True(t, mockSource.decisions["act_1"])

	// Reject
	err = engine.RejectAction(ctx, "c1", "act_2")
	require.NoError(t, err)
	assert.False(t, mockSource.decisions["act_2"])

	// Send Message
	err = engine.SendMessage(ctx, "c1", "execute tests")
	require.NoError(t, err)
	assert.Equal(t, []string{"execute tests"}, mockSource.sentMessages["c1"])
}

func TestEngineAutoScan(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_test_autoscan")
	defer os.RemoveAll(tempDir)

	mockSource := &mockConversationSource{
		summaries: []personalagent.ConversationSummary{
			{ID: "c1", Title: "Task 1"},
		},
		convs: map[string]*personalagent.Conversation{
			"c1": {ID: "c1", Title: "Task 1"},
		},
	}

	cfg := copi.Config{
		SandboxDir:       tempDir,
		AutoScanInterval: 10 * time.Millisecond,
	}

	engine, err := copi.NewEngine(cfg, mockSource)
	require.NoError(t, err)

	var hookCalled bool
	var mu sync.Mutex
	engine.SetUpdateHook(func(r copi.UpdateReport) {
		mu.Lock()
		hookCalled = true
		mu.Unlock()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	engine.StartAutoScan(ctx)
	// idempotent
	engine.StartAutoScan(ctx)

	assert.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return hookCalled
	}, 150*time.Millisecond, 15*time.Millisecond)

	engine.Stop()
}
