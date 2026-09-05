package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// Manager handles project operations
type Manager struct {
	ProjectsDir string
}

// NewManager creates a new Project Manager
func NewManager(projectsDir string) *Manager {
	return &Manager{
		ProjectsDir: projectsDir,
	}
}

// CreateProject creates a new project file
func (m *Manager) CreateProject(slug, title, description string) (*Project, error) {
	if err := os.MkdirAll(m.ProjectsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create projects directory: %w", err)
	}

	path := filepath.Join(m.ProjectsDir, slug+".md")
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("project with slug %s already exists", slug)
	}

	proj := &Project{
		ID:   slug,
		Path: path,
		Metadata: Metadata{
			Title:          title,
			Slug:           slug,
			Description:    description,
			BoardColumns:   []string{"todo", "inProgress", "test", "done"},
			Tasks:          []Task{},
			Repositories:   []string{},
			Workspaces:     []string{},
			KnowledgeBases: []string{},
			Links:          []Link{},
			Tags:           []string{},
			StartDate:      time.Now(),
		},
		Content: "## Tracking Log\n\n",
	}

	if err := m.SaveProject(proj); err != nil {
		return nil, err
	}

	return proj, nil
}

// GetProject loads a project by slug
func (m *Manager) GetProject(slug string) (*Project, error) {
	path := filepath.Join(m.ProjectsDir, slug+".md")
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project not found: %s", slug)
		}
		return nil, fmt.Errorf("failed to read project file: %w", err)
	}

	return ParseProject(slug, path, string(content))
}

// SaveProject serializes and saves a project
func (m *Manager) SaveProject(proj *Project) error {
	content, err := SerializeProject(proj)
	if err != nil {
		return err
	}
	return os.WriteFile(proj.Path, []byte(content), 0644)
}

// ListProjects returns all projects
func (m *Manager) ListProjects() ([]*Project, error) {
	var projects []*Project

	entries, err := os.ReadDir(m.ProjectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return projects, nil
		}
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}

		id := GenerateProjectID(entry.Name())
		path := filepath.Join(m.ProjectsDir, entry.Name())

		content, err := os.ReadFile(path)
		if err != nil {
			continue // Skip unreadable files
		}

		proj, err := ParseProject(id, path, string(content))
		if err == nil {
			projects = append(projects, proj)
		}
	}

	return projects, nil
}

// AddTask adds a task to a project
func (m *Manager) AddTask(slug, title string) error {
	proj, err := m.GetProject(slug)
	if err != nil {
		return err
	}

	// Generate unique task ID using nanosecond precision
	taskID := fmt.Sprintf("TASK-%d", time.Now().UnixNano())
	task := Task{
		ID:        taskID,
		Title:     title,
		Status:    "todo",
		CreatedAt: time.Now(),
	}

	proj.Metadata.Tasks = append(proj.Metadata.Tasks, task)
	return m.SaveProject(proj)
}

// MoveTask changes a task's column
func (m *Manager) MoveTask(slug, taskID, column string) error {
	proj, err := m.GetProject(slug)
	if err != nil {
		return err
	}

	// Validate column exists
	validCol := false
	for _, col := range proj.Metadata.BoardColumns {
		if col == column {
			validCol = true
			break
		}
	}
	if !validCol {
		return fmt.Errorf("column %s does not exist in project board", column)
	}

	taskFound := false
	for i, task := range proj.Metadata.Tasks {
		if task.ID == taskID {
			proj.Metadata.Tasks[i].Status = column
			taskFound = true
			break
		}
	}

	if !taskFound {
		return fmt.Errorf("task %s not found in project %s", taskID, slug)
	}

	return m.SaveProject(proj)
}

// ReorderTask changes a task's position in the list
func (m *Manager) ReorderTask(slug, taskID string, newIndex int) error {
	proj, err := m.GetProject(slug)
	if err != nil {
		return err
	}

	tasks := proj.Metadata.Tasks
	var taskToMove Task
	oldIndex := -1

	for i, t := range tasks {
		if t.ID == taskID {
			taskToMove = t
			oldIndex = i
			break
		}
	}

	if oldIndex == -1 {
		return fmt.Errorf("task %s not found", taskID)
	}

	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(tasks) {
		newIndex = len(tasks) - 1
	}

	// Remove from old position
	tasks = append(tasks[:oldIndex], tasks[oldIndex+1:]...)

	// Insert at new position
	tasks = append(tasks[:newIndex], append([]Task{taskToMove}, tasks[newIndex:]...)...)

	proj.Metadata.Tasks = tasks
	return m.SaveProject(proj)
}

