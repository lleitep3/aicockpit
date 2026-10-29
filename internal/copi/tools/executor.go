package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/lleitep3/aicockpit/internal/copi/storage"
)

// Executor manages tool execution for Copi.
type Executor struct {
	db *storage.DB
}

// NewExecutor creates a new tool Executor.
func NewExecutor(db *storage.DB) *Executor {
	return &Executor{db: db}
}

// ToolCall represents a requested tool invocation parsed from model output.
type ToolCall struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

// SearchKBArgs arguments for search_kb.
type SearchKBArgs struct {
	Query string `json:"query"`
	Limit int    `json:"limit,omitempty"`
}

// SearchConversationsArgs arguments for search_conversations.
type SearchConversationsArgs struct {
	Query string `json:"query"`
}

// RunCommandArgs arguments for run_command.
type RunCommandArgs struct {
	Command string `json:"command"`
}

// Execute processes a tool call by name and args, returning formatted text output.
func (e *Executor) Execute(ctx context.Context, name string, rawArgs []byte) (string, error) {
	switch name {
	case "search_kb":
		var args SearchKBArgs
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return "", fmt.Errorf("invalid search_kb args: %w", err)
		}
		return e.SearchKB(ctx, args.Query, args.Limit)

	case "search_conversations", "search_history":
		var args SearchConversationsArgs
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return "", fmt.Errorf("invalid search_conversations args: %w", err)
		}
		return e.SearchConversations(ctx, args.Query)

	case "run_command":
		var args RunCommandArgs
		if err := json.Unmarshal(rawArgs, &args); err != nil {
			return "", fmt.Errorf("invalid run_command args: %w", err)
		}
		return e.RunCommand(ctx, args.Command)

	default:
		return "", fmt.Errorf("unknown tool %q", name)
	}
}

// SearchKB queries the Cockpit knowledge base via 'cockpit kb search --bm25'.
func (e *Executor) SearchKB(ctx context.Context, query string, limit int) (string, error) {
	if query == "" {
		return "Nenhuma query informada para busca na KB.", nil
	}
	if limit <= 0 {
		limit = 5
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "cockpit", "kb", "search", query, "--bm25", "--limit", fmt.Sprintf("%d", limit))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Erro ao consultar KB: %v (saída: %s)", err, string(out)), nil
	}

	result := strings.TrimSpace(string(out))
	if result == "" || strings.Contains(result, "Found: 0 documents") || strings.Contains(strings.ToLower(result), "no documents found") {
		return fmt.Sprintf("Nenhum documento encontrado na KB para %q.", query), nil
	}

	return result, nil
}

// SearchConversations queries the SQLite conversations database.
func (e *Executor) SearchConversations(ctx context.Context, query string) (string, error) {
	if e.db == nil {
		return "Banco de conversas SQLite não disponível.", nil
	}
	matches, err := e.db.SearchConversations(ctx, query)
	if err != nil {
		return fmt.Sprintf("Erro ao pesquisar conversas: %v", err), nil
	}
	if len(matches) == 0 {
		return fmt.Sprintf("Nenhuma conversa encontrada falando sobre %q.", query), nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Encontrada(s) %d conversa(s) sobre %q:\n", len(matches), query))
	for _, m := range matches {
		b.WriteString(fmt.Sprintf("• [%s] \"%s\"\n", m.ID, m.Title))
	}
	return b.String(), nil
}

// RunCommand executes a safe, read-only system inspection command (git, gh, ps, df, etc).
func (e *Executor) RunCommand(ctx context.Context, command string) (string, error) {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return "Nenhum comando fornecido.", nil
	}

	// Security filter: only allow inspection / status commands
	allowedPrefixes := []string{"git ", "gh ", "cockpit ", "df ", "free ", "uptime", "ps "}
	allowed := false
	for _, p := range allowedPrefixes {
		if strings.HasPrefix(trimmed, p) || trimmed == strings.TrimSpace(p) {
			allowed = true
			break
		}
	}
	if !allowed {
		return fmt.Sprintf("Comando não permitido por segurança: %q. Permitidos apenas comandos de inspeção (git, gh, cockpit, df, etc).", trimmed), nil
	}

	cmdCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", trimmed)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("Erro na execução: %v\nSaída:\n%s", err, string(out)), nil
	}

	outputStr := strings.TrimSpace(string(out))
	if outputStr == "" {
		return "(Comando executado com código 0 sem saída)", nil
	}
	// Limit output to prevent huge context blast
	if len(outputStr) > 4000 {
		outputStr = outputStr[:4000] + "\n...(saída truncada para economizar tokens)..."
	}
	return outputStr, nil
}

// ParseToolCalls inspects text for tool call declarations in the format:
// TOOL_CALL: tool_name {"arg": "val"}
func ParseToolCalls(text string) []ToolCall {
	var calls []ToolCall
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "TOOL_CALL:") {
			rest := strings.TrimSpace(strings.TrimPrefix(line, "TOOL_CALL:"))
			parts := strings.SplitN(rest, " ", 2)
			name := parts[0]
			var argsJSON []byte
			if len(parts) > 1 {
				argsJSON = []byte(strings.TrimSpace(parts[1]))
			} else {
				argsJSON = []byte("{}")
			}
			calls = append(calls, ToolCall{
				Name: name,
				Args: argsJSON,
			})
		}
	}
	return calls
}
