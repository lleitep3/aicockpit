package kb

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileIndexProvider implements IndexProvider using a JSON file cache.
type FileIndexProvider struct {
	indexPath string
	index     *KBIndex
	lastCheck time.Time
}

// NewFileIndexProvider creates a new file-based index provider.
func NewFileIndexProvider(indexPath string) *FileIndexProvider {
	return &FileIndexProvider{
		indexPath: indexPath,
	}
}

// LoadIndex loads the KB index from the JSON file.
func (p *FileIndexProvider) LoadIndex(roots []string) (*KBIndex, error) {
	// Try to load from cache
	if p.index != nil && p.IsValid() {
		return p.index, nil
	}

	// Try to read from file
	data, err := os.ReadFile(p.indexPath)
	if err == nil {
		var index KBIndex
		if err := json.Unmarshal(data, &index); err == nil {
			p.index = &index
			p.lastCheck = time.Now()
			return p.index, nil
		}
	}

	// If file doesn't exist or is invalid, rebuild
	return p.RebuildIndex(roots)
}

// SaveIndex saves the KB index to the JSON file.
func (p *FileIndexProvider) SaveIndex(index *KBIndex) error {
	if index == nil {
		return fmt.Errorf("cannot save nil index")
	}

	// Update timestamp
	index.LastUpdated = time.Now()

	// Marshal to JSON
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal index: %w", err)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(p.indexPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write to file
	if err := os.WriteFile(p.indexPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	// Update cache
	p.index = index
	p.lastCheck = time.Now()

	return nil
}

// InvalidateCache invalidates the cache for a specific document.
func (p *FileIndexProvider) InvalidateCache(docPath string) error {
	// For file-based provider, we need to rebuild the entire index
	// In a more sophisticated implementation, we could update just the changed document
	// But for simplicity, we'll invalidate the cache and let it rebuild on next load
	p.index = nil
	return nil
}

// RebuildIndex rebuilds the entire index from scratch.
func (p *FileIndexProvider) RebuildIndex(roots []string) (*KBIndex, error) {
	index := &KBIndex{
		Version:     "1.0",
		LastUpdated: time.Now(),
		Roots:       make([]RootIndex, 0, len(roots)),
	}

	// Build index for each root
	for _, root := range roots {
		rootIndex, err := p.buildRootIndex(root)
		if err != nil {
			// Log warning but continue with other roots
			fmt.Fprintf(os.Stderr, "Warning: failed to index root %s: %v\n", root, err)
			continue
		}
		index.Roots = append(index.Roots, rootIndex)
	}

	// Save the rebuilt index
	if err := p.SaveIndex(index); err != nil {
		return nil, fmt.Errorf("failed to save rebuilt index: %w", err)
	}

	return index, nil
}

// buildRootIndex builds the index for a single root directory.
func (p *FileIndexProvider) buildRootIndex(root string) (RootIndex, error) {
	rootIndex := RootIndex{
		Path:      root,
		Documents: make([]IndexEntry, 0),
	}

	// Walk the directory
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process markdown files
		if filepath.Ext(path) != ".md" {
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		// Load document
		doc, err := p.loadDocumentForIndex(path)
		if err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to load document %s: %v\n", path, err)
			return nil
		}

		// Create index entry
		entry := IndexEntry{
			ID:          doc.ID,
			Title:       doc.Metadata.Title,
			Description: doc.Metadata.Description,
			Tags:        doc.Metadata.Tags,
			Path:        relPath,
			Hash:        calculateHash(doc.Content),
			Created:     doc.Metadata.Created,
			Modified:    doc.Metadata.Modified,
		}

		rootIndex.Documents = append(rootIndex.Documents, entry)
		return nil
	})

	if err != nil {
		return RootIndex{}, fmt.Errorf("failed to walk directory %s: %w", root, err)
	}

	return rootIndex, nil
}

// loadDocumentForIndex loads a document from disk for indexing.
func (p *FileIndexProvider) loadDocumentForIndex(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Generate ID from filename
	filename := filepath.Base(path)
	id := GenerateDocumentID(filename)

	// Parse document
	doc, err := ParseDocument(id, path, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse document: %w", err)
	}

	return doc, nil
}

// IsValid checks if the index is valid and up-to-date.
func (p *FileIndexProvider) IsValid() bool {
	if p.index == nil {
		return false
	}

	// Check if index file exists
	_, err := os.Stat(p.indexPath)
	return err == nil
}

// calculateHash calculates MD5 hash of content.
func calculateHash(content string) string {
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// GetIndexPath returns the path to the index file.
func (p *FileIndexProvider) GetIndexPath() string {
	return p.indexPath
}
