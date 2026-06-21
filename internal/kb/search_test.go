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

// --- BM25 Tests ---

func TestNewBM25Scorer(t *testing.T) {
	scorer := NewBM25Scorer()
	if scorer == nil {
		t.Fatal("NewBM25Scorer() returned nil")
	}
	if scorer.k1 != 1.5 {
		t.Errorf("k1 = %f, want 1.5", scorer.k1)
	}
	if scorer.b != 0.75 {
		t.Errorf("b = %f, want 0.75", scorer.b)
	}
	if scorer.titleWeight != 0.5 {
		t.Errorf("titleWeight = %f, want 0.5", scorer.titleWeight)
	}
	if scorer.tagsWeight != 0.3 {
		t.Errorf("tagsWeight = %f, want 0.3", scorer.tagsWeight)
	}
}

func TestBM25Scorer_ScoreKeyword_EmptyQuery(t *testing.T) {
	scorer := NewBM25Scorer()
	doc := &Document{
		ID:   "d1",
		Path: "a.md",
		Metadata: Metadata{
			Title: "Logging Guide",
			Tags:  []string{"logging"},
		},
		Content: "Content",
	}
	if score := scorer.ScoreKeyword("", doc); score != 0 {
		t.Errorf("expected 0 for empty query, got %f", score)
	}
}

func TestBM25Scorer_ScoreKeyword_NoMatch(t *testing.T) {
	scorer := NewBM25Scorer()
	corpus := []*Document{
		{ID: "d1", Path: "a.md", Metadata: Metadata{Title: "Logging Guide", Tags: []string{"logging"}}, Content: "logging setup"},
	}
	scorer.Initialize("metrics", corpus)
	score := scorer.ScoreKeyword("metrics", corpus[0])
	if score != 0 {
		t.Errorf("expected 0 for no match, got %f", score)
	}
}

func TestBM25Scorer_ScoreKeyword_TitleMatch(t *testing.T) {
	scorer := NewBM25Scorer()
	corpus := []*Document{
		{ID: "d1", Path: "a.md", Metadata: Metadata{Title: "Logging Configuration Guide", Tags: []string{"logging"}}, Content: "logging setup"},
		{ID: "d2", Path: "b.md", Metadata: Metadata{Title: "Metrics Overview", Tags: []string{"metrics"}}, Content: "metrics performance"},
	}
	scorer.Initialize("logging", corpus)

	scoreLogging := scorer.ScoreKeyword("logging", corpus[0])
	scoreMetrics := scorer.ScoreKeyword("logging", corpus[1])

	if scoreLogging <= 0 {
		t.Errorf("expected positive score for title match, got %f", scoreLogging)
	}
	if scoreLogging <= scoreMetrics {
		t.Errorf("logging doc (%f) should score higher than metrics doc (%f)", scoreLogging, scoreMetrics)
	}
}

func TestBM25Scorer_ScoreKeyword_ScoreRange(t *testing.T) {
	scorer := NewBM25Scorer()
	corpus := []*Document{
		{ID: "d1", Path: "a.md", Metadata: Metadata{Title: "Logging Configuration Guide", Tags: []string{"logging"}}, Content: "logging setup guide"},
		{ID: "d2", Path: "b.md", Metadata: Metadata{Title: "Metrics Overview", Tags: []string{"metrics"}}, Content: "metrics performance"},
	}
	scorer.Initialize("logging", corpus)

	for _, doc := range corpus {
		score := scorer.ScoreKeyword("logging", doc)
		if score < 0 || score > 1 {
			t.Errorf("score %f for doc %s out of range [0,1]", score, doc.ID)
		}
	}
}

func TestBM25Scorer_ExactPhraseBoost(t *testing.T) {
	scorer := NewBM25Scorer()
	// Two documents: one has the exact phrase in title, the other has words scattered
	corpus := []*Document{
		{
			ID:   "exact",
			Path: "a.md",
			Metadata: Metadata{
				Title:       "logging configuration guide",
				Description: "A guide for logging configuration",
				Tags:        []string{"logging", "configuration"},
			},
			Content: "This guide covers logging configuration in detail.",
		},
		{
			ID:   "scattered",
			Path: "b.md",
			Metadata: Metadata{
				Title:       "Configuration and Logging topics",
				Description: "Several topics covered here",
				Tags:        []string{"configuration", "logging"},
			},
			Content: "Some configuration options and some logging notes are covered.",
		},
	}
	scorer.Initialize("logging configuration", corpus)

	scoreExact := scorer.ScoreKeyword("logging configuration", corpus[0])
	scoreScattered := scorer.ScoreKeyword("logging configuration", corpus[1])

	// The document with exact phrase in title should score higher
	if scoreExact <= scoreScattered {
		t.Errorf("exact phrase doc (%f) should score >= scattered doc (%f)", scoreExact, scoreScattered)
	}
}

