package hindsight

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Runtime starts and inspects containers. Implementations must not write Env
// values to disk or logs. Tests use FakeRuntime; production uses DockerCLI.
type Runtime interface {
	Inspect(ctx context.Context, name string) (ContainerState, error)
	Run(ctx context.Context, spec Spec) error
	Stop(ctx context.Context, name string) error
	Remove(ctx context.Context, name string) error
}

// ContainerState is the subset of docker inspect this package needs.
// It deliberately excludes environment variables, which may hold secrets.
type ContainerState struct {
	// ID is the container id returned by docker.
	ID string
	// Name is the container name that was inspected.
	Name string
	// Status is the docker state status, such as running or exited.
	Status string
	// Running is true when the container process is running.
	Running bool
}

// Spec is a request to start the Hindsight container.
// Env may contain secrets and must not be logged or written to a file.
type Spec struct {
	// Name is the container name.
	Name string
	// Image is the docker image reference.
	Image string
	// APIPort is the host port published to APIContainerPort.
	APIPort int
	// UIPort is the host port published to UIContainerPort.
	UIPort int
	// Volume is a named docker volume.
	Volume string
	// Mount is the in-container path, normally PG0Mount.
	Mount string
	// Env is the container environment, including secrets.
	Env []EnvVar
	// ShmSize is passed to docker --shm-size.
	ShmSize string
	// Restart is passed to docker --restart.
	Restart string
}

// EnvVar is one container environment entry.
type EnvVar struct {
	// Name is the environment variable name.
	Name string
	// Value is the environment variable value. Secret values stay in memory.
	Value string
	// Secret marks values that must be redacted from errors and logs.
	Secret bool
}

// SecretValues returns the secret environment values carried by the spec.
func (s Spec) SecretValues() []string {
	values := make([]string, 0, len(s.Env))
	for _, env := range s.Env {
		if env.Secret && env.Value != "" {
			values = append(values, env.Value)
		}
	}
	return values
}

func (s *Spec) zeroSecrets() {
	for i := range s.Env {
		if s.Env[i].Secret {
			s.Env[i].Value = ""
		}
	}
}

func (s Spec) args() ([]string, error) {
	if s.Name == "" || s.Image == "" || s.Volume == "" || s.Mount == "" {
		return nil, errors.New("container spec is incomplete")
	}
	if strings.Contains(s.Mount, "..") {
		return nil, errors.New("invalid container mount")
	}
	args := []string{
		"run", "-d",
		"--name", s.Name,
		"--restart", s.Restart,
		"--shm-size", s.ShmSize,
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", s.APIPort, APIContainerPort),
		"-p", fmt.Sprintf("127.0.0.1:%d:%d", s.UIPort, UIContainerPort),
	}
	for _, env := range s.Env {
		if env.Name == EnvDatabaseURL {
			return nil, fmt.Errorf("refusing to set %s; embedded pg0 is required", EnvDatabaseURL)
		}
		if !envPattern.MatchString(env.Name) {
			return nil, fmt.Errorf("refusing environment variable %q", env.Name)
		}
		if strings.ContainsAny(env.Value, "\r\n\x00") {
			return nil, fmt.Errorf("environment %s contains a newline", env.Name)
		}
		args = append(args, "-e", env.Name+"="+env.Value)
	}
	args = append(args, "-v", s.Volume+":"+s.Mount, s.Image)
	return args, nil
}

type commandExec func(ctx context.Context, bin string, args []string) (string, string, error)

// DockerCLI is a Runtime implemented by the docker binary.
// Environment values are passed as process arguments to docker and are never
// written to an env file.
type DockerCLI struct {
	exec commandExec
	bin  string
}

// NewDockerCLI returns a Runtime that invokes docker on PATH.
func NewDockerCLI() DockerCLI {
	return DockerCLI{}
}

func (d DockerCLI) runner() commandExec {
	if d.exec != nil {
		return d.exec
	}
	return defaultExec
}

