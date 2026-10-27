package memoryhindsight

import (
	"context"

	"github.com/lleitep3/aicockpit/internal/memory"
)

// Client is the Hindsight API surface used by Backend.
// Fake implements it in tests. API adapts the official Go client.
// Implementations must not be required for CI to reach a live server.
type Client interface {
	// Retain stores items in one bank.
	Retain(ctx context.Context, bank string, items []memory.Item) error
	// Recall returns hits for one bank. A missing bank returns no hits.
	Recall(ctx context.Context, bank string, query memory.Query) ([]memory.Hit, error)
	// Reflect returns the synthesized answer and the facts behind it.
	// A missing bank returns an empty answer.
	Reflect(ctx context.Context, bank string, query memory.Query) (string, []memory.Hit, error)
}
