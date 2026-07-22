package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestManager(t *testing.T) (*Manager, string) {
	tempDir, err := os.MkdirTemp("", "cockpit-project-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	mgr := NewManager(tempDir)
	return mgr, tempDir
}

func TestCreateProject(t *testing.T) {
	mgr, cleanup := setupTestManager(t)
	defer os.RemoveAll(cleanup)

	proj, err := mgr.CreateProject("test-project", "Test Project", "Description")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if proj.ID != "test-project" {
		t.Errorf("expected ID 'test-project', got %s", proj.ID)
	}

	// Should fail if creating again
	_, err = mgr.CreateProject("test-project", "Test Project 2", "Desc")
	if err == nil {
		t.Errorf("expected error when creating existing project, got nil")
	}
}

func TestGetAndListProjects(t *testing.T) {
	mgr, cleanup := setupTestManager(t)
	defer os.RemoveAll(cleanup)

	mgr.CreateProject("p1", "Project 1", "Desc 1")
	mgr.CreateProject("p2", "Project 2", "Desc 2")

	// Create a non-md file to ensure it's ignored
	os.WriteFile(filepath.Join(cleanup, "ignore.txt"), []byte("ignore"), 0644)

	projects, err := mgr.ListProjects()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(projects))
	}

	proj, err := mgr.GetProject("p1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if proj.Metadata.Title != "Project 1" {
		t.Errorf("expected 'Project 1', got %s", proj.Metadata.Title)
	}

	_, err = mgr.GetProject("not-exist")
	if err == nil {
		t.Errorf("expected error getting non-existent project")
	}
}

func TestTasks(t *testing.T) {
	mgr, cleanup := setupTestManager(t)
	defer os.RemoveAll(cleanup)

	mgr.CreateProject("tasks-proj", "Tasks Project", "Desc")

	err := mgr.AddTask("tasks-proj", "My first task")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	proj, _ := mgr.GetProject("tasks-proj")
	if len(proj.Metadata.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(proj.Metadata.Tasks))
	}
	taskID := proj.Metadata.Tasks[0].ID
	if proj.Metadata.Tasks[0].Status != "todo" {
		t.Errorf("expected task status 'todo', got %s", proj.Metadata.Tasks[0].Status)
	}

	err = mgr.MoveTask("tasks-proj", taskID, "done")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	proj, _ = mgr.GetProject("tasks-proj")
	if proj.Metadata.Tasks[0].Status != "done" {
		t.Errorf("expected task status 'done', got %s", proj.Metadata.Tasks[0].Status)
	}

	err = mgr.MoveTask("tasks-proj", taskID, "invalid-col")
	if err == nil {
		t.Errorf("expected error for invalid column")
	}

	err = mgr.MoveTask("tasks-proj", "non-existent-task", "done")
	if err == nil {
		t.Errorf("expected error for non-existent task")
	}
}

func TestTaskErrors(t *testing.T) {
	mgr, cleanup := setupTestManager(t)
	defer os.RemoveAll(cleanup)

	err := mgr.AddTask("non-existent", "title")
	if err == nil {
		t.Errorf("expected error when adding task to non-existent project")
	}

	err = mgr.MoveTask("non-existent", "task-1", "done")
	if err == nil {
		t.Errorf("expected error when moving task in non-existent project")
	}

	err = mgr.AddTracking("non-existent", "msg")
	if err == nil {
		t.Errorf("expected error when tracking in non-existent project")
	}
}

func TestTracking(t *testing.T) {
	mgr, cleanup := setupTestManager(t)
	defer os.RemoveAll(cleanup)

	mgr.CreateProject("track-proj", "Track", "Desc")

	err := mgr.AddTracking("track-proj", "This is an update.")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	proj, _ := mgr.GetProject("track-proj")
	if !strings.Contains(proj.Content, "This is an update.") {
		t.Errorf("expected content to contain tracking message")
	}
}
