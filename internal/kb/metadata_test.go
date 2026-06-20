package kb

import (
	"strings"
	"testing"
	"time"
)

func TestParseDocument(t *testing.T) {
	tests := []struct {
		name        string
		id          string
		path        string
		rawContent  string
		wantErr     bool
		wantTitle   string
		wantContent string
	}{
		{
			name: "valid document with metadata",
			id:   "test-doc",
			path: "guides/test.md",
			rawContent: `---
title: "Test Document"
description: "A test document"
tags: ["test", "example"]
author: "Test Author"
version: "1.0"
---
# Content Header

This is the content.`,
			wantErr:     false,
			wantTitle:   "Test Document",
			wantContent: "# Content Header\n\nThis is the content.",
		},
		{
			name: "document without metadata",
			id:   "no-metadata",
			path: "guides/no-meta.md",
			rawContent: `# Just Content

This is content without metadata.`,
			wantErr:     false,
			wantTitle:   "",
			wantContent: "# Just Content\n\nThis is content without metadata.",
		},
		{
			name: "unclosed metadata delimiter",
			id:   "unclosed",
			path: "guides/unclosed.md",
			rawContent: `---
title: "Unclosed"
description: "No closing delimiter"
# Content here`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseDocument(tt.id, tt.path, tt.rawContent)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDocument() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if doc.ID != tt.id {
				t.Errorf("ParseDocument() ID = %s, want %s", doc.ID, tt.id)
			}

			if doc.Metadata.Title != tt.wantTitle {
				t.Errorf("ParseDocument() Title = %s, want %s", doc.Metadata.Title, tt.wantTitle)
			}

			if strings.TrimSpace(doc.Content) != strings.TrimSpace(tt.wantContent) {
				t.Errorf("ParseDocument() Content = %s, want %s", doc.Content, tt.wantContent)
			}
		})
	}
}

func TestExtractMetadata(t *testing.T) {
	tests := []struct {
		name        string
		rawContent  string
		wantErr     bool
		wantTitle   string
		wantTags    []string
		wantContent string
	}{
		{
			name: "complete metadata",
			rawContent: `---
title: "Complete Doc"
description: "Full metadata"
tags: ["tag1", "tag2"]
author: "Author Name"
version: "1.0"
---
Content here`,
			wantErr:     false,
			wantTitle:   "Complete Doc",
			wantTags:    []string{"tag1", "tag2"},
			wantContent: "Content here",
		},
		{
			name: "minimal metadata",
			rawContent: `---
title: "Minimal"
---
Content`,
			wantErr:     false,
			wantTitle:   "Minimal",
			wantContent: "Content",
		},
		{
			name:        "no metadata",
			rawContent:  "Just content",
			wantErr:     false,
			wantTitle:   "",
			wantContent: "Just content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, content, err := extractMetadata(tt.rawContent)

			if (err != nil) != tt.wantErr {
				t.Errorf("extractMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			if metadata.Title != tt.wantTitle {
				t.Errorf("extractMetadata() Title = %s, want %s", metadata.Title, tt.wantTitle)
			}

			if len(metadata.Tags) != len(tt.wantTags) {
				t.Errorf("extractMetadata() Tags length = %d, want %d", len(metadata.Tags), len(tt.wantTags))
			}

			if strings.TrimSpace(content) != strings.TrimSpace(tt.wantContent) {
				t.Errorf("extractMetadata() Content = %s, want %s", content, tt.wantContent)
			}
		})
	}
}

func TestSerializeDocument(t *testing.T) {
	doc := &Document{
		ID:   "test-doc",
		Path: "guides/test.md",
		Metadata: Metadata{
			Title:       "Test Doc",
			Description: "A test",
			Tags:        []string{"test"},
			Author:      "Author",
			Version:     "1.0",
			Created:     time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
			Modified:    time.Date(2026, 6, 20, 0, 0, 0, 0, time.UTC),
		},
		Content: "# Content\n\nTest content",
	}

	serialized, err := SerializeDocument(doc)
	if err != nil {
		t.Fatalf("SerializeDocument() error = %v", err)
	}

	// Verify it contains delimiters
	if !strings.Contains(serialized, MetadataDelimiter) {
		t.Error("SerializeDocument() missing metadata delimiter")
	}

	// Verify it can be parsed back
	parsed, err := ParseDocument(doc.ID, doc.Path, serialized)
	if err != nil {
		t.Fatalf("ParseDocument() error = %v", err)
	}

	if parsed.Metadata.Title != doc.Metadata.Title {
		t.Errorf("Round-trip Title = %s, want %s", parsed.Metadata.Title, doc.Metadata.Title)
	}

	if strings.TrimSpace(parsed.Content) != strings.TrimSpace(doc.Content) {
		t.Errorf("Round-trip Content = %s, want %s", parsed.Content, doc.Content)
	}
}

