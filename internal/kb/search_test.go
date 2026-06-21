package kb

import (
	"strings"
	"testing"
	"time"
)

func TestKeywordSearcher_Search(t *testing.T) {
	scorer := NewDefaultScorer()
	searcher := NewKeywordSearcher(scorer)

	documents := []*Document{
		{
			ID:   "doc1",
			Path: "guides/logging.md",
			Metadata: Metadata{
				Title:       "Logging Configuration Guide",
				Description: "How to configure logging in AICockpit",
				Tags:        []string{"logging", "configuration", "setup"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: "This guide explains how to configure logging. Logging is important for debugging.",
		},
		{
			ID:   "doc2",
			Path: "guides/metrics.md",
			Metadata: Metadata{
				Title:       "Metrics Collection",
				Description: "Understanding metrics in AICockpit",
				Tags:        []string{"metrics", "monitoring"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: "Metrics help you understand performance. Collect metrics regularly.",
		},
		{
			ID:   "doc3",
			Path: "troubleshooting/logging-issues.md",
			Metadata: Metadata{
				Title:       "Troubleshooting Logging Issues",
				Description: "Solutions for common logging problems",
				Tags:        []string{"logging", "troubleshooting"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: "If logging is not working, check your configuration. Logging errors are common.",
		},
	}

	tests := []struct {
		name          string
		query         string
		wantCount     int
		wantFirstID   string
		wantFirstHigh bool
	}{
		{
			name:          "search for logging",
			query:         "logging",
			wantCount:     2,
			wantFirstID:   "doc1",
			wantFirstHigh: true,
		},
		{
			name:          "search for configuration",
			query:         "configuration",
			wantCount:     2,
			wantFirstID:   "doc1",
			wantFirstHigh: true,
		},
		{
			name:          "search for metrics",
			query:         "metrics",
			wantCount:     1,
			wantFirstID:   "doc2",
			wantFirstHigh: true,
		},
		{
			name:      "empty query",
			query:     "",
			wantCount: 0,
		},
		{
			name:      "no matches",
			query:     "nonexistent",
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := searcher.Search(tt.query, documents)
			if err != nil {
				t.Fatalf("Search() error = %v", err)
			}

			if results.Total != tt.wantCount {
				t.Errorf("Search() Total = %d, want %d", results.Total, tt.wantCount)
			}

			if tt.wantCount > 0 {
				if results.Results[0].ID != tt.wantFirstID {
					t.Errorf("Search() first result ID = %s, want %s", results.Results[0].ID, tt.wantFirstID)
				}

				if tt.wantFirstHigh && results.Results[0].Score < 0.5 {
					t.Errorf("Search() first result score = %f, want >= 0.5", results.Results[0].Score)
				}
			}
		})
	}
}

func TestDefaultScorer_ScoreKeyword(t *testing.T) {
	scorer := NewDefaultScorer()

	tests := []struct {
		name      string
		query     string
		doc       *Document
		wantScore float64
		wantRange bool // if true, check range instead of exact value
	}{
		{
			name:  "exact title match",
			query: "logging",
			doc: &Document{
				ID:   "test",
				Path: "test.md",
				Metadata: Metadata{
					Title:       "Logging Configuration",
					Description: "Test",
					Tags:        []string{},
				},
				Content: "Content",
			},
			wantScore: 0.5,
			wantRange: true, // Should be >= 0.5 due to title weight
		},
		{
			name:  "tag match",
			query: "logging",
			doc: &Document{
				ID:   "test",
				Path: "test.md",
				Metadata: Metadata{
					Title:       "Some Title",
					Description: "Description",
					Tags:        []string{"logging", "setup"},
				},
				Content: "Content",
			},
			wantScore: 0.3,
			wantRange: true, // Should be >= 0.3 due to tags weight
		},
		{
			name:  "no match",
			query: "nonexistent",
			doc: &Document{
				ID:   "test",
				Path: "test.md",
				Metadata: Metadata{
					Title:       "Some Title",
					Description: "Description",
					Tags:        []string{"other"},
				},
				Content: "Content",
			},
			wantScore: 0,
		},
		{
			name:  "empty query",
			query: "",
			doc: &Document{
				ID:   "test",
				Path: "test.md",
				Metadata: Metadata{
					Title: "Title",
				},
				Content: "Content",
			},
			wantScore: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.ScoreKeyword(tt.query, tt.doc)

			if tt.wantRange {
				if score < tt.wantScore {
					t.Errorf("ScoreKeyword() = %f, want >= %f", score, tt.wantScore)
				}
			} else {
				if score != tt.wantScore {
					t.Errorf("ScoreKeyword() = %f, want %f", score, tt.wantScore)
				}
			}

			// Score should always be between 0 and 1
			if score < 0 || score > 1 {
				t.Errorf("ScoreKeyword() = %f, want between 0 and 1", score)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantKeys []string
	}{
		{
			name:     "single word",
			query:    "logging",
			wantKeys: []string{"logging"},
		},
		{
			name:     "multiple words",
			query:    "logging configuration setup",
			wantKeys: []string{"logging", "configuration", "setup"},
		},
		{
			name:     "with special characters",
			query:    "how-to configure!",
			wantKeys: []string{"how-to", "configure"},
		},
		{
			name:     "with extra spaces",
			query:    "  logging   setup  ",
			wantKeys: []string{"logging", "setup"},
		},
		{
			name:     "empty string",
			query:    "",
			wantKeys: []string{},
		},
		{
			name:     "only spaces",
			query:    "   ",
			wantKeys: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys := tokenize(tt.query)

			if len(keys) != len(tt.wantKeys) {
				t.Errorf("tokenize() length = %d, want %d", len(keys), len(tt.wantKeys))
				return
			}

			for i, key := range keys {
				if key != tt.wantKeys[i] {
					t.Errorf("tokenize()[%d] = %s, want %s", i, key, tt.wantKeys[i])
				}
			}
		})
	}
}

func TestCalculateMatchScore(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		keywords []string
		want     float64
	}{
		{
			name:     "all keywords match",
			text:     "This is a logging configuration guide",
			keywords: []string{"logging", "configuration"},
			want:     1.0,
		},
		{
			name:     "partial match",
			text:     "This is a logging guide",
			keywords: []string{"logging", "configuration"},
			want:     0.5,
		},
		{
			name:     "no match",
			text:     "This is a guide",
			keywords: []string{"logging", "configuration"},
			want:     0,
		},
		{
			name:     "case insensitive",
			text:     "LOGGING Configuration",
			keywords: []string{"logging", "configuration"},
			want:     1.0,
		},
		{
			name:     "empty text",
			text:     "",
			keywords: []string{"logging"},
			want:     0,
		},
		{
			name:     "empty keywords",
			text:     "logging",
			keywords: []string{},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateMatchScore(tt.text, tt.keywords)
			if score != tt.want {
				t.Errorf("calculateMatchScore() = %f, want %f", score, tt.want)
			}
		})
	}
}

func TestCalculateTagsScore(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		keywords []string
		want     float64
	}{
		{
			name:     "all keywords match tags",
			tags:     []string{"logging", "configuration", "setup"},
			keywords: []string{"logging", "configuration"},
			want:     1.0,
		},
		{
			name:     "partial match",
			tags:     []string{"logging", "metrics"},
			keywords: []string{"logging", "configuration"},
			want:     0.5,
		},
		{
			name:     "no match",
			tags:     []string{"metrics", "monitoring"},
			keywords: []string{"logging", "configuration"},
			want:     0,
		},
		{
			name:     "case insensitive",
			tags:     []string{"LOGGING", "Configuration"},
			keywords: []string{"logging", "configuration"},
			want:     1.0,
		},
		{
			name:     "partial tag match",
			tags:     []string{"logging-setup"},
			keywords: []string{"logging"},
			want:     1.0,
		},
		{
			name:     "empty tags",
			tags:     []string{},
			keywords: []string{"logging"},
			want:     0,
		},
		{
			name:     "empty keywords",
			tags:     []string{"logging"},
			keywords: []string{},
			want:     0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateTagsScore(tt.tags, tt.keywords)
			if score != tt.want {
				t.Errorf("calculateTagsScore() = %f, want %f", score, tt.want)
			}
		})
	}
}

func TestCalculateContentScore(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		keywords []string
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "high frequency",
			content:  "logging logging logging is important. logging helps debugging.",
			keywords: []string{"logging"},
			wantMin:  0.5,
			wantMax:  1.0,
		},
		{
			name:     "low frequency",
			content:  "This is a long document about many topics. logging is mentioned once.",
			keywords: []string{"logging"},
			wantMin:  0,
			wantMax:  1.0,
		},
		{
			name:     "no match",
			content:  "This document talks about metrics and monitoring.",
			keywords: []string{"logging"},
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "empty content",
			content:  "",
			keywords: []string{"logging"},
			wantMin:  0,
			wantMax:  0,
		},
		{
			name:     "empty keywords",
			content:  "logging logging logging",
			keywords: []string{},
			wantMin:  0,
			wantMax:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculateContentScore(tt.content, tt.keywords)
			if score < tt.wantMin || score > tt.wantMax {
				t.Errorf("calculateContentScore() = %f, want between %f and %f", score, tt.wantMin, tt.wantMax)
			}

			// Score should always be between 0 and 1
			if score < 0 || score > 1 {
				t.Errorf("calculateContentScore() = %f, want between 0 and 1", score)
			}
		})
	}
}

