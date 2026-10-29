package copi_test

import (
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
)

func TestSnapshotFromConversation(t *testing.T) {
	assert.Equal(t, copi.SessionSnapshot{}, copi.SnapshotFromConversation(nil))

	inTokens := int64(100)
	outTokens := int64(50)
	cost := 0.002

	conv := &personalagent.Conversation{
		ID:        "sess_1",
		Title:     "Feature Auth",
		ProjectID: "proj_a",
		UpdatedAt: time.Now(),
		Messages:  []personalagent.Message{{ID: "m1"}, {ID: "m2"}},
		Turns: []personalagent.Turn{
			{
				ID:     "turn_1",
				Status: "completed",
				Usage: &personalagent.Usage{
					InputTokens:  &inTokens,
					OutputTokens: &outTokens,
					Cost:         &cost,
				},
			},
		},
		AssistantActions: []personalagent.AssistantAction{
			{
				ID:      "act_1",
				Status:  "pending",
				Content: "rm -rf test",
			},
		},
	}

	snap := copi.SnapshotFromConversation(conv)
	assert.Equal(t, "sess_1", snap.ID)
	assert.Equal(t, "Feature Auth", snap.Title)
	assert.Equal(t, copi.StatusPending, snap.Status)
	assert.Equal(t, 2, snap.MessageCount)
	assert.Equal(t, 1, snap.TurnCount)
	assert.Equal(t, int64(100), snap.InputTokens)
	assert.Equal(t, int64(50), snap.OutputTokens)
	assert.Equal(t, 0.002, snap.Cost)
	assert.Len(t, snap.PendingActions, 1)
	assert.Equal(t, "act_1", snap.PendingActions[0].ID)
}

func TestComputeDiff(t *testing.T) {
	t.Run("no changes", func(t *testing.T) {
		snap := copi.SessionSnapshot{
			ID:           "sess_1",
			Title:        "Task 1",
			Status:       copi.StatusIdle,
			MessageCount: 2,
		}
		diff, changed := copi.ComputeDiff(snap, snap)
		assert.False(t, changed)
		assert.Empty(t, diff.Summary)
	})

	t.Run("first time seen", func(t *testing.T) {
		snap := copi.SessionSnapshot{
			ID:           "sess_1",
			Title:        "Task 1",
			Status:       copi.StatusRunning,
			MessageCount: 1,
		}
		diff, changed := copi.ComputeDiff(copi.SessionSnapshot{}, snap)
		assert.True(t, changed)
		assert.Contains(t, diff.Summary, "Nova sessão identificada")
	})

	t.Run("status change and new pending actions", func(t *testing.T) {
		oldSnap := copi.SessionSnapshot{
			ID:           "sess_1",
			Title:        "Task 1",
			Status:       copi.StatusRunning,
			MessageCount: 2,
		}
		newSnap := copi.SessionSnapshot{
			ID:           "sess_1",
			Title:        "Task 1",
			Status:       copi.StatusPending,
			MessageCount: 4,
			InputTokens:  100,
			PendingActions: []copi.PendingAction{
				{ID: "act_x", Description: "Deploy to prod"},
			},
		}

		diff, changed := copi.ComputeDiff(oldSnap, newSnap)
		assert.True(t, changed)
		assert.Equal(t, 2, diff.NewMessages)
		assert.Equal(t, int64(100), diff.NewTokens)
		assert.Len(t, diff.NewPending, 1)
		assert.Contains(t, diff.Summary, "Status mudou de 'running' para 'pending_decision'")
		assert.Contains(t, diff.Summary, "2 novas mensagens")
		assert.Contains(t, diff.Summary, "1 nova(s) ação(ões) pendente(s)")
	})

	t.Run("resolved pending actions", func(t *testing.T) {
		oldSnap := copi.SessionSnapshot{
			ID:     "sess_1",
			Status: copi.StatusPending,
			PendingActions: []copi.PendingAction{
				{ID: "act_x"},
			},
		}
		newSnap := copi.SessionSnapshot{
			ID:     "sess_1",
			Status: copi.StatusDone,
		}
		diff, changed := copi.ComputeDiff(oldSnap, newSnap)
		assert.True(t, changed)
		assert.Equal(t, []string{"act_x"}, diff.ResolvedPending)
		assert.Contains(t, diff.Summary, "1 ação(ões) resolvida(s)")
	})
}

func TestBuildUpdateReport(t *testing.T) {
	oldSnaps := map[string]copi.SessionSnapshot{
		"sess_1": {ID: "sess_1", Title: "Job A", Status: copi.StatusRunning},
	}
	newSnaps := map[string]copi.SessionSnapshot{
		"sess_1": {ID: "sess_1", Title: "Job A", Status: copi.StatusDone},
	}

	report := copi.BuildUpdateReport(oldSnaps, newSnaps)
	assert.True(t, report.HasChanges)
	assert.Equal(t, 1, report.TotalSessions)
	assert.Equal(t, 0, report.ActiveSessions)
	assert.Contains(t, report.FormattedText, "Alterações detectadas (0 tokens consumidos)")
	assert.Contains(t, report.FormattedText, "Status mudou")

	// No changes report
	reportNoChange := copi.BuildUpdateReport(newSnaps, newSnaps)
	assert.False(t, reportNoChange.HasChanges)
	assert.Contains(t, reportNoChange.FormattedText, "Nenhuma alteração recente")
}
