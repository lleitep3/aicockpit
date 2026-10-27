package personalagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseAntigravityModelsOutput(t *testing.T) {
	raw := "\x1b[?25l⠋ Fetching models...\r\x1b[2K\x1b[?25h" +
		"gemini-4-argon-medium              Gemini 3.1 Pro (Medium)\n" +
		"gemini-4-argon-high                Gemini 3.1 Pro (High)\n" +
		"gemini-3.8-flash-high              Gemini 3 Flash (High)\n" +
		"claude-sonnet-5-5-medium           Claude Sonnet 4.6 (Medium)\n"

	models := parseAntigravityModelsOutput(raw)
	if len(models) != 4 {
		t.Fatalf("expected 4 models, got %d: %+v", len(models), models)
	}
	if models[0].ID != "gemini-4-argon-medium" || !models[0].Default {
		t.Fatalf("expected first model gemini-4-argon-medium default=true, got %+v", models[0])
	}
	if models[0].Name != "Gemini 3.1 Pro (Medium)" {
		t.Fatalf("unexpected model name %q", models[0].Name)
	}
}

func TestAntigravityAdapterRunTurnStreamJSON(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "fake-agy.sh")
	script := `#!/usr/bin/env bash
set -euo pipefail
for arg in "$@"; do
  if [[ "$arg" == "--version" ]]; then
    echo "agy 1.4.8"
    exit 0
  fi
done
echo '{"event":"init","conversation_id":"conv-agy-123","init":{"model":"gemini-4-argon-medium","cwd":"/tmp","permission_mode":"auto"}}'
echo '{"event":"step_update","step_update":{"conversation_id":"conv-agy-123","step_index":1,"state":"ACTIVE","step_type":"agent_response","text_delta":"Olá do "}}'
echo '{"event":"step_update","step_update":{"conversation_id":"conv-agy-123","step_index":1,"state":"DONE","step_type":"agent_response","text_delta":"Antigravity!"}}'
echo '{"event":"result","result":{"conversation_id":"conv-agy-123","status":"SUCCESS","response":"Olá do Antigravity!"}}'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}

	adapter := NewAntigravityAdapter(scriptPath)
	var deltas []string
	result, err := adapter.RunTurn(
		context.Background(),
		RunRequest{
			TurnID:         "turn_1",
			ProviderID:     "antigravity",
			ModelID:        "gemini-4-argon-medium",
			ApprovalPolicy: "never",
			Workspace:      tmp,
			Prompt:         "Diga olá",
		},
		func(text string) {
			deltas = append(deltas, text)
		},
		nil,
	)
	if err != nil {
		t.Fatalf("RunTurn failed: %v", err)
	}
	if result.NativeSessionID != "conv-agy-123" {
		t.Fatalf("expected session conv-agy-123, got %q", result.NativeSessionID)
	}
	if result.CLIVersion != "agy 1.4.8" {
		t.Fatalf("expected CLI version 'agy 1.4.8', got %q", result.CLIVersion)
	}
	if strings.Join(deltas, "") != "Olá do Antigravity!" {
		t.Fatalf("expected deltas 'Olá do Antigravity!', got %q", strings.Join(deltas, ""))
	}
	if len(result.Events) != 2 {
		t.Fatalf("expected 2 execution events, got %d: %+v", len(result.Events), result.Events)
	}
}
