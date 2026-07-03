package scheduler

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func setupStoreTest(t *testing.T) (*JSONStore, func()) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)

	store := NewJSONStore(nil)

	cleanup := func() {
		os.Setenv("HOME", oldHome)
	}

	return store, cleanup
}

func TestNewJSONStore(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	expectedPath := filepath.Join(os.Getenv("HOME"), ".cockpit", "scheduler", "jobs.json")
	if store.path != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, store.path)
	}
}

func TestJSONStoreLoadEmpty(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	jobs, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestJSONStoreAddAndLoad(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	job := Job{
		ID:        GenerateID(),
		Command:   "echo hello",
		Type:      JobTypeCron,
		CronExpr:  "0 9 * * *",
		Status:    JobStatusActive,
		CreatedAt: time.Now(),
	}

	jobs, err := store.Add(job)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 1 {
		t.Errorf("expected 1 loaded job, got %d", len(loaded))
	}
	if loaded[0].Command != job.Command {
		t.Errorf("expected command %s, got %s", job.Command, loaded[0].Command)
	}
}

func TestJSONStoreDuplicateID(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	job := Job{
		ID:        "sched_dup",
		Command:   "echo hello",
		Type:      JobTypeCron,
		CronExpr:  "0 9 * * *",
		Status:    JobStatusActive,
		CreatedAt: time.Now(),
	}

	_, err := store.Add(job)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	_, err = store.Add(job)
	if err == nil {
		t.Fatal("expected duplicate id error")
	}
}

func TestJSONStoreRemove(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	job := Job{
		ID:        "sched_remove",
		Command:   "echo hello",
		Type:      JobTypeCron,
		CronExpr:  "0 9 * * *",
		Status:    JobStatusActive,
		CreatedAt: time.Now(),
	}

	_, err := store.Add(job)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	jobs, err := store.Remove("sched_remove")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs after remove, got %d", len(jobs))
	}

	_, err = store.Remove("nonexistent")
	if err == nil {
		t.Fatal("expected error removing nonexistent job")
	}
}

func TestJSONStoreGet(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	job := Job{
		ID:        "sched_get",
		Command:   "echo hello",
		Type:      JobTypeCron,
		CronExpr:  "0 9 * * *",
		Status:    JobStatusActive,
		CreatedAt: time.Now(),
	}

	_, err := store.Add(job)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, err := store.Get("sched_get")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("expected id %s, got %s", job.ID, got.ID)
	}

	_, err = store.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error getting nonexistent job")
	}
}

func TestJSONStoreUpdate(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	job := Job{
		ID:        "sched_update",
		Command:   "echo hello",
		Type:      JobTypeCron,
		CronExpr:  "0 9 * * *",
		Status:    JobStatusActive,
		CreatedAt: time.Now(),
	}

	_, err := store.Add(job)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	job.Status = JobStatusPaused
	jobs, err := store.Update(job)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Status != JobStatusPaused {
		t.Errorf("expected status paused, got %s", jobs[0].Status)
	}

	_, err = store.Update(Job{ID: "nonexistent"})
	if err == nil {
		t.Fatal("expected error updating nonexistent job")
	}
}

func TestJSONStoreInvalidPath(t *testing.T) {
	path := "/nonexistent_dir/scheduler/jobs.json"
	store := NewJSONStoreWithPath(path)

	job := Job{
		ID:        "sched_invalid",
		Command:   "echo hello",
		Type:      JobTypeCron,
		CronExpr:  "0 9 * * *",
		Status:    JobStatusActive,
		CreatedAt: time.Now(),
	}

	_, err := store.Add(job)
	if err == nil {
		t.Fatal("expected error writing to invalid path")
	}
}

func TestJSONStoreLoadCorrupted(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStoreWithPath(filepath.Join(tmpDir, "jobs.json"))

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, []byte("not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := store.Load()
	if err == nil {
		t.Fatal("expected error for corrupted file")
	}
}

func TestJSONStoreLoadEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	store := NewJSONStoreWithPath(filepath.Join(tmpDir, "jobs.json"))

	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.path, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	jobs, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}
}

func TestJSONStoreSortJobs(t *testing.T) {
	store, cleanup := setupStoreTest(t)
	defer cleanup()

	now := time.Now()
	job1 := Job{ID: "sched_1", Command: "echo 1", Type: JobTypeCron, CronExpr: "0 9 * * *", Status: JobStatusActive, CreatedAt: now.Add(time.Hour)}
	job2 := Job{ID: "sched_2", Command: "echo 2", Type: JobTypeCron, CronExpr: "0 9 * * *", Status: JobStatusActive, CreatedAt: now}

	_, _ = store.Add(job1)
	_, _ = store.Add(job2)

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(loaded))
	}
	if loaded[0].ID != "sched_2" {
		t.Errorf("expected first job to be sched_2, got %s", loaded[0].ID)
	}
}
