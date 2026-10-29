package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi/storage"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageMorePaths(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "more_test.db")
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Get non-existent
	_, err = db.GetConversation(ctx, "non_existent_id")
	assert.Error(t, err)

	// Corrupted json in legacy dir
	jsonDir := filepath.Join(tempDir, "legacy_corrupted")
	require.NoError(t, os.MkdirAll(jsonDir, 0755))
	_ = os.WriteFile(filepath.Join(jsonDir, "broken.json"), []byte("{broken json"), 0644)
	_ = os.WriteFile(filepath.Join(jsonDir, "ignore.txt"), []byte("not json"), 0644)

	imported, err := db.ImportFromJSONDir(ctx, jsonDir)
	require.NoError(t, err)
	assert.Equal(t, 0, imported)

	// Save one archived conversation then list including archived
	convArchived := &personalagent.Conversation{
		ID:        "conv_archived_1",
		Title:     "Archived Session",
		Archived:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, db.SaveConversation(ctx, convArchived))

	allSummaries, err := db.ListSummaries(ctx, true)
	require.NoError(t, err)
	assert.Len(t, allSummaries, 1)
}
