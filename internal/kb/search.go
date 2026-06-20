package kb

import (
	"regexp"
	"sort"
	"strings"
)

// KeywordSearcher implements keyword-based search for KB documents.
type KeywordSearcher struct {
	scorer Scorer
}

// NewKeywordSearcher creates a new keyword searcher.
func NewKeywordSearcher(scorer Scorer) *KeywordSearcher {
	return &KeywordSearcher{
		scorer: scorer,
	}
}

// Search performs a keyword search on documents.
func (ks *KeywordSearcher) Search(query string, documents []*Document) (*SearchResults, error) {
	if query == "" {
		return &SearchResults{
			Query:   query,
			Results: []SearchResult{},
			Total:   0,
		}, nil
	}

	results := make([]SearchResult, 0)

	for _, doc := range documents {
		score := ks.scorer.ScoreKeyword(query, doc)
		if score > 0 {
			result := SearchResult{
				ID:            doc.ID,
				Title:         doc.Metadata.Title,
				Description:   doc.Metadata.Description,
				Path:          doc.Path,
				Score:         score,
				KeywordScore:  score,
				SemanticScore: 0,
				Tags:          doc.Metadata.Tags,
				Excerpt:       ExtractExcerpt(doc.Content, 200),
				Created:       doc.Metadata.Created,
				Modified:      doc.Metadata.Modified,
			}
			results = append(results, result)
		}
	}

	// Sort by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return &SearchResults{
		Query:   query,
		Results: results,
		Total:   len(results),
	}, nil
}

// DefaultScorer implements the Scorer interface with keyword scoring logic.
type DefaultScorer struct {
	titleWeight       float64
	descriptionWeight float64
	tagsWeight        float64
	contentWeight     float64
}

// NewDefaultScorer creates a new default scorer with standard weights.
func NewDefaultScorer() *DefaultScorer {
	return &DefaultScorer{
		titleWeight:       0.5,
		descriptionWeight: 0.2,
		tagsWeight:        0.3,
		contentWeight:     0.1,
	}
}

// ScoreKeyword calculates keyword score for a document.
// Scoring factors:
// - Title match: 0.5 weight
// - Tags match: 0.3 weight
// - Description match: 0.2 weight
// - Content frequency: 0.1 weight
func (ds *DefaultScorer) ScoreKeyword(query string, doc *Document) float64 {
	if query == "" {
		return 0
	}

	keywords := tokenize(query)
	if len(keywords) == 0 {
		return 0
	}

	var totalScore float64

	// Score title matches
	titleScore := calculateMatchScore(doc.Metadata.Title, keywords)
	totalScore += titleScore * ds.titleWeight

	// Score tags matches
	tagsScore := calculateTagsScore(doc.Metadata.Tags, keywords)
	totalScore += tagsScore * ds.tagsWeight

	// Score description matches
	descScore := calculateMatchScore(doc.Metadata.Description, keywords)
	totalScore += descScore * ds.descriptionWeight

	// Score content frequency
	contentScore := calculateContentScore(doc.Content, keywords)
	totalScore += contentScore * ds.contentWeight

	// Normalize to 0-1 range
	if totalScore > 1.0 {
		totalScore = 1.0
	}

	return totalScore
}

// ScoreSemantic is not implemented for keyword searcher.
func (ds *DefaultScorer) ScoreSemantic(query string, doc *Document) float64 {
	return 0
}

// CombineScores combines keyword and semantic scores.
func (ds *DefaultScorer) CombineScores(keywordScore, semanticScore float64) float64 {
	// 40% keyword, 60% semantic
	return (keywordScore * 0.4) + (semanticScore * 0.6)
}

// tokenize splits query into keywords and normalizes them.
func tokenize(query string) []string {
	// Convert to lowercase and split by whitespace
	normalized := strings.ToLower(strings.TrimSpace(query))

	// Remove special characters but keep hyphens and underscores
	re := regexp.MustCompile(`[^\w\s\-]`)
	normalized = re.ReplaceAllString(normalized, "")

	// Split by whitespace
	keywords := strings.Fields(normalized)

	// Filter out empty strings
	filtered := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if len(kw) > 0 {
			filtered = append(filtered, kw)
		}
	}

	return filtered
}

// calculateMatchScore calculates how well keywords match a text.
// Returns a score between 0 and 1.
func calculateMatchScore(text string, keywords []string) float64 {
	if text == "" || len(keywords) == 0 {
		return 0
	}

	normalizedText := strings.ToLower(text)
	matchCount := 0

	for _, keyword := range keywords {
		if strings.Contains(normalizedText, keyword) {
			matchCount++
		}
	}

	// Return ratio of matched keywords to total keywords
	return float64(matchCount) / float64(len(keywords))
}

// calculateTagsScore calculates how well keywords match document tags.
// Returns a score between 0 and 1.
func calculateTagsScore(tags []string, keywords []string) float64 {
	if len(tags) == 0 || len(keywords) == 0 {
		return 0
	}

	matchCount := 0
	for _, keyword := range keywords {
		for _, tag := range tags {
			if strings.EqualFold(keyword, tag) || strings.Contains(strings.ToLower(tag), keyword) {
				matchCount++
				break
			}
		}
	}

	return float64(matchCount) / float64(len(keywords))
}

// calculateContentScore calculates keyword frequency in document content.
// Returns a score between 0 and 1 based on keyword density.
func calculateContentScore(content string, keywords []string) float64 {
	if content == "" || len(keywords) == 0 {
		return 0
	}

	normalizedContent := strings.ToLower(content)
	totalMatches := 0

	for _, keyword := range keywords {
		// Count occurrences of keyword
		count := strings.Count(normalizedContent, keyword)
		totalMatches += count
	}

	if totalMatches == 0 {
		return 0
	}

	// Calculate keyword density (matches per 100 words)
	words := strings.Fields(content)
	if len(words) == 0 {
		return 0
	}

	density := float64(totalMatches) / float64(len(words)) * 100

	// Normalize density to 0-1 range (cap at 5% density = 1.0)
	if density > 5.0 {
		density = 5.0
	}

	return density / 5.0
}