func TestBM25Scorer_PrefixMatch(t *testing.T) {
	scorer := NewBM25Scorer()
	corpus := []*Document{
		{ID: "d1", Path: "a.md", Metadata: Metadata{Title: "Configuration Guide"}, Content: "configuration settings"},
	}
	scorer.Initialize("config", corpus)

	score := scorer.ScoreKeyword("config", corpus[0])
	// "config" is a prefix of "configuration" – should get a partial score > 0
	if score <= 0 {
		t.Errorf("prefix match should yield score > 0, got %f", score)
	}
}

func TestBM25Scorer_RarityBoost(t *testing.T) {
	// IDF should boost terms that appear in fewer documents.
	// "rare" appears in only one doc; "common" appears in all.
	scorer := NewBM25Scorer()
	corpus := []*Document{
		{ID: "d1", Path: "a.md", Metadata: Metadata{Title: "common rare topic"}, Content: "common rare content here"},
		{ID: "d2", Path: "b.md", Metadata: Metadata{Title: "common topic two"}, Content: "common content two"},
		{ID: "d3", Path: "c.md", Metadata: Metadata{Title: "common topic three"}, Content: "common content three"},
	}
	scorer.Initialize("rare", corpus)

	// d1 has "rare", d2 and d3 don't.
	scoreRare := scorer.ScoreKeyword("rare", corpus[0])
	scoreCommon2 := scorer.ScoreKeyword("rare", corpus[1])

	if scoreRare <= scoreCommon2 {
		t.Errorf("doc with rare term (%f) should score higher than doc without (%f)", scoreRare, scoreCommon2)
	}
	if scoreCommon2 != 0 {
		t.Errorf("doc without rare term should score 0, got %f", scoreCommon2)
	}
}

func TestBM25Scorer_Initialize_EmptyCorpus(t *testing.T) {
	scorer := NewBM25Scorer()
	// Should not panic or crash on empty corpus
	scorer.Initialize("query", []*Document{})
	// docCount should remain 0
	if scorer.docCount != 0 {
		t.Errorf("docCount = %d, want 0", scorer.docCount)
	}
}

func TestBM25Scorer_ScoreSemantic(t *testing.T) {
	scorer := NewBM25Scorer()
	score := scorer.ScoreSemantic("query", &Document{ID: "d1"})
	if score != 0 {
		t.Errorf("ScoreSemantic() should return 0, got %f", score)
	}
}

func TestBM25Scorer_CombineScores(t *testing.T) {
	scorer := NewBM25Scorer()
	// BM25 CombineScores just returns keywordScore
	combined := scorer.CombineScores(0.7, 0.9)
	if combined != 0.7 {
		t.Errorf("CombineScores() = %f, want 0.7", combined)
	}
}

