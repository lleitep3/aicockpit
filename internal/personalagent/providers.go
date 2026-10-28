package personalagent

import (
	"context"
	"fmt"
	"os/exec"
	"sync"
)

type Adapter interface {
	Catalog(context.Context) (Catalog, error)
	RunTurn(context.Context, RunRequest, func(string), func(ApprovalRequest) (string, error)) (RunResult, error)
}

type ProviderRegistry struct {
	quotaMu    sync.Mutex
	quotas     map[string]ProviderQuota
	quotaCalls map[string]*quotaCall
	adapters   map[string]Adapter
	paths      map[string]string
}

func NewProviderRegistry() *ProviderRegistry {
	candidates := []struct {
		id       string
		binaries []string
	}{
		{id: "codex", binaries: []string{"codex"}},
		{id: "devin", binaries: []string{"devin"}},
		{id: "claude", binaries: []string{"claude"}},
		{id: "copilot", binaries: []string{"copilot"}},
		{id: "goose", binaries: []string{"goose"}},
		{id: "antigravity", binaries: []string{"agy", "antigravity"}},
	}
	paths := make(map[string]string, len(candidates))
	for _, candidate := range candidates {
		for _, bin := range candidate.binaries {
			if path, err := exec.LookPath(bin); err == nil {
				paths[candidate.id] = path
				break
			}
		}
	}
	registry := &ProviderRegistry{
		adapters: make(map[string]Adapter),
		paths:    paths,
	}
	if path := paths["codex"]; path != "" {
		registry.adapters["codex"] = NewCodexAdapter(path)
	}
	if path := paths["devin"]; path != "" {
		registry.adapters["devin"] = NewDevinAdapter(path)
	}
	if path := paths["antigravity"]; path != "" {
		registry.adapters["antigravity"] = NewAntigravityAdapter(path)
	}
	if path := paths["copilot"]; path != "" {
		registry.adapters["copilot"] = NewCopilotAdapter(path)
	}
	return registry
}

func (r *ProviderRegistry) List() []Provider {
	providers := []Provider{
		{ID: "codex", Name: "OpenAI Codex", Capabilities: []string{"models", "streaming", "history", "approvals", "images", "tools"}},
		{ID: "devin", Name: "Devin CLI", Capabilities: []string{"models"}},
		{ID: "antigravity", Name: "Antigravity (AGY)", Capabilities: []string{"models", "streaming", "history"}},
		{ID: "copilot", Name: "GitHub Copilot CLI", Capabilities: []string{"models", "streaming", "history"}},
		{ID: "claude", Name: "Claude Code", Capabilities: []string{}},
		{ID: "goose", Name: "Goose", Capabilities: []string{}},
	}
	for i := range providers {
		provider := &providers[i]
		provider.Executable = r.paths[provider.ID]
		provider.Detected = provider.Executable != ""
		provider.Supported = r.adapters[provider.ID] != nil
		switch {
		case provider.Supported:
			provider.Status = "detected"
		case provider.Detected:
			provider.Status = "adapter_unavailable"
		default:
			provider.Status = "not_installed"
		}
	}
	return providers
}

func (r *ProviderRegistry) Catalog(ctx context.Context, providerID string) (Catalog, error) {
	if err := r.requireQuota(providerID); err != nil {
		return Catalog{}, err
	}
	adapter := r.adapters[providerID]
	if adapter == nil {
		return Catalog{}, fmt.Errorf("provider %q has no conversation adapter", providerID)
	}
	return adapter.Catalog(ctx)
}

func (r *ProviderRegistry) Adapter(providerID string) (Adapter, bool) {
	adapter, ok := r.adapters[providerID]
	return adapter, ok
}
