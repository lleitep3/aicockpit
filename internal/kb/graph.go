package kb

import (
	"fmt"
)

// GraphSearchEngine implements the GraphSearcher interface.
type GraphSearchEngine struct {
	documents map[string]*Document
	adjacency map[string][]string // Maps a docID to a list of related docIDs
}

// NewGraphSearcher creates a new GraphSearchEngine.
func NewGraphSearcher() *GraphSearchEngine {
	return &GraphSearchEngine{
		documents: make(map[string]*Document),
		adjacency: make(map[string][]string),
	}
}

// BuildGraph constructs the graph from a set of documents.
// It creates a bidirectional graph using the 'Related' metadata field.
func (g *GraphSearchEngine) BuildGraph(documents []*Document) error {
	g.documents = make(map[string]*Document)
	g.adjacency = make(map[string][]string)

	// Add nodes
	for _, doc := range documents {
		g.documents[doc.ID] = doc
		g.adjacency[doc.ID] = make([]string, 0)
	}

	// Helper to add edge if not exists
	addEdge := func(from, to string) {
		for _, existing := range g.adjacency[from] {
			if existing == to {
				return
			}
		}
		g.adjacency[from] = append(g.adjacency[from], to)
	}

	// Add bidirectional edges based on Related field
	for _, doc := range documents {
		for _, relatedID := range doc.Metadata.Related {
			if _, exists := g.documents[relatedID]; exists {
				addEdge(doc.ID, relatedID)
				addEdge(relatedID, doc.ID) // Bidirectional
			}
		}
	}

	return nil
}

// SearchGraph performs a BFS search starting from rootID up to maxDepth.
func (g *GraphSearchEngine) SearchGraph(rootID string, maxDepth int) (*GraphResult, error) {
	if _, exists := g.documents[rootID]; !exists {
		return nil, fmt.Errorf("document with ID %q not found in graph", rootID)
	}

	result := &GraphResult{
		RootID:    rootID,
		MaxDepth:  maxDepth,
		Nodes:     make([]GraphNode, 0),
		TotalDocs: 0,
	}

	visited := make(map[string]bool)
	queue := []GraphNode{{Document: g.documents[rootID], Distance: 0}}
	visited[rootID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		result.Nodes = append(result.Nodes, current)
		result.TotalDocs++

		// If we reached max depth, don't add children
		if current.Distance >= maxDepth {
			continue
		}

		// Add neighbors
		neighbors := g.adjacency[current.Document.ID]
		for _, neighborID := range neighbors {
			if !visited[neighborID] {
				visited[neighborID] = true
				queue = append(queue, GraphNode{
					Document: g.documents[neighborID],
					Distance: current.Distance + 1,
				})
			}
		}
	}

	return result, nil
}
