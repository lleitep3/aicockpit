package kb

import (
	"math"
	"regexp"
	"strings"
)

// BM25Scorer implements advanced BM25 scoring with exact phrase and prefix matching.
type BM25Scorer struct {
	// BM25 parameters
	k1 float64
	b  float64

	// Field weights
	titleWeight       float64
	descriptionWeight float64
	tagsWeight        float64
	contentWeight     float64

	// Corpus statistics (populated on Initialize)
	docCount      int
	avgLenTitle   float64
	avgLenDesc    float64
	avgLenTags    float64
	avgLenContent float64
	docsWithTerm  map[string]int
	initialized   bool
}

// NewBM25Scorer creates a new BM25 scorer with standard weights and parameters.
func NewBM25Scorer() *BM25Scorer {
	return &BM25Scorer{
		k1:                1.5,
		b:                 0.75,
		titleWeight:       0.5,
		descriptionWeight: 0.2,
		tagsWeight:        0.3,
		contentWeight:     0.1,
		docsWithTerm:      make(map[string]int),
	}
}

// Initialize populates corpus statistics dynamically at search time.
func (bs *BM25Scorer) Initialize(query string, corpus []*Document) {
	bs.docCount = len(corpus)
	if bs.docCount == 0 {
		return
	}

	keywords := tokenizeQuery(query)
	bs.docsWithTerm = make(map[string]int)

	var totalLenTitle, totalLenDesc, totalLenTags, totalLenContent float64

	for _, doc := range corpus {
		totalLenTitle += float64(len(strings.Fields(doc.Metadata.Title)))
		totalLenDesc += float64(len(strings.Fields(doc.Metadata.Description)))
		totalLenTags += float64(len(doc.Metadata.Tags))
		totalLenContent += float64(len(strings.Fields(doc.Content)))

		// Count documents containing each query keyword (for IDF)
		for _, kw := range keywords {
			if docContainsTerm(doc, kw) {
				bs.docsWithTerm[kw]++
			}
		}
	}

	N := float64(bs.docCount)
	bs.avgLenTitle = totalLenTitle / N
	bs.avgLenDesc = totalLenDesc / N
	bs.avgLenTags = totalLenTags / N
	bs.avgLenContent = totalLenContent / N
	bs.initialized = true
}

// ScoreKeyword calculates the advanced BM25 score for a document.
func (bs *BM25Scorer) ScoreKeyword(query string, doc *Document) float64 {
	if query == "" || bs.docCount == 0 {
		return 0
	}

	// Auto-initialize if not initialized (fallback for direct calls)
	if !bs.initialized {
		bs.Initialize(query, []*Document{doc})
	}

	keywords := tokenizeQuery(query)
	if len(keywords) == 0 {
		return 0
	}

	var scoreTitle, scoreDesc, scoreTags, scoreContent float64

	for _, kw := range keywords {
		docsWithKw := bs.docsWithTerm[kw]

		// Title
		tfTitle := countTermMatches(doc.Metadata.Title, kw)
		scoreTitle += bs.computeFieldBM25(tfTitle, doc.Metadata.Title, docsWithKw, bs.avgLenTitle)

		// Description
		tfDesc := countTermMatches(doc.Metadata.Description, kw)
		scoreDesc += bs.computeFieldBM25(tfDesc, doc.Metadata.Description, docsWithKw, bs.avgLenDesc)

		// Tags
		tfTags := countTagsMatches(doc.Metadata.Tags, kw)
		scoreTags += bs.computeFieldBM25(tfTags, strings.Join(doc.Metadata.Tags, " "), docsWithKw, bs.avgLenTags)

		// Content
		tfContent := countTermMatches(doc.Content, kw)
		scoreContent += bs.computeFieldBM25(tfContent, doc.Content, docsWithKw, bs.avgLenContent)
	}

	totalScore := (scoreTitle * bs.titleWeight) +
		(scoreDesc * bs.descriptionWeight) +
		(scoreTags * bs.tagsWeight) +
		(scoreContent * bs.contentWeight)

	// Apply Exact Phrase Boost
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))
	if len(keywords) > 1 {
		if strings.Contains(strings.ToLower(doc.Metadata.Title), normalizedQuery) {
			totalScore += 0.4
		}
		if strings.Contains(strings.ToLower(doc.Metadata.Description), normalizedQuery) {
			totalScore += 0.2
		}
		if strings.Contains(strings.ToLower(doc.Content), normalizedQuery) {
			totalScore += 0.1
		}
	}

	// Normalize score to 0-1 range (BM25 can exceed 1, so we scale it)
	// Typically we cap or scale using sigmoid-like function or linear clamp
	if totalScore > 1.0 {
		totalScore = 1.0
	}

	return totalScore
}

