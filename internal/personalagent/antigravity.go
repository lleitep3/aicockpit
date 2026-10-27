package personalagent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const antigravityAdapterVersion = "1"

var ansiEscapeRegex = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]`)

type AntigravityAdapter struct {
	binary string
}

func NewAntigravityAdapter(binary string) *AntigravityAdapter {
	return &AntigravityAdapter{binary: binary}
}

func (a *AntigravityAdapter) Catalog(ctx context.Context) (Catalog, error) {
	version, _ := a.version(ctx)
	catCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(catCtx, a.binary, "models")
	output, err := cmd.CombinedOutput()
	models := parseAntigravityModelsOutput(string(output))
	source := "agy:models"
	if err != nil || len(models) == 0 {
		models = defaultAntigravityModels()
		source = "agy:fallback-catalog"
	}
	return Catalog{
		ProviderID:   "antigravity",
		Source:       source,
		CLIVersion:   version,
		DiscoveredAt: time.Now().UTC(),
		Models:       models,
	}, nil
}

func parseAntigravityModelsOutput(raw string) []Model {
	clean := ansiEscapeRegex.ReplaceAllString(raw, "")
	clean = strings.ReplaceAll(clean, "\r", "\n")
	lines := strings.Split(clean, "\n")
	models := make([]Model, 0, len(lines))
	seen := make(map[string]struct{})
	efforts := []string{"low", "medium", "high", "xhigh"}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "Fetching available models") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		if !looksLikeModelID(id) {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(strings.TrimPrefix(trimmed, id))
		if name == "" {
			name = id
		}
		models = append(models, Model{
			ID:               id,
			Name:             name,
			Default:          len(models) == 0,
			Selectable:       true,
			ReasoningEfforts: efforts,
		})
	}
	return models
}

func looksLikeModelID(id string) bool {
	if len(id) < 4 || !strings.Contains(id, "-") {
		return false
	}
	for _, r := range id {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' || r == '_'
		if !valid {
			return false
		}
	}
	return true
}

func defaultAntigravityModels() []Model {
	efforts := []string{"low", "medium", "high", "xhigh"}
	return []Model{
		{ID: "gemini-4-argon-medium", Name: "Gemini 4 Argon (Medium)", Default: true, Selectable: true, ReasoningEfforts: efforts},
		{ID: "gemini-4-argon-high", Name: "Gemini 4 Argon (High)", Selectable: true, ReasoningEfforts: efforts},
		{ID: "gemini-4-argon-low", Name: "Gemini 4 Argon (Low)", Selectable: true, ReasoningEfforts: efforts},
		{ID: "gemini-3.8-flash-high", Name: "Gemini 3.8 Flash (High)", Selectable: true, ReasoningEfforts: efforts},
		{ID: "gemini-3.8-flash-medium", Name: "Gemini 3.8 Flash (Medium)", Selectable: true, ReasoningEfforts: efforts},
		{ID: "gemini-3.8-flash-low", Name: "Gemini 3.8 Flash (Low)", Selectable: true, ReasoningEfforts: efforts},
		{ID: "claude-sonnet-5-5-medium", Name: "Claude Sonnet 5.5 (Medium)", Selectable: true, ReasoningEfforts: efforts},
		{ID: "claude-opus-5-5-medium", Name: "Claude Opus 5.5 (Medium)", Selectable: true, ReasoningEfforts: efforts},
	}
}

func (a *AntigravityAdapter) RunTurn(
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

	args := []string{"--output-format=stream-json"}
	if request.ModelID != "" {
		args = append(args, "--model="+request.ModelID)
	}
	if request.ReasoningEffort != "" {
		args = append(args, "--effort="+request.ReasoningEffort)
	}
	if request.NativeSessionID != "" {
		args = append(args, "--conversation="+request.NativeSessionID)
	}
	if request.Workspace != "" {
		args = append(args, "--add-dir="+request.Workspace)
	}
	if request.ApprovalPolicy == "never" {
		args = append(args, "--dangerously-skip-permissions", "--mode=auto")
	} else {
		args = append(args, "--mode=accept-edits")
	}
	args = append(args, "--print="+request.Prompt)

	cmd := exec.CommandContext(ctx, a.binary, args...)
	if request.Workspace != "" {
		cmd.Dir = request.Workspace
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return result, fmt.Errorf("open Antigravity output: %w", err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return result, fmt.Errorf("start Antigravity CLI: %w", err)
	}

	var streamedAnyDelta bool
	var finalResultResponse string
	var streamErr error

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 32*1024), 4*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var evt agyStreamEvent
		if err := json.Unmarshal([]byte(line), &evt); err != nil {
			continue
		}
		if evt.ConversationID != "" {
			result.NativeSessionID = evt.ConversationID
		}
		switch evt.Event {
		case "init":
			result.Events = append(result.Events, ExecutionEvent{
				Type:       "agy/init",
				ItemID:     evt.Init.Model,
				OccurredAt: time.Now().UTC(),
			})
			if request.OnStatus != nil {
				request.OnStatus(fmt.Sprintf("Antigravity iniciado (%s)…", evt.Init.Model))
			}
		case "step_update":
			if evt.StepUpdate.ConversationID != "" {
				result.NativeSessionID = evt.StepUpdate.ConversationID
			}
			if evt.StepUpdate.TextDelta != "" {
				streamedAnyDelta = true
				onDelta(evt.StepUpdate.TextDelta)
			}
			if request.OnStatus != nil && evt.StepUpdate.StepType != "" {
				request.OnStatus(fmt.Sprintf("Antigravity · passo %d (%s)…", evt.StepUpdate.StepIndex, evt.StepUpdate.StepType))
			}
			if evt.StepUpdate.StepType != "" && evt.StepUpdate.State == "DONE" {
				result.NativeTurnID = fmt.Sprintf("step_%d", evt.StepUpdate.StepIndex)
				result.Events = append(result.Events, ExecutionEvent{
					Type:       "agy/" + evt.StepUpdate.StepType,
					ItemID:     result.NativeTurnID,
					Decision:   evt.StepUpdate.State,
					OccurredAt: time.Now().UTC(),
				})
			}
		case "result":
			if evt.Result.ConversationID != "" {
				result.NativeSessionID = evt.Result.ConversationID
			}
			finalResultResponse = evt.Result.Response
			if evt.Result.Status != "" && evt.Result.Status != "SUCCESS" {
				streamErr = fmt.Errorf("Antigravity turn ended with status %s", evt.Result.Status)
			}
		}
	}
	if scanErr := scanner.Err(); scanErr != nil && !errors.Is(scanErr, io.EOF) {
		streamErr = scanErr
	}

	waitErr := cmd.Wait()
	if !streamedAnyDelta && finalResultResponse != "" {
		onDelta(finalResultResponse)
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
			return result, fmt.Errorf("Antigravity CLI error: %s", errMsg)
		}
		return result, fmt.Errorf("Antigravity CLI exited with error: %w", waitErr)
	}
	return result, nil
}

func (a *AntigravityAdapter) version(ctx context.Context) (string, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(versionCtx, a.binary, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("read Antigravity CLI version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

type agyStreamEvent struct {
	Event          string `json:"event"`
	ConversationID string `json:"conversation_id"`
	Init           struct {
		Model          string `json:"model"`
		Cwd            string `json:"cwd"`
		PermissionMode string `json:"permission_mode"`
	} `json:"init"`
	StepUpdate struct {
		ConversationID string `json:"conversation_id"`
		StepIndex      int    `json:"step_index"`
		State          string `json:"state"`
		StepType       string `json:"step_type"`
		TextDelta      string `json:"text_delta"`
	} `json:"step_update"`
	Result struct {
		ConversationID string `json:"conversation_id"`
		Status         string `json:"status"`
		Response       string `json:"response"`
	} `json:"result"`
}
