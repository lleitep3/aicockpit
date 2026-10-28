package ask

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/agents"
)

// LiveOptions configures the catalog that talks to local CLIs and the API backend.
type LiveOptions struct {
	// Runner starts CLI probes. Nil uses agents.OSRunner.
	Runner agents.Runner
	// API is the non-secret provider block from configuration.
	API agents.APISettings
	// Tokens reads the provider token at call time.
	Tokens agents.TokenSource
	// Probe reports whether a token is configured. Nil uses Tokens when it implements agents.TokenProbe.
	Probe agents.TokenProbe
	// HTTP sends API requests. Nil uses the default client, which does not follow redirects.
	HTTP agents.HTTPDoer
}

// LiveCatalog is the production Catalog.
// Availability of a CLI is the binary on PATH. Availability of an API is a
// configured provider whose vault token is present. The token value is not read
// during selection and is not included in errors.
type LiveCatalog struct {
	opts LiveOptions
}

// NewLiveCatalog returns a catalog for opts.
func NewLiveCatalog(opts LiveOptions) *LiveCatalog {
	return &LiveCatalog{opts: opts}
}

// Check reports whether ref can run.
func (c *LiveCatalog) Check(ctx context.Context, ref Ref) (Candidate, error) {
	if c == nil {
		return Candidate{Ref: ref, Reason: "unavailable"}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch ref.Kind {
	case KindCLI:
		return c.cliCandidate(ref.ID), nil
	case KindAPI:
		return c.apiCandidate(ctx, ref.ID), nil
	default:
		return Candidate{Ref: ref, Reason: "unknown backend"}, nil
	}
}

// Auto returns claude, codex, and the configured API provider, in that order.
func (c *LiveCatalog) Auto(ctx context.Context) ([]Candidate, error) {
	if c == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	out := make([]Candidate, 0, len(agents.Adapters())+1)
	for _, adapter := range agents.Adapters() {
		out = append(out, c.cliCandidate(adapter.ID()))
	}
	if strings.TrimSpace(c.opts.API.Provider) != "" {
		out = append(out, c.apiCandidate(ctx, ""))
	}
	return out, nil
}

// Open returns the worker for ref.
func (c *LiveCatalog) Open(_ context.Context, ref Ref) (agents.Backend, error) {
	if c == nil {
		return nil, fmt.Errorf("ask: %w", ErrNoBackend)
	}
	switch ref.Kind {
	case KindCLI:
		backend, err := agents.NewCLIBackend(ref.ID, c.opts.Runner)
		if err != nil {
			return nil, err
		}
		return backend, nil
	case KindAPI:
		if c.opts.Tokens == nil {
			return nil, fmt.Errorf("ask: %w", agents.ErrTokenMissing)
		}
		settings := c.apiSettings(ref.ID)
		backend, err := agents.NewAPIBackend(settings, c.opts.Tokens, c.opts.HTTP)
		if err != nil {
			return nil, err
		}
		return backend, nil
	default:
		return nil, fmt.Errorf("ask: backend %q: %w", displayRef(ref), agents.ErrUnknownBackend)
	}
}

func (c *LiveCatalog) cliCandidate(id string) Candidate {
	ref := Ref{Kind: KindCLI, ID: id, Raw: id}
	adapter, err := agents.AdapterByID(id)
	if err != nil {
		ref.Raw = id
		return Candidate{Ref: ref, Reason: "unknown backend"}
	}
	ref.ID = adapter.ID()
	ref.Raw = adapter.ID()
	if c.installed(adapter.Binary()) {
		return Candidate{Ref: ref, Available: true}
	}
	return Candidate{Ref: ref, Reason: adapter.Binary() + " is not installed"}
}

func (c *LiveCatalog) installed(binary string) bool {
	runner := c.opts.Runner
	if runner == nil {
		runner = agents.OSRunner{}
	}
	if checker, ok := runner.(agents.InstallChecker); ok {
		return checker.Installed(binary)
	}
	for _, status := range agents.Detect(context.Background(), runner) {
		if status.Binary == binary {
			return status.Installed
		}
	}
	return false
}

func (c *LiveCatalog) apiCandidate(ctx context.Context, provider string) Candidate {
	settings := c.apiSettings(provider)
	if strings.TrimSpace(settings.Provider) == "" {
		return Candidate{Ref: Ref{Kind: KindAPI, Raw: "api"}, Reason: "agents.api.provider is not set"}
	}
	resolved, err := settings.Resolve()
	if err != nil {
		raw := "api"
		if provider != "" {
			raw = "api:" + provider
		} else if strings.TrimSpace(settings.Provider) != "" {
			raw = "api:" + strings.ToLower(strings.TrimSpace(settings.Provider))
		}
		return Candidate{Ref: Ref{Kind: KindAPI, Raw: raw}, Reason: "configuration is invalid"}
	}
	ref := Ref{Kind: KindAPI, ID: resolved.Provider, Raw: "api:" + resolved.Provider}
	probe := c.opts.Probe
	if probe == nil {
		if asProbe, ok := c.opts.Tokens.(agents.TokenProbe); ok {
			probe = asProbe
		}
	}
	if probe == nil {
		return Candidate{Ref: ref, Reason: "token status unavailable"}
	}
	ok, err := probe.Configured(ctx, resolved.VaultNamespace, resolved.VaultKey)
	if err != nil {
		if errors.Is(err, agents.ErrVaultLocked) {
			return Candidate{Ref: ref, Reason: "vault is locked"}
		}
		return Candidate{Ref: ref, Reason: "token status unavailable"}
	}
	if !ok {
		return Candidate{Ref: ref, Reason: "token is not configured"}
	}
	return Candidate{Ref: ref, Available: true}
}

func (c *LiveCatalog) apiSettings(provider string) agents.APISettings {
	settings := c.opts.API
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return settings
	}
	configured := canonicalProvider(settings.Provider)
	requested := canonicalProvider(provider)
	if configured != "" && requested != "" && configured != requested {
		settings.Model = ""
		settings.BaseURL = ""
		settings.VaultKey = ""
	}
	settings.Provider = provider
	return settings
}

func canonicalProvider(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	resolved, err := (agents.APISettings{Provider: raw}).Resolve()
	if err != nil {
		return ""
	}
	return resolved.Provider
}
