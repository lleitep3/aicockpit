package kb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	manager := NewManager([]string{tmpDir}, indexPath)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if manager.repo == nil {
		t.Error("expected non-nil repository")
	}

	if manager.searcher == nil {
		t.Error("expected non-nil searcher")
	}

	if manager.scorer == nil {
		t.Error("expected non-nil scorer")
	}

	if manager.indexer == nil {
		t.Error("expected non-nil indexer")
	}
}

func TestManager_Search(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory with test document
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	// Create test document
	docPath := filepath.Join(kbDir, "test.md")
	docContent := `---
title: "Test Document"
description: "A test document"
tags: ["test"]
---
# Test Content

This is test content about logging.`

	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Search
	results, err := manager.Search("test")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if results == nil {
		t.Fatal("expected non-nil search results")
	}

	if len(results.Results) == 0 {
		t.Error("expected at least one search result")
	}
}

func TestManager_ListDocuments(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory with test document
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	// Create test document
	docPath := filepath.Join(kbDir, "test.md")
	docContent := `---
title: "Test Document"
---
Content`

	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// List documents
	docs, err := manager.ListDocuments()
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	if len(docs) != 1 {
		t.Errorf("ListDocuments() returned %d documents, want 1", len(docs))
	}

	if docs[0].Metadata.Title != "Test Document" {
		t.Errorf("ListDocuments() Title = %s, want Test Document", docs[0].Metadata.Title)
	}
}

func TestManager_RebuildIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory with test document
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	// Create test document
	docPath := filepath.Join(kbDir, "test.md")
	docContent := `---
title: "Test Document"
---
Content`

	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Rebuild index
	err := manager.RebuildIndex()
	if err != nil {
		t.Fatalf("RebuildIndex() error = %v", err)
	}

	// Verify index file exists
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("Index file not created at %s", indexPath)
	}
}

func TestManager_GetRoots(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	roots := []string{tmpDir, "/another/path"}

	manager := NewManager(roots, indexPath)

	gotRoots := manager.GetRoots()
	if len(gotRoots) != len(roots) {
		t.Errorf("GetRoots() returned %d roots, want %d", len(gotRoots), len(roots))
	}

	for i, root := range roots {
		if gotRoots[i] != root {
			t.Errorf("GetRoots()[%d] = %s, want %s", i, gotRoots[i], root)
		}
	}
}

func TestManager_AddRoot(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir1 := filepath.Join(tmpDir, "kb1")
	kbDir2 := filepath.Join(tmpDir, "kb2")

	// Create directories
	if err := os.MkdirAll(kbDir1, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 1: %v", err)
	}

	if err := os.MkdirAll(kbDir2, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 2: %v", err)
	}

	manager := NewManager([]string{kbDir1}, indexPath)

	// Add root
	err := manager.AddRoot(kbDir2)
	if err != nil {
		t.Fatalf("AddRoot() error = %v", err)
	}

	roots := manager.GetRoots()
	if len(roots) != 2 {
		t.Errorf("AddRoot() resulted in %d roots, want 2", len(roots))
	}

	if roots[1] != kbDir2 {
		t.Errorf("AddRoot() second root = %s, want %s", roots[1], kbDir2)
	}
}

func TestManager_AddRoot_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Try to add duplicate root
	err := manager.AddRoot(kbDir)
	if err == nil {
		t.Error("AddRoot() should fail for duplicate root")
	}
}

func TestManager_AddRoot_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Try to add non-existent root
	err := manager.AddRoot("/non/existent/path")
	if err == nil {
		t.Error("AddRoot() should fail for non-existent root")
	}
}

func TestManager_RemoveRoot(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir1 := filepath.Join(tmpDir, "kb1")
	kbDir2 := filepath.Join(tmpDir, "kb2")

	// Create directories
	if err := os.MkdirAll(kbDir1, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 1: %v", err)
	}

	if err := os.MkdirAll(kbDir2, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 2: %v", err)
	}

	manager := NewManager([]string{kbDir1, kbDir2}, indexPath)

	// Remove root
	err := manager.RemoveRoot(kbDir1)
	if err != nil {
		t.Fatalf("RemoveRoot() error = %v", err)
	}

	roots := manager.GetRoots()
	if len(roots) != 1 {
		t.Errorf("RemoveRoot() resulted in %d roots, want 1", len(roots))
	}

	if roots[0] != kbDir2 {
		t.Errorf("RemoveRoot() remaining root = %s, want %s", roots[0], kbDir2)
	}
}

