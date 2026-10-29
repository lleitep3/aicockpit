package intent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi/intent"
	"github.com/lleitep3/aicockpit/internal/copi/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntentManagerFlow(t *testing.T) {
	tempDir := t.TempDir()

	mgr := intent.NewManager(tempDir)
	assert.Nil(t, mgr.Get())
	assert.Empty(t, mgr.History())

	// Set intent
	act := mgr.SetOrUpdate(router.IntentHostCheck, "Checar CI", "github")
	require.NotNil(t, act)
	assert.Equal(t, router.IntentHostCheck, act.Category)
	assert.Equal(t, "Checar CI", act.Goal)

	// Record turns
	mgr.RecordTurn("como estão os checks?", "rodando 6 checks")
	assert.Len(t, mgr.History(), 2)

	// Update existing intent
	act2 := mgr.SetOrUpdate(router.IntentHostCheck, "Checar CI e branch", "github")
	assert.Equal(t, act.ID, act2.ID)
	assert.Equal(t, "Checar CI e branch", act2.Goal)

	// Change category starts new intent
	act3 := mgr.SetOrUpdate(router.IntentDatabaseQuery, "Limpar conversas", "")
	assert.NotEqual(t, act.ID, act3.ID)
	assert.Equal(t, router.IntentDatabaseQuery, act3.Category)

	// Load existing from disk
	mgrLoaded := intent.NewManager(tempDir)
	assert.NotNil(t, mgrLoaded.Get())
	assert.Equal(t, act3.ID, mgrLoaded.Get().ID)

	// Clear
	mgr.Clear()
	assert.Nil(t, mgr.Get())
	assert.NoFileExists(t, filepath.Join(tempDir, "current_intent.json"))
}

func TestIntentManagerRecordTurnWithoutPriorActive(t *testing.T) {
	tempDir := t.TempDir()
	mgr := intent.NewManager(tempDir)

	mgr.RecordTurn("pergunta solta", "resposta")
	assert.NotNil(t, mgr.Get())
	assert.Equal(t, router.IntentComplexChat, mgr.Get().Category)
	assert.Len(t, mgr.History(), 2)

	// Record more turns to verify window bounding (> 6 turns)
	for i := 0; i < 5; i++ {
		mgr.RecordTurn("msg", "resp")
	}
	assert.LessOrEqual(t, len(mgr.History()), 6)
}

func TestIntentManagerCorruptedFile(t *testing.T) {
	tempDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tempDir, "current_intent.json"), []byte("invalid-json"), 0644)

	mgr := intent.NewManager(tempDir)
	assert.Nil(t, mgr.Get())
}
