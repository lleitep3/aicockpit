// Package events provides a lightweight synchronous pub/sub bus for
// package lifecycle notifications. Subscribers are called in registration
// order; any subscriber error is collected but does not abort the others.
package events

import (
	"sync"
	"time"
)

// Topic identifies the kind of event being published.
type Topic string

const (
	TopicPackageInstalled   Topic = "package.installed"
	TopicPackageUninstalled Topic = "package.uninstalled"
	TopicPackageUpgraded    Topic = "package.upgraded"
	TopicRegistryUpdated    Topic = "registry.updated"
)

// Features mirrors the subset of package features relevant to subscribers.
type Features struct {
	Skills    []string
	Rules     []string
	Agents    []string
	Workflows []string
	KB        []string
}

// PackageInstalledPayload carries data for TopicPackageInstalled events.
type PackageInstalledPayload struct {
	PackageName  string
	Version      string
	Author       string
	Description  string
	Features     Features
	InstallPath  string
	RegistryName string
	Timestamp    time.Time
}

// PackageUninstalledPayload carries data for TopicPackageUninstalled events.
type PackageUninstalledPayload struct {
	PackageName string
	Version     string
	InstallPath string
	Features    Features
	Timestamp   time.Time
}

// PackageUpgradedPayload carries data for TopicPackageUpgraded events.
type PackageUpgradedPayload struct {
	PackageName string
	OldVersion  string
	NewVersion  string
	InstallPath string
	Features    Features
	Timestamp   time.Time
}

// RegistryUpdatedPayload carries data for TopicRegistryUpdated events.
type RegistryUpdatedPayload struct {
	RegistryName string
	Timestamp    time.Time
}

// Event is the envelope passed to every subscriber.
type Event struct {
	Topic   Topic
	Payload interface{}
}

// Handler is a function that processes an event. It may return an error,
// which will be collected but will not prevent other handlers from running.
type Handler func(Event) error

// Bus is a synchronous pub/sub event bus. It is safe for concurrent use.
type Bus struct {
	mu       sync.RWMutex
	handlers map[Topic][]Handler
}

// New creates a new, empty Bus.
func New() *Bus {
	return &Bus{handlers: make(map[Topic][]Handler)}
}

// Subscribe registers h to be called for every event on topic.
func (b *Bus) Subscribe(topic Topic, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[topic] = append(b.handlers[topic], h)
}

// Publish dispatches an event to all registered handlers for its topic.
// All handlers are called; errors are collected and returned as a slice.
// A nil slice means no errors occurred.
func (b *Bus) Publish(e Event) []error {
	b.mu.RLock()
	handlers := make([]Handler, len(b.handlers[e.Topic]))
	copy(handlers, b.handlers[e.Topic])
	b.mu.RUnlock()

	var errs []error
	for _, h := range handlers {
		if err := h(e); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// Subscribers returns the number of handlers registered for a topic.
func (b *Bus) Subscribers(topic Topic) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[topic])
}
