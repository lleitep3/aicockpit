package memory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNormalizeItem(t *testing.T) {
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("UTC-3", -3*60*60))
	longTitle := strings.Repeat("á", titleLimit+5)

	tests := []struct {
		name     string
		item     Item
		wantErr  error
		wantText string
		check    func(t *testing.T, got Item)
	}{
		{
			name:    "blank content",
			item:    Item{Content: " \n\t "},
			wantErr: ErrEmptyContent,
		},
		{
			name:    "invalid id",
			item:    Item{ID: "../secret", Content: "kept"},
			wantErr: ErrInvalidID,
		},
		{
			name:    "invalid document id",
			item:    Item{Content: "kept", DocumentID: "has space"},
			wantErr: ErrInvalidID,
		},
		{
			name:    "invalid bank",
			item:    Item{Content: "kept", Bank: "Bad Bank"},
			wantErr: ErrInvalidBank,
		},
		{
			name:     "invalid metadata key",
			item:     Item{Content: "kept", Metadata: map[string]string{"bad key": "v"}},
			wantText: "invalid metadata key",
		},
		{
			name:     "metadata newline",
			item:     Item{Content: "kept", Metadata: map[string]string{"source": "a\nb"}},
			wantText: "contains a newline",
		},
		{
			name: "defaults and heading title",
			item: Item{
				Content: "  # Bridge\n\nThe span is closed.  ",
				Tags:    []string{" Civil ", " ", "memory", "bank:nope", "civil", "Alpha", "alpha"},
			},
			check: func(t *testing.T, got Item) {
				t.Helper()
				if got.Content != "# Bridge\n\nThe span is closed." {
					t.Fatalf("content = %q", got.Content)
				}
				if got.Title != "Bridge" {
					t.Fatalf("title = %q", got.Title)
				}
				if got.Bank != defaultBank {
					t.Fatalf("bank = %q", got.Bank)
				}
				if got.ID == "" || !idPattern.MatchString(got.ID) {
					t.Fatalf("id = %q", got.ID)
				}
				if got.Timestamp.IsZero() || got.Timestamp.Location() != time.UTC {
					t.Fatalf("timestamp = %v", got.Timestamp)
				}
				if len(got.Tags) != 2 || got.Tags[0] != "Civil" || got.Tags[1] != "Alpha" {
					t.Fatalf("tags = %#v", got.Tags)
				}
			},
		},
		{
			name: "explicit fields",
			item: Item{
				ID:         "note1",
				Title:      longTitle,
				Content:    "###",
				Context:    "  site  ",
				Bank:       "Field",
				DocumentID: "doc-9",
				Metadata:   map[string]string{"source": "crew"},
				Timestamp:  stamp,
			},
			check: func(t *testing.T, got Item) {
				t.Helper()
				if got.ID != "note1" || got.Bank != "field" || got.DocumentID != "doc-9" {
					t.Fatalf("identity = %+v", got)
				}
				if got.Title == longTitle || !strings.HasSuffix(got.Title, "...") {
					t.Fatalf("title = %q", got.Title)
				}
				if got.Context != "site" {
					t.Fatalf("context = %q", got.Context)
				}
				if !got.Timestamp.Equal(stamp) || got.Timestamp.Location() != time.UTC {
					t.Fatalf("timestamp = %v", got.Timestamp)
				}
				if got.Metadata["source"] != "crew" {
					t.Fatalf("metadata = %#v", got.Metadata)
				}
				if got.Title == "memory" {
					t.Fatal("explicit title was replaced")
				}
			},
		},
		{
			name: "hash-only content uses fallback title",
			item: Item{Content: "###"},
			check: func(t *testing.T, got Item) {
				t.Helper()
				if got.Title != fallbackTitle {
					t.Fatalf("title = %q", got.Title)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeItem(tt.item)
			if tt.wantErr != nil || tt.wantText != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				if tt.wantText != "" && !strings.Contains(err.Error(), tt.wantText) {
					t.Fatalf("error = %v, want text %q", err, tt.wantText)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeItem: %v", err)
			}
			tt.check(t, got)
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   Query
		want    string
		wantErr error
	}{
		{name: "empty", query: Query{Text: "  ", Context: " "}, wantErr: ErrEmptyQuery},
		{name: "invalid bank", query: Query{Text: "aqueduct", Bank: "../x"}, wantErr: ErrInvalidBank},
		{name: "text", query: Query{Text: " aqueduct "}, want: "aqueduct"},
		{name: "context only", query: Query{Context: " inspection "}, want: "inspection"},
		{name: "both", query: Query{Text: "aqueduct", Context: "inspection"}, want: "aqueduct inspection"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, search, err := NormalizeQuery("recall", tt.query)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeQuery: %v", err)
			}
			if search != tt.want {
				t.Fatalf("search = %q, want %q", search, tt.want)
			}
			if displayQuery(got) != strings.TrimSpace(firstNonEmpty(tt.query.Text, tt.query.Context)) {
				t.Fatalf("display = %q, want %q", displayQuery(got), strings.TrimSpace(firstNonEmpty(tt.query.Text, tt.query.Context)))
			}
		})
	}
}

