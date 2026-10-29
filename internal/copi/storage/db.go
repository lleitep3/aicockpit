package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lleitep3/aicockpit/internal/personalagent"
	_ "modernc.org/sqlite"
)

// DB encapsulates the embedded SQLite storage for Cockpit.
type DB struct {
	dbPath string
	db     *sql.DB
}

// Open creates or connects to the embedded SQLite database at path,
// applying WAL mode and standard schema tables.
func Open(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	// Performance pragmatic settings: WAL mode & busy timeout
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA synchronous = NORMAL;",
		"PRAGMA busy_timeout = 5000;",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("exec pragma %q: %w", p, err)
		}
	}

	s := &DB{
		dbPath: dbPath,
		db:     db,
	}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *DB) Close() error {
	if s != nil && s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *DB) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		schema_version INTEGER NOT NULL,
		title TEXT NOT NULL,
		project_id TEXT,
		parent_id TEXT,
		root_id TEXT,
		assistant_mode BOOLEAN DEFAULT FALSE,
		archived BOOLEAN DEFAULT FALSE,
		workspace TEXT,
		selection_json TEXT,
		messages_json TEXT,
		turns_json TEXT,
		assistant_actions_json TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_conversations_archived ON conversations(archived);
	CREATE INDEX IF NOT EXISTS idx_conversations_updated_at ON conversations(updated_at DESC);
	`
	_, err := s.db.Exec(schema)
	return err
}

// SaveConversation upserts a conversation into the database.
func (s *DB) SaveConversation(ctx context.Context, conv *personalagent.Conversation) error {
	if conv == nil || conv.ID == "" {
		return fmt.Errorf("invalid conversation")
	}

	selData, _ := json.Marshal(conv.Selection)
	msgsData, _ := json.Marshal(conv.Messages)
	turnsData, _ := json.Marshal(conv.Turns)
	actionsData, _ := json.Marshal(conv.AssistantActions)

	query := `
	INSERT INTO conversations (
		id, schema_version, title, project_id, parent_id, root_id,
		assistant_mode, archived, workspace, selection_json,
		messages_json, turns_json, assistant_actions_json,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		title = excluded.title,
		project_id = excluded.project_id,
		parent_id = excluded.parent_id,
		root_id = excluded.root_id,
		assistant_mode = excluded.assistant_mode,
		archived = excluded.archived,
		workspace = excluded.workspace,
		selection_json = excluded.selection_json,
		messages_json = excluded.messages_json,
		turns_json = excluded.turns_json,
		assistant_actions_json = excluded.assistant_actions_json,
		updated_at = excluded.updated_at;
	`

	_, err := s.db.ExecContext(ctx, query,
		conv.ID, conv.SchemaVersion, conv.Title, conv.ProjectID,
		conv.ParentConversationID, conv.RootConversationID,
		conv.AssistantMode, conv.Archived, conv.Workspace,
		string(selData), string(msgsData), string(turnsData), string(actionsData),
		conv.CreatedAt, conv.UpdatedAt,
	)
	return err
}

// GetConversation loads a conversation by ID.
func (s *DB) GetConversation(ctx context.Context, id string) (*personalagent.Conversation, error) {
	query := `
	SELECT id, schema_version, title, project_id, parent_id, root_id,
	       assistant_mode, archived, workspace, selection_json,
	       messages_json, turns_json, assistant_actions_json,
	       created_at, updated_at
	FROM conversations WHERE id = ? LIMIT 1;
	`

	row := s.db.QueryRowContext(ctx, query, id)

	var conv personalagent.Conversation
	var selStr, msgsStr, turnsStr, actionsStr string

	err := row.Scan(
		&conv.ID, &conv.SchemaVersion, &conv.Title, &conv.ProjectID,
		&conv.ParentConversationID, &conv.RootConversationID,
		&conv.AssistantMode, &conv.Archived, &conv.Workspace,
		&selStr, &msgsStr, &turnsStr, &actionsStr,
		&conv.CreatedAt, &conv.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("conversation not found")
	}
	if err != nil {
		return nil, err
	}

	_ = json.Unmarshal([]byte(selStr), &conv.Selection)
	_ = json.Unmarshal([]byte(msgsStr), &conv.Messages)
	_ = json.Unmarshal([]byte(turnsStr), &conv.Turns)
	_ = json.Unmarshal([]byte(actionsStr), &conv.AssistantActions)

	return &conv, nil
}

// ListSummaries returns summaries of conversations.
func (s *DB) ListSummaries(ctx context.Context, includeArchived bool) ([]personalagent.ConversationSummary, error) {
	query := `
	SELECT id, assistant_mode, archived, project_id, parent_id, root_id,
	       title, updated_at, messages_json
	FROM conversations
	`
	if !includeArchived {
		query += " WHERE archived = FALSE"
	}
	query += " ORDER BY updated_at DESC;"

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []personalagent.ConversationSummary
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

		var msgs []personalagent.Message
		_ = json.Unmarshal([]byte(msgsJSON), &msgs)
		sum.MessageCount = len(msgs)

		summaries = append(summaries, sum)
	}

	return summaries, nil
}

// DeleteConversation deletes a conversation record.
func (s *DB) DeleteConversation(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "DELETE FROM conversations WHERE id = ?", id)
	return err
}

// ImportFromJSONDir auto-migrates existing json files from the legacy folder into SQLite.
func (s *DB) ImportFromJSONDir(ctx context.Context, jsonDir string) (int, error) {
	entries, err := os.ReadDir(jsonDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	imported := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(jsonDir, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var conv personalagent.Conversation
		if err := json.Unmarshal(data, &conv); err != nil {
			continue
		}

		if conv.ID != "" {
			if err := s.SaveConversation(ctx, &conv); err == nil {
				imported++
			}
		}
	}

	return imported, nil
}
