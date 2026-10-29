package bot

import (
	"context"
	"testing"
)

func TestCreateAndListObservers(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "O", Handle: "o"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	// Observers are managed through the bot store; here we verify the
	// observer CRUD boundary returns stable ids and round-trips state.
	obs, err := svc.CreateObserver(ctx, bot.ID, BotObserverKindEmail, ObserverConfig{})
	if err != nil {
		t.Fatalf("CreateObserver() error = %v", err)
	}
	if obs.ID == "" {
		t.Fatalf("CreateObserver() returned empty id")
	}
	if obs.BotID != bot.ID {
		t.Errorf("BotID = %q, want %q", obs.BotID, bot.ID)
	}
	if obs.Kind != BotObserverKindEmail {
		t.Errorf("Kind = %q, want %q", obs.Kind, BotObserverKindEmail)
	}

	list, err := svc.ListObservers(ctx, bot.ID)
	if err != nil {
		t.Fatalf("ListObservers() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListObservers() = %d, want 1", len(list))
	}
	// Verify the observer round-trips: same kind and a non-empty, consistent
	// id across both the writer and the reader.
	if list[0].Kind != BotObserverKindEmail {
		t.Errorf("Kind = %q, want %q", list[0].Kind, BotObserverKindEmail)
	}
	if list[0].ID == "" {
		t.Errorf("ListObservers() returned empty id")
	}
	if obs.ID != "" && list[0].ID != obs.ID {
		t.Errorf("ListObservers() id mismatch: got %q, want %q", list[0].ID, obs.ID)
	}
}

func TestGetObserverMissing(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "G", Handle: "g"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	if _, err := svc.GetObserver(ctx, bot.ID, "missing"); err != nil {
		t.Fatalf("GetObserver() error = %v", err)
	}
}

func TestUpdateObserverStatus(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "U", Handle: "u"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	obs, err := svc.CreateObserver(ctx, bot.ID, BotObserverKindPr, ObserverConfig{})
	if err != nil {
		t.Fatalf("CreateObserver() error = %v", err)
	}

	updated, err := svc.UpdateObserver(ctx, bot.ID, obs.ID, BotObserverStatusActive)
	if err != nil {
		t.Fatalf("UpdateObserver() error = %v", err)
	}
	if updated.Status != BotObserverStatusActive {
		t.Errorf("Status = %v, want %v", updated.Status, BotObserverStatusActive)
	}
}

func TestDeleteObserver(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "D", Handle: "d"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	obs, err := svc.CreateObserver(ctx, bot.ID, BotObserverKindWeb, ObserverConfig{})
	if err != nil {
		t.Fatalf("CreateObserver() error = %v", err)
	}
	if err := svc.DeleteObserver(ctx, bot.ID, obs.ID); err != nil {
		t.Fatalf("DeleteObserver() error = %v", err)
	}
	// GetObserver returns (nil, nil) for a deleted observer, not an error.
	if obs, err := svc.GetObserver(ctx, bot.ID, obs.ID); obs != nil || err != nil {
		t.Fatalf("GetObserver() after delete expected (nil, nil), got (%v, %v)", obs, err)
	}
}

func TestStartAllWithNoActiveObservers(t *testing.T) {
	svc, _ := newTestService(t)
	ctx := context.Background()

	bot, err := svc.CreateBot(ctx, &BotProfile{Name: "S", Handle: "s"})
	if err != nil {
		t.Fatalf("CreateBot() error = %v", err)
	}

	out, err := svc.StartAll(ctx, bot.ID)
	if err != nil {
		t.Fatalf("StartAll() error = %v", err)
	}
	// With no active observers, the hub closes the channel immediately.
	select {
	case <-out:
		// A closed, unbuffered-less channel is immediately readable:
		// the zero value is returned and ok = false. If the channel
		// still delivers a value (ok == true), it means an event.
		_, ok := <-out
		if ok {
			t.Fatalf("StartAll() returned an event with no active observers")
		}
		// Channel closed - expected behaviour when no active observers.
	case <-ctx.Done():
		t.Fatalf("StartAll() closed due to context: %v", ctx.Err())
	}
}