func TestApplyLimitAndTruncate(t *testing.T) {
	limits := []struct {
		in, want int
	}{
		{0, defaultLimit},
		{-3, defaultLimit},
		{1, 1},
		{maxLimit, maxLimit},
		{maxLimit + 10, maxLimit},
	}
	for _, tt := range limits {
		if got := applyLimit(tt.in); got != tt.want {
			t.Errorf("applyLimit(%d) = %d, want %d", tt.in, got, tt.want)
		}
	}

	cuts := []struct {
		in   string
		max  int
		want string
	}{
		{in: "abc", max: 10, want: "abc"},
		{in: "abcd", max: 4, want: "abcd"},
		{in: "abcdefg", max: 6, want: "abc..."},
		{in: "abcdef", max: 2, want: "ab"},
		{in: "áááááá", max: 4, want: "á..."},
	}
	for _, tt := range cuts {
		if got := truncate(tt.in, tt.max); got != tt.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{name: "empty", in: "  ", want: nil},
		{name: "punctuation", in: "???", want: nil},
		{name: "words", in: "Hello, World", want: []string{"hello", "world"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("tokenize(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("tokenize(%q) = %#v, want %#v", tt.in, got, tt.want)
				}
			}
		})
	}
}

func TestSummarize(t *testing.T) {
	long := strings.Repeat("a", snippetLimit+25)
	tests := []struct {
		name string
		hits []Hit
		want string
	}{
		{name: "none", hits: nil, want: noMatchText},
		{
			name: "one",
			hits: []Hit{{Item: Item{Title: "T", Content: "body"}}},
			want: "Based on 1 local memory:\n- T: body",
		},
		{
			name: "untitled",
			hits: []Hit{{Item: Item{ID: "id1", Content: "body"}}},
			want: "Based on 1 local memory:\n- id1: body",
		},
		{
			name: "blank",
			hits: []Hit{{Item: Item{Content: "body"}}},
			want: "Based on 1 local memory:\n- memory: body",
		},
		{
			name: "two",
			hits: []Hit{
				{Item: Item{Title: "A", Content: "one"}},
				{Item: Item{Title: "B", Content: "two\nthree"}},
			},
			want: "Based on 2 local memories:\n- A: one\n- B: two three",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarize("q", tt.hits)
			if got.Text != tt.want {
				t.Fatalf("text = %q, want %q", got.Text, tt.want)
			}
			if got.Query != "q" {
				t.Fatalf("query = %q", got.Query)
			}
			if got.Hits == nil {
				t.Fatal("nil hits")
			}
		})
	}

	got := summarize("q", []Hit{{Item: Item{Title: "T", Content: long}}})
	if !strings.Contains(got.Text, "...") {
		t.Fatalf("text = %q", got.Text)
	}
}

func TestNewMemoryIDUnique(t *testing.T) {
	first := newMemoryID("same")
	second := newMemoryID("same")
	if first == second || !idPattern.MatchString(first) || !idPattern.MatchString(second) {
		t.Fatalf("ids = %s %s", first, second)
	}
}

func TestMemoryContract(t *testing.T) {
	backends := []struct {
		name string
		new  func(t *testing.T) Memory
	}{
		{name: "inmem", new: func(*testing.T) Memory { return NewInMemory() }},
		{name: "kb", new: func(t *testing.T) Memory { return newTestKB(t) }},
	}

	for _, backend := range backends {
		t.Run(backend.name, func(t *testing.T) {
			t.Run("round trip", func(t *testing.T) {
				testRoundTrip(t, backend.new(t))
			})
			t.Run("banks and tags", func(t *testing.T) {
				testBanksAndTags(t, backend.new(t))
			})
			t.Run("limit and upsert", func(t *testing.T) {
				testLimitAndUpsert(t, backend.new(t))
			})
			t.Run("empty and canceled", func(t *testing.T) {
				testEmptyAndCanceled(t, backend.new(t))
			})
			t.Run("no match and punctuation", func(t *testing.T) {
				testNoMatch(t, backend.new(t))
			})
		})
	}
}

