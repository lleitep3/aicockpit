package hindsight

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const healthTimeout = 2 * time.Second

// Prober checks the Hindsight API health endpoint.
type Prober interface {
	Probe(ctx context.Context, url string) error
}

// HTTPProber probes with HTTP GET. A nil Client uses a 2 second timeout.
// Response bodies are discarded and are never included in errors.
type HTTPProber struct {
	Client *http.Client
}

// Probe reports whether url returned an HTTP 2xx response.
func (p HTTPProber) Probe(ctx context.Context, rawURL string) error {
	if ctx == nil {
		return fmt.Errorf("hindsight health: %w", ErrNilContext)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: healthTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("hindsight health: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("hindsight health: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("hindsight health: HTTP %d", resp.StatusCode)
	}
	return nil
}
