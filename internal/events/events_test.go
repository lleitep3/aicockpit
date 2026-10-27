package events

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Bus construction
// ---------------------------------------------------------------------------

func TestNew_ReturnsNonNilBus(t *testing.T) {
	b := New()
	if b == nil {
		t.Fatal("New() returned nil")
	}
}

// ---------------------------------------------------------------------------
// Subscribe + Publish basic flow
// ---------------------------------------------------------------------------

func TestPublish_SingleSubscriber(t *testing.T) {
	b := New()
	var got Event

	b.Subscribe(TopicPackageInstalled, func(e Event) error {
		got = e
		return nil
	})

	sent := Event{Topic: TopicPackageInstalled, Payload: "hello"}
	errs := b.Publish(sent)

	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if got.Topic != TopicPackageInstalled {
		t.Errorf("got topic %q, want %q", got.Topic, TopicPackageInstalled)
	}
	if got.Payload != "hello" {
		t.Errorf("got payload %v, want %q", got.Payload, "hello")
	}
}

func TestPublish_MultipleSubscribersSameTopic(t *testing.T) {
	b := New()
	calls := 0

	inc := func(_ Event) error { calls++; return nil }
	b.Subscribe(TopicPackageInstalled, inc)
	b.Subscribe(TopicPackageInstalled, inc)
	b.Subscribe(TopicPackageInstalled, inc)

	b.Publish(Event{Topic: TopicPackageInstalled})

	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestPublish_NoSubscribers_NoPanic(t *testing.T) {
	b := New()
	// Should not panic and should return nil slice
	errs := b.Publish(Event{Topic: TopicRegistryUpdated})
	if len(errs) != 0 {
		t.Errorf("expected no errors for no-subscriber publish, got %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Error collection — one fails, others still run
// ---------------------------------------------------------------------------

func TestPublish_HandlerErrorCollection(t *testing.T) {
	b := New()
	sentinel := errors.New("handler error")
	secondCalled := false

	b.Subscribe(TopicPackageUpgraded, func(_ Event) error { return sentinel })
	b.Subscribe(TopicPackageUpgraded, func(_ Event) error {
		secondCalled = true
		return nil
	})

	errs := b.Publish(Event{Topic: TopicPackageUpgraded})

	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !errors.Is(errs[0], sentinel) {
		t.Errorf("unexpected error: %v", errs[0])
	}
	if !secondCalled {
		t.Error("second handler was not called despite first handler returning an error")
	}
}

func TestPublish_MultipleErrors(t *testing.T) {
	b := New()
	err1 := errors.New("err1")
	err2 := errors.New("err2")

	b.Subscribe(TopicPackageUninstalled, func(_ Event) error { return err1 })
	b.Subscribe(TopicPackageUninstalled, func(_ Event) error { return err2 })

	errs := b.Publish(Event{Topic: TopicPackageUninstalled})
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
}

func TestPublish_NoErrors_ReturnsNil(t *testing.T) {
	b := New()
	b.Subscribe(TopicPackageInstalled, func(_ Event) error { return nil })

	errs := b.Publish(Event{Topic: TopicPackageInstalled})
	if errs != nil {
		t.Errorf("expected nil errors slice, got %v", errs)
	}
}

// ---------------------------------------------------------------------------
// Subscribers() count
// ---------------------------------------------------------------------------

func TestSubscribers_Count(t *testing.T) {
	b := New()

	if b.Subscribers(TopicPackageInstalled) != 0 {
		t.Error("expected 0 subscribers on a new bus")
	}

	b.Subscribe(TopicPackageInstalled, func(_ Event) error { return nil })
	if b.Subscribers(TopicPackageInstalled) != 1 {
		t.Errorf("expected 1 subscriber, got %d", b.Subscribers(TopicPackageInstalled))
	}

	b.Subscribe(TopicPackageInstalled, func(_ Event) error { return nil })
	if b.Subscribers(TopicPackageInstalled) != 2 {
		t.Errorf("expected 2 subscribers, got %d", b.Subscribers(TopicPackageInstalled))
	}

	// Different topic must not affect count
	if b.Subscribers(TopicPackageUninstalled) != 0 {
		t.Error("unrelated topic count should be 0")
	}
}

// ---------------------------------------------------------------------------
// Concurrent Subscribe + Publish (race detector)
// ---------------------------------------------------------------------------

func TestConcurrentSubscribeAndPublish(t *testing.T) {
	b := New()
	var wg sync.WaitGroup
	const goroutines = 20

	// Concurrent subscribes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Subscribe(TopicPackageInstalled, func(_ Event) error { return nil })
		}()
	}

	// Concurrent publishes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b.Publish(Event{Topic: TopicPackageInstalled})
		}()
	}

	wg.Wait()
}

func TestConcurrentSubscribersRead(t *testing.T) {
	b := New()
	b.Subscribe(TopicRegistryUpdated, func(_ Event) error { return nil })

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = b.Subscribers(TopicRegistryUpdated)
		}()
	}
	wg.Wait()
}

// ---------------------------------------------------------------------------
// All payload types can be created and published
// ---------------------------------------------------------------------------