func testRoundTrip(t *testing.T, mem Memory) {
	t.Helper()
	stamp := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	stored, err := mem.Retain(context.Background(), Item{
		ID:         "note1",
		Title:      "Inspection",
		Content:    "Remember the aqueduct inspection.",
		Context:    "site visit: north arch",
		Tags:       []string{"civil"},
		Bank:       "Field",
		DocumentID: "doc-9",
		Metadata:   map[string]string{"source": "crew", "zone": "north"},
		Timestamp:  stamp,
	})
	if err != nil {
		t.Fatalf("Retain: %v", err)
	}
	if stored.Bank != "field" || stored.ID != "note1" {
		t.Fatalf("stored = %+v", stored)
	}

	recalled, err := mem.Recall(context.Background(), Query{Text: "aqueduct"})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if recalled.Query != "aqueduct" || recalled.Total != 1 || len(recalled.Hits) != 1 {
		t.Fatalf("recall = %+v", recalled)
	}
	hit := recalled.Hits[0].Item
	if hit.Content != "Remember the aqueduct inspection." || hit.Title != "Inspection" {
		t.Fatalf("hit = %+v", hit)
	}
	if hit.Context != "site visit: north arch" || hit.Bank != "field" || hit.DocumentID != "doc-9" {
		t.Fatalf("hit context = %+v", hit)
	}
	if len(hit.Tags) != 1 || hit.Tags[0] != "civil" {
		t.Fatalf("tags = %#v", hit.Tags)
	}
	if hit.Metadata["source"] != "crew" || hit.Metadata["zone"] != "north" {
		t.Fatalf("metadata = %#v", hit.Metadata)
	}
	if !hit.Timestamp.Equal(stamp) {
		t.Fatalf("timestamp = %v", hit.Timestamp)
	}
	if recalled.Hits[0].Score <= 0 {
		t.Fatalf("score = %v", recalled.Hits[0].Score)
	}

	reflected, err := mem.Reflect(context.Background(), Query{Context: "aqueduct"})
	if err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if reflected.Query != "aqueduct" || !strings.Contains(reflected.Text, "Remember the aqueduct inspection.") {
		t.Fatalf("reflect = %+v", reflected)
	}
}

func testBanksAndTags(t *testing.T, mem Memory) {
	t.Helper()
	ctx := context.Background()
	items := []Item{
		{ID: "shared", Bank: "kitchen", Content: "kitchen shared pomegranate recipe", Tags: []string{"Recipe"}},
		{ID: "shared", Bank: "studio", Content: "studio shared pomegranate painting"},
	}
	for _, item := range items {
		if _, err := mem.Retain(ctx, item); err != nil {
			t.Fatalf("Retain: %v", err)
		}
	}

	kitchen, err := mem.Recall(ctx, Query{Text: "pomegranate", Bank: "kitchen"})
	if err != nil {
		t.Fatalf("Recall bank: %v", err)
	}
	if kitchen.Total != 1 || !strings.Contains(kitchen.Hits[0].Item.Content, "recipe") {
		t.Fatalf("kitchen = %+v", kitchen)
	}

	all, err := mem.Recall(ctx, Query{Text: "pomegranate"})
	if err != nil {
		t.Fatalf("Recall all: %v", err)
	}
	if all.Total != 2 {
		t.Fatalf("total = %d", all.Total)
	}

	tagged, err := mem.Recall(ctx, Query{Text: "pomegranate", Tags: []string{"recipe"}})
	if err != nil {
		t.Fatalf("Recall tags: %v", err)
	}
	if tagged.Total != 1 || tagged.Hits[0].Item.Bank != "kitchen" {
		t.Fatalf("tagged = %+v", tagged)
	}
}

