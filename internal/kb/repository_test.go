package kb

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileRepository_SaveAndLoadDocument(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	repo := NewFileRepository(tmpDir)

	// Create and save document
	doc := &Document{
		ID:   "test",
		Path: "guides/test.md",
		Metadata: Metadata{
			Title:       "Test Document",
			Description: "A test document",
			Tags:        []string{"test", "example"},
			Author:      "Test Author",
			Version:     "1.0",
			Created:     time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
			Modified:    time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
		},
		Content: "# Test Content\n\nThis is test content.",
	}

	// Save document
	err := repo.SaveDocument(doc)
	if err != nil {
		t.Fatalf("SaveDocument() error = %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, doc.Path)
	if _, err := os.Stat(fullPath); err != nil {
		t.Errorf("Document file not created at %s", fullPath)
	}

	// Load document
	loaded, err := repo.LoadDocument(doc.Path)
	if err != nil {
		t.Fatalf("LoadDocument() error = %v", err)
	}

	// Verify loaded document
	if loaded.ID != doc.ID {
		t.Errorf("LoadDocument() ID = %s, want %s", loaded.ID, doc.ID)
	}

	if loaded.Metadata.Title != doc.Metadata.Title {
		t.Errorf("LoadDocument() Title = %s, want %s", loaded.Metadata.Title, doc.Metadata.Title)
	}
}

func TestFileRepository_ListDocuments(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	repo := NewFileRepository(tmpDir)

	// Create multiple documents
	docs := []*Document{
		{
			ID:   "doc1",
			Path: "guides/doc1.md",
			Metadata: Metadata{
				Title:       "Document 1",
				Description: "First document",
				Tags:        []string{"guide"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: "Content 1",
		},
		{
			ID:   "doc2",
			Path: "guides/doc2.md",
			Metadata: Metadata{
				Title:       "Document 2",
				Description: "Second document",
				Tags:        []string{"guide"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: "Content 2",
		},
		{
			ID:   "doc3",
			Path: "troubleshooting/doc3.md",
			Metadata: Metadata{
				Title:       "Document 3",
				Description: "Third document",
				Tags:        []string{"troubleshooting"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: "Content 3",
		},
	}

	// Save all documents
	for _, doc := range docs {
		if err := repo.SaveDocument(doc); err != nil {
			t.Fatalf("SaveDocument() error = %v", err)
		}
	}

	// List documents
	listed, err := repo.ListDocuments(".")
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	if len(listed) != len(docs) {
		t.Errorf("ListDocuments() count = %d, want %d", len(listed), len(docs))
	}

	// Verify all documents are listed
	idMap := make(map[string]bool)
	for _, doc := range listed {
		idMap[doc.ID] = true
	}

	for _, doc := range docs {
		if !idMap[doc.ID] {
			t.Errorf("Document %s not found in list", doc.ID)
		}
	}
}

func TestFileRepository_ListDocuments_Subdirectory(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	repo := NewFileRepository(tmpDir)

	// Create documents in subdirectories
	docs := []*Document{
		{
			ID:   "guide1",
			Path: "guides/guide1.md",
			Metadata: Metadata{
				Title:   "Guide 1",
				Created: time.Now(),
				Modified: time.Now(),
			},
			Content: "Content",
		},
		{
			ID:   "guide2",
			Path: "guides/guide2.md",
			Metadata: Metadata{
				Title:   "Guide 2",
				Created: time.Now(),
				Modified: time.Now(),
			},
			Content: "Content",
		},
	}

	for _, doc := range docs {
		if err := repo.SaveDocument(doc); err != nil {
			t.Fatalf("SaveDocument() error = %v", err)
		}
	}

	// List only guides
	listed, err := repo.ListDocuments("guides")
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	if len(listed) != 2 {
		t.Errorf("ListDocuments() count = %d, want 2", len(listed))
	}
}

func TestFileRepository_DeleteDocument(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	repo := NewFileRepository(tmpDir)

	// Create and save document
	doc := &Document{
		ID:   "test-doc",
		Path: "guides/test.md",
		Metadata: Metadata{
			Title:   "Test",
			Created: time.Now(),
			Modified: time.Now(),
		},
		Content: "Content",
	}

	if err := repo.SaveDocument(doc); err != nil {
		t.Fatalf("SaveDocument() error = %v", err)
	}

	// Verify file exists
	fullPath := filepath.Join(tmpDir, doc.Path)
	if _, err := os.Stat(fullPath); err != nil {
		t.Fatalf("Document not created")
	}

	// Delete document
	if err := repo.DeleteDocument(doc.Path); err != nil {
		t.Fatalf("DeleteDocument() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(fullPath); err == nil {
		t.Error("Document still exists after deletion")
	}
}

func TestFileRepository_LoadDocument_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFileRepository(tmpDir)

	_, err := repo.LoadDocument("nonexistent.md")
	if err == nil {
		t.Error("LoadDocument() expected error for nonexistent file")
	}
}

func TestFileRepository_ListDocuments_InvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFileRepository(tmpDir)

	_, err := repo.ListDocuments("nonexistent")
	if err == nil {
		t.Error("ListDocuments() expected error for nonexistent directory")
	}
}

func TestFileRepository_ListDocuments_SkipsNonMarkdown(t *testing.T) {
	tmpDir := t.TempDir()
	repo := NewFileRepository(tmpDir)

	// Create a markdown document
	mdDoc := &Document{
		ID:   "doc1",
		Path: "guides/doc1.md",
		Metadata: Metadata{
			Title:   "Document 1",
			Created: time.Now(),
			Modified: time.Now(),
		},
		Content: "Content",
	}

	if err := repo.SaveDocument(mdDoc); err != nil {
		t.Fatalf("SaveDocument() error = %v", err)
	}

	// Create a non-markdown file
	nonMdPath := filepath.Join(tmpDir, "guides", "readme.txt")
	if err := os.WriteFile(nonMdPath, []byte("Not markdown"), 0o644); err != nil {
		t.Fatalf("Failed to create non-markdown file")
	}

	// List documents
	listed, err := repo.ListDocuments("guides")
	if err != nil {
		t.Fatalf("ListDocuments() error = %v", err)
	}

	// Should only find markdown file
	if len(listed) != 1 {
		t.Errorf("ListDocuments() count = %d, want 1", len(listed))
	}

	if listed[0].ID != "doc1" {
		t.Errorf("ListDocuments() ID = %s, want doc1", listed[0].ID)
	}
}
