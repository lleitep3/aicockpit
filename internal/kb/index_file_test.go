package kb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileIndexProvider_SaveAndLoadIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Create test index
	index := &KBIndex{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Roots: []RootIndex{
			{
				Path: tmpDir,
				Documents: []IndexEntry{
					{
						ID:          "test-doc",
						Title:       "Test Document",
						Description: "A test document",
						Tags:        []string{"test"},
						Path:        "test.md",
						Hash:        "abc123",
						Created:     time.Now(),
						Modified:    time.Now(),
					},
				},
			},
		},
	}

	// Save index
	err := provider.SaveIndex(index)
	if err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("Index file not created at %s", indexPath)
	}

	// Load index
	loaded, err := provider.LoadIndex([]string{tmpDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	// Verify loaded index
	if loaded.Version != "1.0" {
		t.Errorf("LoadIndex() Version = %s, want 1.0", loaded.Version)
	}

	if len(loaded.Roots) != 1 {
		t.Errorf("LoadIndex() Roots length = %d, want 1", len(loaded.Roots))
	}

	if len(loaded.Roots[0].Documents) != 1 {
		t.Errorf("LoadIndex() Documents length = %d, want 1", len(loaded.Roots[0].Documents))
	}

	if loaded.Roots[0].Documents[0].ID != "test-doc" {
		t.Errorf("LoadIndex() Document ID = %s, want test-doc", loaded.Roots[0].Documents[0].ID)
	}
}

func TestFileIndexProvider_RebuildIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory with test documents
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

This is test content.`

	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatalf("Failed to create test document: %v", err)
	}

	provider := NewFileIndexProvider(indexPath)

	// Rebuild index
	index, err := provider.RebuildIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("RebuildIndex() error = %v", err)
	}

	// Verify index
	if index.Version != "1.0" {
		t.Errorf("RebuildIndex() Version = %s, want 1.0", index.Version)
	}

	if len(index.Roots) != 1 {
		t.Errorf("RebuildIndex() Roots length = %d, want 1", len(index.Roots))
	}

	if len(index.Roots[0].Documents) != 1 {
		t.Errorf("RebuildIndex() Documents length = %d, want 1", len(index.Roots[0].Documents))
	}

	doc := index.Roots[0].Documents[0]
	if doc.Title != "Test Document" {
		t.Errorf("RebuildIndex() Title = %s, want Test Document", doc.Title)
	}

	if doc.ID != "test" {
		t.Errorf("RebuildIndex() ID = %s, want test", doc.ID)
	}
}

func TestFileIndexProvider_RebuildIndex_MultipleRoots(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir1 := filepath.Join(tmpDir, "kb1")
	kbDir2 := filepath.Join(tmpDir, "kb2")

	// Create first KB directory
	if err := os.MkdirAll(kbDir1, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 1: %v", err)
	}

	// Create second KB directory
	if err := os.MkdirAll(kbDir2, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory 2: %v", err)
	}

	// Create documents in first KB
	doc1Path := filepath.Join(kbDir1, "doc1.md")
	doc1Content := `---
title: "Document 1"
---
Content 1`
	if err := os.WriteFile(doc1Path, []byte(doc1Content), 0o644); err != nil {
		t.Fatalf("Failed to create document 1: %v", err)
	}

	// Create documents in second KB
	doc2Path := filepath.Join(kbDir2, "doc2.md")
	doc2Content := `---
title: "Document 2"
---
Content 2`
	if err := os.WriteFile(doc2Path, []byte(doc2Content), 0o644); err != nil {
		t.Fatalf("Failed to create document 2: %v", err)
	}

	provider := NewFileIndexProvider(indexPath)

	// Rebuild index for both roots
	index, err := provider.RebuildIndex([]string{kbDir1, kbDir2})
	if err != nil {
		t.Fatalf("RebuildIndex() error = %v", err)
	}

	// Verify index has both roots
	if len(index.Roots) != 2 {
		t.Errorf("RebuildIndex() Roots length = %d, want 2", len(index.Roots))
	}

	// Verify first root
	if len(index.Roots[0].Documents) != 1 {
		t.Errorf("RebuildIndex() Root 0 Documents length = %d, want 1", len(index.Roots[0].Documents))
	}

	// Verify second root
	if len(index.Roots[1].Documents) != 1 {
		t.Errorf("RebuildIndex() Root 1 Documents length = %d, want 1", len(index.Roots[1].Documents))
	}
}

func TestFileIndexProvider_InvalidateCache(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Create and save index
	index := &KBIndex{
		Version: "1.0",
		Roots:   []RootIndex{},
	}
	if err := provider.SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	// Verify cache is loaded
	if provider.index == nil {
		t.Error("Index should be cached after SaveIndex()")
	}

	// Invalidate cache
	err := provider.InvalidateCache("test.md")
	if err != nil {
		t.Errorf("InvalidateCache() error = %v", err)
	}

	// Verify cache is cleared
	if provider.index != nil {
		t.Error("Index should be cleared after InvalidateCache()")
	}
}

func TestFileIndexProvider_IsValid(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Should be invalid without index
	if provider.IsValid() {
		t.Error("IsValid() should return false without index file")
	}

	// Create and save index
	index := &KBIndex{
		Version: "1.0",
		Roots:   []RootIndex{},
	}
	if err := provider.SaveIndex(index); err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	// Should be valid now
	if !provider.IsValid() {
		t.Error("IsValid() should return true after SaveIndex()")
	}
}

func TestFileIndexProvider_GetIndexPath(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	if provider.GetIndexPath() != indexPath {
		t.Errorf("GetIndexPath() = %s, want %s", provider.GetIndexPath(), indexPath)
	}
}

func TestCalculateHash(t *testing.T) {
	content1 := "test content"
	content2 := "test content"
	content3 := "different content"

	hash1 := calculateHash(content1)
	hash2 := calculateHash(content2)
	hash3 := calculateHash(content3)

	if hash1 != hash2 {
		t.Error("Same content should produce same hash")
	}

	if hash1 == hash3 {
		t.Error("Different content should produce different hash")
	}

	// Verify hash is not empty
	if hash1 == "" {
		t.Error("Hash should not be empty")
	}
}

func TestFileIndexProvider_LoadIndex_FromCache(t *testing.T) {
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

	provider := NewFileIndexProvider(indexPath)

	// First load - should rebuild
	index1, err := provider.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	if len(index1.Roots) != 1 {
		t.Errorf("LoadIndex() Roots length = %d, want 1", len(index1.Roots))
	}

	// Verify cache is set
	if provider.index == nil {
		t.Error("Provider cache should be set after LoadIndex()")
	}

	// Second load - should use cache (same provider instance)
	index2, err := provider.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	if len(index2.Roots) != 1 {
		t.Errorf("LoadIndex() Roots length = %d, want 1", len(index2.Roots))
	}

	// Verify it's the same instance (from cache)
	if index1 != index2 {
		t.Error("LoadIndex() should return cached index")
	}
}

func TestFileIndexProvider_RebuildIndex_WithError(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Try to rebuild with non-existent root
	// Should not fail, just skip the root
	index, err := provider.RebuildIndex([]string{"/non/existent/path"})
	if err != nil {
		t.Fatalf("RebuildIndex() error = %v", err)
	}

	if index == nil {
		t.Error("RebuildIndex() should return index even with non-existent roots")
	}
}

func TestFileIndexProvider_SaveIndex_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "subdir", ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Create and save index
	index := &KBIndex{
		Version: "1.0",
		Roots:   []RootIndex{},
	}

	err := provider.SaveIndex(index)
	if err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(filepath.Dir(indexPath)); err != nil {
		t.Errorf("Directory not created: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(indexPath); err != nil {
		t.Errorf("Index file not created: %v", err)
	}
}

func TestFileIndexProvider_SaveIndex_NilIndex(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Try to save nil index
	err := provider.SaveIndex(nil)
	if err == nil {
		t.Error("SaveIndex() should fail for nil index")
	}
}

func TestFileIndexProvider_SaveIndex_UpdatesCache(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Create and save index
	index := &KBIndex{
		Version: "1.0",
		Roots: []RootIndex{
			{
				Path: tmpDir,
				Documents: []IndexEntry{
					{
						ID:    "test",
						Title: "Test",
						Path:  "test.md",
					},
				},
			},
		},
	}

	err := provider.SaveIndex(index)
	if err != nil {
		t.Fatalf("SaveIndex() error = %v", err)
	}

	// Verify cache is updated
	if provider.index == nil {
		t.Error("Cache should be set after SaveIndex()")
	}

	if provider.index.Version != "1.0" {
		t.Errorf("Cached index version = %s, want 1.0", provider.index.Version)
	}
}

func TestFileIndexProvider_LoadIndex_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")
	kbDir := filepath.Join(tmpDir, "kb")

	// Create KB directory
	if err := os.MkdirAll(kbDir, 0o755); err != nil {
		t.Fatalf("Failed to create KB directory: %v", err)
	}

	// Create invalid JSON file
	if err := os.WriteFile(indexPath, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("Failed to create invalid index file: %v", err)
	}

	provider := NewFileIndexProvider(indexPath)

	// Load should rebuild on invalid JSON
	index, err := provider.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	if index == nil {
		t.Error("LoadIndex() should return index even with invalid JSON")
	}
}

func TestFileIndexProvider_LoadIndex_EmptyRoots(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, ".index.json")

	provider := NewFileIndexProvider(indexPath)

	// Load with empty roots
	index, err := provider.LoadIndex([]string{})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	if index == nil {
		t.Error("LoadIndex() should return index even with empty roots")
	}

	if len(index.Roots) != 0 {
		t.Errorf("LoadIndex() Roots length = %d, want 0", len(index.Roots))
	}
}

func TestFileIndexProvider_LoadIndex_FromFile(t *testing.T) {
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

	// Create first provider and save index
	provider1 := NewFileIndexProvider(indexPath)
	index1, err := provider1.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	// Create second provider (different instance) and load from file
	provider2 := NewFileIndexProvider(indexPath)
	index2, err := provider2.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	// Both should have same content
	if len(index1.Roots) != len(index2.Roots) {
		t.Errorf("Roots length mismatch: %d vs %d", len(index1.Roots), len(index2.Roots))
	}

	if len(index1.Roots[0].Documents) != len(index2.Roots[0].Documents) {
		t.Errorf("Documents length mismatch: %d vs %d", len(index1.Roots[0].Documents), len(index2.Roots[0].Documents))
	}
}

func TestFileIndexProvider_LoadIndex_CacheWithValidFile(t *testing.T) {
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

	provider := NewFileIndexProvider(indexPath)

	// First load
	index1, err := provider.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	// Verify cache is set and file exists
	if provider.index == nil {
		t.Error("Cache should be set")
	}

	if !provider.IsValid() {
		t.Error("IsValid() should return true")
	}

	// Second load should return cached version
	index2, err := provider.LoadIndex([]string{kbDir})
	if err != nil {
		t.Fatalf("LoadIndex() error = %v", err)
	}

	// Should be same instance
	if index1 != index2 {
		t.Error("Should return cached index")
	}
}
