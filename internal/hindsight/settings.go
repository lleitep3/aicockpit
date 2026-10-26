package hindsight

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	// DefaultImage is the official standalone image (API and control plane).
	DefaultImage = "ghcr.io/vectorize-io/hindsight:latest"
	// DefaultAPIPort is the host port published for the Hindsight API.
	DefaultAPIPort = 8888
	// DefaultUIPort is the host port published for the Hindsight control plane.
	DefaultUIPort = 9999
	// APIContainerPort is the port the image listens on for the API.
	APIContainerPort = 8888
	// UIContainerPort is the port the image listens on for the control plane.
	UIContainerPort = 9999
	// DefaultContainer is the container name cockpit manages.
	DefaultContainer = "aicockpit-hindsight"
	// DefaultVolume is the named volume for the embedded pg0 data directory.
	DefaultVolume = "aicockpit-hindsight-data"
	// PG0Mount is the in-container path of the embedded pg0 database.
	// Leaving HINDSIGHT_API_DATABASE_URL unset selects pg0 and stores data here.
	PG0Mount = "/home/hindsight/.pg0"
	// HealthPath is the API health path the image waits on during startup.
	HealthPath = "/health"
	// DefaultProvider is the LLM provider used when none is configured.
	DefaultProvider = "openai"
	// DefaultVaultNamespace is the vault namespace that holds Hindsight secrets.
	DefaultVaultNamespace = "hindsight"
	// DefaultAPIKeyName is the vault key for HINDSIGHT_API_LLM_API_KEY.
	DefaultAPIKeyName = "llm_api_key"
	// DefaultShmSize matches the shared-memory size from the official Docker quick start.
	DefaultShmSize = "1g"
	// DefaultRestart asks Docker to bring the container back after a daemon restart.
	DefaultRestart = "unless-stopped"

	// EnvLLMProvider is the Hindsight LLM provider variable.
	EnvLLMProvider = "HINDSIGHT_API_LLM_PROVIDER"
	// EnvLLMAPIKey is the Hindsight LLM API key variable.
	EnvLLMAPIKey = "HINDSIGHT_API_LLM_API_KEY"
	// EnvLLMModel is the optional Hindsight LLM model variable.
	EnvLLMModel = "HINDSIGHT_API_LLM_MODEL"
	// EnvWorkerID is a stable worker identity so restarts can reclaim tasks.
	EnvWorkerID = "HINDSIGHT_API_WORKER_ID"
	// EnvDatabaseURL would select an external database. Cockpit refuses it and uses pg0.
	EnvDatabaseURL = "HINDSIGHT_API_DATABASE_URL"
)

var (
	imagePattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._:/-]{0,200}$`)
	namePattern      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,127}$`)
	providerPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,63}$`)
	modelPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@/+-]{0,127}$`)
	namespacePattern = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	vaultKeyPattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)
	envPattern       = regexp.MustCompile(`^HINDSIGHT_[A-Z0-9_]+$`)
)

// Settings is the non-secret Hindsight block stored in the cockpit config.
// Secret values are not fields of Settings. They are read from the vault when
// a container is created.
type Settings struct {
	// Enabled turns the optional server on. False leaves doctor non-fatal.
	Enabled bool `yaml:"enabled"`
	// Image is the Docker image. Empty uses DefaultImage.
	Image string `yaml:"image,omitempty"`
	// APIPort is the host port mapped to APIContainerPort. Empty uses DefaultAPIPort.
	APIPort int `yaml:"api_port,omitempty"`
	// UIPort is the host port mapped to UIContainerPort. Empty uses DefaultUIPort.
	UIPort int `yaml:"ui_port,omitempty"`
	// ContainerName is the Docker container name. Empty uses DefaultContainer.
	ContainerName string `yaml:"container_name,omitempty"`
	// Volume is a Docker named volume, not a host path. Empty uses DefaultVolume.
	Volume string `yaml:"volume,omitempty"`
	// LLMProvider is HINDSIGHT_API_LLM_PROVIDER. Empty uses DefaultProvider.
	LLMProvider string `yaml:"llm_provider,omitempty"`
	// LLMModel is HINDSIGHT_API_LLM_MODEL when set.
	LLMModel string `yaml:"llm_model,omitempty"`
	// VaultNamespace is the vault namespace for secrets. Empty uses DefaultVaultNamespace.
	VaultNamespace string `yaml:"vault_namespace,omitempty"`
	// APIKeyName is the vault key for the LLM API key. Empty uses DefaultAPIKeyName.
	APIKeyName string `yaml:"api_key_name,omitempty"`
	// APIKeyOptional skips the LLM API key for providers that do not need one.
	APIKeyOptional bool `yaml:"api_key_optional,omitempty"`
	// Secrets maps additional HINDSIGHT_* environment variables to vault keys.
	// HINDSIGHT_API_DATABASE_URL is rejected so the server stays on embedded pg0.
	Secrets map[string]string `yaml:"secrets,omitempty"`
}

// Resolved is Settings with defaults filled in and checks applied.
type Resolved struct {
	// Image is the docker image reference.
	Image string
	// APIPort is the host port for the API.
	APIPort int
	// UIPort is the host port for the control plane.
	UIPort int
	// ContainerName is the docker container name.
	ContainerName string
	// Volume is the named volume for pg0 data.
	Volume string
	// LLMProvider is the Hindsight LLM provider.
	LLMProvider string
	// LLMModel is the optional model name.
	LLMModel string
	// VaultNamespace is the vault namespace that holds secrets.
	VaultNamespace string
	// APIKeyName is the vault key for the LLM API key.
	APIKeyName string
	// APIKeyOptional reports whether the LLM API key can be omitted.
	APIKeyOptional bool
	// Secrets maps environment variable names to vault keys.
	Secrets map[string]string
}

