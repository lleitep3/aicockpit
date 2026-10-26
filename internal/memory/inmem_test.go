package memory

import (
	"context"
	"sync"
	"testing"
)

func TestInMemoryRankingAndIsolation(t *testing.T) {
	mem := NewInMemory()
	ctx := context.Background()
	items := []Item{
		{ID: "b", Content: "red green"},
		{ID: "a", Content: "red blue"},
		{ID: "c", Content: "unrelated moss"},
	}
	for _, item := range items {
		if _, err := mem.Retain(ctx, item); err != nil {
			t.Fatalf("Retain: %v", err)
		}
	}

	ranked, err := mem.Recall(ctx, Query{Text: "red blue"})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if ranked.Total != 2 {
		t.Fatalf("total = %d", ranked.Total)
	}
	if ranked.Hits[0].Item.ID != "a" || ranked.Hits[1].Item.ID != "b" {
		t.Fatalf("order = %s %s", ranked.Hits[0].Item.ID, ranked.Hits[1].Item.ID)
	}
	if ranked.Hits[0].Score <= ranked.Hits[1].Score {
		t.Fatalf("scores = %v %v", ranked.Hits[0].Score, ranked.Hits[1].Score)
	}

	tied := NewInMemory()
	for _, item := range []Item{
		{ID: "same", Bank: "studio", Content: "sameword token"},
		{ID: "same", Bank: "kitchen", Content: "sameword token"},
	} {
		if _, err := tied.Retain(ctx, item); err != nil {
			t.Fatalf("Retain tied: %v", err)
		}
	}
	ties, err := tied.Recall(ctx, Query{Text: "sameword"})
	if err != nil {
		t.Fatalf("Recall tied: %v", err)
	}
	if len(ties.Hits) != 2 || ties.Hits[0].Item.Bank != "kitchen" || ties.Hits[1].Item.Bank != "studio" {
		t.Fatalf("tie order = %+v %+v", ties.Hits[0].Item, ties.Hits[1].Item)
	}

	original := Item{Content: "isolated saffron", Tags: []string{"spice"}, Metadata: map[string]string{"k": "v"}}
	stored, err := mem.Retain(ctx, original)
	if err != nil {
		t.Fatalf("Retain isolated: %v", err)
	}
	original.Tags[0] = "mutated"
	original.Metadata["k"] = "mutated"
	stored.Tags[0] = "also"
	stored.Metadata["k"] = "also"
	found, err := mem.Recall(ctx, Query{Text: "saffron"})
	if err != nil {
		t.Fatalf("Recall isolated: %v", err)
	}
	if found.Hits[0].Item.Tags[0] != "spice" || found.Hits[0].Item.Metadata["k"] != "v" {
		t.Fatalf("isolated = %+v", found.Hits[0].Item)
	}
}

func TestInMemoryConcurrent(t *testing.T) {
	mem := NewInMemory()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			item := Item{Content: "concurrent memory " + string(rune('a'+i)) + " unique"}
			if _, err := mem.Retain(context.Background(), item); err != nil {
				t.Errorf("retain: %v", err)
			}
		}(i)
	}
	wg.Wait()

	res, err := mem.Recall(context.Background(), Query{Text: "concurrent", Limit: maxLimit})
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if res.Total != 20 {
		t.Fatalf("total = %d", res.Total)
	}
}
