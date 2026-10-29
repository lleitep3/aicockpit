package storage_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi/storage"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStorageErrorPaths(t *testing.T) {
	tempDir := t.TempDir()

	// Invalid dir path to trigger MkdirAll error in Open
	invalidPath := filepath.Join("/dev/null", "cannot_create", "test.db")
	_, err := storage.Open(invalidPath)
	assert.Error(t, err)

	// Open valid db
	dbPath := filepath.Join(tempDir, "err_test.db")
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Save nil/empty conversation
	err = db.SaveConversation(ctx, nil)
	assert.Error(t, err)

	err = db.SaveConversation(ctx, &personalagent.Conversation{ID: ""})
	assert.Error(t, err)

	// Import from non-existent directory
	imported, err := db.ImportFromJSONDir(ctx, filepath.Join(tempDir, "does_not_exist"))
	require.NoError(t, err)
	assert.Equal(t, 0, imported)

	// Close nil safely
	var nilDB *storage.DB
	assert.NoError(t, nilDB.Close())
}
