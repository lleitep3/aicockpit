package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

	taskID := fmt.Sprintf("TASK-%d", time.Now().Unix())
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
		return fmt.Errorf("task %s not found", taskID)
	}

	return m.SaveProject(proj)
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