func testLimitAndUpsert(t *testing.T, mem Memory) {
	t.Helper()
	ctx := context.Background()
	for _, content := range []string{"quota-sample-1", "quota-sample-2", "quota-sample-3"} {
		if _, err := mem.Retain(ctx, Item{Content: content}); err != nil {
			t.Fatalf("Retain: %v", err)
		}
	}
	limited, err := mem.Recall(ctx, Query{Text: "quota-sample", Limit: 1})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if limited.Total != 3 || len(limited.Hits) != 1 {
		t.Fatalf("limited total=%d hits=%d", limited.Total, len(limited.Hits))
	}

	if _, err := mem.Retain(ctx, Item{ID: "note", Content: "ledger-old-token"}); err != nil {
		t.Fatalf("Retain first: %v", err)
	}
	if _, err := mem.Retain(ctx, Item{ID: "note", Content: "ledger-new-token"}); err != nil {
		t.Fatalf("Retain second: %v", err)
	}
	found, err := mem.Recall(ctx, Query{Text: "ledger-new-token"})
	if err != nil {
		t.Fatalf("Recall new: %v", err)
	}
	if found.Total != 1 || !strings.Contains(found.Hits[0].Item.Content, "ledger-new-token") {
		t.Fatalf("found = %+v", found)
	}
	gone, err := mem.Recall(ctx, Query{Text: "ledger-old-token"})
	if err != nil {
		t.Fatalf("Recall alpha: %v", err)
	}
	if gone.Total != 0 || len(gone.Hits) != 0 {
		t.Fatalf("alpha hits = %+v", gone)
	}
}

func testEmptyAndCanceled(t *testing.T, mem Memory) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name    string
		ctx     context.Context
		call    func(context.Context) error
		wantErr error
	}{
		{
			name:    "nil retain",
			call:    func(ctx context.Context) error { _, err := mem.Retain(ctx, Item{Content: "x"}); return err },
			wantErr: ErrNilContext,
		},
		{
			name:    "nil recall",
			call:    func(ctx context.Context) error { _, err := mem.Recall(ctx, Query{Text: "x"}); return err },
			wantErr: ErrNilContext,
		},
		{
			name:    "nil reflect",
			call:    func(ctx context.Context) error { _, err := mem.Reflect(ctx, Query{Text: "x"}); return err },
			wantErr: ErrNilContext,
		},
		{
			name:    "canceled retain",
			ctx:     ctx,
			call:    func(ctx context.Context) error { _, err := mem.Retain(ctx, Item{Content: "x"}); return err },
			wantErr: context.Canceled,
		},
		{
			name:    "canceled recall",
			ctx:     ctx,
			call:    func(ctx context.Context) error { _, err := mem.Recall(ctx, Query{Text: "x"}); return err },
			wantErr: context.Canceled,
		},
		{
			name:    "canceled reflect",
			ctx:     ctx,
			call:    func(ctx context.Context) error { _, err := mem.Reflect(ctx, Query{Text: "x"}); return err },
			wantErr: context.Canceled,
		},
		{
			name:    "empty content",
			ctx:     context.Background(),
			call:    func(ctx context.Context) error { _, err := mem.Retain(ctx, Item{Content: " "}); return err },
			wantErr: ErrEmptyContent,
		},
		{
			name:    "empty recall",
			ctx:     context.Background(),
			call:    func(ctx context.Context) error { _, err := mem.Recall(ctx, Query{}); return err },
			wantErr: ErrEmptyQuery,
		},
		{
			name:    "empty reflect",
			ctx:     context.Background(),
			call:    func(ctx context.Context) error { _, err := mem.Reflect(ctx, Query{}); return err },
			wantErr: ErrEmptyQuery,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !errors.Is(tt.call(tt.ctx), tt.wantErr) {
				err := tt.call(tt.ctx)
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func testNoMatch(t *testing.T, mem Memory) {
	t.Helper()
	ctx := context.Background()
	if _, err := mem.Retain(ctx, Item{Content: "stored aqueduct note"}); err != nil {
		t.Fatalf("Retain: %v", err)
	}
	miss, err := mem.Recall(ctx, Query{Text: "zzzz-not-a-term"})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if miss.Total != 0 || len(miss.Hits) != 0 {
		t.Fatalf("miss = %+v", miss)
	}
	reflected, err := mem.Reflect(ctx, Query{Text: "zzzz-not-a-term"})
	if err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if reflected.Text != noMatchText {
		t.Fatalf("text = %q", reflected.Text)
	}

	punct, err := mem.Recall(ctx, Query{Text: "???"})
	if err != nil {
		t.Fatalf("Recall punctuation: %v", err)
	}
	if punct.Total != 0 {
		t.Fatalf("punctuation total = %d", punct.Total)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
