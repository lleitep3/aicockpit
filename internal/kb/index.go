package kb

import (
	"time"
)

// IndexProvider defines the interface for KB index implementations.
// Different providers (file-based, SQLite, Redis, etc.) can implement this.
type IndexProvider interface {
	// LoadIndex loads the KB index from the provider
	LoadIndex(roots []string) (*KBIndex, error)

	// SaveIndex saves the KB index to the provider
	SaveIndex(index *KBIndex) error

	// InvalidateCache invalidates the cache for a specific document
	InvalidateCache(docPath string) error

	// RebuildIndex rebuilds the entire index from scratch
	RebuildIndex(roots []string) (*KBIndex, error)

	// IsValid checks if the index is valid and up-to-date
	IsValid() bool
}

// KBIndex represents the complete knowledge base index.
type KBIndex struct {
	Version     string      `json:"version"`
	LastUpdated time.Time   `json:"last_updated"`
	Roots       []RootIndex `json:"roots"`
}

// RootIndex represents the index for a single KB root directory.
type RootIndex struct {
	Path      string       `json:"path"`
	Documents []IndexEntry `json:"documents"`
}

// IndexEntry represents a single document in the index.
type IndexEntry struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Path        string    `json:"path"`
	Hash        string    `json:"hash"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
}
