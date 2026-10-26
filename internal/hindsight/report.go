package hindsight

import (
	"fmt"
	"strings"
)

// Report is the public status of the optional Hindsight server.
// It never contains secrets.
type Report struct {
	// Configured is true when memory.hindsight.enabled is set.
	Configured bool `json:"configured"`
	// Backend is "hindsight" or "none".
	Backend string `json:"backend"`
	// Image is the container image when the server is configured.
	Image string `json:"image,omitempty"`
	// Container is the docker container name.
	Container string `json:"container,omitempty"`
	// Provider is the configured LLM provider. It is not a secret.
	Provider string `json:"llm_provider,omitempty"`
	// State is not_configured, absent, unavailable, or a docker status.
	State string `json:"state"`
	// Running is true when the container process is running.
	Running bool `json:"running"`
	// Healthy is true when the API /health endpoint returned success.
	Healthy bool `json:"healthy"`
	// Health is skipped, ok, or unreachable.
	Health string `json:"health"`
	// APIURL is the local API base URL.
	APIURL string `json:"api_url,omitempty"`
	// UIURL is the local control-plane URL.
	UIURL string `json:"ui_url,omitempty"`
	// Message is a short human-readable status.
	Message string `json:"message"`
}

// DoctorView is the doctor row for Hindsight.
type DoctorView struct {
	// Status is ok, skipped, or error.
	Status string
	// Message describes the row.
	Message string
	// Fatal is false when the server is not configured.
	Fatal bool
	// Fix is a command that can repair a fatal row.
	Fix string
}

// FormatText renders a multi-line status without secrets.
func (r Report) FormatText() string {
	var b strings.Builder
	b.WriteString("Hindsight memory\n")
	fmt.Fprintf(&b, "  configured: %t\n", r.Configured)
	if r.Provider != "" {
		fmt.Fprintf(&b, "  provider: %s\n", r.Provider)
	}
	if r.Container != "" {
		fmt.Fprintf(&b, "  container: %s\n", r.Container)
	}
	if r.State != "" {
		fmt.Fprintf(&b, "  state: %s\n", r.State)
	}
	if r.Health != "" {
		fmt.Fprintf(&b, "  health: %s\n", r.Health)
	}
	if r.APIURL != "" {
		fmt.Fprintf(&b, "  api: %s\n", r.APIURL)
	}
	if r.UIURL != "" {
		fmt.Fprintf(&b, "  ui: %s\n", r.UIURL)
	}
	if r.Message != "" {
		fmt.Fprintf(&b, "  message: %s\n", r.Message)
	}
	return b.String()
}

// Doctor maps a report onto the doctor check.
// A server that is not configured is skipped and is not fatal.
func (r Report) Doctor() DoctorView {
	if !r.Configured {
		return DoctorView{Status: "skipped", Message: "Hindsight is not configured", Fatal: false}
	}
	if r.Running && r.Healthy {
		return DoctorView{Status: "ok", Message: "Hindsight API is healthy", Fatal: false}
	}
	if r.Running {
		message := r.Message
		if message == "" {
			message = "Hindsight is running but /health failed"
		}
		return DoctorView{Status: "error", Message: message, Fatal: true, Fix: "cockpit memory status"}
	}
	message := r.Message
	if message == "" {
		message = "Hindsight is configured but the container is not running"
	}
	return DoctorView{Status: "error", Message: message, Fatal: true, Fix: "cockpit memory up"}
}

func notConfiguredReport() Report {
	return Report{
		Configured: false,
		Backend:    "none",
		State:      "not_configured",
		Health:     "skipped",
		Message:    "Hindsight is not configured. Set memory.hindsight.enabled: true and store the LLM API key in the vault.",
	}
}

func reportFrom(resolved Resolved, state ContainerState) Report {
	status := state.Status
	if status == "" {
		if state.Running {
			status = "running"
		} else {
			status = "exited"
		}
	}
	return Report{
		Configured: true,
		Backend:    "hindsight",
		Image:      resolved.Image,
		Container:  resolved.ContainerName,
		Provider:   resolved.LLMProvider,
		State:      status,
		Running:    state.Running,
		Health:     "skipped",
		APIURL:     resolved.APIURL(),
		UIURL:      resolved.UIURL(),
	}
}

func absentReport(resolved Resolved, message string) Report {
	report := reportFrom(resolved, ContainerState{Status: "absent"})
	report.Message = message
	return report
}
