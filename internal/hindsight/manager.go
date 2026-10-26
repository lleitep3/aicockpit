package hindsight

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var (
	// ErrNotConfigured is returned when memory.hindsight.enabled is false.
	ErrNotConfigured = errors.New("hindsight is not configured")
	// ErrNotFound is returned when the managed container does not exist.
	ErrNotFound = errors.New("container not found")
	// ErrNilContext is returned when a method is called with a nil context.
	ErrNilContext = errors.New("nil context")
	// ErrRuntimeMissing is returned when docker is not available.
	ErrRuntimeMissing = errors.New("docker not found")
)

// SecretSource reads one vault secret. Implementations must not log the value.
type SecretSource interface {
	Get(ctx context.Context, namespace, key string) (string, error)
}

// SecretFunc adapts a function to SecretSource.
type SecretFunc func(ctx context.Context, namespace, key string) (string, error)

// Get calls the function.
func (f SecretFunc) Get(ctx context.Context, namespace, key string) (string, error) {
	return f(ctx, namespace, key)
}

// Manager runs up, down, and status for the optional local server.
type Manager struct {
	// Settings is the non-secret configuration.
	Settings Settings
	// Runtime talks to Docker. Nil is an error once the server is enabled.
	Runtime Runtime
	// Probe checks /health. Nil reports the API as not ready.
	Probe Prober
	// Secrets reads vault values. Nil is an error when a secret is required.
	Secrets SecretSource
}

// NewManager returns a manager. runtime, probe, and secrets may be nil only
// for the paths that do not need them.
func NewManager(settings Settings, runtime Runtime, probe Prober, secrets SecretSource) *Manager {
	return &Manager{Settings: settings, Runtime: runtime, Probe: probe, Secrets: secrets}
}

// Status reports configuration, container state, and /health.
// A disabled server returns a report and a nil error.
func (m *Manager) Status(ctx context.Context) (Report, error) {
	if !m.Settings.Enabled {
		return notConfiguredReport(), nil
	}
	if err := checkCtx(ctx, "status"); err != nil {
		return Report{}, err
	}
	resolved, err := m.Settings.Resolve()
	if err != nil {
		return Report{Configured: true, Backend: "hindsight", State: "unavailable", Health: "skipped", Message: err.Error()}, wrapOp("status", err, nil)
	}
	if m.Runtime == nil {
		return Report{Configured: true, Backend: "hindsight", State: "unavailable", Health: "skipped", Message: ErrRuntimeMissing.Error()}, wrapOp("status", ErrRuntimeMissing, nil)
	}
	state, err := m.Runtime.Inspect(ctx, resolved.ContainerName)
	if errors.Is(err, ErrNotFound) {
		return absentReport(resolved, "Hindsight is configured but the container is not running"), nil
	}
	if err != nil {
		return Report{Configured: true, Backend: "hindsight", Container: resolved.ContainerName, State: "unavailable", Health: "skipped", Message: err.Error()}, wrapOp("status", err, nil)
	}
	return m.observe(ctx, resolved, state)
}

// Up creates the container when it is not already running.
// An already running container is left in place so its vault-provided
// environment is not rebuilt. Stop it with Down before changing secrets.
// Up returns a nil error once the container is running, even if /health is
// not ready yet. The first boot can spend several minutes loading models.
func (m *Manager) Up(ctx context.Context) (Report, error) {
	if err := checkCtx(ctx, "up"); err != nil {
		return Report{}, err
	}
	if !m.Settings.Enabled {
		return Report{}, wrapOp("up", ErrNotConfigured, nil)
	}
	resolved, err := m.Settings.Resolve()
	if err != nil {
		return Report{}, wrapOp("up", err, nil)
	}
	if m.Runtime == nil {
		return Report{}, wrapOp("up", ErrRuntimeMissing, nil)
	}
	state, err := m.Runtime.Inspect(ctx, resolved.ContainerName)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return Report{}, wrapOp("up", err, nil)
	}
	if err == nil && state.Running {
		return m.observe(ctx, resolved, state)
	}
	if err == nil {
		if removeErr := m.Runtime.Remove(ctx, resolved.ContainerName); removeErr != nil {
			return Report{}, wrapOp("up", fmt.Errorf("remove stopped container: %w", removeErr), nil)
		}
	}
	values, err := m.loadSecrets(ctx, resolved)
	if err != nil {
		return Report{}, wrapOp("up", err, nil)
	}
	spec, err := resolved.Spec(values)
	clearValues(values)
	if err != nil {
		return Report{}, wrapOp("up", err, nil)
	}
	secrets := spec.SecretValues()
	runErr := m.Runtime.Run(ctx, spec)
	spec.zeroSecrets()
	if runErr != nil {
		return Report{}, wrapOp("up", runErr, secrets)
	}
	state, err = m.Runtime.Inspect(ctx, resolved.ContainerName)
	if err != nil {
		return Report{}, wrapOp("up", fmt.Errorf("started but inspect failed: %w", err), secrets)
	}
	return m.observe(ctx, resolved, state)
}

