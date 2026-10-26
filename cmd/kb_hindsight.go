package cmd

import (
	"context"
	"strings"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/hindsight"
	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/memoryhindsight"
	"github.com/lleitep3/aicockpit/internal/services"
)

// newMemoryContributor builds the optional memory-hindsight KB extension.
// Tests replace it. A nil contributor leaves search and index on the local KB.
var newMemoryContributor = memoryContributor

// newHindsightIntegration opens the Hindsight client. Tests replace it.
var newHindsightIntegration = func(settings hindsight.Settings, id memoryhindsight.Identity, token string) (kb.Contributor, error) {
	return memoryhindsight.Open(settings, id, token)
}

// readHindsightAPIToken reads the optional API bearer token. Tests replace it.
var readHindsightAPIToken = func(namespace, key string) (string, error) {
	return vaultHindsightSecrets{}.Get(context.Background(), namespace, key)
}

func attachMemoryContributor(svc services.KBService, cfg *config.Config) {
	if svc == nil || newMemoryContributor == nil {
		return
	}
	svc.AddContributors(newMemoryContributor(cfg))
}

func memoryContributor(cfg *config.Config) kb.Contributor {
	if cfg == nil || !cfg.Memory.Hindsight.Enabled {
		return nil
	}
	token, err := hindsightAPIToken(cfg.Memory.Hindsight)
	if err != nil {
		return memoryhindsight.Failed(err)
	}
	contributor, err := newHindsightIntegration(cfg.Memory.Hindsight, memoryhindsight.CurrentIdentity(), token)
	if err != nil {
		return memoryhindsight.Failed(err)
	}
	if contributor == nil {
		return nil
	}
	if gate, ok := contributor.(interface{ Enabled() bool }); ok && !gate.Enabled() {
		return nil
	}
	return contributor
}

func hindsightAPIToken(settings hindsight.Settings) (string, error) {
	key := strings.TrimSpace(settings.APITokenName)
	if key == "" {
		return "", nil
	}
	namespace := strings.TrimSpace(settings.VaultNamespace)
	if namespace == "" {
		namespace = hindsight.DefaultVaultNamespace
	}
	token, err := readHindsightAPIToken(namespace, key)
	if err != nil {
		return "", err
	}
	return token, nil
}
