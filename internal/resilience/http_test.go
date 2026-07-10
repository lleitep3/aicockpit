package resilience

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRetryableClient_Get_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, DefaultConfig(), nil)
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestRetryableClient_Get_RetriesThenSucceeds(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			http.Error(w, "boom", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := Config{MaxAttempts: 5, InitialDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond, Multiplier: 2}
	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, cfg, nil)
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "ok" {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestRetryableClient_Get_RetryableStatusMaxAttempts(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	cfg := Config{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond}
	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, cfg, nil)
	_, err := client.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetryableClient_Get_NonRetryableStatus(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, DefaultConfig(), nil)
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRetryableClient_Post_NoRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, DefaultConfig(), nil)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL, http.NoBody)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()

	if calls != 1 {
		t.Fatalf("expected 1 call for POST, got %d", calls)
	}
}

func TestRetryableClient_CircuitBreaker(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 1, ResetTimeout: 1 * time.Hour})
	cfg := Config{MaxAttempts: 1, InitialDelay: 1 * time.Millisecond}
	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, cfg, cb)

	_, err := client.Get(context.Background(), server.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	// First call records the failure and opens the circuit.
	_, err = client.Get(context.Background(), server.URL)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected circuit open, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 server call before open, got %d", calls)
	}
}

func TestRetryableClient_NilClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewRetryableClient(nil, DefaultConfig(), nil)
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer resp.Body.Close()
}

func TestRetryableClient_Get_InvalidURL(t *testing.T) {
	client := NewRetryableClient(&http.Client{Timeout: 5 * time.Second}, DefaultConfig(), nil)
	_, err := client.Get(context.Background(), "://invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestIsRetryableStatus(t *testing.T) {
	retryable := []int{408, 429, 500, 502, 503, 504}
	for _, code := range retryable {
		if !isRetryableStatus(code) {
			t.Fatalf("expected %d to be retryable", code)
		}
	}
	non := []int{200, 301, 400, 401, 403, 404, 418}
	for _, code := range non {
		if isRetryableStatus(code) {
			t.Fatalf("expected %d to be non-retryable", code)
		}
	}
}
