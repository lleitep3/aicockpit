package services

import (
	"context"
	"path/filepath"
	"time"

	"github.com/lleitep3/aicockpit/internal/kb"
	"github.com/lleitep3/aicockpit/internal/logging"
	"github.com/lleitep3/aicockpit/internal/packages"
)

// KBService abstracts knowledge base operations used by cmd/kb.go.
type KBService interface {
	// Search performs a BM25 keyword search.
	Search(query string) (*kb.SearchResults, error)

	// RunSearchExtensions runs package search scripts and in-process contributors.
	// Supplement is set even when no package script succeeded.
	RunSearchExtensions(ctx context.Context, query string, limit int) (kb.SearchExtensionResult, error)

	// SearchSupplement runs in-process contributors such as memory-hindsight.
	// Package scripts are not started. The result is empty when nothing is enabled.
	SearchSupplement(ctx context.Context, query string, limit int) string

	// RunIndexExtensions triggers index extensions after a rebuild.
	RunIndexExtensions(ctx context.Context, roots []string, fast bool) error

	// AddContributors registers in-process extensions used by search and index.
	AddContributors(contributors ...kb.Contributor)

	// ListDocuments returns all documents in the knowledge base.
	ListDocuments() ([]*kb.Document, error)

	// AddRoot adds a new root path to the knowledge base.
	AddRoot(rootPath string) error

	// RemoveRoot removes a root path from the knowledge base.
	RemoveRoot(rootPath string) error

	// GetRoots returns the current list of root paths.
	GetRoots() []string

	// RebuildIndex rebuilds the full-text search index.
	RebuildIndex() error

	// GetLastIndexUpdate returns the time the index was last updated.
	GetLastIndexUpdate() (time.Time, error)

	// NewGraphSearcher returns a graph searcher for document graph traversal.
	NewGraphSearcher() kb.GraphSearcher
}

// DefaultKBService is the production implementation of KBService.
type DefaultKBService struct {
	manager      *kb.Manager
	pm           *packages.PackageManager
	contributors []kb.Contributor
}

// NewKBService creates a DefaultKBService.
func NewKBService(roots []string, cockpitDir string, log *logging.Manager) KBService {
	indexPath := filepath.Join(cockpitDir, ".kb-index.json")
	manager := kb.NewManagerWithLogger(roots, indexPath, log)
	pm := packages.NewPackageManager(cockpitDir)

	return &DefaultKBService{
		manager: manager,
		pm:      pm,
	}
}

func (s *DefaultKBService) Search(query string) (*kb.SearchResults, error) {
	return s.manager.Search(query)
}

func (s *DefaultKBService) RunSearchExtensions(ctx context.Context, query string, limit int) (kb.SearchExtensionResult, error) {
	return kb.RunSearchExtensions(ctx, s.pm, query, limit, s.contributors...)
}

func (s *DefaultKBService) SearchSupplement(ctx context.Context, query string, limit int) string {
	return kb.SearchSupplement(ctx, query, limit, s.contributors...)
}

func (s *DefaultKBService) RunIndexExtensions(ctx context.Context, roots []string, fast bool) error {
	return kb.RunIndexExtensions(ctx, s.pm, roots, fast, s.contributors...)
}

func (s *DefaultKBService) AddContributors(contributors ...kb.Contributor) {
	for _, contributor := range contributors {
		if contributor != nil {
			s.contributors = append(s.contributors, contributor)
		}
	}
}

func (s *DefaultKBService) ListDocuments() ([]*kb.Document, error) {
	return s.manager.ListDocuments()
}

func (s *DefaultKBService) AddRoot(rootPath string) error {
	return s.manager.AddRoot(rootPath)
}

func (s *DefaultKBService) RemoveRoot(rootPath string) error {
	return s.manager.RemoveRoot(rootPath)
}

func (s *DefaultKBService) GetRoots() []string {
	return s.manager.GetRoots()
}

func (s *DefaultKBService) RebuildIndex() error {
	return s.manager.RebuildIndex()
}

func (s *DefaultKBService) GetLastIndexUpdate() (time.Time, error) {
	return s.manager.GetLastIndexUpdate()
}

func (s *DefaultKBService) NewGraphSearcher() kb.GraphSearcher {
	return kb.NewGraphSearcher()
}
