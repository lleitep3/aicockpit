package kb

import (
	"testing"
)

func TestGraphSearcher(t *testing.T) {
	doc1 := &Document{
		ID: "doc1",
		Metadata: Metadata{
			Title:   "Document 1",
			Related: []string{"doc2"},
		},
	}
	doc2 := &Document{
		ID: "doc2",
		Metadata: Metadata{
			Title:   "Document 2",
			Related: []string{"doc3"},
		},
	}
	doc3 := &Document{
		ID: "doc3",
		Metadata: Metadata{
			Title: "Document 3",
		},
	}
	doc4 := &Document{
		ID: "doc4",
		Metadata: Metadata{
			Title: "Document 4",
		},
	}

	docs := []*Document{doc1, doc2, doc3, doc4}

	searcher := NewGraphSearcher()
	err := searcher.BuildGraph(docs)
	if err != nil {
		t.Fatalf("Failed to build graph: %v", err)
	}

	// Test SearchGraph with depth 1
	res, err := searcher.SearchGraph("doc1", 1)
	if err != nil {
		t.Fatalf("Failed to search graph: %v", err)
	}

	if res.TotalDocs != 2 {
		t.Errorf("Expected 2 docs (doc1 + doc2), got %d", res.TotalDocs)
	}

	// Test SearchGraph with depth 2
	res, err = searcher.SearchGraph("doc1", 2)
	if err != nil {
		t.Fatalf("Failed to search graph: %v", err)
	}

	if res.TotalDocs != 3 {
		t.Errorf("Expected 3 docs (doc1 + doc2 + doc3), got %d", res.TotalDocs)
	}

	// Test isolated node
	res, err = searcher.SearchGraph("doc4", 1)
	if err != nil {
		t.Fatalf("Failed to search graph: %v", err)
	}

	if res.TotalDocs != 1 {
		t.Errorf("Expected 1 doc (doc4), got %d", res.TotalDocs)
	}

	// Test non-existent node
	_, err = searcher.SearchGraph("doc_not_exists", 1)
	if err == nil {
		t.Errorf("Expected error for non-existent doc, got nil")
	}
}