// Down stops and removes the container. The named volume is kept.
// A disabled server returns a not-configured report and a nil error.
func (m *Manager) Down(ctx context.Context) (Report, error) {
	if err := checkCtx(ctx, "down"); err != nil {
		return Report{}, err
	}
	if !m.Settings.Enabled {
		return notConfiguredReport(), nil
	}
	resolved, err := m.Settings.Resolve()
	if err != nil {
		return Report{}, wrapOp("down", err, nil)
	}
	if m.Runtime == nil {
		return Report{}, wrapOp("down", ErrRuntimeMissing, nil)
	}
	state, err := m.Runtime.Inspect(ctx, resolved.ContainerName)
	if errors.Is(err, ErrNotFound) {
		return absentReport(resolved, "Hindsight is already stopped"), nil
	}
	if err != nil {
		return Report{}, wrapOp("down", err, nil)
	}
	if state.Running {
		if stopErr := m.Runtime.Stop(ctx, resolved.ContainerName); stopErr != nil {
			return Report{}, wrapOp("down", stopErr, nil)
		}
	}
	if removeErr := m.Runtime.Remove(ctx, resolved.ContainerName); removeErr != nil {
		return Report{}, wrapOp("down", removeErr, nil)
	}
	return absentReport(resolved, "Hindsight container stopped"), nil
}

func (m *Manager) observe(ctx context.Context, resolved Resolved, state ContainerState) (Report, error) {
	report := reportFrom(resolved, state)
	if !state.Running {
		report.Message = "Hindsight container is " + report.State
		return report, nil
	}
	if m.Probe == nil {
		report.Health = "unreachable"
		report.Message = "Hindsight is running but /health was not checked"
		return report, nil
	}
	if err := m.Probe.Probe(ctx, resolved.HealthURL()); err != nil {
		report.Health = "unreachable"
		report.Message = "Hindsight is running but /health is not ready"
		return report, nil
	}
	report.Healthy = true
	report.Health = "ok"
	report.Message = "Hindsight API is healthy"
	return report, nil
}

func (m *Manager) loadSecrets(ctx context.Context, resolved Resolved) (map[string]string, error) {
	if len(resolved.Secrets) == 0 {
		return map[string]string{}, nil
	}
	if m.Secrets == nil {
		return nil, errors.New("vault secret source is not configured")
	}
	keys := make([]string, 0, len(resolved.Secrets))
	seen := make(map[string]struct{}, len(resolved.Secrets))
	envByKey := make(map[string][]string, len(resolved.Secrets))
	for envName, key := range resolved.Secrets {
		envByKey[key] = append(envByKey[key], envName)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	loaded := make(map[string]string, len(keys))
	for _, key := range keys {
		value, err := m.Secrets.Get(ctx, resolved.VaultNamespace, key)
		if err != nil {
			clearValues(loaded)
			return nil, fmt.Errorf("vault key %s: %w", key, redactError(err, []string{value}))
		}
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00") {
			clearValues(loaded)
			return nil, fmt.Errorf("vault key %s is empty or contains a newline", key)
		}
		loaded[key] = value
	}
	values := make(map[string]string, len(resolved.Secrets))
	for key, envNames := range envByKey {
		for _, envName := range envNames {
			values[envName] = loaded[key]
		}
	}
	clearValues(loaded)
	return values, nil
}

func checkCtx(ctx context.Context, op string) error {
	if ctx == nil {
		return fmt.Errorf("hindsight %s: %w", op, ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("hindsight %s: %w", op, err)
	}
	return nil
}
