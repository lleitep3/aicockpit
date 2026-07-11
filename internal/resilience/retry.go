package resilience

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Default retry configuration values.
const (
	DefaultMaxAttempts  = 3
	DefaultInitialDelay = 500 * time.Millisecond
	DefaultMaxDelay     = 10 * time.Second
	DefaultMultiplier   = 2.0
)

// Config configures retry behavior.
type Config struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultConfig returns a sensible default retry configuration.
func DefaultConfig() Config {
	return Config{
		MaxAttempts:  DefaultMaxAttempts,
		InitialDelay: DefaultInitialDelay,
		MaxDelay:     DefaultMaxDelay,
		Multiplier:   DefaultMultiplier,
	}
}

// RetryableError is returned when an operation fails in a retryable way.
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	if e.Err == nil {
		return "retryable error"
	}
	return fmt.Sprintf("retryable: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// AsRetryable wraps an error as retryable.
func AsRetryable(err error) error {
	if err == nil {
		return nil
	}
	return &RetryableError{Err: err}
}

// IsRetryable reports whether err is a retryable error.
func IsRetryable(err error) bool {
	var re *RetryableError
	return err != nil && As(err, &re)
}

// As is a tiny helper that wraps errors.As for use inside the package.
// Exported as a variable so tests can override it if needed.
var As = func(err error, target interface{}) bool {
	return errors.As(err, target)
}

// Retry runs the provided operation until it succeeds, the context is cancelled,
// or the maximum number of attempts is reached. It sleeps with exponential backoff
// between attempts. The operation should return AsRetryable(err) for failures that
// merit a retry.
func Retry(ctx context.Context, cfg Config, op func() error) error {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = DefaultMaxAttempts
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = DefaultInitialDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = DefaultMaxDelay
	}
	if cfg.Multiplier <= 1 {
		cfg.Multiplier = DefaultMultiplier
	}

	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := op()
		if err == nil {
			return nil
		}

		lastErr = err
		if !IsRetryable(err) {
			return err
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		sleepCtx, cancel := context.WithTimeout(ctx, delay)
		<-sleepCtx.Done()
		cancel()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
