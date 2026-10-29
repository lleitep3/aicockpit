package storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/personalagent"
)

// SearchConversations queries conversations matching text in title or messages (0 tokens).
func (s *DB) SearchConversations(ctx context.Context, keyword string) ([]personalagent.ConversationSummary, error) {
	clean := strings.TrimSpace(keyword)
	if clean == "" {
		return nil, nil
	}

	query := `
	SELECT id, assistant_mode, archived, project_id, parent_id, root_id,
	       title, updated_at, messages_json
	FROM conversations
	WHERE archived = FALSE AND (title LIKE ? OR messages_json LIKE ?)
	ORDER BY updated_at DESC;
	`

	pattern := "%" + clean + "%"
	rows, err := s.db.QueryContext(ctx, query, pattern, pattern)
	if err != nil {
		return nil, fmt.Errorf("search query error: %w", err)
	}
	defer rows.Close()

	var results []personalagent.ConversationSummary
	for rows.Next() {
		var sum personalagent.ConversationSummary
		var msgsJSON string
		err := rows.Scan(
			&sum.ID, &sum.AssistantMode, &sum.Archived, &sum.ProjectID,
			&sum.ParentConversationID, &sum.RootConversationID,
			&sum.Title, &sum.UpdatedAt, &msgsJSON,
		)
		if err != nil {
			continue
		}
		results = append(results, sum)
	}

	return results, nil
}
