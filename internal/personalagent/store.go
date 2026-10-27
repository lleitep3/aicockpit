package personalagent

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/lleitep3/aicockpit/internal/config"
	"github.com/lleitep3/aicockpit/internal/project"
)

type Store struct {
	directory   string
	projectsDir string
	mu          sync.RWMutex
}

func NewStore(directory string) (*Store, error) {
	return NewStoreWithProjectsDir(directory, filepath.Join(config.GetCockpitDir(), "workspace", "projects"))
}

func NewStoreWithProjectsDir(directory, projectsDir string) (*Store, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create conversation directory: %w", err)
	}
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create projects directory: %w", err)
	}
	store := &Store{directory: directory, projectsDir: projectsDir}
	_ = store.migrateLegacyJSONProjects()
	return store, nil
}

func NewID(prefix string) (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate identifier: %w", err)
	}
	return prefix + hex.EncodeToString(value[:]), nil
}

func (s *Store) Save(conversation Conversation) error {
	if !validConversationID(conversation.ID) {
		return fmt.Errorf("invalid conversation id")
	}
	if conversation.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported conversation schema version %d", conversation.SchemaVersion)
	}
	data, err := json.MarshalIndent(conversation, "", "  ")
	if err != nil {
		return fmt.Errorf("encode conversation: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	temporary, err := os.CreateTemp(s.directory, ".conversation-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary conversation file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict temporary conversation permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write conversation: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync conversation: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close conversation: %w", err)
	}
	if err := os.Rename(temporaryPath, s.filePath(conversation.ID)); err != nil {
		return fmt.Errorf("persist conversation: %w", err)
	}
	return nil
}

func (s *Store) Load(id string) (Conversation, error) {
	if !validConversationID(id) {
		return Conversation{}, fmt.Errorf("invalid conversation id")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.filePath(id))
	if err != nil {
		return Conversation{}, fmt.Errorf("read conversation: %w", err)
	}
	var conversation Conversation
	if err := json.Unmarshal(data, &conversation); err != nil {
		return Conversation{}, fmt.Errorf("decode conversation: %w", err)
	}
	if conversation.ID != id {
		return Conversation{}, fmt.Errorf("conversation identity is invalid")
	}
	if err := migrateConversation(&conversation); err != nil {
		return Conversation{}, err
	}
	return conversation, nil
}

func (s *Store) List() ([]Conversation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	conversations := make([]Conversation, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !validConversationID(id) {
			continue
		}
		data, err := os.ReadFile(s.filePath(id))
		if err != nil {
			return nil, fmt.Errorf("read conversation %s: %w", id, err)
		}
		var conversation Conversation
		if err := json.Unmarshal(data, &conversation); err != nil {
			return nil, fmt.Errorf("decode conversation %s: %w", id, err)
		}
		if conversation.ID != id {
			return nil, fmt.Errorf("conversation %s has invalid identity", id)
		}
		if err := migrateConversation(&conversation); err != nil {
			return nil, fmt.Errorf("migrate conversation %s: %w", id, err)
		}
		conversations = append(conversations, conversation)
	}
	return conversations, nil
}

