// Package personalagent defines the domain types and errors shared by the
// local AI agent layer: conversations, messages, turns and the services that
// drive them.
package personalagent

import (
	"errors"
)

// Sentinel errors used across the personalagent layer.
var (
	// ErrNotFound is returned when an entity does not exist.
	ErrNotFound = errors.New("personalagent: not found")
)
