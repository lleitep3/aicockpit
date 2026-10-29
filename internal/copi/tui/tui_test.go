package tui_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi"
	"github.com/lleitep3/aicockpit/internal/copi/tui"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSource struct {
	summaries []personalagent.ConversationSummary
	convs     map[string]*personalagent.Conversation
	decisions map[string]bool
}

func (m *mockSource) List(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error) {
	return m.summaries, nil
}

func (m *mockSource) Get(ctx context.Context, id string) (*personalagent.Conversation, error) {
	return m.convs[id], nil
}

func (m *mockSource) SendMessage(ctx context.Context, convID string, message string) error {
	return nil
}

func (m *mockSource) DecideAction(ctx context.Context, convID string, actionID string, approved bool) error {
	if m.decisions == nil {
		m.decisions = make(map[string]bool)
	}
	m.decisions[actionID] = approved
	return nil
}

func (m *mockSource) ArchiveSession(ctx context.Context, convID string) error {
	return nil
}

func (m *mockSource) Ask(ctx context.Context, prompt string) (string, error) {
	return "Entendido e executado com sucesso.", nil
}

func TestTUIRunAndMenuOptions(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_tui_test")
	defer os.RemoveAll(tempDir)

	src := &mockSource{
		summaries: []personalagent.ConversationSummary{{ID: "c1", Title: "Task 1"}},
		convs: map[string]*personalagent.Conversation{
			"c1": {
				ID:    "c1",
				Title: "Task 1",
				AssistantActions: []personalagent.AssistantAction{
					{ID: "act_1", Status: "pending", Content: "Deploy"},
				},
			},
		},
	}

	engine, err := copi.NewEngine(copi.DefaultConfig(tempDir), src)
	require.NoError(t, err)

	input := strings.NewReader("1\n2\n3\n4\n5\n6\nhello copi\nsair\n")
	output := &bytes.Buffer{}

	ui := tui.NewUI(engine, input, output)
	err = ui.Run(context.Background())
	require.NoError(t, err)

	outStr := output.String()
	assert.Contains(t, outStr, "🤖 Copi — Cockpit Bot Assistant")
	assert.Contains(t, outStr, "Atalhos Rápidos (0 Tokens):")
	assert.Contains(t, outStr, "Relatório Copi")
	assert.Contains(t, outStr, "Sessões Monitoradas")
	assert.Contains(t, outStr, "Ação [act_1] aprovada com sucesso")
	assert.Contains(t, outStr, "Para enviar mensagem, utilize")
	assert.Contains(t, outStr, "Até logo!")
	assert.True(t, src.decisions["act_1"])
}

func TestTUIContextCancellation(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_tui_cancel_test")
	defer os.RemoveAll(tempDir)

	src := &mockSource{}
	engine, err := copi.NewEngine(copi.DefaultConfig(tempDir), src)
	require.NoError(t, err)

	inPipeR, inPipeW, err := os.Pipe()
	require.NoError(t, err)
	defer inPipeR.Close()
	defer inPipeW.Close()

	output := &bytes.Buffer{}
	ui := tui.NewUI(engine, inPipeR, output)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err = ui.Run(ctx)
	assert.NoError(t, err)
	assert.Contains(t, output.String(), "Encerrando Copi. Até logo!")
}