// ScoreSemantic is unused in advanced keyword search.
func (bs *BM25Scorer) ScoreSemantic(query string, doc *Document) float64 {
	return 0
}

// CombineScores combines keyword and semantic scores.
func (bs *BM25Scorer) CombineScores(keywordScore, semanticScore float64) float64 {
	return keywordScore
}

// computeFieldBM25 computes BM25 score for a single field given its Term Frequency and average length.
func (bs *BM25Scorer) computeFieldBM25(tf float64, text string, docsWithTerm int, avgLen float64) float64 {
	if tf == 0 {
		return 0
	}

	// Calculate IDF: idf = ln((N - n + 0.5) / (n + 0.5) + 1)
	n := float64(docsWithTerm)
	N := float64(bs.docCount)
	idf := math.Log((N-n+0.5)/(n+0.5) + 1.0)
	if idf < 0 {
		idf = 0.0001
	}

	// Calculate Document Length Normalization
	words := strings.Fields(text)
	dl := float64(len(words))
	if avgLen == 0 {
		avgLen = 1.0
	}

	// TF scaling formula: (tf * (k1 + 1)) / (tf + k1 * (1 - b + b * (dl / avgLen)))
	numerator := tf * (bs.k1 + 1.0)
	denominator := tf + bs.k1*(1.0-bs.b+bs.b*(dl/avgLen))

	return idf * (numerator / denominator)
}

// tokenizeQuery splits queries and normalizes keywords.
func tokenizeQuery(query string) []string {
	normalized := strings.ToLower(strings.TrimSpace(query))
	re := regexp.MustCompile(`[^\w\s\-]`)
	normalized = re.ReplaceAllString(normalized, "")
	keywords := strings.Fields(normalized)

	filtered := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if len(kw) > 0 {
			filtered = append(filtered, kw)
		}
	}
	return filtered
}

// docContainsTerm checks if any field in the document contains the query term (for document frequency).
func docContainsTerm(doc *Document, term string) bool {
	term = strings.ToLower(term)
	if strings.Contains(strings.ToLower(doc.Metadata.Title), term) {
		return true
	}
	if strings.Contains(strings.ToLower(doc.Metadata.Description), term) {
		return true
	}
	for _, tag := range doc.Metadata.Tags {
		if strings.Contains(strings.ToLower(tag), term) {
			return true
		}
	}
	return strings.Contains(strings.ToLower(doc.Content), term)
}

// countTermMatches counts matches of the term in a text with prefix matching support.
func countTermMatches(text string, term string) float64 {
	if text == "" || term == "" {
		return 0
	}

	// Tokenize text into words
	normalized := strings.ToLower(text)
	re := regexp.MustCompile(`[^\w\s\-]`)
	normalized = re.ReplaceAllString(normalized, "")
	words := strings.Fields(normalized)

	term = strings.ToLower(term)
	var count float64

	for _, w := range words {
		if w == term {
			count += 1.0
		} else if strings.HasPrefix(w, term) {
			// Prefix match gets partial score (e.g. 0.4)
			count += 0.4
		}
	}

	return count
}

// countTagsMatches counts tags matching the query term.
func countTagsMatches(tags []string, term string) float64 {
	if len(tags) == 0 || term == "" {
		return 0
	}
	term = strings.ToLower(term)
	var count float64
	for _, t := range tags {
		tLower := strings.ToLower(t)
		if tLower == term {
			count += 1.0
		} else if strings.HasPrefix(tLower, term) {
			count += 0.4
		}
	}
	return count
}
