package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lleitep3/aicockpit/internal/kb"
)

func TestNewKBService(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewKBService([]string{tmpDir}, tmpDir, nil)
	if svc == nil {
		t.Fatal("expected non-nil KB service")
	}
}

func TestDefaultKBService_Operations(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewKBService([]string{tmpDir}, tmpDir, nil)

	// RebuildIndex on an empty directory should succeed.
	if err := svc.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex() unexpected error: %v", err)
	}

	// Search should return a non-nil result even when empty.
	results, err := svc.Search("query")
	if err != nil {
		t.Errorf("Search() unexpected error: %v", err)
	}
	if results == nil {
		t.Error("Search() should return a non-nil result")
	}

	// ListDocuments should be empty on a fresh directory.
	docs, err := svc.ListDocuments()
	if err != nil {
		t.Errorf("ListDocuments() unexpected error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("ListDocuments() should be empty, got %d", len(docs))
	}

	// GetLastIndexUpdate should return a recent time after rebuilding.
	lastUpdate, err := svc.GetLastIndexUpdate()
	if err != nil {
		t.Errorf("GetLastIndexUpdate() unexpected error: %v", err)
	}
	if time.Since(lastUpdate) > time.Minute {
		t.Errorf("GetLastIndexUpdate() returned stale time: %v", lastUpdate)
	}

	roots := svc.GetRoots()
	if len(roots) != 1 || roots[0] != tmpDir {
		t.Errorf("GetRoots() = %v, want [%s]", roots, tmpDir)
	}

	extraRoot := filepath.Join(tmpDir, "extra")
	if err := os.MkdirAll(extraRoot, 0o755); err != nil {
		t.Fatalf("failed to create extra root: %v", err)
	}
	if err := svc.AddRoot(extraRoot); err != nil {
		t.Errorf("AddRoot() unexpected error: %v", err)
	}
	if err := svc.RemoveRoot(extraRoot); err != nil {
		t.Errorf("RemoveRoot() unexpected error: %v", err)
	}

	if gs := svc.NewGraphSearcher(); gs == nil {
		t.Error("NewGraphSearcher() should return a non-nil searcher")
	}

	ctx := context.Background()
	if _, err := svc.RunSearchExtensions(ctx, "q"); err == nil {
		t.Error("RunSearchExtensions() should return an error when no packages are installed")
	}

	if err := svc.RunIndexExtensions(ctx, []string{}, false); err != nil {
		t.Errorf("RunIndexExtensions() unexpected error: %v", err)
	}
}

func TestDefaultKBService_SearchWithDocument(t *testing.T) {
	tmpDir := t.TempDir()
	docPath := filepath.Join(tmpDir, "test.md")
	content := `---
title: "Test Document"
description: "A document about testing"
tags: ["test"]
---
# Test Content

This is content about testing.`
	if err := os.WriteFile(docPath, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to create test document: %v", err)
	}

	svc := NewKBService([]string{tmpDir}, tmpDir, nil)
	results, err := svc.Search("test")
	if err != nil {
		t.Fatalf("Search() unexpected error: %v", err)
	}
	if results == nil {
		t.Fatal("Search() should return a non-nil result")
	}
	if results.Total == 0 {
		t.Error("Search(test) should return at least one result")
	}
}

func TestDefaultKBService_NewGraphSearcher(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewKBService([]string{tmpDir}, tmpDir, nil)
	gs := svc.NewGraphSearcher()
	if gs == nil {
		t.Fatal("NewGraphSearcher() should return a non-nil searcher")
	}

	// Build and search a tiny graph to confirm the returned searcher is usable.
	docs := []*kb.Document{
		{
			ID:       "doc-a",
			Metadata: kb.Metadata{Related: []string{"doc-b"}},
		},
		{
			ID:       "doc-b",
			Metadata: kb.Metadata{},
		},
	}
	if err := gs.BuildGraph(docs); err != nil {
		t.Fatalf("BuildGraph() unexpected error: %v", err)
	}
	result, err := gs.SearchGraph("doc-a", 1)
	if err != nil {
		t.Fatalf("SearchGraph() unexpected error: %v", err)
	}
	if result == nil || result.TotalDocs == 0 {
		t.Error("SearchGraph() should return non-empty result")
	}
}