func (d DockerCLI) binary() string {
	if d.bin != "" {
		return d.bin
	}
	return "docker"
}

// Inspect reads container state without requesting the environment block.
func (d DockerCLI) Inspect(ctx context.Context, name string) (ContainerState, error) {
	if err := checkCtx(ctx, "inspect"); err != nil {
		return ContainerState{}, err
	}
	stdout, stderr, err := d.runner()(ctx, d.binary(), []string{
		"inspect", "--format", "{{.State.Status}}|{{.State.Running}}|{{.Id}}", name,
	})
	if err != nil {
		if errors.Is(err, ErrRuntimeMissing) {
			return ContainerState{}, fmt.Errorf("inspect %s: %w", name, ErrRuntimeMissing)
		}
		if isNotFound(stderr, err) {
			return ContainerState{}, fmt.Errorf("inspect %s: %w", name, ErrNotFound)
		}
		return ContainerState{}, fmt.Errorf("inspect %s: %s", name, clip(stderr, err))
	}
	state, err := parseInspect(stdout)
	if err != nil {
		return ContainerState{}, fmt.Errorf("inspect %s: %w", name, err)
	}
	state.Name = name
	return state, nil
}

// Run starts a detached container. It does not use --env-file.
func (d DockerCLI) Run(ctx context.Context, spec Spec) error {
	if err := checkCtx(ctx, "run"); err != nil {
		return err
	}
	args, err := spec.args()
	if err != nil {
		return err
	}
	secrets := spec.SecretValues()
	_, stderr, runErr := d.runner()(ctx, d.binary(), args)
	if runErr == nil {
		return nil
	}
	if errors.Is(runErr, ErrRuntimeMissing) {
		return fmt.Errorf("docker run %s: %w", spec.Name, ErrRuntimeMissing)
	}
	detail := redact(clip(stderr, runErr), secrets)
	return fmt.Errorf("docker run %s: %s", spec.Name, detail)
}

// Stop stops a running container.
func (d DockerCLI) Stop(ctx context.Context, name string) error {
	return d.simple(ctx, "stop", name, []string{"stop", name})
}

// Remove deletes the container and leaves its named volume in place.
func (d DockerCLI) Remove(ctx context.Context, name string) error {
	return d.simple(ctx, "rm", name, []string{"rm", name})
}

func (d DockerCLI) simple(ctx context.Context, op, name string, args []string) error {
	if err := checkCtx(ctx, op); err != nil {
		return err
	}
	_, stderr, err := d.runner()(ctx, d.binary(), args)
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrRuntimeMissing) {
		return fmt.Errorf("docker %s %s: %w", op, name, ErrRuntimeMissing)
	}
	return fmt.Errorf("docker %s %s: %s", op, name, clip(stderr, err))
}

func defaultExec(ctx context.Context, bin string, args []string) (string, string, error) {
	if ctx == nil {
		return "", "", fmt.Errorf("docker: %w", ErrNilContext)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if errors.Is(err, exec.ErrNotFound) {
		return stdout.String(), stderr.String(), fmt.Errorf("%w", ErrRuntimeMissing)
	}
	return stdout.String(), stderr.String(), err
}

func parseInspect(stdout string) (ContainerState, error) {
	parts := strings.Split(strings.TrimSpace(stdout), "|")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return ContainerState{}, errors.New("unexpected docker inspect output")
	}
	running, err := strconv.ParseBool(parts[1])
	if err != nil {
		return ContainerState{}, errors.New("unexpected docker inspect running flag")
	}
	return ContainerState{Status: parts[0], Running: running, ID: parts[2]}, nil
}

func isNotFound(stderr string, err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(stderr + " " + err.Error())
	return strings.Contains(text, "no such object") || strings.Contains(text, "no such container")
}

func clip(stderr string, err error) string {
	text := strings.TrimSpace(stderr)
	if text == "" && err != nil {
		text = err.Error()
	}
	if len(text) > 300 {
		text = text[:300]
	}
	return text
}
