package tui_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi"
	"github.com/lleitep3/aicockpit/internal/copi/tui"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorSource struct {
	decideErr bool
}

func (e *errorSource) List(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error) {
	return nil, errors.New("list failed")
}
func (e *errorSource) Get(ctx context.Context, id string) (*personalagent.Conversation, error) {
	return nil, errors.New("get failed")
}
func (e *errorSource) SendMessage(ctx context.Context, convID string, message string) error {
	return errors.New("send failed")
}
func (e *errorSource) DecideAction(ctx context.Context, convID string, actionID string, approved bool) error {
	return errors.New("decide failed")
}

func TestTUIErrorBranchesAndEmptyState(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_tui_err_test")
	defer os.RemoveAll(tempDir)

	errSrc := &errorSource{}
	engine, err := copi.NewEngine(copi.DefaultConfig(tempDir), errSrc)
	require.NoError(t, err)

	input := strings.NewReader("1\n2\n3\n4\n5\n\nquit\n")
	output := &bytes.Buffer{}

	ui := tui.NewUI(engine, input, output)
	err = ui.Run(context.Background())
	require.NoError(t, err)

	outStr := output.String()
	assert.Contains(t, outStr, "Erro ao verificar atualizações")
	assert.Contains(t, outStr, "Nenhuma ação pendente de aprovação")
	assert.Contains(t, outStr, "Nenhuma sessão disponível para envio")
}

func TestTUIApproveExecution(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_tui_appr_exec")
	defer os.RemoveAll(tempDir)

	validSrc := &mockSource{
		summaries: []personalagent.ConversationSummary{{ID: "c1", Title: "Task"}},
		convs: map[string]*personalagent.Conversation{
			"c1": {
				ID: "c1",
				AssistantActions: []personalagent.AssistantAction{
					{ID: "act_ok", Status: "pending"},
				},
			},
		},
	}
	engineValid, err := copi.NewEngine(copi.DefaultConfig(tempDir), validSrc)
	require.NoError(t, err)
	_, _ = engineValid.ScanUpdates(context.Background())

	input := strings.NewReader("3\nquit\n")
	output := &bytes.Buffer{}
	ui := tui.NewUI(engineValid, input, output)
	err = ui.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, output.String(), "Ação [act_ok] aprovada")
}
