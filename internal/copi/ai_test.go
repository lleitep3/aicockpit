package copi_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type aiMockSource struct {
	mockConversationSource
	archived []string
	lastAsk  string
}

func (m *aiMockSource) ArchiveSession(ctx context.Context, convID string) error {
	m.archived = append(m.archived, convID)
	return nil
}

func (m *aiMockSource) Ask(ctx context.Context, prompt string) (string, error) {
	m.lastAsk = prompt
	return "Rotina criada com sucesso usando automação.", nil
}

func TestAskCopi(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_test_ai")
	defer os.RemoveAll(tempDir)

	mockSrc := &aiMockSource{}
	engine, err := copi.NewEngine(copi.DefaultConfig(tempDir), mockSrc)
	require.NoError(t, err)

	ctx := context.Background()

	// Empty request error
	_, err = engine.AskCopi(ctx, "   ")
	assert.Error(t, err)

	// Valid Ask
	resp, err := engine.AskCopi(ctx, "crie uma rotina de limpeza")
	require.NoError(t, err)
	assert.Contains(t, resp, "Rotina criada")
	assert.Contains(t, mockSrc.lastAsk, "Você é o Copi")
	assert.Contains(t, mockSrc.lastAsk, "AUTOMATION FIRST")
	assert.Contains(t, mockSrc.lastAsk, "crie uma rotina de limpeza")
}

func TestCleanIdleEmptySessions(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_test_clean")
	defer os.RemoveAll(tempDir)

	// Test nil source error
	nilEngine, _ := copi.NewEngine(copi.DefaultConfig(tempDir), nil)
	_, err := nilEngine.CleanIdleEmptySessions(context.Background())
	assert.Error(t, err)

	// Test list error
	errMock := &aiMockSource{
		mockConversationSource: mockConversationSource{
			listErr: assert.AnError,
		},
	}
	errEngine, _ := copi.NewEngine(copi.DefaultConfig(tempDir), errMock)
	_, err = errEngine.CleanIdleEmptySessions(context.Background())
	assert.Error(t, err)

	snapshots := make(map[string]copi.SessionSnapshot)
	for i := 0; i < 12; i++ {
		id := "sess_" + string(rune('a'+i))
		snapshots[id] = copi.SessionSnapshot{ID: id, Title: "Session", Status: copi.StatusIdle}
	}
	prompt := copi.CopiSystemPrompt("test", tempDir, snapshots)
	assert.Contains(t, prompt, "outras sessões omitidas")

	mockSrc := &aiMockSource{
		mockConversationSource: mockConversationSource{
			summaries: []personalagent.ConversationSummary{
				{ID: "c_empty"},
				{ID: "c_busy"},
			},
			convs: map[string]*personalagent.Conversation{
				"c_empty": {ID: "c_empty"},
				"c_busy": {
					ID:       "c_busy",
					Messages: []personalagent.Message{{ID: "m1"}},
				},
			},
		},
	}

	engine, err := copi.NewEngine(copi.DefaultConfig(tempDir), mockSrc)
	require.NoError(t, err)

	ctx := context.Background()
	cleaned, err := engine.CleanIdleEmptySessions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, cleaned)
	assert.Equal(t, []string{"c_empty"}, mockSrc.archived)
}
