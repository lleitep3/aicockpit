package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestCircuitBreaker_ClosesAfterSuccess(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 1 * time.Hour})
	if cb.State() != stateClosed {
		t.Fatal("expected closed state initially")
	}

	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.State() != stateClosed {
		t.Fatalf("expected still closed after success, got %d", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterFailures(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 2, ResetTimeout: 1 * time.Hour})
	_ = cb.Call(func() error { return errors.New("boom") })
	if cb.State() != stateClosed {
		t.Fatalf("expected closed after first failure, got %d", cb.State())
	}

	_ = cb.Call(func() error { return errors.New("boom") })
	if cb.State() != stateOpen {
		t.Fatalf("expected open after threshold, got %d", cb.State())
	}

	err := cb.Call(func() error { return nil })
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpenThenClose(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 1, ResetTimeout: 1 * time.Millisecond})
	_ = cb.Call(func() error { return errors.New("boom") })
	if cb.State() != stateOpen {
		t.Fatalf("expected open, got %d", cb.State())
	}

	time.Sleep(10 * time.Millisecond)

	err := cb.Call(func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.State() != stateClosed {
		t.Fatalf("expected closed after half-open success, got %d", cb.State())
	}
}

func TestCircuitBreaker_HalfOpenThenOpenAgain(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{FailureThreshold: 1, ResetTimeout: 1 * time.Millisecond})
	_ = cb.Call(func() error { return errors.New("boom") })
	if cb.State() != stateOpen {
		t.Fatalf("expected open, got %d", cb.State())
	}

	time.Sleep(10 * time.Millisecond)

	err := cb.Call(func() error { return errors.New("boom") })
	if err == nil {
		t.Fatal("expected error")
	}
	if cb.State() != stateOpen {
		t.Fatalf("expected open again after half-open failure, got %d", cb.State())
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	cfg := DefaultCircuitBreakerConfig()
	if cfg.FailureThreshold == 0 || cfg.ResetTimeout <= 0 {
		t.Fatal("expected non-zero defaults")
	}
	cb := NewCircuitBreaker(CircuitBreakerConfig{})
	if cb.cfg.FailureThreshold != cfg.FailureThreshold {
		t.Fatal("expected fallback to default threshold")
	}
}