// DeleteTask removes a task from a project and optionally deletes its GitHub issue
func (m *Manager) DeleteTask(slug, taskID string, deleteGitHubIssue bool) error {
	proj, err := m.GetProject(slug)
	if err != nil {
		return err
	}

	// Find task and get issue info before deletion
	var taskToDelete *Task
	taskIdx := -1
	for i, t := range proj.Metadata.Tasks {
		if t.ID == taskID {
			taskToDelete = &proj.Metadata.Tasks[i]
			taskIdx = i
			break
		}
	}

	if taskIdx == -1 {
		return fmt.Errorf("task %s not found in project %s", taskID, slug)
	}

	// Delete GitHub issue if requested and issue exists
	if deleteGitHubIssue && taskToDelete.IssueNumber > 0 {
		if err := m.deleteGitHubIssue(taskToDelete); err != nil {
			return fmt.Errorf("failed to delete GitHub issue: %w", err)
		}
	}

	// Remove task from project
	proj.Metadata.Tasks = append(proj.Metadata.Tasks[:taskIdx], proj.Metadata.Tasks[taskIdx+1:]...)

	// Log the deletion
	dateStr := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("- **%s**: Task deletada: %s (ID: %s)\n", dateStr, taskToDelete.Title, taskID)
	proj.Content = strings.TrimRight(proj.Content, "\n") + "\n" + entry

	return m.SaveProject(proj)
}

// deleteGitHubIssue deletes a GitHub issue
func (m *Manager) deleteGitHubIssue(task *Task) error {
	if task.Repository == "" || task.IssueNumber == 0 {
		return fmt.Errorf("task does not have GitHub issue information")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return fmt.Errorf("GITHUB_TOKEN not found in environment")
	}

	repoURL := task.Repository
	repoName := repoURL
	if strings.Contains(repoURL, "github.com/") {
		parts := strings.Split(repoURL, "github.com/")
		repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	ownerRepo := strings.Split(repoName, "/")
	if len(ownerRepo) != 2 {
		return fmt.Errorf("invalid repository format: %s. Expected owner/repo", repoName)
	}
	owner, repo := ownerRepo[0], ownerRepo[1]

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Close the issue instead of deleting (GitHub doesn't allow issue deletion)
	issueReq := &github.IssueRequest{
		State: github.Ptr("closed"),
	}

	_, _, err := client.Issues.Edit(ctx, owner, repo, task.IssueNumber, issueReq)
	if err != nil {
		return fmt.Errorf("failed to close issue #%d: %w", task.IssueNumber, err)
	}

	return nil
}

// AddTracking appends a tracking log entry
func (m *Manager) AddTracking(slug, message string) error {
	proj, err := m.GetProject(slug)
	if err != nil {
		return err
	}

	dateStr := time.Now().Format("2006-01-02 15:04")
	entry := fmt.Sprintf("- **%s**: %s\n", dateStr, message)

	proj.Content = strings.TrimRight(proj.Content, "\n") + "\n" + entry
	return m.SaveProject(proj)
}

// SyncGitHubIssue syncs a task with a GitHub issue
func (m *Manager) SyncGitHubIssue(slug, taskID string) (*Task, error) {
	proj, err := m.GetProject(slug)
	if err != nil {
		return nil, err
	}

	var task *Task
	taskIdx := -1
	for i, t := range proj.Metadata.Tasks {
		if t.ID == taskID {
			task = &proj.Metadata.Tasks[i]
			taskIdx = i
			break
		}
	}

	if task == nil {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	repoURL := task.Repository
	if repoURL == "" {
		return nil, fmt.Errorf("repository not defined for this task")
	}

	repoName := repoURL
	if strings.Contains(repoURL, "github.com/") {
		parts := strings.Split(repoURL, "github.com/")
		repoName = strings.TrimSuffix(parts[len(parts)-1], ".git")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN not found in environment")
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	ownerRepo := strings.Split(repoName, "/")
	if len(ownerRepo) != 2 {
		return nil, fmt.Errorf("invalid repository format: %s. Expected owner/repo", repoName)
	}
	owner, repo := ownerRepo[0], ownerRepo[1]

	if task.IssueNumber > 0 {
		// Update existing issue
		issueReq := &github.IssueRequest{}
		needsUpdate := false

		// fetch issue first to compare
		issue, _, err := client.Issues.Get(ctx, owner, repo, task.IssueNumber)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch issue: %w", err)
		}

		if task.Title != "" && task.Title != issue.GetTitle() {
			issueReq.Title = github.Ptr(task.Title)
			needsUpdate = true
		}
		if task.Description != "" && task.Description != issue.GetBody() {
			issueReq.Body = github.Ptr(task.Description)
			needsUpdate = true
		}

		stateMap := map[string]string{"open": "open", "closed": "closed"}
		if state, ok := stateMap[task.State]; ok && state != issue.GetState() {
			issueReq.State = github.Ptr(state)
			needsUpdate = true
		}

		if needsUpdate {
			issue, _, err = client.Issues.Edit(ctx, owner, repo, task.IssueNumber, issueReq)
			if err != nil {
				return nil, fmt.Errorf("failed to update issue: %w", err)
			}
		}

		// Sync back
		task.Title = issue.GetTitle()
		task.Description = issue.GetBody()
		task.State = issue.GetState()
	} else {
		// Create new issue
		issueReq := &github.IssueRequest{
			Title: github.Ptr(task.Title),
		}
		if task.Description != "" {
			issueReq.Body = github.Ptr(task.Description)
		}

		issue, _, err := client.Issues.Create(ctx, owner, repo, issueReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create issue: %w", err)
		}

		task.IssueNumber = issue.GetNumber()
		task.IssueURL = issue.GetHTMLURL()
	}

	proj.Metadata.Tasks[taskIdx] = *task

	if err := m.SaveProject(proj); err != nil {
		return nil, err
	}

	return task, nil
}
