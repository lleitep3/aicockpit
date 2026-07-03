package scheduler

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Store defines the persistence interface for scheduled jobs.
type Store interface {
	Load() ([]Job, error)
	Save(jobs []Job) error
	Add(job Job) ([]Job, error)
	Remove(id string) ([]Job, error)
	Get(id string) (Job, error)
	Update(job Job) ([]Job, error)
}

// JSONStore persists jobs in a JSON file.
type JSONStore struct {
	mu       sync.RWMutex
	path     string
	homeFunc func() string
}

// NewJSONStore creates a new JSON-backed store.
func NewJSONStore(homeFunc func() string) *JSONStore {
	if homeFunc == nil {
		homeFunc = defaultHomeDir
	}
	return &JSONStore{
		path:     filepath.Join(homeFunc(), ".cockpit", "scheduler", "jobs.json"),
		homeFunc: homeFunc,
	}
}

// NewJSONStoreWithPath creates a store with an explicit path.
func NewJSONStoreWithPath(path string) *JSONStore {
	return &JSONStore{path: path}
}

func defaultHomeDir() string {
	return os.Getenv("HOME")
}

// Load reads jobs from the JSON file.
func (s *JSONStore) Load() ([]Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, err := os.Stat(s.path); os.IsNotExist(err) {
		return []Job{}, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to read jobs file: %w", err)
	}

	var jobs []Job
	if len(data) == 0 {
		return []Job{}, nil
	}

	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, fmt.Errorf("failed to parse jobs file: %w", err)
	}

	return jobs, nil
}

// Save writes jobs to the JSON file.
func (s *JSONStore) Save(jobs []Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("failed to create scheduler directory: %w", err)
	}

	data, err := json.MarshalIndent(jobs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal jobs: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write jobs file: %w", err)
	}

	return nil
}

// Add appends a job to the store.
func (s *JSONStore) Add(job Job) ([]Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}

	for _, j := range jobs {
		if j.ID == job.ID {
			return nil, fmt.Errorf("job with id %s already exists", job.ID)
		}
	}

	jobs = append(jobs, job)
	sortJobs(jobs)

	if err := s.Save(jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Remove deletes a job by id.
func (s *JSONStore) Remove(id string) ([]Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}

	found := false
	filtered := make([]Job, 0, len(jobs))
	for _, j := range jobs {
		if j.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, j)
	}

	if !found {
		return nil, fmt.Errorf("job %s not found: %w", id, errors.New("not found"))
	}

	if err := s.Save(filtered); err != nil {
		return nil, err
	}

	return filtered, nil
}

// Get retrieves a single job by id.
func (s *JSONStore) Get(id string) (Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return Job{}, err
	}

	for _, j := range jobs {
		if j.ID == id {
			return j, nil
		}
	}

	return Job{}, fmt.Errorf("job %s not found: %w", id, errors.New("not found"))
}

// Update modifies an existing job.
func (s *JSONStore) Update(job Job) ([]Job, error) {
	jobs, err := s.Load()
	if err != nil {
		return nil, err
	}

	found := false
	for i, j := range jobs {
		if j.ID == job.ID {
			jobs[i] = job
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("job %s not found: %w", job.ID, errors.New("not found"))
	}

	sortJobs(jobs)
	if err := s.Save(jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

// Now returns the current time. Exposed for testability.
var Now = func() time.Time {
	return time.Now()
}

func sortJobs(jobs []Job) {
	sort.Slice(jobs, func(i, j int) bool {
		return jobs[i].CreatedAt.Before(jobs[j].CreatedAt)
	})
}
