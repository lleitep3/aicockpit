package resilience

import (
	"context"
	"fmt"
	"net/http"
)

// RetryableClient wraps an http.Client and retries idempotent HTTP requests.
type RetryableClient struct {
	client  *http.Client
	config  Config
	breaker *CircuitBreaker
}

// NewRetryableClient creates a new RetryableClient.
// A nil breaker disables circuit breaking.
func NewRetryableClient(client *http.Client, config Config, breaker *CircuitBreaker) *RetryableClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &RetryableClient{
		client:  client,
		config:  config,
		breaker: breaker,
	}
}

// Do performs the HTTP request, retrying on transient failures.
// It currently retries only GET and HEAD requests (or requests with no Body).
func (c *RetryableClient) Do(req *http.Request) (*http.Response, error) {
	if !isRetryableRequest(req) {
		return c.client.Do(req)
	}

	var resp *http.Response
	op := func() error {
		var err error
		resp, err = c.client.Do(req.Clone(req.Context()))
		if err != nil {
			return AsRetryable(fmt.Errorf("http request failed: %w", err))
		}

		if isRetryableStatus(resp.StatusCode) {
			// Close body so connection can be reused; the caller will not see this response.
			_ = resp.Body.Close()
			return AsRetryable(fmt.Errorf("http status %d", resp.StatusCode))
		}

		return nil
	}

	var err error
	if c.breaker != nil {
		err = c.breaker.Call(func() error { return Retry(req.Context(), c.config, op) })
	} else {
		err = Retry(req.Context(), c.config, op)
	}

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Get is a convenience helper for GET requests.
func (c *RetryableClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return c.Do(req)
}

// isRetryableRequest reports whether the request is safe to retry.
func isRetryableRequest(req *http.Request) bool {
	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}
	if req.Body != nil {
		return false
	}
	return true
}

// isRetryableStatus reports whether an HTTP status code is transient.
func isRetryableStatus(status int) bool {
	switch status {
	case http.StatusRequestTimeout, // 408
		http.StatusTooManyRequests,     // 429
		http.StatusInternalServerError, // 500
		http.StatusBadGateway,          // 502
		http.StatusServiceUnavailable,  // 503
		http.StatusGatewayTimeout:      // 504
		return true
	}
	return false
}
