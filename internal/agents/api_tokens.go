package agents

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lleitep3/aicockpit/internal/vault"
)

// vaultActor is the cockpit CLI identity used for vault lock checks.
const vaultActor = "cockpit-cli"

// TokenProbe reports whether a vault entry exists.
// It must not return or log the secret value.
type TokenProbe interface {
	Configured(ctx context.Context, namespace, key string) (bool, error)
}

// TokenStatus is the non-secret presence of one provider token.
type TokenStatus struct {
	// Provider is the canonical provider id. Empty when the configured id is invalid.
	Provider string
	// Name is the doctor label.
	Name string
	// Namespace is the vault namespace that was checked. Empty when the reference is invalid.
	Namespace string
	// Key is the vault key that was checked. Empty when the reference is invalid.
	Key string
	// Configured is true when the vault index lists the key.
	Configured bool
	// Locked is true when the vault refused access. The secret was not read.
	Locked bool
	// Unavailable is true when the index could not be read.
	Unavailable bool
	// Invalid is true when provider settings cannot be used. The raw value is discarded.
	Invalid bool
}

// Doctor maps token presence onto a non-fatal doctor row.
// The message never includes a secret value.
func (s TokenStatus) Doctor() DoctorView {
	name := s.Name
	if name == "" {
		name = "API provider"
	}
	switch {
	case s.Invalid:
		return DoctorView{Name: name, Status: "skipped", Message: name + " configuration is invalid", Fatal: false}
	case s.Locked:
		return DoctorView{Name: name, Status: "skipped", Message: name + " token status unknown because the vault is locked", Fatal: false}
	case s.Unavailable:
		return DoctorView{Name: name, Status: "skipped", Message: name + " token status unavailable", Fatal: false}
	case s.Configured:
		return DoctorView{Name: name, Status: "ok", Message: name + " token is configured", Fatal: false}
	default:
		return DoctorView{Name: name, Status: "skipped", Message: name + " token is not configured", Fatal: false}
	}
}

// VaultTokenSource reads namespaced vault secrets.
// Get returns the token for a provider call. Configured reports index presence
// and does not load the secret value.
type VaultTokenSource struct {
	// Access reports whether the vault is open. Nil uses the cockpit lock manager.
	Access func() error
	// Read returns one secret. Nil uses the namespaced vault. The value must not be logged.
	Read func(namespace, key string) (string, error)
	// List returns key names in a namespace. Nil uses the vault index and not the secret values.
	List func(namespace string) ([]string, error)
}

// NewVaultTokenSource returns a source that uses the cockpit vault.
func NewVaultTokenSource() VaultTokenSource {
	return VaultTokenSource{
		Access: defaultVaultAccess,
		Read:   readVaultValue,
		List:   listVaultKeys,
	}
}

// Get reads one token. A locked vault or a missing value returns before the
// caller can place the token on a request. Error text does not include the value.
func (s VaultTokenSource) Get(ctx context.Context, namespace, key string) (string, error) {
	if err := s.gate(ctx, namespace, key); err != nil {
		return "", err
	}
	read := s.Read
	if read == nil {
		read = readVaultValue
	}
	value, err := read(namespace, key)
	if err != nil {
		return "", fmt.Errorf("agents api: %s/%s: %w", namespace, key, ErrTokenUnavailable)
	}
	if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n\x00") {
		return "", fmt.Errorf("agents api: %s/%s: %w", namespace, key, ErrTokenMissing)
	}
	return value, nil
}

// Configured reports whether the vault index contains key.
// The secret value is not loaded.
func (s VaultTokenSource) Configured(ctx context.Context, namespace, key string) (bool, error) {
	if err := s.gate(ctx, namespace, key); err != nil {
		return false, err
	}
	list := s.List
	if list == nil {
		list = listVaultKeys
	}
	names, err := list(namespace)
	if err != nil {
		return false, fmt.Errorf("agents api: %w", ErrTokenUnavailable)
	}
	for _, name := range names {
		if name == key {
			return true, nil
		}
	}
	return false, nil
}