func TestGenerateDocumentID(t *testing.T) {
	tests := []struct {
		filename string
		wantID   string
	}{
		{"logging-setup.md", "logging-setup"},
		{"test-file.txt", "test-file"},
		{"document.yaml", "document"},
		{"no-extension", "no-extension"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			id := GenerateDocumentID(tt.filename)
			if id != tt.wantID {
				t.Errorf("GenerateDocumentID() = %s, want %s", id, tt.wantID)
			}
		})
	}
}

func TestExtractExcerpt(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		maxLength  int
		wantMaxLen int
		wantText   string
	}{
		{
			name:       "normal content",
			content:    "This is a test content that should be extracted as excerpt.",
			maxLength:  50,
			wantMaxLen: 53,
			wantText:   "This is a test content that should be extracted",
		},
		{
			name:      "with markdown headers",
			content:   "# Header\n\nThis is content after header.",
			maxLength: 100,
			wantText:  "This is content after header.",
		},
		{
			name:      "with list items",
			content:   "- Item 1\n- Item 2\n- Item 3",
			maxLength: 100,
			wantText:  "Item 1 Item 2 Item 3",
		},
		{
			name:      "empty content",
			content:   "",
			maxLength: 100,
			wantText:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excerpt := ExtractExcerpt(tt.content, tt.maxLength)

			if tt.wantMaxLen > 0 && len(excerpt) > tt.wantMaxLen {
				t.Errorf("ExtractExcerpt() length = %d, want <= %d", len(excerpt), tt.wantMaxLen)
			}

			if tt.wantText != "" && !strings.Contains(excerpt, tt.wantText) {
				t.Errorf("ExtractExcerpt() = %s, want to contain %s", excerpt, tt.wantText)
			}
		})
	}
}

func TestParseDocumentWithoutMetadata(t *testing.T) {
	tests := []struct {
		name            string
		id              string
		path            string
		rawContent      string
		wantTitle       string
		wantDescription string
		wantTags        int
	}{
		{
			name: "document with header",
			id:   "test-doc",
			path: "test.md",
			rawContent: `# Test Document

This is a test document with a header.

More content here.`,
			wantTitle:       "Test Document",
			wantDescription: "This is a test document with a header.",
			wantTags:        0,
		},
		{
			name: "document without header",
			id:   "no-header",
			path: "no-header.md",
			rawContent: `This is content without a header.

More content here.`,
			wantTitle:       "no-header",
			wantDescription: "This is content without a header.",
			wantTags:        0,
		},
		{
			name: "document with multiple headers",
			id:   "multi-header",
			path: "multi.md",
			rawContent: `# First Header

This is the first paragraph.

## Second Header

This is another section.`,
			wantTitle:       "First Header",
			wantDescription: "This is the first paragraph.",
			wantTags:        0,
		},
		{
			name: "empty document",
			id:   "empty",
			path: "empty.md",
			rawContent: ``,
			wantTitle:       "empty",
			wantDescription: "",
			wantTags:        0,
		},
		{
			name: "document with list items",
			id:   "with-list",
			path: "list.md",
			rawContent: `# List Document

- Item 1
- Item 2

This is a paragraph after the list.`,
			wantTitle:       "List Document",
			wantDescription: "This is a paragraph after the list.",
			wantTags:        0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := ParseDocumentWithoutMetadata(tt.id, tt.path, tt.rawContent)

			if err != nil {
				t.Fatalf("ParseDocumentWithoutMetadata() error = %v", err)
			}

			if doc.ID != tt.id {
				t.Errorf("ID = %s, want %s", doc.ID, tt.id)
			}

			if doc.Metadata.Title != tt.wantTitle {
				t.Errorf("Title = %s, want %s", doc.Metadata.Title, tt.wantTitle)
			}

			if doc.Metadata.Description != tt.wantDescription {
				t.Errorf("Description = %s, want %s", doc.Metadata.Description, tt.wantDescription)
			}

			if len(doc.Metadata.Tags) != tt.wantTags {
				t.Errorf("Tags length = %d, want %d", len(doc.Metadata.Tags), tt.wantTags)
			}

			if doc.Metadata.Created.IsZero() {
				t.Error("Created should not be zero")
			}

			if doc.Metadata.Modified.IsZero() {
				t.Error("Modified should not be zero")
			}
		})
	}
}
