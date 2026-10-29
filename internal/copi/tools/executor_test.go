package tools_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lleitep3/aicockpit/internal/copi/storage"
	"github.com/lleitep3/aicockpit/internal/copi/tools"
	"github.com/lleitep3/aicockpit/internal/personalagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolCalls(t *testing.T) {
	text := `Vou pesquisar sobre isso:
TOOL_CALL: search_kb {"query": "partilhar aba"}
E também checar git:
TOOL_CALL: run_command {"command": "git status -s"}
TOOL_CALL: search_history {"query": "teste"}
TOOL_CALL: empty_tool`

	calls := tools.ParseToolCalls(text)
	require.Len(t, calls, 4)

	assert.Equal(t, "search_kb", calls[0].Name)
	assert.JSONEq(t, `{"query": "partilhar aba"}`, string(calls[0].Args))

	assert.Equal(t, "run_command", calls[1].Name)
	assert.JSONEq(t, `{"command": "git status -s"}`, string(calls[1].Args))

	assert.Equal(t, "search_history", calls[2].Name)
	assert.JSONEq(t, `{"query": "teste"}`, string(calls[2].Args))

	assert.Equal(t, "empty_tool", calls[3].Name)
	assert.JSONEq(t, `{}`, string(calls[3].Args))
}

func TestExecutorRunCommand(t *testing.T) {
	exec := tools.NewExecutor(nil)
	ctx := context.Background()

	// Allowed command
	out, err := exec.RunCommand(ctx, "git status")
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Disallowed command
	out, err = exec.RunCommand(ctx, "rm -rf /")
	assert.NoError(t, err)
	assert.Contains(t, out, "não permitido por segurança")

	// Empty command
	out, err = exec.RunCommand(ctx, "")
	assert.NoError(t, err)
	assert.Equal(t, "Nenhum comando fornecido.", out)

	// Command with no output
	out, err = exec.RunCommand(ctx, "ps > /dev/null")
	assert.NoError(t, err)
	assert.Contains(t, out, "código 0 sem saída")
}

func TestExecutorSearchKB(t *testing.T) {
	exec := tools.NewExecutor(nil)
	ctx := context.Background()

	// Empty query
	out, err := exec.SearchKB(ctx, "", 0)
	assert.NoError(t, err)
	assert.Contains(t, out, "Nenhuma query informada")

	// Valid query
	out, err = exec.SearchKB(ctx, "partilhar", 2)
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// Non-existent query
	out, err = exec.SearchKB(ctx, "xyzabc123nonexistent", 2)
	assert.NoError(t, err)
	assert.Contains(t, out, "Nenhum documento encontrado")
}

func TestExecutorSearchConversations(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(tempDB)
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	_ = db.SaveConversation(ctx, &personalagent.Conversation{
		ID:    "c1",
		Title: "Discussão sobre partilhar",
	})

	exec := tools.NewExecutor(db)

	out, err := exec.SearchConversations(ctx, "partilhar")
	assert.NoError(t, err)
	assert.Contains(t, out, "c1")

	outNotFound, err := exec.SearchConversations(ctx, "inexistente")
	assert.NoError(t, err)
	assert.Contains(t, outNotFound, "Nenhuma conversa encontrada")

	// Nil DB
	nilExec := tools.NewExecutor(nil)
	outNil, err := nilExec.SearchConversations(ctx, "foo")
	assert.NoError(t, err)
	assert.Contains(t, outNil, "não disponível")
}

func TestExecutorExecute(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.Open(tempDB)
	require.NoError(t, err)
	defer db.Close()

	exec := tools.NewExecutor(db)
	ctx := context.Background()

	// search_kb
	out, err := exec.Execute(ctx, "search_kb", []byte(`{"query": "partilhar", "limit": 2}`))
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// search_conversations
	out, err = exec.Execute(ctx, "search_conversations", []byte(`{"query": "partilhar"}`))
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// run_command
	out, err = exec.Execute(ctx, "run_command", []byte(`{"command": "git status -s"}`))
	assert.NoError(t, err)
	assert.NotEmpty(t, out)

	// invalid json for all
	_, err = exec.Execute(ctx, "search_kb", []byte(`invalid`))
	assert.Error(t, err)

	_, err = exec.Execute(ctx, "search_conversations", []byte(`invalid`))
	assert.Error(t, err)

	_, err = exec.Execute(ctx, "run_command", []byte(`invalid`))
	assert.Error(t, err)

	// unknown tool
	_, err = exec.Execute(ctx, "unknown_tool", []byte(`{}`))
	assert.Error(t, err)
}
