package memoryhindsight

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/memory"
)

func testBanks() Names {
	return Names{Project: "project-demo", User: "user-ada"}
}

func TestNewBackend(t *testing.T) {
	if _, err := New(nil, testBanks()); err == nil {
		t.Fatal("nil client should fail")
	}
	if _, err := New(&Fake{}, Names{Project: "project-demo", User: "project-demo"}); err == nil {
		t.Fatal("equal banks should fail")
	}
	backend, err := New(&Fake{}, testBanks())
	if err != nil {
		t.Fatal(err)
	}
	if backend == nil {
		t.Fatal("missing backend")
	}
}

func TestBackendRetainRecallReflect(t *testing.T) {
	stamp := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	fake := &Fake{
		RecallHits: map[string][]memory.Hit{
			"project-demo": {
				{Item: memory.Item{ID: "b", Content: "beta", Title: "Beta", Tags: []string{"keep"}}, Score: 0.2},
				{Item: memory.Item{ID: "a", Content: "alpha", Title: "Alpha", Tags: []string{"keep"}}, Score: 0.9},
				{Item: memory.Item{ID: "c", Content: "drop", Title: "Drop", Tags: []string{"other"}}, Score: 1},
			},
			"user-ada": {
				{Item: memory.Item{ID: "u", Content: "user fact", Title: "User", Tags: []string{"keep"}}, Score: 0.4},
			},
		},
		ReflectText: map[string]string{
			"project-demo": "Project answer",
			"user-ada":     "User answer",
		},
		ReflectHits: map[string][]memory.Hit{
			"project-demo": {{Item: memory.Item{ID: "a", Content: "alpha", Tags: []string{"keep"}}}},
			"user-ada":     {{Item: memory.Item{ID: "u", Content: "user fact", Tags: []string{"keep"}}}},
		},
	}
	backend, err := New(fake, testBanks())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	stored, err := backend.Retain(ctx, memory.Item{
		Content:    "The aqueduct is open",
		Context:    "inspection",
		Tags:       []string{"crew"},
		DocumentID: "doc-1",
		Metadata:   map[string]string{"source": "kb"},
		Timestamp:  stamp,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stored.Bank != "project-demo" || stored.ID == "" {
		t.Fatalf("stored = %+v", stored)
	}

	userItem, err := backend.Retain(ctx, memory.Item{Content: "Ada prefers morning shifts", Bank: "user"})
	if err != nil {
		t.Fatal(err)
	}
	if userItem.Bank != "user-ada" {
		t.Fatalf("user bank = %s", userItem.Bank)
	}

	if _, err := backend.Retain(ctx, memory.Item{Content: "nope", Bank: "other"}); !errors.Is(err, memory.ErrInvalidBank) {
		t.Fatalf("invalid bank error = %v", err)
	}
	if _, err := backend.Retain(ctx, memory.Item{}); !errors.Is(err, memory.ErrEmptyContent) {
		t.Fatalf("empty content error = %v", err)
	}

	recalled, err := backend.Recall(ctx, memory.Query{Text: "shift", Tags: []string{"keep"}, Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if recalled.Total != 3 || len(recalled.Hits) != 2 {
		t.Fatalf("recall = %+v", recalled)
	}
	if recalled.Hits[0].Item.ID != "a" || recalled.Hits[1].Item.ID != "u" {
		t.Fatalf("order = %+v", recalled.Hits)
	}
	if recalled.Hits[0].Item.Bank != "project-demo" || recalled.Hits[1].Item.Bank != "user-ada" {
		t.Fatalf("banks = %+v", recalled.Hits)
	}

	projectOnly, err := backend.Recall(ctx, memory.Query{Text: "shift", Bank: "project", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if projectOnly.Total != 3 {
		t.Fatalf("project total = %d", projectOnly.Total)
	}
	for _, hit := range projectOnly.Hits {
		if hit.Item.Bank != "project-demo" {
			t.Fatalf("unexpected bank %s", hit.Item.Bank)
		}
	}

	reflected, err := backend.Reflect(ctx, memory.Query{Text: "shift", Context: "crew", Tags: []string{"keep"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(reflected.Text, "Project answer") || !strings.Contains(reflected.Text, "User answer") {
		t.Fatalf("reflect = %q", reflected.Text)
	}
	if !strings.Contains(reflected.Text, "### project-demo") || len(reflected.Hits) != 2 {
		t.Fatalf("reflect = %+v", reflected)
	}
	if reflected.Query != "shift" {
		t.Fatalf("query = %q", reflected.Query)
	}

	calls := fake.Snapshot()
	var retains, recalls, reflects int
	for _, call := range calls {
		switch call.Op {
		case "retain":
			retains++
			if len(call.Items) != 1 || call.Items[0].Content == "" {
				t.Fatalf("retain call = %+v", call)
			}
		case "recall":
			recalls++
		case "reflect":
			reflects++
			if !strings.Contains(call.Query.Text, "crew") {
				t.Fatalf("reflect query = %+v", call.Query)
			}
		default:
			t.Fatalf("op = %s", call.Op)
		}
	}
	if retains != 2 || recalls < 2 || reflects != 2 {
		t.Fatalf("calls retain=%d recall=%d reflect=%d", retains, recalls, reflects)
	}
}

func TestBackendDegradesPerBank(t *testing.T) {
	fake := &Fake{
		RecallErr:   map[string]error{"user-ada": errors.New("down")},
		RecallHits:  map[string][]memory.Hit{"project-demo": {{Item: memory.Item{ID: "a", Content: "alpha"}, Score: 1}}},
		ReflectErr:  map[string]error{"project-demo": errors.New("down")},
		ReflectText: map[string]string{"user-ada": ""},
	}
	backend, err := New(fake, testBanks())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	recalled, err := backend.Recall(ctx, memory.Query{Text: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if len(recalled.Hits) != 1 || recalled.Hits[0].Item.Bank != "project-demo" {
		t.Fatalf("recall = %+v", recalled)
	}
	reflected, err := backend.Reflect(ctx, memory.Query{Text: "alpha", Bank: "user-ada"})
	if err != nil {
		t.Fatal(err)
	}
	if reflected.Text != "No Hindsight memories matched the query." {
		t.Fatalf("text = %q", reflected.Text)
	}

	fake.RecallErr["project-demo"] = errors.New("down too")
	if _, err := backend.Recall(ctx, memory.Query{Text: "alpha"}); err == nil {
		t.Fatal("expected both banks to fail")
	}
	fake.ReflectErr["user-ada"] = errors.New("down")
	if _, err := backend.Reflect(ctx, memory.Query{Text: "alpha"}); err == nil {
		t.Fatal("expected reflect to fail")
	}
}

func TestBackendContextAndEmptyQuery(t *testing.T) {
	backend, err := New(&Fake{}, testBanks())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := backend.Retain(nil, memory.Item{Content: "x"}); !errors.Is(err, memory.ErrNilContext) { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatalf("retain nil = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := backend.Recall(ctx, memory.Query{Text: "x"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("recall canceled = %v", err)
	}
	if _, err := backend.Reflect(context.Background(), memory.Query{}); !errors.Is(err, memory.ErrEmptyQuery) {
		t.Fatalf("empty query = %v", err)
	}
	if _, err := backend.Recall(context.Background(), memory.Query{Text: "x", Bank: "../nope"}); !errors.Is(err, memory.ErrInvalidBank) {
		t.Fatalf("bad bank = %v", err)
	}
	contextOnly, err := backend.Recall(context.Background(), memory.Query{Context: "inspection"})
	if err != nil || contextOnly.Query != "inspection" {
		t.Fatalf("context only = %+v %v", contextOnly, err)
	}
	if _, err := backend.Reflect(nil, memory.Query{Text: "x"}); !errors.Is(err, memory.ErrNilContext) { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatalf("reflect nil = %v", err)
	}
	wide, err := backend.Recall(context.Background(), memory.Query{Text: "x", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if wide.Total != 0 || len(wide.Hits) != 0 {
		t.Fatalf("empty recall = %+v", wide)
	}
}

func TestLimitHits(t *testing.T) {
	hits := []memory.Hit{{Score: 1}, {Score: 2}, {Score: 3}}
	limited, total := limitHits(hits, 0)
	if total != 3 || len(limited) != 3 {
		t.Fatalf("default limit = %d/%d", len(limited), total)
	}
	limited, total = limitHits(hits, 80)
	if total != 3 || len(limited) != 3 {
		t.Fatalf("capped unused = %d", len(limited))
	}
	if applyLimit(-1) != 10 || applyLimit(80) != 50 || applyLimit(3) != 3 {
		t.Fatalf("applyLimit defaults failed")
	}
}
