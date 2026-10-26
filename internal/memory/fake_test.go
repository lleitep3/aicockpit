package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
)

func TestFakeRecordsAndInjectsErrors(t *testing.T) {
	fake := NewFake()
	ctx := context.Background()
	item := Item{Content: "stored token", Tags: []string{"one"}, Metadata: map[string]string{"k": "v"}}
	if _, err := fake.Retain(ctx, item); err != nil {
		t.Fatalf("Retain: %v", err)
	}
	item.Tags[0] = "mutated"
	item.Metadata["k"] = "mutated"
	if fake.RetainCalls[0].Tags[0] != "one" || fake.RetainCalls[0].Metadata["k"] != "v" {
		t.Fatalf("recorded = %+v", fake.RetainCalls[0])
	}

	query := Query{Text: "stored", Tags: []string{"one"}}
	if _, err := fake.Recall(ctx, query); err != nil {
		t.Fatalf("Recall: %v", err)
	}
	query.Tags[0] = "mutated"
	if fake.RecallCalls[0].Tags[0] != "one" {
		t.Fatalf("recorded query = %+v", fake.RecallCalls[0])
	}
	reflected, err := fake.Reflect(ctx, Query{Text: "stored"})
	if err != nil {
		t.Fatalf("Reflect: %v", err)
	}
	if len(fake.ReflectCalls) != 1 || reflected.Hits[0].Item.Content != "stored token" {
		t.Fatalf("reflect calls = %d text = %q", len(fake.ReflectCalls), reflected.Text)
	}

	fake.RetainErr = errors.New("retain failed")
	if _, err := fake.Retain(ctx, Item{Content: "should not stick"}); !errors.Is(err, fake.RetainErr) {
		t.Fatalf("RetainErr = %v", err)
	}
	found, err := fake.Recall(ctx, Query{Text: "stick"})
	if err != nil {
		t.Fatalf("Recall after failed retain: %v", err)
	}
	if found.Total != 0 {
		t.Fatalf("stored a failed retain: %+v", found)
	}

	fake.RecallErr = errors.New("recall failed")
	if _, err := fake.Recall(ctx, Query{Text: "stored"}); !errors.Is(err, fake.RecallErr) {
		t.Fatalf("RecallErr = %v", err)
	}
	fake.ReflectErr = ErrReflectNotSupported
	_, err = fake.Reflect(ctx, Query{Text: "stored"})
	if !errors.Is(err, ErrReflectNotSupported) {
		t.Fatalf("ReflectErr = %v", err)
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	fake.RetainErr = errors.New("ignored when canceled")
	_, err = fake.Retain(canceled, Item{Content: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled retain = %v", err)
	}
	_, err = fake.Recall(canceled, Query{Text: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled recall = %v", err)
	}
	_, err = fake.Reflect(canceled, Query{Text: "x"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reflect = %v", err)
	}
}

func TestFakeConcurrent(t *testing.T) {
	fake := NewFake()
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, err := fake.Retain(context.Background(), Item{Content: "concurrent fake token"}); err != nil {
			t.Errorf("retain: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if _, err := fake.Recall(context.Background(), Query{Text: "concurrent"}); err != nil {
			t.Errorf("recall: %v", err)
		}
	}()
	wg.Wait()
	if len(fake.RetainCalls) != 1 {
		t.Fatalf("retain calls = %d", len(fake.RetainCalls))
	}
}
