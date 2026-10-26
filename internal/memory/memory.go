package memory

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	tagMemory     = "memory"
	bankTagPrefix = "bank:"
	docRefPrefix  = "doc:"
	metaRefPrefix = "meta:"
	memoryAuthor  = "aicockpit-memory"
	memoryVersion = "1"
	defaultBank   = "default"
	memoryDir     = "memory"
	defaultLimit  = 10
	maxLimit      = 50
	titleLimit    = 120
	snippetLimit  = 500
	noMatchText   = "No local memories matched the query."
	fallbackTitle = "memory"
)

var (
	// ErrEmptyContent is returned by Retain when the item content is blank.
	ErrEmptyContent = errors.New("empty content")
	// ErrEmptyQuery is returned by Recall and Reflect when Text and Context are both blank.
	ErrEmptyQuery = errors.New("empty query")
	// ErrInvalidID is returned when a memory id or document id is not a safe token.
	ErrInvalidID = errors.New("invalid id")
	// ErrInvalidBank is returned when a bank name is not a safe token.
	ErrInvalidBank = errors.New("invalid bank")
	// ErrNilContext is returned when a method is called with a nil context.
	ErrNilContext = errors.New("nil context")
	// ErrReflectNotSupported is returned by backends that cannot summarize memories.
	// KB and InMemory aggregate matches instead. A later backend may return this error.
	ErrReflectNotSupported = errors.New("reflect is not supported")
)

var (
	idPattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	bankPattern    = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)
	metaKeyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
	memorySeq      atomic.Uint64
)

// Memory retains facts, recalls them, and reflects over what was stored.
// Implementations must honor cancellation and wrap returned errors.
type Memory interface {
	// Retain stores one item and returns the item as stored, including
	// any id, bank, or timestamp the backend filled in.
	Retain(ctx context.Context, item Item) (Item, error)
	// Recall searches stored memories and existing knowledge-base documents.
	Recall(ctx context.Context, query Query) (*RecallResult, error)
	// Reflect summarizes memories that match the query.
	Reflect(ctx context.Context, query Query) (*ReflectResult, error)
}

// Item is one memory. Fields are the subset a knowledge-base document and a
// later Hindsight retain item can both carry.
type Item struct {
	// ID is the memory id. Empty on input asks the backend to assign one.
	// KB accepts only path-safe tokens.
	ID string
	// Title is an optional short label. Empty on input derives one from Content.
	Title string
	// Content is the retained text. It is required.
	Content string
	// Context is the situation the fact was observed in, such as a conversation
	// or a document. It is searchable and is not a hard filter.
	Context string
	// Tags are caller labels. The names "memory" and "bank:*" are reserved.
	Tags []string
	// Bank partitions memories. Empty becomes "default" on Retain.
	// An empty Query.Bank searches every bank.
	Bank string
	// DocumentID groups a memory with a source document.
	DocumentID string
	// Metadata is string metadata copied through with the item.
	Metadata map[string]string
	// Timestamp is when the fact was observed. Zero becomes the current time.
	Timestamp time.Time
}

// Query selects memories for Recall and Reflect.
type Query struct {
	// Text is the search text. Recall and Reflect require Text or Context.
	Text string
	// Context is appended to the search text so situational words participate.
	// It does not drop hits that lack an exact context match.
	Context string
	// Tags, when set, keep only items that carry every tag.
	Tags []string
	// Bank, when set, keeps only items stored in that bank.
	Bank string
	// Limit caps the number of hits. Zero or negative uses 10. Values above 50
	// are capped at 50.
	Limit int
}

// Hit is one recalled memory.
type Hit struct {
	// Item is the matched memory.
	Item Item
	// Score is a backend relevance score. Higher is better.
	Score float64
	// Path is the knowledge-base path when the hit came from KB. Other
	// backends leave it empty.
	Path string
}

// RecallResult is the outcome of Recall.
type RecallResult struct {
	// Query is the text that was searched, or the context when text was empty.
	Query string
	// Hits are the matches, best first, after Limit is applied.
	Hits []Hit
	// Total is the number of matches before Limit.
	Total int
}

// ReflectResult is the outcome of Reflect.
type ReflectResult struct {
	// Query is the text that was summarized.
	Query string
	// Text is the local summary. KB and InMemory join the matching excerpts.
	// A Hindsight backend replaces this with its synthesized answer.
	Text string
	// Hits are the memories included in Text.
	Hits []Hit
}

func checkCtx(ctx context.Context, op string) error {
	if ctx == nil {
		return fmt.Errorf("memory %s: %w", op, ErrNilContext)
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("memory %s: %w", op, err)
	}
	return nil
}

