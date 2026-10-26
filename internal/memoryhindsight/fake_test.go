package memoryhindsight

import (
	"context"
	"errors"
	"testing"

	"github.com/lleitep3/aicockpit/internal/memory"
)

func TestFakeRecordsAndCopies(t *testing.T) {
	fake := &Fake{
		RetainErr:   map[string]error{"project-demo": errors.New("retain down")},
		RecallHits:  map[string][]memory.Hit{"project-demo": {{Item: memory.Item{Content: "fact", Tags: []string{"a"}, Metadata: map[string]string{"k": "v"}}, Score: 1}}},
		ReflectText: map[string]string{"user-ada": "answer"},
		ReflectHits: map[string][]memory.Hit{"user-ada": {{Item: memory.Item{Content: "because"}}}},
	}
	ctx := context.Background()
	if err := fake.Retain(ctx, "project-demo", []memory.Item{{Content: "one", Tags: []string{"t"}}}); err == nil {
		t.Fatal("expected retain error")
	}
	hits, err := fake.Recall(ctx, "project-demo", memory.Query{Text: "one", Tags: []string{"t"}})
	if err != nil || len(hits) != 1 {
		t.Fatalf("recall = %+v, %v", hits, err)
	}
	hits[0].Item.Tags[0] = "changed"
	hits[0].Item.Metadata["k"] = "changed"
	again, err := fake.Recall(ctx, "missing", memory.Query{Text: "one"})
	if err != nil || again != nil {
		t.Fatalf("missing bank = %+v, %v", again, err)
	}
	text, facts, err := fake.Reflect(ctx, "user-ada", memory.Query{Text: "why"})
	if err != nil || text != "answer" || len(facts) != 1 {
		t.Fatalf("reflect = %q %+v %v", text, facts, err)
	}
	if _, _, err := fake.Reflect(nil, "user-ada", memory.Query{}); !errors.Is(err, memory.ErrNilContext) { //nolint:staticcheck // SA1012: nil context is the case under test.
		t.Fatalf("nil ctx = %v", err)
	}
	calls := fake.Snapshot()
	if len(calls) != 4 {
		t.Fatalf("calls = %d", len(calls))
	}
	if calls[1].Query.Tags[0] != "t" || calls[1].Op != "recall" {
		t.Fatalf("recall call = %+v", calls[1])
	}
	fresh, err := fake.Recall(ctx, "project-demo", memory.Query{Text: "one"})
	if err != nil || fresh[0].Item.Tags[0] != "a" || fresh[0].Item.Metadata["k"] != "v" {
		t.Fatalf("stored hit mutated: %+v", fresh)
	}
}
