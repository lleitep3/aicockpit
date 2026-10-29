package storage_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi/storage"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBOpenSaveGetListDelete(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test_conversations.db")

	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Initial empty list
	summaries, err := db.ListSummaries(ctx, true)
	require.NoError(t, err)
	assert.Empty(t, summaries)

	conv := &personalagent.Conversation{
		SchemaVersion: personalagent.CurrentSchemaVersion,
		ID:            "conv_1234567890abcdef1234567890abcdef",
		Title:         "Refactor API",
		ProjectID:     "proj_alpha",
		Workspace:     "/home/user/workspace",
		Archived:      false,
		CreatedAt:     time.Now().UTC().Truncate(time.Second),
		UpdatedAt:     time.Now().UTC().Truncate(time.Second),
		Messages: []personalagent.Message{
			{ID: "m1", Content: "Hello"},
			{ID: "m2", Content: "World"},
		},
		Turns: []personalagent.Turn{
			{ID: "t1", Status: "completed"},
		},
		AssistantActions: []personalagent.AssistantAction{
			{ID: "a1", Content: "Run tests"},
		},
	}

	// Save
	err = db.SaveConversation(ctx, conv)
	require.NoError(t, err)

	// Get
	loaded, err := db.GetConversation(ctx, conv.ID)
	require.NoError(t, err)
	assert.Equal(t, conv.ID, loaded.ID)
	assert.Equal(t, conv.Title, loaded.Title)
	assert.Equal(t, 2, len(loaded.Messages))
	assert.Equal(t, 1, len(loaded.Turns))
	assert.Equal(t, 1, len(loaded.AssistantActions))

	// List
	summaries, err = db.ListSummaries(ctx, false)
	require.NoError(t, err)
	assert.Len(t, summaries, 1)
	assert.Equal(t, 2, summaries[0].MessageCount)

	// Delete
	err = db.DeleteConversation(ctx, conv.ID)
	require.NoError(t, err)

	_, err = db.GetConversation(ctx, conv.ID)
	assert.Error(t, err)
}

func TestDBImportFromJSONDir(t *testing.T) {
	tempDir := t.TempDir()
	jsonDir := filepath.Join(tempDir, "legacy_jsons")
	require.NoError(t, os.MkdirAll(jsonDir, 0755))

	conv1 := personalagent.Conversation{
		ID:        "conv_test_1",
		Title:     "Old Session 1",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	data, _ := json.Marshal(conv1)
	err := os.WriteFile(filepath.Join(jsonDir, "conv_test_1.json"), data, 0644)
	require.NoError(t, err)

	dbPath := filepath.Join(tempDir, "import_test.db")
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	imported, err := db.ImportFromJSONDir(ctx, jsonDir)
	require.NoError(t, err)
	assert.Equal(t, 1, imported)

	loaded, err := db.GetConversation(ctx, "conv_test_1")
	require.NoError(t, err)
	assert.Equal(t, "Old Session 1", loaded.Title)
}
