package personalagent

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCopilotConfigModels(t *testing.T) {
	raw := `Configuration Settings:
  ` + "`model`: AI model to use for Copilot CLI; can be changed with /model command or --model flag option.\n" +
		`    - "claude-sonnet-5.5"
    - "gpt-6.1-sol"
    - "mai-code-1.1-flash"
    - "gemini-4-argon"

  ` + "`contextTier`: context window tier\n"

	models := parseCopilotConfigModels(raw)
	if len(models) != 5 {
		t.Fatalf("expected 5 models (auto + 4), got %d: %+v", len(models), models)
	}
	if models[0].ID != "auto" || !models[0].Default {
		t.Fatalf("expected first model auto default=true, got %+v", models[0])
	}
	if models[1].ID != "claude-sonnet-5.5" {
		t.Fatalf("expected second model claude-sonnet-5.5, got %+v", models[1])
	}
}

func TestCopilotAdapterRunTurnStreamJSON(t *testing.T) {
	tmp := t.TempDir()
	scriptPath := filepath.Join(tmp, "fake-copilot.sh")
	script := `#!/usr/bin/env bash
set -euo pipefail
for arg in "$@"; do
  if [[ "$arg" == "--version" ]]; then
    echo "GitHub Copilot CLI 1.0.103."
    exit 0
  fi
done
echo '{"type":"session.auto_mode_resolved","data":{"chosenModel":"mai-code-1.1-flash"}}'
echo '{"type":"model.call_start","data":{"turnId":"0","model":"mai-code-1.1-flash"}}'
echo '{"type":"assistant.message_delta","data":{"messageId":"msg-1","deltaContent":"Olá do "}}'
echo '{"type":"assistant.message_delta","data":{"messageId":"msg-1","deltaContent":"Copilot!"}}'
echo '{"type":"model.call_finished","data":{"turnId":"0","outcome":"success"}}'
echo '{"type":"assistant.message","data":{"messageId":"msg-1","content":"Olá do Copilot!"}}'
echo '{"type":"result","sessionId":"805167dd-2f81-4c07-a2d6-3ff5274bed4a","exitCode":0}'
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake script: %v", err)
	}

	adapter := NewCopilotAdapter(scriptPath)
	var deltas []string
	result, err := adapter.RunTurn(
		context.Background(),
		RunRequest{
			TurnID:         "turn_1",
			ProviderID:     "copilot",
			ModelID:        "auto",
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
	if result.NativeSessionID != "805167dd-2f81-4c07-a2d6-3ff5274bed4a" {
		t.Fatalf("expected session 805167dd-2f81-4c07-a2d6-3ff5274bed4a, got %q", result.NativeSessionID)
	}
	if result.CLIVersion != "GitHub Copilot CLI 1.0.103." {
		t.Fatalf("unexpected CLI version %q", result.CLIVersion)
	}
	if strings.Join(deltas, "") != "Olá do Copilot!" {
		t.Fatalf("expected deltas 'Olá do Copilot!', got %q", strings.Join(deltas, ""))
	}
	if len(result.Events) != 3 {
		t.Fatalf("expected 3 execution events, got %d: %+v", len(result.Events), result.Events)
	}
}
