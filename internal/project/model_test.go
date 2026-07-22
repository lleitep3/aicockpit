package project

import (
	"strings"
	"testing"
)

func TestParseProject(t *testing.T) {
	rawContent := `---
title: Test Project
slug: test-project
description: A test project description
board_columns:
    - todo
    - done
tasks:
    - id: task-1
      title: First task
      status: todo
      created_at: 2026-07-22T00:00:00Z
repositories:
    - https://github.com/user/repo
workspaces:
    - /home/user/workspace
knowledge_bases:
    - kb-1
links:
    - title: Design
      url: https://figma.com
tags:
    - test
start_date: 2026-07-22T00:00:00Z
---
## Tracking Log
- 2026-07-22: Started project`

	proj, err := ParseProject("test-project", "/path/to/test-project.md", rawContent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if proj.ID != "test-project" {
		t.Errorf("expected ID 'test-project', got %s", proj.ID)
	}
	if proj.Metadata.Title != "Test Project" {
		t.Errorf("expected Title 'Test Project', got %s", proj.Metadata.Title)
	}
	if len(proj.Metadata.BoardColumns) != 2 {
		t.Errorf("expected 2 columns, got %d", len(proj.Metadata.BoardColumns))
	}
	if len(proj.Metadata.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(proj.Metadata.Tasks))
	}
	if proj.Metadata.Tasks[0].ID != "task-1" {
		t.Errorf("expected task ID 'task-1', got %s", proj.Metadata.Tasks[0].ID)
	}
	if !strings.Contains(proj.Content, "## Tracking Log") {
		t.Errorf("expected content to contain '## Tracking Log', got %s", proj.Content)
	}
}

func TestParseProjectDefaults(t *testing.T) {
	rawContent := `---
title: Basic Project
slug: basic-project
---`
	proj, err := ParseProject("basic-project", "/path/to/basic-project.md", rawContent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(proj.Metadata.BoardColumns) != 4 { // Default columns
		t.Errorf("expected 4 default columns, got %d", len(proj.Metadata.BoardColumns))
	}
	if proj.Metadata.StartDate.IsZero() {
		t.Errorf("expected start date to be set")
	}
}

func TestParseProjectNoMetadata(t *testing.T) {
	rawContent := `Just some text`
	proj, err := ParseProject("no-meta", "/path/to/no-meta.md", rawContent)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if proj.Content != rawContent {
		t.Errorf("expected content to match, got %s", proj.Content)
	}
	if len(proj.Metadata.BoardColumns) != 4 {
		t.Errorf("expected default columns, got %d", len(proj.Metadata.BoardColumns))
	}
}

func TestSerializeProject(t *testing.T) {
	proj := &Project{
		ID: "test",
		Metadata: Metadata{
			Title:        "Test",
			Slug:         "test",
			BoardColumns: []string{"todo"},
		},
		Content: "## Tracking Log",
	}

	serialized, err := SerializeProject(proj)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.HasPrefix(serialized, "---") {
		t.Errorf("expected serialized to start with ---")
	}
	if !strings.Contains(serialized, "title: Test") {
		t.Errorf("expected serialized to contain 'title: Test'")
	}
	if !strings.Contains(serialized, "## Tracking Log") {
		t.Errorf("expected serialized to contain '## Tracking Log'")
	}
}

func TestParseProjectErrors(t *testing.T) {
	_, err := ParseProject("test", "test.md", "---\ntitle: [invalid yaml\n---")
	if err == nil {
		t.Errorf("expected error for invalid yaml")
	}

	_, err = ParseProject("test", "test.md", "---\ntitle: unclosed")
	if err == nil {
		t.Errorf("expected error for unclosed metadata")
	}
}

func TestGenerateProjectID(t *testing.T) {
	id := GenerateProjectID("my-project.md")
	if id != "my-project" {
		t.Errorf("expected 'my-project', got %s", id)
	}
}
