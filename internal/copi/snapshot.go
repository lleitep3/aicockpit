package copi

import (
	"fmt"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

// SnapshotFromConversation extracts a token-free snapshot from a personalagent Conversation.
func SnapshotFromConversation(conv *personalagent.Conversation) SessionSnapshot {
	if conv == nil {
		return SessionSnapshot{}
	}

	snap := SessionSnapshot{
		ID:           conv.ID,
		Title:        conv.Title,
		ProjectID:    conv.ProjectID,
		Status:       StatusIdle,
		MessageCount: len(conv.Messages),
		TurnCount:    len(conv.Turns),
		UpdatedAt:    conv.UpdatedAt,
	}

	for _, turn := range conv.Turns {
		if turn.Usage != nil {
			if turn.Usage.InputTokens != nil {
				snap.InputTokens += *turn.Usage.InputTokens
			}
			if turn.Usage.OutputTokens != nil {
				snap.OutputTokens += *turn.Usage.OutputTokens
			}
			if turn.Usage.Cost != nil {
				snap.Cost += *turn.Usage.Cost
			}
		}
	}

	if len(conv.Turns) > 0 {
		latest := conv.Turns[len(conv.Turns)-1]
		snap.LatestTurnID = latest.ID
		switch strings.ToLower(latest.Status) {
		case "running", "in_progress":
			snap.Status = StatusRunning
		case "completed", "done", "success":
			snap.Status = StatusDone
		case "failed", "error":
			snap.Status = StatusFailed
		default:
			snap.Status = StatusIdle
		}
	}

	for _, action := range conv.AssistantActions {
		if strings.EqualFold(action.Status, "pending") || action.Status == "" {
			snap.PendingActions = append(snap.PendingActions, PendingAction{
				ID:             action.ID,
				ConversationID: conv.ID,
				TurnID:         action.TurnID,
				Description:    action.Content,
				PayloadHash:    action.PayloadHash,
				CreatedAt:      action.CreatedAt,
				ExpiresAt:      action.ExpiresAt,
			})
		}
	}

	if len(snap.PendingActions) > 0 {
		snap.Status = StatusPending
	}

	return snap
}

// ComputeDiff compares an old snapshot with a new snapshot and produces a deterministic diff.
func ComputeDiff(oldSnap, newSnap SessionSnapshot) (SessionDiff, bool) {
	diff := SessionDiff{
		SessionID:     newSnap.ID,
		SessionTitle:  newSnap.Title,
		PreviousState: oldSnap.Status,
		CurrentState:  newSnap.Status,
	}

	newMsgs := newSnap.MessageCount - oldSnap.MessageCount
	if newMsgs > 0 {
		diff.NewMessages = newMsgs
	}

	oldTokens := oldSnap.InputTokens + oldSnap.OutputTokens
	newTokens := newSnap.InputTokens + newSnap.OutputTokens
	if newTokens > oldTokens {
		diff.NewTokens = newTokens - oldTokens
	}

	// Identify new pending actions
	oldPendingMap := make(map[string]bool)
	for _, p := range oldSnap.PendingActions {
		oldPendingMap[p.ID] = true
	}

	newPendingMap := make(map[string]bool)
	for _, p := range newSnap.PendingActions {
		newPendingMap[p.ID] = true
		if !oldPendingMap[p.ID] {
			diff.NewPending = append(diff.NewPending, p)
		}
	}

	// Identify resolved actions
	for _, p := range oldSnap.PendingActions {
		if !newPendingMap[p.ID] {
			diff.ResolvedPending = append(diff.ResolvedPending, p.ID)
		}
	}

	hasChanges := oldSnap.ID == "" || // first time seen
		oldSnap.Status != newSnap.Status ||
		diff.NewMessages > 0 ||
		len(diff.NewPending) > 0 ||
		len(diff.ResolvedPending) > 0

	if !hasChanges {
		return diff, false
	}

	var parts []string
	if oldSnap.ID == "" {
		parts = append(parts, fmt.Sprintf("Nova sessão identificada (status: %s)", newSnap.Status))
	} else if oldSnap.Status != newSnap.Status {
		parts = append(parts, fmt.Sprintf("Status mudou de '%s' para '%s'", oldSnap.Status, newSnap.Status))
	}

	if diff.NewMessages > 0 {
		parts = append(parts, fmt.Sprintf("%d novas mensagens (+%d tokens)", diff.NewMessages, diff.NewTokens))
	}

	if len(diff.NewPending) > 0 {
		parts = append(parts, fmt.Sprintf("⚠️ %d nova(s) ação(ões) pendente(s) aguardando decisão", len(diff.NewPending)))
	}

	if len(diff.ResolvedPending) > 0 {
		parts = append(parts, fmt.Sprintf("%d ação(ões) resolvida(s)", len(diff.ResolvedPending)))
	}

	diff.Summary = strings.Join(parts, " | ")
	return diff, true
}

// BuildUpdateReport aggregates changes from all sessions into a human-readable zero-token report.
func BuildUpdateReport(oldSnapshots, newSnapshots map[string]SessionSnapshot) UpdateReport {
	report := UpdateReport{
		Timestamp:     time.Now(),
		TotalSessions: len(newSnapshots),
	}

	var diffLines []string

	for id, newSnap := range newSnapshots {
		if newSnap.Status == StatusRunning {
			report.ActiveSessions++
		}
		report.PendingCount += len(newSnap.PendingActions)

		oldSnap := oldSnapshots[id]
		if diff, changed := ComputeDiff(oldSnap, newSnap); changed {
			report.Diffs = append(report.Diffs, diff)
			line := fmt.Sprintf("  • [%s] \"%s\": %s", diff.SessionID, diff.SessionTitle, diff.Summary)
			diffLines = append(diffLines, line)
		}
	}

	report.HasChanges = len(report.Diffs) > 0

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("📊 Relatório Copi (%s) — %d sessões (%d ativas, %d ações pendentes)\n",
		report.Timestamp.Format("15:04:05"), report.TotalSessions, report.ActiveSessions, report.PendingCount))

	if report.HasChanges {
		builder.WriteString("Alterações detectadas (0 tokens consumidos):\n")
		builder.WriteString(strings.Join(diffLines, "\n"))
	} else {
		builder.WriteString("Nenhuma alteração recente nas sessões. Tudo calmo.")
	}

	report.FormattedText = builder.String()
	return report
}