func TestManager_RemoveRoot_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Try to remove non-existent root
	err := manager.RemoveRoot("/non/existent/path")
	if err == nil {
		t.Error("RemoveRoot() should fail for non-existent root")
	}
}

func TestManager_GetIndexPath(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	manager := NewManager([]string{tmpDir}, indexPath)

	if manager.GetIndexPath() != indexPath {
		t.Errorf("GetIndexPath() = %s, want %s", manager.GetIndexPath(), indexPath)
	}
}

func TestManager_GetIndexProvider(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	manager := NewManager([]string{tmpDir}, indexPath)

	provider := manager.GetIndexProvider()
	if provider == nil {
		t.Error("GetIndexProvider() returned nil")
	}
}

func TestManager_GetLastIndexUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Rebuild index
	if err := manager.RebuildIndex(); err != nil {
		t.Fatalf("RebuildIndex() error = %v", err)
	}

	// Get last update time
	lastUpdate, err := manager.GetLastIndexUpdate()
	if err != nil {
		t.Fatalf("GetLastIndexUpdate() error = %v", err)
	}

	// Verify it's recent (within 1 second)
	now := time.Now()
	if now.Sub(lastUpdate) > time.Second {
		t.Errorf("GetLastIndexUpdate() returned old time: %v", lastUpdate)
	}
}

func TestManager_AddDocument(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Create document
	doc := &Document{
		ID:   "new-doc",
		Path: "guides/new.md",
		Metadata: Metadata{
			Title:       "New Document",
			Description: "A new document",
			Tags:        []string{"new"},
			Created:     time.Now(),
			Modified:    time.Now(),
		},
		Content: "# New Document\n\nContent here.",
	}

	// Add document
	err := manager.AddDocument(doc, 0)
	if err != nil {
		t.Fatalf("AddDocument() error = %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(kbDir, doc.Path)
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("Document file not created at %s", fullPath)
	}
}

func TestManager_AddDocument_InvalidRoot(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Create document
	doc := &Document{
		ID:   "new-doc",
		Path: "guides/new.md",
		Metadata: Metadata{
			Title:   "New Document",
			Created: time.Now(),
		},
		Content: "Content",
	}

	// Try to add to invalid root
	err := manager.AddDocument(doc, 5)
	if err == nil {
		t.Error("AddDocument() should fail for invalid root index")
	}
}

func TestManager_RemoveDocument(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory with document
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	docPath := filepath.Join(kbDir, "test.md")
	docContent := `---
title: "Test"
---
Content`

	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Remove document
	err := manager.RemoveDocument("test.md")
	if err != nil {
		t.Fatalf("RemoveDocument() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(docPath); err == nil {
		t.Error("Document still exists after removal")
	}
}

func TestManager_RemoveDocument_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	manager := NewManager([]string{kbDir}, indexPath)

	// Try to remove non-existent document
	err := manager.RemoveDocument("nonexistent.md")
	if err == nil {
		t.Error("RemoveDocument() should fail for non-existent file")
	}
}

func TestManager_Search_WithMultipleRoots(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir1 := filepath.Join(tmpDir, "kb1")
	kbDir2 := filepath.Join(tmpDir, "kb2")

	// Create first KB directory with document
	if err := os.MkdirAll(kbDir1, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 1: %v", err)
	}

	docPath1 := filepath.Join(kbDir1, "doc1.md")
	docContent1 := `---
title: "Document 1"
tags: ["test"]
---
# Test Content

This is test content.`

	if err := os.WriteFile(docPath1, []byte(docContent1), 0o644); err != nil {
		t.Fatalf("Failed to create test document 1: %v", err)
	}

	// Create second KB directory with document
	if err := os.MkdirAll(kbDir2, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 2: %v", err)
	}

	docPath2 := filepath.Join(kbDir2, "doc2.md")
	docContent2 := `---
title: "Document 2"
tags: ["test"]
---
# Another Test

Another test document.`

	if err := os.WriteFile(docPath2, []byte(docContent2), 0o644); err != nil {
		t.Fatalf("Failed to create test document 2: %v", err)
	}

	manager := NewManager([]string{kbDir1, kbDir2}, indexPath)

	// Search
	results, err := manager.Search("test")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}

	if len(results.Results) != 2 {
		t.Errorf("Search() returned %d results, want 2", len(results.Results))
	}
}

func TestManager_NewManager_NoRoots(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	// Create manager with no roots
	manager := NewManager([]string{}, indexPath)

	if manager == nil {
		t.Fatal("expected non-nil manager")
	}

	if len(manager.GetRoots()) != 0 {
		t.Errorf("GetRoots() returned %d roots, want 0", len(manager.GetRoots()))
	}
}
