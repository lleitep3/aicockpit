package memoryhindsight

import (
	"context"
	"sync"

	"github.com/lleitep3/aicockpit/internal/memory"
)

// Call is one recorded client invocation.
type Call struct {
	// Op is retain, recall, or reflect.
	Op string
	// Bank is the Hindsight bank id.
	Bank string
	// Items are the retained items. Recall and reflect leave this empty.
	Items []memory.Item
	// Query is the recall or reflect query.
	Query memory.Query
}

// Fake is a Client that records calls and returns scripted results.
// It does not open a network connection.
type Fake struct {
	mu sync.Mutex
	// Calls are the invocations in order.
	Calls []Call
	// RetainErr fails Retain for a bank when set.
	RetainErr map[string]error
	// RecallErr fails Recall for a bank when set.
	RecallErr map[string]error
	// ReflectErr fails Reflect for a bank when set.
	ReflectErr map[string]error
	// RecallHits are returned by Recall. A missing bank returns no hits.
	RecallHits map[string][]memory.Hit
	// ReflectText is the synthesized answer for a bank.
	ReflectText map[string]string
	// ReflectHits are the facts returned with ReflectText.
	ReflectHits map[string][]memory.Hit
}

var _ Client = (*Fake)(nil)

// Retain records items.
func (f *Fake) Retain(ctx context.Context, bank string, items []memory.Item) error {
	if err := contextErr(ctx); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	copied := make([]memory.Item, len(items))
	for i, item := range items {
		copied[i] = cloneItem(item)
	}
	f.Calls = append(f.Calls, Call{Op: "retain", Bank: bank, Items: copied})
	return f.RetainErr[bank]
}

// Recall records the query and returns the scripted hits.
func (f *Fake) Recall(ctx context.Context, bank string, query memory.Query) ([]memory.Hit, error) {
	if err := contextErr(ctx); err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Op: "recall", Bank: bank, Query: cloneQuery(query)})
	if err := f.RecallErr[bank]; err != nil {
		return nil, err
	}
	return cloneHits(f.RecallHits[bank]), nil
}

// Reflect records the query and returns the scripted answer.
func (f *Fake) Reflect(ctx context.Context, bank string, query memory.Query) (string, []memory.Hit, error) {
	if err := contextErr(ctx); err != nil {
		return "", nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Calls = append(f.Calls, Call{Op: "reflect", Bank: bank, Query: cloneQuery(query)})
	if err := f.ReflectErr[bank]; err != nil {
		return "", nil, err
	}
	return f.ReflectText[bank], cloneHits(f.ReflectHits[bank]), nil
}

// Snapshot returns a copy of the recorded calls.
func (f *Fake) Snapshot() []Call {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]Call, len(f.Calls))
	for i, call := range f.Calls {
		out[i] = Call{Op: call.Op, Bank: call.Bank, Items: cloneItems(call.Items), Query: cloneQuery(call.Query)}
	}
	return out
}

func cloneItems(items []memory.Item) []memory.Item {
	if len(items) == 0 {
		return nil
	}
	out := make([]memory.Item, len(items))
	for i, item := range items {
		out[i] = cloneItem(item)
	}
	return out
}

func cloneHits(hits []memory.Hit) []memory.Hit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]memory.Hit, len(hits))
	for i, hit := range hits {
		hit.Item = cloneItem(hit.Item)
		out[i] = hit
	}
	return out
}

func cloneItem(item memory.Item) memory.Item {
	item.Tags = append([]string(nil), item.Tags...)
	item.Metadata = copyMeta(item.Metadata)
	return item
}

func contextErr(ctx context.Context) error {
	if ctx == nil {
		return memory.ErrNilContext
	}
	return ctx.Err()
}

func cloneQuery(query memory.Query) memory.Query {
	query.Tags = append([]string(nil), query.Tags...)
	return query
}