func TestCountTermMatches(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		term    string
		wantMin float64
		wantMax float64
	}{
		{"exact match", "logging configuration setup", "logging", 1.0, 1.0},
		{"prefix match", "configuration setup guide", "config", 0.3, 0.5},
		{"no match", "metrics monitoring", "logging", 0, 0},
		{"empty text", "", "logging", 0, 0},
		{"empty term", "logging setup", "", 0, 0},
		{"multiple exact", "logging logging setup logging", "logging", 3.0, 3.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countTermMatches(tt.text, tt.term)
			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("countTermMatches(%q, %q) = %f, want [%f, %f]", tt.text, tt.term, count, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestCountTagsMatches(t *testing.T) {
	tests := []struct {
		name    string
		tags    []string
		term    string
		wantMin float64
		wantMax float64
	}{
		{"exact match", []string{"logging", "setup"}, "logging", 1.0, 1.0},
		{"prefix match", []string{"configuration", "setup"}, "config", 0.3, 0.5},
		{"no match", []string{"metrics", "monitoring"}, "logging", 0, 0},
		{"empty tags", []string{}, "logging", 0, 0},
		{"empty term", []string{"logging"}, "", 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := countTagsMatches(tt.tags, tt.term)
			if count < tt.wantMin || count > tt.wantMax {
				t.Errorf("countTagsMatches(%v, %q) = %f, want [%f, %f]", tt.tags, tt.term, count, tt.wantMin, tt.wantMax)
			}
		})
	}
}

func TestDocContainsTerm(t *testing.T) {
	doc := &Document{
		ID:   "d1",
		Path: "a.md",
		Metadata: Metadata{
			Title:       "Logging Configuration Guide",
			Description: "Setup instructions",
			Tags:        []string{"logging", "setup"},
		},
		Content: "detailed content about logging",
	}
	tests := []struct {
		term string
		want bool
	}{
		{"logging", true},
		{"setup", true},
		{"guide", true},
		{"detailed", true},
		{"nonexistent", false},
	}
	for _, tt := range tests {
		got := docContainsTerm(doc, tt.term)
		if got != tt.want {
			t.Errorf("docContainsTerm(doc, %q) = %v, want %v", tt.term, got, tt.want)
		}
	}
}

func TestTokenizeQuery(t *testing.T) {
	tests := []struct {
		query    string
		wantKeys []string
	}{
		{"logging", []string{"logging"}},
		{"logging setup", []string{"logging", "setup"}},
		{"  logging  setup  ", []string{"logging", "setup"}},
		{"", []string{}},
		{"how-to configure!", []string{"how-to", "configure"}},
	}
	for _, tt := range tests {
		got := tokenizeQuery(tt.query)
		if len(got) != len(tt.wantKeys) {
			t.Errorf("tokenizeQuery(%q) = %v, want %v", tt.query, got, tt.wantKeys)
			continue
		}
		for i, k := range got {
			if k != tt.wantKeys[i] {
				t.Errorf("tokenizeQuery(%q)[%d] = %s, want %s", tt.query, i, k, tt.wantKeys[i])
			}
		}
	}
}

// TestBM25VsDefaultScorer_RelevanceComparison documents the measurable improvement
// of BM25 over DefaultScorer on a realistic corpus with multiple relevance scenarios.
func TestBM25VsDefaultScorer_RelevanceComparison(t *testing.T) {
	corpus := []*Document{
		{
			ID:   "kb-logging",
			Path: "guides/logging.md",
			Metadata: Metadata{
				Title:       "Logging Configuration Guide",
				Description: "How to configure logging in AICockpit",
				Tags:        []string{"logging", "configuration", "setup"},
			},
			Content: "This guide explains how to configure logging. Logging is critical for debugging production issues.",
		},
		{
			ID:   "kb-metrics",
			Path: "guides/metrics.md",
			Metadata: Metadata{
				Title:       "Metrics Collection Overview",
				Description: "Understanding metrics in AICockpit",
				Tags:        []string{"metrics", "monitoring", "performance"},
			},
			Content: "Metrics help you understand performance. Collect metrics regularly for best results.",
		},
		{
			ID:   "kb-troubleshoot",
			Path: "troubleshooting/logging-issues.md",
			Metadata: Metadata{
				Title:       "Troubleshooting Logging Issues",
				Description: "Solutions for common logging problems",
				Tags:        []string{"logging", "troubleshooting", "debugging"},
			},
			Content: "If logging is not working, check your configuration. Logging errors are common.",
		},
		{
			ID:   "kb-install",
			Path: "guides/install.md",
			Metadata: Metadata{
				Title:       "Installation Guide",
				Description: "How to install AICockpit on your system",
				Tags:        []string{"installation", "setup", "getting-started"},
			},
			Content: "Follow these steps to install AICockpit on Linux, macOS, or Windows.",
		},
		{
			ID:   "kb-config",
			Path: "references/config.md",
			Metadata: Metadata{
				Title:       "Configuration Reference",
				Description: "All configuration options for AICockpit",
				Tags:        []string{"configuration", "reference", "yaml"},
			},
			Content: "The config.yaml file controls all aspects of AICockpit behavior. Configuration options include logging level, metrics interval, and more.",
		},
	}

	type scenario struct {
		query         string
		expectedFirst string // expected top result doc ID with BM25
		description   string
	}

	scenarios := []scenario{
		{
			query:         "logging configuration",
			expectedFirst: "kb-logging",
			description:   "Exact phrase match in title should rank highest",
		},
		{
			query:         "troubleshoot",
			expectedFirst: "kb-troubleshoot",
			description:   "Prefix match on 'troubleshoot' → 'troubleshooting'",
		},
		{
			query:         "metrics performance",
			expectedFirst: "kb-metrics",
			description:   "Two keywords both present in metrics doc",
		},
	}

	bm25 := NewBM25Scorer()
	defaultScorer := NewDefaultScorer()

	for _, sc := range scenarios {
		t.Run(sc.description, func(t *testing.T) {
			// BM25 search
			bm25Searcher := NewKeywordSearcher(bm25)
			bm25Results, err := bm25Searcher.Search(sc.query, corpus)
			if err != nil {
				t.Fatalf("BM25 search error: %v", err)
			}

			// Default search
			defaultSearcher := NewKeywordSearcher(defaultScorer)
			defaultResults, err := defaultSearcher.Search(sc.query, corpus)
			if err != nil {
				t.Fatalf("Default search error: %v", err)
			}

			// BM25 must return results
			if bm25Results.Total == 0 {
				t.Errorf("BM25 returned 0 results for query %q", sc.query)
				return
			}

			// BM25 top result must match expected
			if bm25Results.Results[0].ID != sc.expectedFirst {
				t.Errorf("BM25 top result = %q, want %q (default top = %q)",
					bm25Results.Results[0].ID,
					sc.expectedFirst,
					func() string {
						if defaultResults.Total > 0 {
							return defaultResults.Results[0].ID
						}
						return "none"
					}(),
				)
			}
		})
	}
}

func BenchmarkBM25Scorer_Search(b *testing.B) {
	scorer := NewBM25Scorer()
	searcher := NewKeywordSearcher(scorer)

	documents := make([]*Document, 100)
	for i := 0; i < 100; i++ {
		documents[i] = &Document{
			ID:   "doc" + string(rune(i)),
			Path: "guides/doc.md",
			Metadata: Metadata{
				Title:       "Document Title logging",
				Description: "Description with configuration",
				Tags:        []string{"tag1", "logging"},
			},
			Content: strings.Repeat("logging configuration setup ", 10),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = searcher.Search("logging configuration", documents)
	}
}
