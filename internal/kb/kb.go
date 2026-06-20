package kb

import (
	"time"
)

// Metadata represents the metadata of a KB document.
type Metadata struct {
	Title       string    `yaml:"title"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags"`
	Created     time.Time `yaml:"created"`
	Modified    time.Time `yaml:"modified"`
	Related     []string  `yaml:"related"`
	Author      string    `yaml:"author"`
	Version     string    `yaml:"version"`
}

// Document represents a KB document with metadata and content.
type Document struct {
	ID       string
	Path     string
	Metadata Metadata
	Content  string
}

// SearchResult represents a single search result with scoring information.
type SearchResult struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	Path           string  `json:"path"`
	Score          float64 `json:"score"`
	KeywordScore   float64 `json:"keyword_score"`
	SemanticScore  float64 `json:"semantic_score"`
	Tags           []string `json:"tags"`
	Excerpt        string  `json:"excerpt"`
	Created        time.Time `json:"created"`
	Modified       time.Time `json:"modified"`
}

// SearchResults represents a collection of search results.
type SearchResults struct {
	Query   string          `json:"query"`
	Results []SearchResult  `json:"results"`
	Total   int             `json:"total"`
}

// Repository defines the interface for KB operations.
type Repository interface {
	// LoadDocument loads a document from disk
	LoadDocument(path string) (*Document, error)

	// ListDocuments lists all documents in a directory
	ListDocuments(dir string) ([]*Document, error)

	// SaveDocument saves a document to disk
	SaveDocument(doc *Document) error

	// DeleteDocument deletes a document from disk
	DeleteDocument(path string) error
}

// Searcher defines the interface for search operations.
type Searcher interface {
	// Search performs a keyword search on documents
	Search(query string, documents []*Document) (*SearchResults, error)
}

// SemanticSearcher defines the interface for semantic search operations.
type SemanticSearcher interface {
	// SemanticSearch performs a semantic search on documents
	SemanticSearch(query string, documents []*Document) (*SearchResults, error)
}

// Scorer defines the interface for scoring search results.
type Scorer interface {
	// ScoreKeyword calculates keyword score for a document
	ScoreKeyword(query string, doc *Document) float64

	// ScoreSemantic calculates semantic score for a document
	ScoreSemantic(query string, doc *Document) float64

	// CombineScores combines keyword and semantic scores
	CombineScores(keywordScore, semanticScore float64) float64
}
