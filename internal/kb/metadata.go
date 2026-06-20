package kb

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// MetadataDelimiter marks the end of metadata section
	MetadataDelimiter = "---"
)

// ParseDocument parses a KB document from raw content.
// Expected format:
// ---
// title: "Title"
// description: "Description"
// tags: ["tag1", "tag2"]
// created: "2026-06-20"
// modified: "2026-06-20"
// related: ["doc-id-1"]
// author: "Author"
// version: "1.0"
// ---
// # Content here
func ParseDocument(id, path, rawContent string) (*Document, error) {
	metadata, content, err := extractMetadata(rawContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse document %s: %w", id, err)
	}

	return &Document{
		ID:       id,
		Path:     path,
		Metadata: metadata,
		Content:  content,
	}, nil
}

// extractMetadata extracts metadata and content from raw document.
func extractMetadata(rawContent string) (Metadata, string, error) {
	lines := strings.Split(rawContent, "\n")

	// Check if first line is delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != MetadataDelimiter {
		return Metadata{}, rawContent, nil
	}

	// Find closing delimiter
	closingIndex := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == MetadataDelimiter {
			closingIndex = i
			break
		}
	}

	if closingIndex == -1 {
		return Metadata{}, rawContent, fmt.Errorf("metadata section not closed with %s", MetadataDelimiter)
	}

	// Extract metadata and content
	metadataStr := strings.Join(lines[1:closingIndex], "\n")
	contentLines := lines[closingIndex+1:]
	content := strings.TrimSpace(strings.Join(contentLines, "\n"))

	// Parse YAML metadata
	var metadata Metadata
	if err := yaml.Unmarshal([]byte(metadataStr), &metadata); err != nil {
		return Metadata{}, "", fmt.Errorf("failed to parse metadata YAML: %w", err)
	}

	// Set defaults for empty fields
	if metadata.Created.IsZero() {
		metadata.Created = time.Now()
	}
	if metadata.Modified.IsZero() {
		metadata.Modified = time.Now()
	}
	if metadata.Tags == nil {
		metadata.Tags = []string{}
	}
	if metadata.Related == nil {
		metadata.Related = []string{}
	}

	return metadata, content, nil
}

// SerializeDocument serializes a document to raw content with metadata header.
func SerializeDocument(doc *Document) (string, error) {
	// Marshal metadata to YAML
	metadataBytes, err := yaml.Marshal(doc.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Build document with metadata header
	var sb strings.Builder
	sb.WriteString(MetadataDelimiter + "\n")
	sb.Write(metadataBytes)
	sb.WriteString(MetadataDelimiter + "\n")
	sb.WriteString(doc.Content)

	return sb.String(), nil
}

// GenerateDocumentID generates a document ID from filename.
// Example: "logging-setup.md" -> "logging-setup"
func GenerateDocumentID(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name
}

// ExtractExcerpt extracts a brief excerpt from document content.
// Returns first N characters or first paragraph, whichever is shorter.
func ExtractExcerpt(content string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 200
	}

	// Remove markdown headers and formatting
	lines := strings.Split(content, "\n")
	var excerpt strings.Builder

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Remove markdown formatting
		cleaned := strings.TrimPrefix(trimmed, "- ")
		cleaned = strings.TrimPrefix(cleaned, "* ")
		cleaned = strings.TrimPrefix(cleaned, "> ")

		excerpt.WriteString(cleaned + " ")

		if excerpt.Len() >= maxLength {
			break
		}
	}

	result := strings.TrimSpace(excerpt.String())
	if len(result) > maxLength {
		result = result[:maxLength] + "..."
	}

	return result
}
