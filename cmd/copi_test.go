package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCopiCommandFlags(t *testing.T) {
	cmd := NewCopiCommand()
	assert.Equal(t, "copi", cmd.Use)
	assert.NotNil(t, cmd.Flags().Lookup("sandbox"))
	assert.NotNil(t, cmd.Flags().Lookup("daemon"))
}

func TestCopiCommandExecution(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_cmd_test")
	defer os.RemoveAll(tempDir)

	cmd := NewCopiCommand()
	cmd.SetArgs([]string{"--sandbox", tempDir})

	input := strings.NewReader("sair\n")
	output := &bytes.Buffer{}
	cmd.SetIn(input)
	cmd.SetOut(output)

	err := cmd.Execute()
	require.NoError(t, err)

	assert.Contains(t, output.String(), "🤖 Copi — Cockpit Bot Assistant")
	assert.Contains(t, output.String(), "Até logo!")
}

func TestCopiCommandDaemonMode(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_cmd_daemon_test")
	defer os.RemoveAll(tempDir)

	cmd := NewCopiCommand()
	cmd.SetArgs([]string{"--sandbox", tempDir, "--daemon"})

	output := &bytes.Buffer{}
	cmd.SetOut(output)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	err := cmd.ExecuteContext(ctx)
	require.NoError(t, err)
	assert.Contains(t, output.String(), "modo daemon headless")
}

func TestPersonalAgentAdapter(t *testing.T) {
	tempDir := filepath.Join(os.TempDir(), "copi_adapter_test")
	defer os.RemoveAll(tempDir)

	convDir := filepath.Join(tempDir, "conversations")
	store, err := personalagent.NewStore(convDir)
	require.NoError(t, err)

	// Persist a conversation directly in store to test Adapter.Get and Adapter.List
	convID, err := personalagent.NewID("conv_")
	require.NoError(t, err)

	testConv := personalagent.Conversation{
		SchemaVersion: personalagent.CurrentSchemaVersion,
		ID:            convID,
		Title:         "Sample Session",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	err = store.Save(testConv)
	require.NoError(t, err)

	paSvc, err := personalagent.NewService(store, personalagent.NewProviderRegistry(), tempDir)
	require.NoError(t, err)

	adapter := &personalAgentAdapter{svc: paSvc}
	ctx := context.Background()

	// List
	summaries, err := adapter.List(ctx, false)
	require.NoError(t, err)
	assert.Len(t, summaries, 1)

	// Get
	gotConv, err := adapter.Get(ctx, convID)
	require.NoError(t, err)
	assert.Equal(t, convID, gotConv.ID)

	// SendMessage & DecideAction branches
	_ = adapter.SendMessage(ctx, convID, "hello")
	_ = adapter.DecideAction(ctx, convID, "act_x", true)
	_ = adapter.DecideAction(ctx, convID, "act_y", false)
}
