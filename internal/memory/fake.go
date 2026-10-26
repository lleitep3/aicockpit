package memory

import (
	"context"
	"sync"
)

// Fake is an in-memory Memory test double. It records calls and stores items
// with the same rules as InMemory. Set RetainErr, RecallErr, or ReflectErr to
// return that error instead of touching storage. ReflectErr may be
// ErrReflectNotSupported. Fake is safe for concurrent use.
type Fake struct {
	mu    sync.Mutex
	inner *InMemory
	// RetainErr, when set, is returned by Retain after the call is recorded.
	RetainErr error
	// RecallErr, when set, is returned by Recall after the call is recorded.
	RecallErr error
	// ReflectErr, when set, is returned by Reflect after the call is recorded.
	ReflectErr error
	// RetainCalls are the items passed to Retain, in order.
	RetainCalls []Item
	// RecallCalls are the queries passed to Recall, in order.
	RecallCalls []Query
	// ReflectCalls are the queries passed to Reflect, in order.
	ReflectCalls []Query
}

// NewFake returns an empty Fake.
func NewFake() *Fake {
	return &Fake{inner: NewInMemory()}
}

// Retain records the call, then stores the item unless RetainErr is set.
func (f *Fake) Retain(ctx context.Context, item Item) (Item, error) {
	f.mu.Lock()
	f.RetainCalls = append(f.RetainCalls, cloneItem(item))
	injected := f.RetainErr
	backend := f.inner
	f.mu.Unlock()

	if err := checkCtx(ctx, "retain"); err != nil {
		return Item{}, err
	}
	if injected != nil {
		return Item{}, injected
	}
	return backend.Retain(ctx, item)
}

// Recall records the call, then searches unless RecallErr is set.
func (f *Fake) Recall(ctx context.Context, query Query) (*RecallResult, error) {
	f.mu.Lock()
	f.RecallCalls = append(f.RecallCalls, cloneQuery(query))
	injected := f.RecallErr
	backend := f.inner
	f.mu.Unlock()

	if err := checkCtx(ctx, "recall"); err != nil {
		return nil, err
	}
	if injected != nil {
		return nil, injected
	}
	return backend.Recall(ctx, query)
}

// Reflect records the call. When ReflectErr is set, including
// ErrReflectNotSupported, Reflect returns it. Otherwise it summarizes matches.
func (f *Fake) Reflect(ctx context.Context, query Query) (*ReflectResult, error) {
	f.mu.Lock()
	f.ReflectCalls = append(f.ReflectCalls, cloneQuery(query))
	injected := f.ReflectErr
	backend := f.inner
	f.mu.Unlock()

	if err := checkCtx(ctx, "reflect"); err != nil {
		return nil, err
	}
	if injected != nil {
		return nil, injected
	}
	return backend.Reflect(ctx, query)
}

func cloneQuery(query Query) Query {
	query.Tags = append([]string(nil), query.Tags...)
	return query
}