func TestDefaultScorer_CombineScores(t *testing.T) {
	scorer := NewDefaultScorer()

	tests := []struct {
		name          string
		keywordScore  float64
		semanticScore float64
		want          float64
	}{
		{
			name:          "equal scores",
			keywordScore:  0.5,
			semanticScore: 0.5,
			want:          0.5,
		},
		{
			name:          "semantic higher",
			keywordScore:  0.2,
			semanticScore: 0.8,
			want:          0.56, // (0.2 * 0.4) + (0.8 * 0.6) = 0.08 + 0.48 = 0.56
		},
		{
			name:          "keyword higher",
			keywordScore:  0.8,
			semanticScore: 0.2,
			want:          0.44, // (0.8 * 0.4) + (0.2 * 0.6) = 0.32 + 0.12 = 0.44
		},
		{
			name:          "both zero",
			keywordScore:  0,
			semanticScore: 0,
			want:          0,
		},
		{
			name:          "both one",
			keywordScore:  1,
			semanticScore: 1,
			want:          1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := scorer.CombineScores(tt.keywordScore, tt.semanticScore)
			// Use tolerance for floating point comparison
			const epsilon = 1e-9
			if score < tt.want-epsilon || score > tt.want+epsilon {
				t.Errorf("CombineScores() = %f, want %f", score, tt.want)
			}
		})
	}
}

