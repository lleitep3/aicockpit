package personalagent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const (
	copilotCatalogTimeout = 10 * time.Second
	copilotAdapterVersion = "copilot-jsonl-v1"
)

type CopilotAdapter struct {
	binary string
}

func NewCopilotAdapter(binary string) *CopilotAdapter {
	return &CopilotAdapter{binary: binary}
}

func (a *CopilotAdapter) Catalog(ctx context.Context) (Catalog, error) {
	catalogCtx, cancel := context.WithTimeout(ctx, copilotCatalogTimeout)
	defer cancel()

	cliVersion, _ := a.version(catalogCtx)
	cmd := exec.CommandContext(catalogCtx, a.binary, "help", "config")
	out, err := cmd.CombinedOutput()
	models := parseCopilotConfigModels(string(out))
	source := "copilot:help-config"
	if err != nil || len(models) == 0 {
		models = defaultCopilotModels()
		source = "copilot:fallback-catalog"
	}

	return Catalog{
		ProviderID:   "copilot",
		Source:       source,
		CLIVersion:   cliVersion,
		DiscoveredAt: time.Now().UTC(),
		Models:       models,
	}, nil
}

func parseCopilotConfigModels(raw string) []Model {
	efforts := []string{"minimal", "low", "medium", "high", "xhigh"}
	var models []Model
	seen := map[string]bool{
		"auto": true,
	}
	models = append(models, Model{
		ID:               "auto",
		Name:             "Copilot Auto (Fusion)",
		Description:      "Roteamento automático do GitHub Copilot CLI",
		Default:          true,
		Selectable:       true,
		ReasoningEfforts: efforts,
	})

	inModelSection := false
	for _, rawLine := range strings.Split(raw, "\n") {
		line := strings.TrimSpace(ansiEscapeRegex.ReplaceAllString(rawLine, ""))
		if strings.HasPrefix(line, "`model`:") {
			inModelSection = true
			continue
		}
		if !inModelSection {
			continue
		}
		if line == "" {
			if len(models) > 1 {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "`") {
			break
		}
		if strings.HasPrefix(line, "- \"") && strings.HasSuffix(line, "\"") {
			id := strings.TrimSuffix(strings.TrimPrefix(line, "- \""), "\"")
			id = strings.TrimSpace(id)
			if id == "" || seen[id] || !looksLikeModelID(id) {
				continue
			}
			seen[id] = true
			models = append(models, Model{
				ID:               id,
				Name:             formatCopilotModelName(id),
				Selectable:       true,
				ReasoningEfforts: efforts,
			})
		}
	}
	if len(models) <= 1 {
		return nil
	}
	return models
}

func formatCopilotModelName(id string) string {
	parts := strings.Split(id, "-")
	for i, p := range parts {
		switch strings.ToLower(p) {
		case "gpt":
			parts[i] = "GPT"
		case "claude":
			parts[i] = "Claude"
		case "gemini":
			parts[i] = "Gemini"
		case "mai":
			parts[i] = "MAI"
		default:
			if len(p) > 0 {
				parts[i] = strings.ToUpper(p[:1]) + p[1:]
			}
		}
	}
	return strings.Join(parts, " ")
}

func defaultCopilotModels() []Model {
	efforts := []string{"minimal", "low", "medium", "high", "xhigh"}
	ids := []string{
		"claude-sonnet-5.5",
		"claude-opus-5.5",
		"gpt-6.1-sol",
		"gpt-5.4",
		"gpt-5.3-codex",
		"gemini-4-argon",
		"mai-code-1.1-flash",
	}
	models := make([]Model, 0, len(ids)+1)
	models = append(models, Model{
		ID:               "auto",
		Name:             "Copilot Auto (Fusion)",
		Description:      "Roteamento automático do GitHub Copilot CLI",
		Default:          true,
		Selectable:       true,
		ReasoningEfforts: efforts,
	})
	for _, id := range ids {
		models = append(models, Model{
			ID:               id,
			Name:             formatCopilotModelName(id),
			Selectable:       true,
			ReasoningEfforts: efforts,
		})
	}
	return models
}

func (a *CopilotAdapter) RunTurn(
	ctx context.Context,
	request RunRequest,
	onDelta func(string),
	_ func(ApprovalRequest) (string, error),
) (RunResult, error) {
	cliVersion, _ := a.version(ctx)
	result := RunResult{
		CLIVersion:      cliVersion,
		NativeSessionID: request.NativeSessionID,
	}

	args := []string{
		"--output-format", "json",
		"--stream", "on",
		"--no-ask-user",
	}
	if request.ModelID != "" {
		args = append(args, "--model", request.ModelID)
	}
	if request.ReasoningEffort != "" {
		args = append(args, "--reasoning-effort", request.ReasoningEffort)
	}
	if request.NativeSessionID != "" {
		args = append(args, "--resume="+request.NativeSessionID)
	}
	if request.Workspace != "" {
		args = append(args, "--add-dir", request.Workspace)
	}
	if request.ApprovalPolicy == "never" {
		args = append(args, "--allow-all-tools")
	} else {
		args = append(args, "--allow-tool=write")
	}
	args = append(args, "-p", request.Prompt)

	cmd := exec.CommandContext(ctx, a.binary, args...)
	if request.Workspace != "" {
		cmd.Dir = request.Workspace
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("open Copilot CLI output: %w", err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("start Copilot CLI: %w", err)
	}

	var streamedAnyDelta bool
	var finalMessageContent string
	var streamErr error

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 32*1024), 8*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var evt copilotStreamEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		switch evt.Type {
		case "session.auto_mode_resolved":
			if evt.Data.ChosenModel != "" {
				result.Events = append(result.Events, ExecutionEvent{
					Type:       "copilot/auto_mode_resolved",
					ItemID:     evt.Data.ChosenModel,
					OccurredAt: time.Now().UTC(),
				})
				if request.OnStatus != nil {
					request.OnStatus(fmt.Sprintf("Copilot Auto → %s…", evt.Data.ChosenModel))
				}
			}
		case "model.call_start":
			if evt.Data.TurnID != "" {
				result.NativeTurnID = evt.Data.TurnID
			}
			result.Events = append(result.Events, ExecutionEvent{
				Type:       "copilot/model.call_start",
				ItemID:     evt.Data.Model,
				OccurredAt: time.Now().UTC(),
			})
			if request.OnStatus != nil {
				request.OnStatus(fmt.Sprintf("Copilot gerando resposta (%s)…", evt.Data.Model))
			}
		case "assistant.message_delta":
			if evt.Data.DeltaContent != "" {
				streamedAnyDelta = true
				onDelta(evt.Data.DeltaContent)
			}
		case "assistant.message":
			if evt.Data.Content != "" {
				finalMessageContent = evt.Data.Content
			}
		case "model.call_finished":
			result.Events = append(result.Events, ExecutionEvent{
				Type:       "copilot/model.call_finished",
				ItemID:     evt.Data.TurnID,
				Decision:   evt.Data.Outcome,
				OccurredAt: time.Now().UTC(),
			})
		case "result":
			if evt.SessionID != "" {
				result.NativeSessionID = evt.SessionID
			}
			if evt.ExitCode != nil && *evt.ExitCode != 0 {
				streamErr = fmt.Errorf("Copilot CLI exited with code %d", *evt.ExitCode)
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil && !errors.Is(scanErr, io.EOF) {
		streamErr = scanErr
	}

	waitErr := cmd.Wait()
	if !streamedAnyDelta && finalMessageContent != "" {
		onDelta(finalMessageContent)
	}
	if streamErr != nil {
		return result, streamErr
	}
	if waitErr != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return result, fmt.Errorf("Copilot CLI error: %s", errMsg)
		}
		return result, fmt.Errorf("Copilot CLI exited with error: %w", waitErr)
	}
	return result, nil
}

func (a *CopilotAdapter) version(ctx context.Context) (string, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(versionCtx, a.binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("read Copilot CLI version: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0]), nil
	}
	return strings.TrimSpace(string(output)), nil
}

type copilotStreamEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	ExitCode  *int   `json:"exitCode"`
	Data      struct {
		ChosenModel  string `json:"chosenModel"`
		Model        string `json:"model"`
		TurnID       string `json:"turnId"`
		MessageID    string `json:"messageId"`
		DeltaContent string `json:"deltaContent"`
		Content      string `json:"content"`
		Outcome      string `json:"outcome"`
	} `json:"data"`
}
