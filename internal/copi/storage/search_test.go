package storage_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi/storage"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBSearchConversations(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "search_test.db")
	db, err := storage.Open(dbPath)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Empty query returns nil
	res, err := db.SearchConversations(ctx, "   ")
	require.NoError(t, err)
	assert.Nil(t, res)

	// Insert matching and non-matching conversations
	convMatch := &personalagent.Conversation{
		ID:        "conv_partilhar_1",
		Title:     "Projeto Partilhar ABA",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Messages:  []personalagent.Message{{ID: "m1", Content: "Contexto partilhar"}},
	}
	require.NoError(t, db.SaveConversation(ctx, convMatch))

	convOther := &personalagent.Conversation{
		ID:        "conv_other_2",
		Title:     "Outro assunto",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, db.SaveConversation(ctx, convOther))

	// Search
	results, err := db.SearchConversations(ctx, "partilhar")
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, "conv_partilhar_1", results[0].ID)
}
