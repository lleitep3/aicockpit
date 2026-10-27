package personalagent

import (
	"sync"
	"time"
)

const maximumBufferedTurnEvents = 4096
const turnSubscriberBuffer = 256

type eventHub struct {
	mu          sync.Mutex
	sequence    map[string]uint64
	history     map[string][]TurnEvent
	subscribers map[string]map[chan TurnEvent]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{
		sequence:    make(map[string]uint64),
		history:     make(map[string][]TurnEvent),
		subscribers: make(map[string]map[chan TurnEvent]struct{}),
	}
}

func (h *eventHub) Publish(turnID, eventType string, data map[string]any) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sequence[turnID]++
	event := TurnEvent{
		ID:         h.sequence[turnID],
		Type:       eventType,
		Data:       data,
		OccurredAt: time.Now().UTC(),
	}
	history := append(h.history[turnID], event)
	if len(history) > maximumBufferedTurnEvents {
		history = append([]TurnEvent(nil), history[len(history)-maximumBufferedTurnEvents:]...)
	}
	h.history[turnID] = history
	for subscriber := range h.subscribers[turnID] {
		select {
		case subscriber <- event:
		default:
			delete(h.subscribers[turnID], subscriber)
			close(subscriber)
		}
	}
}

func (h *eventHub) Subscribe(turnID string, after uint64) ([]TurnEvent, <-chan TurnEvent, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()
	history := make([]TurnEvent, 0)
	for _, event := range h.history[turnID] {
		if event.ID > after {
			history = append(history, event)
		}
	}
	bufferSize := turnSubscriberBuffer
	if len(history) > bufferSize {
		bufferSize = len(history)
	}
	subscriber := make(chan TurnEvent, bufferSize)
	if h.subscribers[turnID] == nil {
		h.subscribers[turnID] = make(map[chan TurnEvent]struct{})
	}
	h.subscribers[turnID][subscriber] = struct{}{}
	unsubscribe := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, exists := h.subscribers[turnID][subscriber]; exists {
			delete(h.subscribers[turnID], subscriber)
			close(subscriber)
		}
	}
	return history, subscriber, unsubscribe
}
