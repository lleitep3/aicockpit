package personalagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	devinAdapterVersion = "1"
	devinCatalogTimeout = 20 * time.Second
)

type DevinAdapter struct {
	binary string
}

func NewDevinAdapter(binary string) *DevinAdapter {
	return &DevinAdapter{binary: binary}
}

func (a *DevinAdapter) Catalog(ctx context.Context) (Catalog, error) {
	catalogCtx, cancel := context.WithTimeout(ctx, devinCatalogTimeout)
	defer cancel()

	version, err := a.version(catalogCtx)
	if err != nil {
		return Catalog{}, err
	}
	cmd := exec.CommandContext(catalogCtx, a.binary, "models", "list", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		if catalogCtx.Err() != nil {
			return Catalog{}, catalogCtx.Err()
		}
		return Catalog{}, fmt.Errorf("query Devin model catalog: %w; check Devin CLI authentication and retry", err)
	}
	models, err := parseDevinModels(output)
	if err != nil {
		return Catalog{}, err
	}
	return Catalog{
		ProviderID:   "devin",
		Source:       "devin-cli:models-list-json",
		CLIVersion:   version,
		DiscoveredAt: time.Now().UTC(),
		Models:       models,
	}, nil
}

func parseDevinModels(data []byte) ([]Model, error) {
	var response struct {
		Families []struct {
			Label    string `json:"family_label"`
			Variants []struct {
				ID          string `json:"model_uid"`
				Name        string `json:"label"`
				Description string `json:"description"`
			} `json:"variants"`
		} `json:"families"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode Devin model catalog: %w", err)
	}

	models := make([]Model, 0)
	seen := make(map[string]struct{})
	for _, family := range response.Families {
		for _, variant := range family.Variants {
			id := strings.TrimSpace(variant.ID)
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}

			name := strings.TrimSpace(variant.Name)
			if name == "" {
				name = id
			}
			description := strings.TrimSpace(variant.Description)
			if description == "" {
				description = strings.TrimSpace(family.Label)
			}
			models = append(models, Model{
				ID:          id,
				Name:        name,
				Description: description,
				Selectable:  true,
			})
		}
	}
	if len(models) == 0 {
		return nil, errors.New("Devin model catalog contains no models")
	}
	return models, nil
}

func (a *DevinAdapter) RunTurn(
	ctx context.Context,
	request RunRequest,
	onDelta func(string),
	_ func(ApprovalRequest) (string, error),
) (RunResult, error) {
	result := RunResult{NativeSessionID: request.NativeSessionID}
	if request.ReasoningEffort != "" {
		return result, errors.New("Devin CLI does not expose a separate reasoning effort setting")
	}

	permissionMode := "auto"
	switch request.ApprovalPolicy {
	case "", "on-request":
	case "never":
		permissionMode = "dangerous"
	default:
		return result, fmt.Errorf("unsupported Devin approval policy %q", request.ApprovalPolicy)
	}

	version, err := a.version(ctx)
	if err != nil {
		return result, err
	}
	result.CLIVersion = version

	args := make([]string, 0, 10)
	if request.ModelID != "" {
		args = append(args, "--model", request.ModelID)
	}
	args = append(args, "--permission-mode", permissionMode)
	if request.NativeSessionID != "" {
		args = append(args, "--resume", request.NativeSessionID)
	}
	args = append(args, "--print", "--", request.Prompt)

	cmd := exec.CommandContext(ctx, a.binary, args...)
	if request.Workspace != "" {
		cmd.Dir = request.Workspace
	}
	if request.OnStatus != nil {
		request.OnStatus(fmt.Sprintf("Devin CLI · %s", request.ModelID))
	}
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return result, fmt.Errorf("Devin CLI exited with status %d; check workspace trust and CLI authentication", exitErr.ExitCode())
		}
		return result, fmt.Errorf("run Devin CLI: %w", err)
	}
	response := strings.TrimSpace(string(output))
	if response == "" {
		return result, errors.New("Devin CLI returned an empty response")
	}
	onDelta(response)
	return result, nil
}

func (a *DevinAdapter) version(ctx context.Context) (string, error) {
	versionCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(versionCtx, a.binary, "--version").Output()
	if err != nil {
		if versionCtx.Err() != nil {
			return "", versionCtx.Err()
		}
		return "", fmt.Errorf("read Devin CLI version: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