func TestPayload_PackageInstalledPayload(t *testing.T) {
	b := New()
	var received PackageInstalledPayload

	b.Subscribe(TopicPackageInstalled, func(e Event) error {
		if p, ok := e.Payload.(PackageInstalledPayload); ok {
			received = p
		}
		return nil
	})

	payload := PackageInstalledPayload{
		PackageName:  "mypkg",
		Version:      "1.2.3",
		Author:       "alice",
		Description:  "A test package",
		Features:     Features{Skills: []string{"skill1"}, Rules: []string{"rule1"}},
		InstallPath:  "/opt/pkg",
		RegistryName: "default",
		Timestamp:    time.Now(),
	}
	b.Publish(Event{Topic: TopicPackageInstalled, Payload: payload})

	if received.PackageName != "mypkg" {
		t.Errorf("PackageName mismatch: got %q", received.PackageName)
	}
	if received.Version != "1.2.3" {
		t.Errorf("Version mismatch: got %q", received.Version)
	}
	if len(received.Features.Skills) != 1 || received.Features.Skills[0] != "skill1" {
		t.Errorf("Features.Skills mismatch: %v", received.Features.Skills)
	}
}

func TestPayload_PackageUninstalledPayload(t *testing.T) {
	b := New()
	var received PackageUninstalledPayload

	b.Subscribe(TopicPackageUninstalled, func(e Event) error {
		if p, ok := e.Payload.(PackageUninstalledPayload); ok {
			received = p
		}
		return nil
	})

	payload := PackageUninstalledPayload{
		PackageName: "mypkg",
		Version:     "1.0.0",
		InstallPath: "/opt/pkg",
		Features:    Features{Agents: []string{"agent1"}, Workflows: []string{"wf1"}, KB: []string{"kb1"}},
		Timestamp:   time.Now(),
	}
	b.Publish(Event{Topic: TopicPackageUninstalled, Payload: payload})

	if received.PackageName != "mypkg" {
		t.Errorf("PackageName mismatch: got %q", received.PackageName)
	}
}

func TestPayload_PackageUpgradedPayload(t *testing.T) {
	b := New()
	var received PackageUpgradedPayload

	b.Subscribe(TopicPackageUpgraded, func(e Event) error {
		if p, ok := e.Payload.(PackageUpgradedPayload); ok {
			received = p
		}
		return nil
	})

	payload := PackageUpgradedPayload{
		PackageName: "mypkg",
		OldVersion:  "1.0.0",
		NewVersion:  "2.0.0",
		InstallPath: "/opt/pkg",
		Features:    Features{},
		Timestamp:   time.Now(),
	}
	b.Publish(Event{Topic: TopicPackageUpgraded, Payload: payload})

	if received.OldVersion != "1.0.0" || received.NewVersion != "2.0.0" {
		t.Errorf("version mismatch: old=%q new=%q", received.OldVersion, received.NewVersion)
	}
}

func TestPayload_RegistryUpdatedPayload(t *testing.T) {
	b := New()
	var received RegistryUpdatedPayload

	b.Subscribe(TopicRegistryUpdated, func(e Event) error {
		if p, ok := e.Payload.(RegistryUpdatedPayload); ok {
			received = p
		}
		return nil
	})

	payload := RegistryUpdatedPayload{
		RegistryName: "community",
		Timestamp:    time.Now(),
	}
	b.Publish(Event{Topic: TopicRegistryUpdated, Payload: payload})

	if received.RegistryName != "community" {
		t.Errorf("RegistryName mismatch: got %q", received.RegistryName)
	}
}

// ---------------------------------------------------------------------------
// Topic constant values
// ---------------------------------------------------------------------------

func TestTopicValues(t *testing.T) {
	if TopicPackageInstalled != "package.installed" {
		t.Errorf("unexpected TopicPackageInstalled: %q", TopicPackageInstalled)
	}
	if TopicPackageUninstalled != "package.uninstalled" {
		t.Errorf("unexpected TopicPackageUninstalled: %q", TopicPackageUninstalled)
	}
	if TopicPackageUpgraded != "package.upgraded" {
		t.Errorf("unexpected TopicPackageUpgraded: %q", TopicPackageUpgraded)
	}
	if TopicRegistryUpdated != "registry.updated" {
		t.Errorf("unexpected TopicRegistryUpdated: %q", TopicRegistryUpdated)
	}
}

// ---------------------------------------------------------------------------
// Registration order is preserved
// ---------------------------------------------------------------------------

func TestPublish_OrderPreserved(t *testing.T) {
	b := New()
	order := make([]int, 0, 3)

	for i := 0; i < 3; i++ {
		n := i
		b.Subscribe(TopicPackageInstalled, func(_ Event) error {
			order = append(order, n)
			return nil
		})
	}

	b.Publish(Event{Topic: TopicPackageInstalled})

	if len(order) != 3 || order[0] != 0 || order[1] != 1 || order[2] != 2 {
		t.Errorf("handlers not called in registration order: %v", order)
	}
}

// ---------------------------------------------------------------------------
// Different topics are isolated
// ---------------------------------------------------------------------------

func TestPublish_TopicIsolation(t *testing.T) {
	b := New()
	var installCalled, uninstallCalled bool

	b.Subscribe(TopicPackageInstalled, func(_ Event) error {
		installCalled = true
		return nil
	})
	b.Subscribe(TopicPackageUninstalled, func(_ Event) error {
		uninstallCalled = true
		return nil
	})

	b.Publish(Event{Topic: TopicPackageInstalled})

	if !installCalled {
		t.Error("install handler not called")
	}
	if uninstallCalled {
		t.Error("uninstall handler should not have been called")
	}
}