func (s VaultTokenSource) gate(ctx context.Context, namespace, key string) error {
	if ctx == nil {
		return fmt.Errorf("agents api: %w", ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !apiNamespacePattern.MatchString(namespace) || !apiVaultKeyPattern.MatchString(key) {
		return fmt.Errorf("agents api: %w", ErrInvalidVaultRef)
	}
	access := s.Access
	if access == nil {
		access = defaultVaultAccess
	}
	if err := access(); err != nil {
		return fmt.Errorf("agents api: %w", ErrVaultLocked)
	}
	return nil
}

func defaultVaultAccess() error {
	lm := vault.NewLockManager("")
	if lm.InitializationError() != nil || !lm.CanPackageAccess(vaultActor) {
		return ErrVaultLocked
	}
	return nil
}

func readVaultValue(namespace, key string) (string, error) {
	return vault.NewNamespacedVault(namespace).Get(key)
}

func listVaultKeys(namespace string) ([]string, error) {
	infos, err := vault.NewNamespacedVault(namespace).List()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(infos))
	for _, info := range infos {
		names = append(names, info.Key)
	}
	return names, nil
}

// ProbeTokens reports which provider tokens are configured.
// A nil probe uses the cockpit vault. Status text never includes secret values.
// An invalid provider or vault reference is reported and is not echoed.
func ProbeTokens(ctx context.Context, settings APISettings, probe TokenProbe) []TokenStatus {
	if ctx == nil {
		ctx = context.Background()
	}
	if probe == nil {
		src := NewVaultTokenSource()
		probe = src
	}
	refs := tokenRefs(settings)
	out := make([]TokenStatus, 0, len(refs))
	for _, ref := range refs {
		out = append(out, probeRef(ctx, probe, ref))
	}
	return out
}

type tokenRef struct {
	provider  string
	name      string
	namespace string
	key       string
	invalid   bool
}

func tokenRefs(settings APISettings) []tokenRef {
	provider := normalizeProvider(settings.Provider)
	namespace := strings.TrimSpace(settings.VaultNamespace)
	key := strings.TrimSpace(settings.VaultKey)
	namespaceOK := namespace == "" || apiNamespacePattern.MatchString(namespace)
	keyOK := key == "" || apiVaultKeyPattern.MatchString(key)

	var refs []tokenRef
	if strings.TrimSpace(settings.Provider) != "" && provider == "" {
		refs = append(refs, tokenRef{name: "API provider", invalid: true})
	}
	for _, id := range KnownAPIProviders() {
		ref := tokenRef{
			provider:  id,
			name:      providerLabel(id),
			namespace: DefaultAPIVaultNamespace,
			key:       defaultAPIVaultKey(id),
		}
		if provider == id {
			if !namespaceOK || !keyOK {
				ref.invalid = true
				ref.namespace = ""
				ref.key = ""
			} else {
				if namespace != "" {
					ref.namespace = namespace
				}
				if key != "" {
					ref.key = key
				}
			}
		}
		refs = append(refs, ref)
	}
	return refs
}

func probeRef(ctx context.Context, probe TokenProbe, ref tokenRef) TokenStatus {
	status := TokenStatus{
		Provider:  ref.provider,
		Name:      ref.name,
		Namespace: ref.namespace,
		Key:       ref.key,
	}
	if ref.invalid {
		status.Invalid = true
		status.Namespace = ""
		status.Key = ""
		return status
	}
	if probe == nil {
		status.Unavailable = true
		return status
	}
	ok, err := probe.Configured(ctx, ref.namespace, ref.key)
	switch {
	case err == nil:
		status.Configured = ok
	case errors.Is(err, ErrVaultLocked):
		status.Locked = true
	default:
		status.Unavailable = true
	}
	return status
}
