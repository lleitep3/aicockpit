// Package bot manages observers: sources (email, PR, web) a bot watches and
// converts into BotEvents the bot's conversational loop can act on. Backends
// implement the ObserverBackend interface below and are registered by kind.
package bot

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ObserverBackend is implemented by each watching source.
type ObserverBackend interface {
	// Kind is the observer kind this backend serves.
	Kind() BotObserverKind

	// Start begins watching; events are pushed on the returned channel.
	Start(ctx context.Context, bot *Bot, cfg ObserverConfig) (<-chan BotEvent, error)

	// Stop ends watching.
	Stop(ctx context.Context, bot *Bot, cfg ObserverConfig) error
}

// backendRegistry maps an observer kind to its backend.
type backendRegistry struct {
	mu       sync.RWMutex
	backends map[BotObserverKind]ObserverBackend
}

var (
	registry = &backendRegistry{backends: make(map[BotObserverKind]ObserverBackend)}
	once     sync.Once
)

// Register adds a backend for a kind. It panics on nil backend or duplicate
// kind (internal invariant).
func Register(b Backend) {
	once.Do(func() {
		registry.mu.Lock()
		registry.backends = make(map[BotObserverKind]ObserverBackend)
		registry.mu.Unlock()
	})
	if b == nil {
		panic("bot: Register called with nil backend")
	}
	kind := b.Kind()
	if kind == "" {
		panic(fmt.Sprintf("bot: backend kind is empty: %T", b))
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.backends[kind]; exists {
		panic(fmt.Sprintf("bot: backend already registered for kind %q", kind))
	}
	registry.backends[kind] = b
}

// Backend is the interface a backend satisfies before registration.
type Backend interface {
	ObserverBackend
	// identity method so Register can be called with a concrete type.
}

// Kind returns the registered kind for a backend.
func Kind(b Backend) BotObserverKind {
	return b.Kind()
}

// Get returns the backend registered for kind, or nil when unknown.
func Get(kind BotObserverKind) ObserverBackend {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	b, ok := registry.backends[kind]
	if !ok {
		return nil
	}
	return b
}

// Start begins watching a single observer, pushing BotEvents on the returned
// channel. The caller must keep the channel open (or close it on Stop) and
// cancel ctx to end the watch.
func Start(ctx context.Context, bot *Bot, kind BotObserverKind, cfg ObserverConfig) (<-chan BotEvent, error) {
	b := Get(kind)
	if b == nil {
		return nil, fmt.Errorf("no backend for observer kind %q", kind)
	}
	return b.Start(ctx, bot, cfg)
}

// Stop ends watching a single observer.
func Stop(ctx context.Context, bot *Bot, kind BotObserverKind, cfg ObserverConfig) error {
	b := Get(kind)
	if b == nil {
		return fmt.Errorf("no backend for observer kind %q", kind)
	}
	return b.Stop(ctx, bot, cfg)
}

// CreateObserver persists a new observer on a bot.
func (s *Service) CreateObserver(ctx context.Context, botID string, kind BotObserverKind, cfg ObserverConfig) (*BotObserver, error) {
	if kind == "" {
		return nil, errors.New("observer kind is required")
	}
	bot, err := s.botStore.Load(ctx, botID)
	if err != nil {
		return nil, err
	}
	b, err := newIDBytes()
	if err != nil {
		return nil, err
	}
	observer := &BotObserver{
		ID:        botID + "/obs_" + hex.EncodeToString(b[:8]),
		BotID:     botID,
		Kind:      kind,
		Config:    cfg,
		Status:    BotObserverStatusInactive,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	bot.Observers = append(bot.Observers, *observer)
	if err := s.botStore.Save(ctx, bot); err != nil {
		return nil, err
	}
	return observer, nil
}

// GetObserver returns an observer by id, or nil when not found.
func (s *Service) GetObserver(ctx context.Context, botID, id string) (*BotObserver, error) {
	bot, err := s.botStore.Load(ctx, botID)
	if err != nil {
		return nil, err
	}
	for i := range bot.Observers {
		if bot.Observers[i].ID == id {
			return &bot.Observers[i], nil
		}
	}
	return nil, nil
}

// ListObservers returns every observer for a bot, most recently configured.
func (s *Service) ListObservers(ctx context.Context, botID string) ([]*BotObserver, error) {
	bot, err := s.botStore.Load(ctx, botID)
	if err != nil {
		return nil, err
	}
	out := make([]*BotObserver, 0, len(bot.Observers))
	for _, o := range bot.Observers {
		out = append(out, &o)
	}
	return out, nil
}

// UpdateObserver applies the supplied fields to an observer.
func (s *Service) UpdateObserver(ctx context.Context, botID, id string, status BotObserverStatus) (*BotObserver, error) {
	bot, err := s.botStore.Load(ctx, botID)
	if err != nil {
		return nil, err
	}
	for i := range bot.Observers {
		if bot.Observers[i].ID == id {
			obs := &bot.Observers[i]
			obs.Status = status
			obs.UpdatedAt = time.Now().UTC()
			return obs, s.botStore.Save(ctx, bot)
		}
	}
	return nil, errors.New("observer not found")
}

// DeleteObserver removes an observer from a bot.
func (s *Service) DeleteObserver(ctx context.Context, botID, id string) error {
	bot, err := s.botStore.Load(ctx, botID)
	if err != nil {
		return err
	}
	for i, o := range bot.Observers {
		if o.ID == id {
			bot.Observers = append(bot.Observers[:i], bot.Observers[i+1:]...)
			return s.botStore.Save(ctx, bot)
		}
	}
	return errors.New("observer not found")
}

// StartAll begins watching every active observer for a bot, mirroring events
// onto a single shared channel returned to the caller.
func (s *Service) StartAll(ctx context.Context, botID string) (<-chan BotEvent, error) {
	bot, err := s.botStore.Load(ctx, botID)
	if err != nil {
		return nil, err
	}
	out := make(chan BotEvent, 256)
	var wg sync.WaitGroup

	for i := range bot.Observers {
		observer := &bot.Observers[i]
		if observer.Status != BotObserverStatusActive {
			continue
		}
		wg.Add(1)
		go func(obs *BotObserver) {
			defer wg.Done()
			ch, err := Start(ctx, bot, observer.Kind, observer.Config)
			if err != nil {
				return
			}
			for {
				select {
				case <-ctx.Done():
					return
				case ev, ok := <-ch:
					if !ok {
						return
					}
					select {
					case out <- ev:
					case <-ctx.Done():
						return
					}
				}
			}
		}(observer)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out, nil
}
