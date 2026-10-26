package memoryhindsight

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/lleitep3/aicockpit/internal/memory"
)

// Backend is a memory.Memory that stores facts in Hindsight.
// It is safe for concurrent use when Client is.
type Backend struct {
	client Client
	banks  Names
}

var _ memory.Memory = (*Backend)(nil)

// New returns a Hindsight memory backend. client and bank names are required.
func New(client Client, banks Names) (*Backend, error) {
	if client == nil {
		return nil, errors.New("memory-hindsight: nil client")
	}
	if err := banks.validate(); err != nil {
		return nil, fmt.Errorf("memory-hindsight: %w", err)
	}
	return &Backend{client: client, banks: banks}, nil
}

// Retain stores item in the project or user bank.
// An empty bank and "default" use the project bank. "user" uses the user bank.
func (b *Backend) Retain(ctx context.Context, item memory.Item) (memory.Item, error) {
	if err := checkCtx(ctx, "retain"); err != nil {
		return memory.Item{}, err
	}
	stored, err := memory.NormalizeItem(item)
	if err != nil {
		return memory.Item{}, err
	}
	bank, err := b.banks.retainBank(stored.Bank)
	if err != nil {
		return memory.Item{}, fmt.Errorf("memory retain: %w", err)
	}
	stored.Bank = bank
	if err := b.client.Retain(ctx, bank, []memory.Item{stored}); err != nil {
		return memory.Item{}, fmt.Errorf("memory retain: %w", err)
	}
	return stored, nil
}

// Recall searches the selected banks and merges the hits.
func (b *Backend) Recall(ctx context.Context, query memory.Query) (*memory.RecallResult, error) {
	return b.recall(ctx, "recall", query)
}

// Reflect asks the selected banks for a synthesized answer.
func (b *Backend) Reflect(ctx context.Context, query memory.Query) (*memory.ReflectResult, error) {
	if err := checkCtx(ctx, "reflect"); err != nil {
		return nil, err
	}
	normalized, search, err := memory.NormalizeQuery("reflect", query)
	if err != nil {
		return nil, err
	}
	banks, err := b.banks.queryBanks(normalized.Bank)
	if err != nil {
		return nil, fmt.Errorf("memory reflect: %w", err)
	}
	shown := displayQuery(normalized)
	clientQuery := memory.Query{Text: search, Tags: normalized.Tags, Limit: normalized.Limit}
	var parts []reflectPart
	var failures []error
	for _, bank := range banks {
		if err := checkCtx(ctx, "reflect"); err != nil {
			return nil, err
		}
		text, hits, err := b.client.Reflect(ctx, bank, clientQuery)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		parts = append(parts, reflectPart{bank: bank, text: strings.TrimSpace(text), hits: tagHits(bank, hits, normalized.Tags)})
	}
	if len(failures) == len(banks) {
		return nil, fmt.Errorf("memory reflect: %w", errors.Join(failures...))
	}
	hits := make([]memory.Hit, 0)
	for _, part := range parts {
		hits = append(hits, part.hits...)
	}
	limited, _ := limitHits(hits, normalized.Limit)
	return &memory.ReflectResult{Query: shown, Text: joinReflect(parts), Hits: limited}, nil
}

func (b *Backend) recall(ctx context.Context, op string, query memory.Query) (*memory.RecallResult, error) {
	if err := checkCtx(ctx, op); err != nil {
		return nil, err
	}
	normalized, search, err := memory.NormalizeQuery(op, query)
	if err != nil {
		return nil, err
	}
	banks, err := b.banks.queryBanks(normalized.Bank)
	if err != nil {
		return nil, fmt.Errorf("memory %s: %w", op, err)
	}
	clientQuery := memory.Query{Text: search, Tags: normalized.Tags, Limit: normalized.Limit}
	hits := make([]memory.Hit, 0)
	var failures []error
	for _, bank := range banks {
		if err := checkCtx(ctx, op); err != nil {
			return nil, err
		}
		found, err := b.client.Recall(ctx, bank, clientQuery)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		hits = append(hits, tagHits(bank, found, normalized.Tags)...)
	}
	if len(failures) == len(banks) {
		return nil, fmt.Errorf("memory %s: %w", op, errors.Join(failures...))
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score == hits[j].Score {
			if hits[i].Item.Bank == hits[j].Item.Bank {
				return hits[i].Item.ID < hits[j].Item.ID
			}
			return hits[i].Item.Bank < hits[j].Item.Bank
		}
		return hits[i].Score > hits[j].Score
	})
	limited, total := limitHits(hits, normalized.Limit)
	return &memory.RecallResult{Query: displayQuery(normalized), Hits: limited, Total: total}, nil
}

type reflectPart struct {
	bank string
	text string
	hits []memory.Hit
}

func joinReflect(parts []reflectPart) string {
	written := 0
	var b strings.Builder
	for _, part := range parts {
		if part.text == "" && len(part.hits) == 0 {
			continue
		}
		if written > 0 {
			b.WriteString("\n\n")
		}
		if len(parts) > 1 {
			fmt.Fprintf(&b, "### %s\n", part.bank)
		}
		if part.text != "" {
			b.WriteString(part.text)
		} else {
			b.WriteString("No Hindsight memories matched the query.")
		}
		written++
	}
	if written == 0 {
		return "No Hindsight memories matched the query."
	}
	return b.String()
}

func tagHits(bank string, hits []memory.Hit, tags []string) []memory.Hit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]memory.Hit, 0, len(hits))
	for _, hit := range hits {
		if !hasTags(hit.Item.Tags, tags) {
			continue
		}
		hit.Item.Bank = bank
		hit.Item.Tags = append([]string(nil), hit.Item.Tags...)
		hit.Item.Metadata = copyMeta(hit.Item.Metadata)
		out = append(out, hit)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func hasTags(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	for _, wanted := range want {
		found := false
		for _, got := range have {
			if strings.EqualFold(got, wanted) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func checkCtx(ctx context.Context, op string) error {
	if ctx == nil {
		return fmt.Errorf("memory %s: %w", op, memory.ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("memory %s: %w", op, err)
	}
	return nil
}

func displayQuery(query memory.Query) string {
	if query.Text != "" {
		return query.Text
	}
	return query.Context
}

func applyLimit(limit int) int {
	if limit <= 0 {
		return 10
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func limitHits(hits []memory.Hit, limit int) ([]memory.Hit, int) {
	total := len(hits)
	limit = applyLimit(limit)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]memory.Hit, len(hits))
	copy(out, hits)
	return out, total
}
