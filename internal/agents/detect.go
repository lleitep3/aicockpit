package agents

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"
)

// probeTimeout bounds a version command so doctor cannot hang on a CLI.
var probeTimeout = 3 * time.Second

// Status is the installation state of one agent CLI.
type Status struct {
	// ID is the backend id.
	ID string
	// Name is the doctor label.
	Name string
	// Binary is the executable name.
	Binary string
	// Installed is true when the binary is on PATH.
	Installed bool
	// Version is the first line of the version command, when it produced one.
	Version string
	// Detail is a short non-secret note when the version command failed.
	Detail string
}

// DoctorView is one non-fatal doctor row for an agent CLI.
type DoctorView struct {
	// Name is the check label.
	Name string
	// Status is ok or skipped. A missing CLI is skipped.
	Status string
	// Message describes installation and version.
	Message string
	// Fatal is always false. A missing CLI does not fail doctor.
	Fatal bool
}

// Doctor maps installation state onto a doctor row.
// The row is never fatal.
func (s Status) Doctor() DoctorView {
	name := s.Name
	if name == "" {
		name = s.Binary
	}
	if !s.Installed {
		return DoctorView{Name: name, Status: "skipped", Message: name + " is not installed", Fatal: false}
	}
	if s.Version != "" {
		return DoctorView{Name: name, Status: "ok", Message: name + " " + s.Version, Fatal: false}
	}
	message := name + " is installed (version unknown)"
	if s.Detail != "" {
		message = name + " is installed (version unknown: " + s.Detail + ")"
	}
	return DoctorView{Name: name, Status: "ok", Message: message, Fatal: false}
}

// Detect probes every known CLI. A nil runner uses OSRunner.
// A missing binary is reported and is not an error.
func Detect(ctx context.Context, runner Runner) []Status {
	if ctx == nil {
		ctx = context.Background()
	}
	if runner == nil {
		runner = OSRunner{}
	}
	adapters := Adapters()
	out := make([]Status, 0, len(adapters))
	for _, adapter := range adapters {
		out = append(out, probe(ctx, runner, adapter))
	}
	return out
}

func probe(ctx context.Context, runner Runner, adapter Adapter) Status {
	status := Status{ID: adapter.ID(), Name: adapter.DisplayName(), Binary: adapter.Binary()}
	if checker, ok := runner.(InstallChecker); ok && !checker.Installed(adapter.Binary()) {
		return status
	}
	probeCtx, cancel := context.WithTimeout(ctx, probeTimeout)
	defer cancel()
	proc, err := runner.Start(probeCtx, Spec{
		Binary:  adapter.Binary(),
		Args:    adapter.VersionArgs(),
		DropEnv: adapter.DropEnv(),
	})
	if err != nil {
		if isNotInstalled(err) {
			return status
		}
		status.Installed = true
		status.Detail = clip(err.Error())
		return status
	}
	stdout, stderr, code, waitErr := readProc(proc)
	status.Installed = true
	status.Version = firstLine(stdout)
	if status.Version == "" {
		status.Version = firstLine(stderr)
	}
	if status.Version == "" {
		detail := clip(stderr)
		if detail == "" && waitErr != nil {
			detail = clip(waitErr.Error())
		}
		if detail == "" && code != 0 {
			detail = "version command failed"
		}
		status.Detail = detail
	}
	return status
}

func isNotInstalled(err error) bool {
	return errors.Is(err, ErrNotInstalled)
}

func readProc(proc Proc) (stdout, stderr string, code int, err error) {
	var out limitedWriter
	var errBuf limitedWriter
	out.max = 4096
	errBuf.max = 4096
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(&errBuf, proc.Stderr())
	}()
	_, _ = io.Copy(&out, proc.Stdout())
	wg.Wait()
	code, err = proc.Wait()
	return out.String(), errBuf.String(), code, err
}

func firstLine(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if i := strings.IndexAny(text, "\r\n"); i >= 0 {
		text = strings.TrimSpace(text[:i])
	}
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}
