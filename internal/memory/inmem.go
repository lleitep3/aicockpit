package memory

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// InMemory is a process-local Memory. It is safe for concurrent use.
// Tests and callers without a knowledge base use it. Nothing is written to disk.
type InMemory struct {
	mu    sync.Mutex
	items []Item
}

// NewInMemory returns an empty in-memory backend.
func NewInMemory() *InMemory {
	return &InMemory{}
}

// Retain stores item, replacing an existing item with the same id and bank.
func (m *InMemory) Retain(ctx context.Context, item Item) (Item, error) {
	if err := checkCtx(ctx, "retain"); err != nil {
		return Item{}, err
	}
	stored, err := NormalizeItem(item)
	if err != nil {
		return Item{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, existing := range m.items {
		if existing.ID == stored.ID && existing.Bank == stored.Bank {
			m.items[i] = cloneItem(stored)
			return cloneItem(stored), nil
		}
	}
	m.items = append(m.items, cloneItem(stored))
	return cloneItem(stored), nil
}

// Recall returns items whose title, content, context, or tags match the query.
func (m *InMemory) Recall(ctx context.Context, query Query) (*RecallResult, error) {
	return m.recall(ctx, "recall", query)
}

// Reflect recalls matching items and joins their excerpts.
func (m *InMemory) Reflect(ctx context.Context, query Query) (*ReflectResult, error) {
	recalled, err := m.recall(ctx, "reflect", query)
	if err != nil {
		return nil, err
	}
	return summarize(recalled.Query, recalled.Hits), nil
}

func (m *InMemory) recall(ctx context.Context, op string, query Query) (*RecallResult, error) {
	if err := checkCtx(ctx, op); err != nil {
		return nil, err
	}
	normalized, search, err := NormalizeQuery(op, query)
	if err != nil {
		return nil, err
	}
	tokens := tokenize(search)
	m.mu.Lock()
	snapshot := make([]Item, len(m.items))
	for i := range m.items {
		snapshot[i] = cloneItem(m.items[i])
	}
	m.mu.Unlock()

	if len(tokens) == 0 {
		return &RecallResult{Query: displayQuery(normalized), Hits: []Hit{}, Total: 0}, nil
	}

	hits := make([]Hit, 0)
	for _, item := range snapshot {
		if normalized.Bank != "" && item.Bank != normalized.Bank {
			continue
		}
		if !hasTags(item.Tags, normalized.Tags) {
			continue
		}
		score := scoreItem(item, tokens)
		if score == 0 {
			continue
		}
		hits = append(hits, Hit{Item: item, Score: score})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if hits[i].Item.ID != hits[j].Item.ID {
			return hits[i].Item.ID < hits[j].Item.ID
		}
		return hits[i].Item.Bank < hits[j].Item.Bank
	})
	limited, total := limitHits(hits, normalized.Limit)
	return &RecallResult{Query: displayQuery(normalized), Hits: limited, Total: total}, nil
}

func scoreItem(item Item, tokens []string) float64 {
	haystack := strings.ToLower(strings.Join([]string{
		item.Title,
		item.Content,
		item.Context,
		strings.Join(item.Tags, " "),
	}, " "))
	matched := 0
	for _, token := range tokens {
		if strings.Contains(haystack, token) {
			matched++
		}
	}
	if matched == 0 {
		return 0
	}
	return float64(matched) / float64(len(tokens))
}