// Resolve checks settings and fills defaults. Disabled settings return ErrNotConfigured.
func (s Settings) Resolve() (Resolved, error) {
	if !s.Enabled {
		return Resolved{}, ErrNotConfigured
	}

	apiPort, err := normalizePort(s.APIPort, DefaultAPIPort, "api")
	if err != nil {
		return Resolved{}, err
	}
	uiPort, err := normalizePort(s.UIPort, DefaultUIPort, "ui")
	if err != nil {
		return Resolved{}, err
	}
	if apiPort == uiPort {
		return Resolved{}, fmt.Errorf("api and ui ports must differ")
	}

	resolved := Resolved{
		Image:          defaultString(s.Image, DefaultImage),
		APIPort:        apiPort,
		UIPort:         uiPort,
		ContainerName:  defaultString(s.ContainerName, DefaultContainer),
		Volume:         defaultString(s.Volume, DefaultVolume),
		LLMProvider:    defaultString(s.LLMProvider, DefaultProvider),
		LLMModel:       strings.TrimSpace(s.LLMModel),
		VaultNamespace: defaultString(s.VaultNamespace, DefaultVaultNamespace),
		APIKeyName:     defaultString(s.APIKeyName, DefaultAPIKeyName),
		APIKeyOptional: s.APIKeyOptional,
		Secrets:        make(map[string]string, len(s.Secrets)+1),
	}
	for name, key := range s.Secrets {
		resolved.Secrets[strings.TrimSpace(name)] = strings.TrimSpace(key)
	}
	if _, ok := resolved.Secrets[EnvLLMAPIKey]; !ok && !resolved.APIKeyOptional {
		resolved.Secrets[EnvLLMAPIKey] = resolved.APIKeyName
	}
	if err := resolved.validate(); err != nil {
		return Resolved{}, err
	}
	return resolved, nil
}

func (r Resolved) validate() error {
	if !imagePattern.MatchString(r.Image) {
		return fmt.Errorf("invalid hindsight image %q", r.Image)
	}
	if !namePattern.MatchString(r.ContainerName) {
		return fmt.Errorf("invalid container name %q", r.ContainerName)
	}
	if strings.Contains(r.Volume, "/") || strings.Contains(r.Volume, `\`) || !namePattern.MatchString(r.Volume) {
		return fmt.Errorf("volume must be a docker named volume, not a host path")
	}
	if !providerPattern.MatchString(r.LLMProvider) {
		return fmt.Errorf("invalid llm provider %q", r.LLMProvider)
	}
	if r.LLMModel != "" && !modelPattern.MatchString(r.LLMModel) {
		return fmt.Errorf("invalid llm model %q", r.LLMModel)
	}
	if !namespacePattern.MatchString(r.VaultNamespace) {
		return fmt.Errorf("invalid vault namespace %q", r.VaultNamespace)
	}
	if !r.APIKeyOptional && !vaultKeyPattern.MatchString(r.APIKeyName) {
		return fmt.Errorf("invalid api key name %q", r.APIKeyName)
	}
	for name, key := range r.Secrets {
		if name == EnvDatabaseURL {
			return fmt.Errorf("refusing %s so embedded pg0 stays in use", EnvDatabaseURL)
		}
		if !envPattern.MatchString(name) {
			return fmt.Errorf("refusing environment variable %q", name)
		}
		if !vaultKeyPattern.MatchString(key) {
			return fmt.Errorf("invalid vault key for %s", name)
		}
	}
	return nil
}

// HealthURL is the local API health endpoint.
func (r Resolved) HealthURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", r.APIPort, HealthPath)
}

// APIURL is the local API base URL.
func (r Resolved) APIURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", r.APIPort)
}

// UIURL is the local control-plane URL.
func (r Resolved) UIURL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", r.UIPort)
}

// Spec builds the container request. values maps environment names to secret
// values already read from the vault. The returned Spec holds those secrets
// and must not be logged.
func (r Resolved) Spec(values map[string]string) (Spec, error) {
	env := []EnvVar{
		{Name: EnvLLMProvider, Value: r.LLMProvider},
		{Name: EnvWorkerID, Value: r.ContainerName},
	}
	if r.LLMModel != "" {
		env = append(env, EnvVar{Name: EnvLLMModel, Value: r.LLMModel})
	}
	names := make([]string, 0, len(r.Secrets))
	for name := range r.Secrets {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		value := values[name]
		if strings.TrimSpace(value) == "" {
			return Spec{}, fmt.Errorf("missing value for %s", name)
		}
		if strings.ContainsAny(value, "\r\n\x00") {
			return Spec{}, fmt.Errorf("environment %s contains a newline", name)
		}
		env = append(env, EnvVar{Name: name, Value: value, Secret: true})
	}
	return Spec{
		Name:    r.ContainerName,
		Image:   r.Image,
		APIPort: r.APIPort,
		UIPort:  r.UIPort,
		Volume:  r.Volume,
		Mount:   PG0Mount,
		Env:     env,
		ShmSize: DefaultShmSize,
		Restart: DefaultRestart,
	}, nil
}

func normalizePort(value, fallback int, name string) (int, error) {
	if value == 0 {
		value = fallback
	}
	if value < 1 || value > 65535 {
		return 0, fmt.Errorf("invalid %s port %d", name, value)
	}
	return value, nil
}

func defaultString(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
