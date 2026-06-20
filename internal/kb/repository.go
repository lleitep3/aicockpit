package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileRepository implements Repository interface using the file system.
type FileRepository struct {
	basePath string
}

// NewFileRepository creates a new file-based repository.
func NewFileRepository(basePath string) *FileRepository {
	return &FileRepository{
		basePath: basePath,
	}
}

// LoadDocument loads a document from disk.
func (fr *FileRepository) LoadDocument(path string) (*Document, error) {
	fullPath := filepath.Join(fr.basePath, path)

	// Read file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document %s: %w", path, err)
	}

	// Generate ID from filename
	filename := filepath.Base(path)
	id := GenerateDocumentID(filename)

	// Parse document
	doc, err := ParseDocument(id, path, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse document %s: %w", path, err)
	}

	return doc, nil
}

// ListDocuments lists all documents in a directory recursively.
func (fr *FileRepository) ListDocuments(dir string) ([]*Document, error) {
	fullPath := filepath.Join(fr.basePath, dir)

	// Check if directory exists
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access directory %s: %w", dir, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	var documents []*Document

	// Walk directory
	err = filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process markdown files
		if !strings.HasSuffix(path, ".md") {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(fr.basePath, path)
		if err != nil {
			return err
		}

		// Load document
		doc, err := fr.LoadDocument(relPath)
		if err != nil {
			// Log error but continue processing other files
			fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", relPath, err)
			return nil
		}

		documents = append(documents, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list documents in %s: %w", dir, err)
	}

	return documents, nil
}

// SaveDocument saves a document to disk.
func (fr *FileRepository) SaveDocument(doc *Document) error {
	fullPath := filepath.Join(fr.basePath, doc.Path)

	// Create directory if it doesn't exist
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Serialize document
	content, err := SerializeDocument(doc)
	if err != nil {
		return fmt.Errorf("failed to serialize document: %w", err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("failed to write document %s: %w", doc.Path, err)
	}

	return nil
}

// DeleteDocument deletes a document from disk.
func (fr *FileRepository) DeleteDocument(path string) error {
	fullPath := filepath.Join(fr.basePath, path)

	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete document %s: %w", path, err)
	}

	return nil
}
