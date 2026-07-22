package project

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

// Task represents a kanban board item
type Task struct {
	ID        string    `yaml:"id"`
	Title     string    `yaml:"title"`
	Status    string    `yaml:"status"`
	CreatedAt time.Time `yaml:"created_at"`
}

// Link represents an external resource link
type Link struct {
	Title string `yaml:"title"`
	URL   string `yaml:"url"`
}

// Metadata holds all the YAML frontmatter structural data
type Metadata struct {
	Title          string    `yaml:"title"`
	Slug           string    `yaml:"slug"`
	Description    string    `yaml:"description"`
	BoardColumns   []string  `yaml:"board_columns"`
	Tasks          []Task    `yaml:"tasks"`
	Repositories   []string  `yaml:"repositories"`
	Workspaces     []string  `yaml:"workspaces"`
	KnowledgeBases []string  `yaml:"knowledge_bases"`
	Links          []Link    `yaml:"links"`
	Tags           []string  `yaml:"tags"`
	StartDate      time.Time `yaml:"start_date"`
}

// Project represents the complete project file context
type Project struct {
	ID       string
	Path     string
	Metadata Metadata
	Content  string
}

// ParseProject parses a Project document from raw Markdown content.
func ParseProject(id, path, rawContent string) (*Project, error) {
	metadata, content, err := extractMetadata(rawContent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project %s: %w", id, err)
	}

	return &Project{
		ID:       id,
		Path:     path,
		Metadata: metadata,
		Content:  content,
	}, nil
}

// extractMetadata extracts metadata and content from raw project document.
func extractMetadata(rawContent string) (Metadata, string, error) {
	lines := strings.Split(rawContent, "\n")

	// Check if first line is delimiter
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != MetadataDelimiter {
		metadata := Metadata{}
		applyDefaults(&metadata)
		return metadata, rawContent, nil
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

	applyDefaults(&metadata)
	return metadata, content, nil
}

func applyDefaults(metadata *Metadata) {
	if metadata.StartDate.IsZero() {
		metadata.StartDate = time.Now()
	}
	if len(metadata.BoardColumns) == 0 {
		metadata.BoardColumns = []string{"todo", "inProgress", "test", "done"}
	}
	if metadata.Tasks == nil {
		metadata.Tasks = []Task{}
	}
	if metadata.Repositories == nil {
		metadata.Repositories = []string{}
	}
	if metadata.Workspaces == nil {
		metadata.Workspaces = []string{}
	}
	if metadata.KnowledgeBases == nil {
		metadata.KnowledgeBases = []string{}
	}
	if metadata.Links == nil {
		metadata.Links = []Link{}
	}
	if metadata.Tags == nil {
		metadata.Tags = []string{}
	}
}

// SerializeProject serializes a project to raw content with metadata header.
func SerializeProject(proj *Project) (string, error) {
	metadataBytes, err := yaml.Marshal(proj.Metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(MetadataDelimiter + "\n")
	sb.Write(metadataBytes)
	sb.WriteString(MetadataDelimiter + "\n")

	// Add tracking header if empty body
	content := strings.TrimSpace(proj.Content)
	if content == "" {
		content = "## Tracking Log\n\n"
	}

	sb.WriteString(content)
	sb.WriteString("\n")

	return sb.String(), nil
}

// GenerateProjectID generates a project ID from filename.
func GenerateProjectID(filename string) string {
	return strings.TrimSuffix(filename, filepath.Ext(filename))
}