func TestNewKeywordSearcher(t *testing.T) {
	scorer := NewDefaultScorer()
	searcher := NewKeywordSearcher(scorer)

	if searcher == nil {
		t.Fatal("NewKeywordSearcher() returned nil")
	}

	if searcher.scorer != scorer {
		t.Error("NewKeywordSearcher() did not set scorer correctly")
	}
}

func TestNewDefaultScorer(t *testing.T) {
	scorer := NewDefaultScorer()

	if scorer == nil {
		t.Fatal("NewDefaultScorer() returned nil")
	}

	if scorer.titleWeight != 0.5 {
		t.Errorf("titleWeight = %f, want 0.5", scorer.titleWeight)
	}

	if scorer.tagsWeight != 0.3 {
		t.Errorf("tagsWeight = %f, want 0.3", scorer.tagsWeight)
	}

	if scorer.descriptionWeight != 0.2 {
		t.Errorf("descriptionWeight = %f, want 0.2", scorer.descriptionWeight)
	}

	if scorer.contentWeight != 0.1 {
		t.Errorf("contentWeight = %f, want 0.1", scorer.contentWeight)
	}
}

// Benchmark tests
func BenchmarkKeywordSearcher_Search(b *testing.B) {
	scorer := NewDefaultScorer()
	searcher := NewKeywordSearcher(scorer)

	// Create 100 documents
	documents := make([]*Document, 100)
	for i := 0; i < 100; i++ {
		documents[i] = &Document{
			ID:   "doc" + string(rune(i)),
			Path: "guides/doc.md",
			Metadata: Metadata{
				Title:       "Document Title",
				Description: "Description",
				Tags:        []string{"tag1", "tag2"},
				Created:     time.Now(),
				Modified:    time.Now(),
			},
			Content: strings.Repeat("logging configuration setup ", 10),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		searcher.Search("logging configuration", documents)
	}
}

func BenchmarkTokenize(b *testing.B) {
	query := "logging configuration setup troubleshooting"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tokenize(query)
	}
}