func (s *Store) ListConversationSummaries() ([]ConversationSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("list conversation summaries: %w", err)
	}
	summaries := make([]ConversationSummary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !validConversationID(id) {
			continue
		}
		data, err := os.ReadFile(s.filePath(id))
		if err != nil {
			return nil, fmt.Errorf("read conversation %s summary: %w", id, err)
		}
		var record struct {
			ID                   string    `json:"id"`
			AssistantMode        bool      `json:"assistant_mode"`
			Archived             bool      `json:"archived"`
			ProjectID            string    `json:"project_id"`
			ParentConversationID string    `json:"parent_conversation_id"`
			RootConversationID   string    `json:"root_conversation_id"`
			Title                string    `json:"title"`
			UpdatedAt            time.Time `json:"updated_at"`
			Turns                []struct {
				Status string `json:"status"`
			} `json:"turns"`
		}
		if err := json.Unmarshal(data, &record); err != nil {
			return nil, fmt.Errorf("decode conversation %s summary: %w", id, err)
		}
		if record.ID != id {
			return nil, fmt.Errorf("conversation %s has invalid identity", id)
		}
		parentID := record.ParentConversationID
		rootID := record.RootConversationID
		if parentID != "" && rootID == "" {
			rootID = parentID
		} else if rootID != "" && parentID == "" {
			parentID = rootID
		}
		summary := ConversationSummary{
			ID:                   record.ID,
			AssistantMode:        record.AssistantMode,
			Archived:             record.Archived,
			ProjectID:            record.ProjectID,
			ParentConversationID: parentID,
			RootConversationID:   rootID,
			Title:                record.Title,
			UpdatedAt:            record.UpdatedAt,
		}
		if len(record.Turns) > 0 {
			summary.LatestTurnStatus = record.Turns[len(record.Turns)-1].Status
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (s *Store) LoadRouting() (RoutingConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, err := os.ReadFile(filepath.Join(s.directory, "routing.config"))
	if err != nil {
		if os.IsNotExist(err) {
			return RoutingConfig{}, nil
		}
		return RoutingConfig{}, fmt.Errorf("read routing configuration: %w", err)
	}
	var config RoutingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return RoutingConfig{}, fmt.Errorf("decode routing configuration: %w", err)
	}
	return config, nil
}

func (s *Store) SaveRouting(config RoutingConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("encode routing configuration: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	temporary, err := os.CreateTemp(s.directory, ".routing-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary routing file: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return fmt.Errorf("restrict routing configuration permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write routing configuration: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return fmt.Errorf("sync routing configuration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close routing configuration: %w", err)
	}
	if err := os.Rename(temporaryPath, filepath.Join(s.directory, "routing.config")); err != nil {
		return fmt.Errorf("persist routing configuration: %w", err)
	}
	return nil
}

func (s *Store) filePath(id string) string {
	return filepath.Join(s.directory, id+".json")
}

func (s *Store) projectFilePath(id string) string {
	return filepath.Join(s.projectsDir, id+".md")
}

func (s *Store) migrateLegacyJSONProjects() error {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(entry.Name(), ".json")
		if !strings.HasPrefix(id, "proj_") {
			continue
		}
		jsonPath := filepath.Join(s.directory, entry.Name())
		data, err := os.ReadFile(jsonPath)
		if err != nil {
			continue
		}
		var legacy Project
		if err := json.Unmarshal(data, &legacy); err != nil || legacy.ID == "" {
			continue
		}
		if err := s.SaveProject(legacy); err == nil {
			_ = os.Remove(jsonPath)
		}
	}
	return nil
}

func (s *Store) SaveProject(proj Project) error {
	if !validProjectID(proj.ID) {
		return fmt.Errorf("invalid project id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	mgr := project.NewManager(s.projectsDir)
	existing, err := mgr.GetProject(proj.ID)
	now := time.Now().UTC()
	if err != nil {
		startDate := proj.CreatedAt
		if startDate.IsZero() {
			startDate = now
		}
		existing = &project.Project{
			ID:   proj.ID,
			Path: s.projectFilePath(proj.ID),
			Metadata: project.Metadata{
				Slug:           proj.ID,
				BoardColumns:   []string{"todo", "inProgress", "test", "done"},
				Tasks:          []project.Task{},
				Repositories:   []string{},
				Workspaces:     []string{},
				KnowledgeBases: []string{},
				Links:          []project.Link{},
				Tags:           []string{},
				StartDate:      startDate,
			},
			Content: "## Tracking Log\n\n",
		}
	}

	existing.Metadata.Title = proj.Name
	existing.Metadata.Slug = proj.ID
	existing.Metadata.Description = proj.Description
	existing.Metadata.Instructions = proj.Instructions
	existing.Metadata.ProviderID = proj.ProviderID
	existing.Metadata.ModelID = proj.ModelID
	existing.Metadata.ReasoningEffort = proj.ReasoningEffort
	existing.Metadata.ApprovalPolicy = proj.ApprovalPolicy
	existing.Metadata.RoutingMode = proj.RoutingMode
	if proj.Repositories != nil {
		existing.Metadata.Repositories = proj.Repositories
	}
	if proj.KnowledgeBases != nil {
		existing.Metadata.KnowledgeBases = proj.KnowledgeBases
	}
	if proj.Tags != nil {
		existing.Metadata.Tags = proj.Tags
	}
	workspaces := make([]string, 0, len(proj.Workspaces)+1)
	if proj.Workspace != "" {
		workspaces = append(workspaces, proj.Workspace)
	}
	for _, ws := range proj.Workspaces {
		if ws != "" && ws != proj.Workspace {
			workspaces = append(workspaces, ws)
		}
	}
	existing.Metadata.Workspaces = workspaces

	return mgr.SaveProject(existing)
}

func (s *Store) LoadProject(id string) (Project, error) {
	if !validProjectID(id) {
		return Project{}, fmt.Errorf("invalid project id")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	mgr := project.NewManager(s.projectsDir)
	loaded, err := mgr.GetProject(id)
	if err != nil {
		return Project{}, fmt.Errorf("read project: %w", err)
	}
	var updatedAt time.Time
	if info, statErr := os.Stat(loaded.Path); statErr == nil {
		updatedAt = info.ModTime().UTC()
	}
	return convertCockpitProject(loaded, updatedAt), nil
}

func (s *Store) ListProjects() ([]Project, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mgr := project.NewManager(s.projectsDir)
	list, err := mgr.ListProjects()
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	projects := make([]Project, 0, len(list))
	for _, item := range list {
		var updatedAt time.Time
		if info, statErr := os.Stat(item.Path); statErr == nil {
			updatedAt = info.ModTime().UTC()
		}
		projects = append(projects, convertCockpitProject(item, updatedAt))
	}
	return projects, nil
}

func convertCockpitProject(item *project.Project, updatedAt time.Time) Project {
	primaryWorkspace := ""
	for _, ws := range item.Metadata.Workspaces {
		if strings.TrimSpace(ws) != "" {
			primaryWorkspace = strings.TrimSpace(ws)
			break
		}
	}
	createdAt := item.Metadata.StartDate
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	name := strings.TrimSpace(item.Metadata.Title)
	if name == "" {
		name = item.ID
	}
	return Project{
		ID:              item.ID,
		Slug:            item.ID,
		Name:            name,
		Description:     item.Metadata.Description,
		Workspace:       primaryWorkspace,
		Workspaces:      item.Metadata.Workspaces,
		Repositories:    item.Metadata.Repositories,
		KnowledgeBases:  item.Metadata.KnowledgeBases,
		Tags:            item.Metadata.Tags,
		TaskCount:       len(item.Metadata.Tasks),
		Instructions:    item.Metadata.Instructions,
		ProviderID:      item.Metadata.ProviderID,
		ModelID:         item.Metadata.ModelID,
		ReasoningEffort: item.Metadata.ReasoningEffort,
		ApprovalPolicy:  item.Metadata.ApprovalPolicy,
		RoutingMode:     item.Metadata.RoutingMode,
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	}
}

func (s *Store) DeleteProject(id string) error {
	if !validProjectID(id) {
		return fmt.Errorf("invalid project id")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.Remove(s.projectFilePath(id)); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

func (s *Store) NextProjectSlug(name string) string {
	base := slugifyProjectName(name)
	if base == "" {
		base = "novo-projeto"
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	candidate := base
	for index := 2; index < 1000; index++ {
		if _, err := os.Stat(s.projectFilePath(candidate)); os.IsNotExist(err) {
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", base, index)
	}
	id, err := NewID("proj_")
	if err != nil {
		return base
	}
	return id
}

func slugifyProjectName(name string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			builder.WriteRune(r)
			lastDash = false
		case r == 'á' || r == 'à' || r == 'â' || r == 'ã':
			builder.WriteByte('a')
			lastDash = false
		case r == 'é' || r == 'ê':
			builder.WriteByte('e')
			lastDash = false
		case r == 'í':
			builder.WriteByte('i')
			lastDash = false
		case r == 'ó' || r == 'ô' || r == 'õ':
			builder.WriteByte('o')
			lastDash = false
		case r == 'ú' || r == 'ü':
			builder.WriteByte('u')
			lastDash = false
		case r == 'ç':
			builder.WriteByte('c')
			lastDash = false
		case unicode.IsSpace(r) || r == '-' || r == '_' || r == '.' || r == '/':
			if builder.Len() > 0 && !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
		if builder.Len() >= 64 {
			break
		}
	}
	return strings.Trim(builder.String(), "-")
}

func validConversationID(id string) bool {
	if !strings.HasPrefix(id, "conv_") || len(id) != len("conv_")+32 {
		return false
	}
	_, err := hex.DecodeString(strings.TrimPrefix(id, "conv_"))
	return err == nil
}

func validProjectID(id string) bool {
	if id == "" || len(id) > 80 {
		return false
	}
	for i, r := range id {
		valid := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !valid || (i == 0 && (r == '-' || r == '_')) {
			return false
		}
	}
	return true
}
