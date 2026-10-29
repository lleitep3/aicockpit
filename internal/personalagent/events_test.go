package personalagent

import "testing"

func TestEventHubReplaysStreamsAndUnsubscribes(t *testing.T) {
	hub := newEventHub()
	hub.Publish("turn_1", "status", map[string]any{"value": "started"})

	history, updates, unsubscribe := hub.Subscribe("turn_1", 0)
	defer unsubscribe()
	if len(history) != 1 || history[0].ID != 1 {
		t.Fatalf("history = %#v, want first event", history)
	}

	hub.Publish("turn_1", "delta", map[string]any{"value": "response"})
	select {
	case event := <-updates:
		if event.ID != 2 || event.Type != "delta" {
			t.Fatalf("stream event = %#v, want second delta", event)
		}
	default:
		t.Fatal("new event was not streamed")
	}

	unsubscribe()
	if _, open := <-updates; open {
		t.Fatal("subscriber channel remained open after unsubscribe")
	}
}

func TestEventHubBoundsHistory(t *testing.T) {
	hub := newEventHub()
	for index := 0; index <= maximumBufferedTurnEvents; index++ {
		hub.Publish("turn_1", "delta", map[string]any{"index": index})
	}

	history, _, unsubscribe := hub.Subscribe("turn_1", 0)
	defer unsubscribe()
	if len(history) != maximumBufferedTurnEvents {
		t.Fatalf("history length = %d, want %d", len(history), maximumBufferedTurnEvents)
	}
	if history[0].ID != 2 || history[len(history)-1].ID != maximumBufferedTurnEvents+1 {
		t.Fatalf("history IDs = %d..%d, want 2..%d", history[0].ID, history[len(history)-1].ID, maximumBufferedTurnEvents+1)
	}
}

func TestEventHubRemovesSlowSubscribers(t *testing.T) {
	hub := newEventHub()
	_, updates, unsubscribe := hub.Subscribe("turn_1", 0)
	for index := 0; index <= turnSubscriberBuffer; index++ {
		hub.Publish("turn_1", "delta", map[string]any{"index": index})
	}

	if _, open := <-updates; !open {
		t.Fatal("buffered subscriber events were discarded")
	}
	if _, open := <-updates; !open {
		t.Fatal("slow subscriber channel closed before buffer drained")
	}
	for range updates {
	}
	hub.mu.Lock()
	subscriberCount := len(hub.subscribers["turn_1"])
	hub.mu.Unlock()
	if subscriberCount != 0 {
		t.Fatalf("slow subscriber count = %d, want 0", subscriberCount)
	}
	unsubscribe()
}
