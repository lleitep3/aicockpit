package resilience

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Circuit state values.
const (
	stateClosed = iota
	stateOpen
	stateHalfOpen
)

// CircuitBreakerConfig configures the circuit breaker.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold uint32
	// ResetTimeout is how long the circuit stays open before moving to half-open.
	ResetTimeout time.Duration
}

// DefaultCircuitBreakerConfig returns a sensible default configuration.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		ResetTimeout:     30 * time.Second,
	}
}

// CircuitBreaker is a simple in-memory circuit breaker.
type CircuitBreaker struct {
	cfg             CircuitBreakerConfig
	mu              sync.Mutex
	state           int
	failures        uint32
	lastFailureTime time.Time
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is open.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// NewCircuitBreaker creates a new CircuitBreaker.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = DefaultCircuitBreakerConfig().FailureThreshold
	}
	if cfg.ResetTimeout <= 0 {
		cfg.ResetTimeout = DefaultCircuitBreakerConfig().ResetTimeout
	}
	return &CircuitBreaker{cfg: cfg}
}

// Call executes op if the circuit allows it. It records successes and failures.
// When the circuit is open, Call returns ErrCircuitOpen without running op.
func (cb *CircuitBreaker) Call(op func() error) error {
	cb.mu.Lock()
	cb.transition()
	if cb.state == stateOpen {
		cb.mu.Unlock()
		return ErrCircuitOpen
	}
	wasHalfOpen := cb.state == stateHalfOpen
	cb.mu.Unlock()

	err := op()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failures++
		cb.lastFailureTime = time.Now()
		if cb.failures >= cb.cfg.FailureThreshold {
			cb.state = stateOpen
		}
		if wasHalfOpen {
			cb.state = stateOpen
		}
		return fmt.Errorf("circuit breaker recorded failure: %w", err)
	}

	cb.failures = 0
	cb.state = stateClosed
	return nil
}

// transition moves from open to half-open after the reset timeout.
func (cb *CircuitBreaker) transition() {
	if cb.state == stateOpen && time.Since(cb.lastFailureTime) > cb.cfg.ResetTimeout {
		cb.state = stateHalfOpen
	}
}

// State returns the current state of the circuit breaker for tests/observability.
// 0 = closed, 1 = open, 2 = half-open.
func (cb *CircuitBreaker) State() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transition()
	return cb.state
}
