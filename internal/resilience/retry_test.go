package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetry_SuccessFirstTry(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), Config{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond}, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetry_SuccessAfterFailures(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), Config{MaxAttempts: 5, InitialDelay: 1 * time.Millisecond, Multiplier: 2}, func() error {
		calls++
		if calls < 3 {
			return AsRetryable(errors.New("transient"))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_NonRetryableStops(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), Config{MaxAttempts: 5, InitialDelay: 1 * time.Millisecond}, func() error {
		calls++
		return errors.New("fatal")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetry_MaxAttemptsExceeded(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), Config{MaxAttempts: 3, InitialDelay: 1 * time.Millisecond}, func() error {
		calls++
		return AsRetryable(errors.New("transient"))
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Retry(ctx, DefaultConfig(), func() error {
		return AsRetryable(errors.New("transient"))
	})
	if err != ctx.Err() {
		t.Fatalf("expected context error, got %v", err)
	}
}

func TestAsRetryable(t *testing.T) {
	if AsRetryable(nil) != nil {
		t.Fatal("AsRetryable(nil) should return nil")
	}
	err := AsRetryable(errors.New("boom"))
	if !IsRetryable(err) {
		t.Fatal("expected retryable")
	}
	var target *RetryableError
	if !errors.As(err, &target) {
		t.Fatal("expected *RetryableError")
	}
}

func TestIsRetryable_False(t *testing.T) {
	if IsRetryable(errors.New("plain")) {
		t.Fatal("plain error should not be retryable")
	}
	if IsRetryable(nil) {
		t.Fatal("nil should not be retryable")
	}
}

func TestRetryableError_Unwrap(t *testing.T) {
	inner := errors.New("inner")
	err := AsRetryable(inner)
	if !errors.Is(err, inner) {
		t.Fatal("expected errors.Is to find inner")
	}
}

func TestRetryableError_Error(t *testing.T) {
	err := &RetryableError{Err: nil}
	if err.Error() != "retryable error" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.MaxAttempts != DefaultMaxAttempts {
		t.Fatalf("unexpected MaxAttempts")
	}
	if cfg.InitialDelay != DefaultInitialDelay {
		t.Fatalf("unexpected InitialDelay")
	}
	if cfg.MaxDelay != DefaultMaxDelay {
		t.Fatalf("unexpected MaxDelay")
	}
	if cfg.Multiplier != DefaultMultiplier {
		t.Fatalf("unexpected Multiplier")
	}
}
