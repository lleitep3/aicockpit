package kb

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Manager orchestrates KB operations using Repository, Searcher, Scorer, and IndexProvider.
type Manager struct {
	repo      Repository
	searcher  Searcher
	scorer    Scorer
	indexer   IndexProvider
	roots     []string
	indexPath string
}

// NewManager creates a new KB Manager.
func NewManager(roots []string, indexPath string) *Manager {
	// Use the first root as base path for repository
	basePath := ""
	if len(roots) > 0 {
		basePath = roots[0]
	}

	return &Manager{
		repo:      NewFileRepository(basePath),
		searcher:  NewKeywordSearcher(NewBM25Scorer()),
		scorer:    NewBM25Scorer(),
		indexer:   NewFileIndexProvider(indexPath),
		roots:     roots,
		indexPath: indexPath,
	}
}

// Search searches for documents across all roots.
func (m *Manager) Search(query string) (*SearchResults, error) {
	// Load index
	index, err := m.indexer.LoadIndex(m.roots)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Convert index to documents
	documents, err := m.indexToDocuments(index)
	if err != nil {
		return nil, fmt.Errorf("failed to convert index to documents: %w", err)
	}

	// Search
	return m.searcher.Search(query, documents)
}

// ListDocuments lists all documents across all roots.
func (m *Manager) ListDocuments() ([]*Document, error) {
	// Load index
	index, err := m.indexer.LoadIndex(m.roots)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Convert index to documents
	documents, err := m.indexToDocuments(index)
	if err != nil {
		return nil, fmt.Errorf("failed to convert index to documents: %w", err)
	}

	return documents, nil
}

// AddDocument adds a new document to the KB.
func (m *Manager) AddDocument(doc *Document, rootIndex int) error {
	if rootIndex < 0 || rootIndex >= len(m.roots) {
		return fmt.Errorf("invalid root index: %d", rootIndex)
	}

	root := m.roots[rootIndex]

	// Determine the path
	docPath := filepath.Join(root, doc.Path)

	// Create directory if needed
	dir := filepath.Dir(docPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save document
	if err := m.repo.SaveDocument(doc); err != nil {
		return fmt.Errorf("failed to save document: %w", err)
	}

	// Invalidate cache
	if err := m.indexer.InvalidateCache(docPath); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	return nil
}

// RemoveDocument removes a document from the KB.
func (m *Manager) RemoveDocument(docPath string) error {
	// Delete document
	if err := m.repo.DeleteDocument(docPath); err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	// Invalidate cache
	if err := m.indexer.InvalidateCache(docPath); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	return nil
}

// RebuildIndex rebuilds the entire index.
func (m *Manager) RebuildIndex() error {
	_, err := m.indexer.RebuildIndex(m.roots)
	return err
}

// GetRoots returns the configured KB roots.
func (m *Manager) GetRoots() []string {
	return m.roots
}

// AddRoot adds a new KB root.
func (m *Manager) AddRoot(root string) error {
	// Verify root exists
	if _, err := os.Stat(root); err != nil {
		return fmt.Errorf("root path does not exist: %w", err)
	}

	// Check if root already exists
	for _, r := range m.roots {
		if r == root {
			return fmt.Errorf("root already exists: %s", root)
		}
	}

	// Add root
	m.roots = append(m.roots, root)

	// Invalidate cache to force rebuild
	if err := m.indexer.InvalidateCache(""); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	return nil
}

// RemoveRoot removes a KB root.
func (m *Manager) RemoveRoot(root string) error {
	// Find and remove root
	found := false
	for i, r := range m.roots {
		if r == root {
			m.roots = append(m.roots[:i], m.roots[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("root not found: %s", root)
	}

	// Invalidate cache to force rebuild
	if err := m.indexer.InvalidateCache(""); err != nil {
		return fmt.Errorf("failed to invalidate cache: %w", err)
	}

	return nil
}

// indexToDocuments converts KBIndex to Document slice.
func (m *Manager) indexToDocuments(index *KBIndex) ([]*Document, error) {
	documents := make([]*Document, 0)

	for _, rootIndex := range index.Roots {
		for _, entry := range rootIndex.Documents {
			// Construct full path
			fullPath := filepath.Join(rootIndex.Path, entry.Path)

			// Load document directly from disk (not through repository)
			data, err := os.ReadFile(fullPath)
			if err != nil {
				// Log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: failed to load document %s: %v\n", fullPath, err)
				continue
			}

			// Parse document
			doc, err := ParseDocument(entry.ID, entry.Path, string(data))
			if err != nil {
				// Log warning but continue
				fmt.Fprintf(os.Stderr, "Warning: failed to parse document %s: %v\n", fullPath, err)
				continue
			}

			documents = append(documents, doc)
		}
	}

	return documents, nil
}

// GetIndexPath returns the path to the index file.
func (m *Manager) GetIndexPath() string {
	return m.indexPath
}

// GetIndexProvider returns the index provider.
func (m *Manager) GetIndexProvider() IndexProvider {
	return m.indexer
}

// GetLastIndexUpdate returns the last time the index was updated.
func (m *Manager) GetLastIndexUpdate() (time.Time, error) {
	index, err := m.indexer.LoadIndex(m.roots)
	if err != nil {
		return time.Time{}, err
	}
	return index.LastUpdated, nil
}