func normalizeItem(item Item) (Item, error) {
	if strings.TrimSpace(item.Content) == "" {
		return Item{}, fmt.Errorf("memory retain: %w", ErrEmptyContent)
	}
	item.Content = strings.TrimSpace(item.Content)
	item.Context = strings.TrimSpace(item.Context)
	item.Title = strings.TrimSpace(item.Title)
	if item.Title == "" {
		item.Title = deriveTitle(item.Content)
	} else {
		item.Title = truncate(item.Title, titleLimit)
	}

	bank, err := normalizeBank(item.Bank, false)
	if err != nil {
		return Item{}, fmt.Errorf("memory retain: %w", err)
	}
	item.Bank = bank

	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = newMemoryID(item.Content)
	} else if !idPattern.MatchString(item.ID) {
		return Item{}, fmt.Errorf("memory retain: %w", ErrInvalidID)
	}

	item.DocumentID = strings.TrimSpace(item.DocumentID)
	if item.DocumentID != "" && !idPattern.MatchString(item.DocumentID) {
		return Item{}, fmt.Errorf("memory retain: document id: %w", ErrInvalidID)
	}

	item.Tags = normalizeTags(item.Tags)
	meta, err := normalizeMeta(item.Metadata)
	if err != nil {
		return Item{}, err
	}
	item.Metadata = meta

	if item.Timestamp.IsZero() {
		item.Timestamp = time.Now().UTC()
	} else {
		item.Timestamp = item.Timestamp.UTC()
	}
	return cloneItem(item), nil
}

func normalizeBank(bank string, allowEmpty bool) (string, error) {
	bank = strings.ToLower(strings.TrimSpace(bank))
	if bank == "" {
		if allowEmpty {
			return "", nil
		}
		return defaultBank, nil
	}
	if !bankPattern.MatchString(bank) {
		return "", ErrInvalidBank
	}
	return bank, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		lower := strings.ToLower(tag)
		if lower == tagMemory || strings.HasPrefix(lower, bankTagPrefix) {
			continue
		}
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, tag)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeMeta(meta map[string]string) (map[string]string, error) {
	if len(meta) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(meta))
	for key, value := range meta {
		key = strings.TrimSpace(key)
		if !metaKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("memory retain: invalid metadata key %q", key)
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, fmt.Errorf("memory retain: metadata %q contains a newline", key)
		}
		out[key] = value
	}
	return out, nil
}

func normalizeQuery(op string, query Query) (Query, string, error) {
	query.Text = strings.TrimSpace(query.Text)
	query.Context = strings.TrimSpace(query.Context)
	if query.Text == "" && query.Context == "" {
		return Query{}, "", fmt.Errorf("memory %s: %w", op, ErrEmptyQuery)
	}
	bank, err := normalizeBank(query.Bank, true)
	if err != nil {
		return Query{}, "", fmt.Errorf("memory %s: %w", op, err)
	}
	query.Bank = bank
	query.Tags = normalizeTags(query.Tags)
	search := query.Text
	if query.Context != "" {
		if search != "" {
			search += " "
		}
		search += query.Context
	}
	return query, search, nil
}

func deriveTitle(content string) string {
	line := content
	if i := strings.IndexAny(content, "\r\n"); i >= 0 {
		line = content[:i]
	}
	line = strings.TrimSpace(strings.TrimLeft(line, "#"))
	if line == "" {
		return fallbackTitle
	}
	return truncate(line, 80)
}

func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}

func newMemoryID(content string) string {
	n := memorySeq.Add(1)
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d|%s|%s", n, time.Now().UTC().Format(time.RFC3339Nano), content)))
	return "mem-" + hex.EncodeToString(sum[:8])
}

func cloneItem(item Item) Item {
	item.Tags = append([]string(nil), item.Tags...)
	if len(item.Metadata) > 0 {
		meta := make(map[string]string, len(item.Metadata))
		for key, value := range item.Metadata {
			meta[key] = value
		}
		item.Metadata = meta
	}
	return item
}

func displayQuery(query Query) string {
	if query.Text != "" {
		return query.Text
	}
	return query.Context
}

func applyLimit(limit int) int {
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return limit
}

func limitHits(hits []Hit, limit int) ([]Hit, int) {
	total := len(hits)
	limit = applyLimit(limit)
	if len(hits) > limit {
		hits = hits[:limit]
	}
	out := make([]Hit, len(hits))
	copy(out, hits)
	return out, total
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

func tokenize(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.Trim(field, ".,;:!?\"'()[]{}")
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func summarize(query string, hits []Hit) *ReflectResult {
	if hits == nil {
		hits = []Hit{}
	}
	var b strings.Builder
	switch len(hits) {
	case 0:
		b.WriteString(noMatchText)
	default:
		noun := "memories"
		if len(hits) == 1 {
			noun = "memory"
		}
		fmt.Fprintf(&b, "Based on %d local %s:\n", len(hits), noun)
		for i, hit := range hits {
			if i > 0 {
				b.WriteByte('\n')
			}
			title := strings.TrimSpace(hit.Item.Title)
			if title == "" {
				title = hit.Item.ID
			}
			if title == "" {
				title = fallbackTitle
			}
			snippet := truncate(strings.Join(strings.Fields(hit.Item.Content), " "), snippetLimit)
			fmt.Fprintf(&b, "- %s: %s", title, snippet)
		}
	}
	return &ReflectResult{Query: query, Text: b.String(), Hits: hits}
}
