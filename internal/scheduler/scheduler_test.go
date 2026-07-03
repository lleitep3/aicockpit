package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// mockExecutor is an executor that records executions and returns controlled results.
type mockExecutor struct {
	commands []string
}

func (m *mockExecutor) Execute(ctx context.Context, command string) ExecutionResult {
	m.commands = append(m.commands, command)
	return ExecutionResult{Output: "ok", ExitCode: 0}
}

// mockStore is a memory-backed store for testing.
type mockStore struct {
	jobs []Job
}

func (m *mockStore) Load() ([]Job, error) {
	return append([]Job{}, m.jobs...), nil
}

func (m *mockStore) Save(jobs []Job) error {
	m.jobs = append([]Job{}, jobs...)
	return nil
}

func (m *mockStore) Add(job Job) ([]Job, error) {
	for _, j := range m.jobs {
		if j.ID == job.ID {
			return nil, fmt.Errorf("duplicate id")
		}
	}
	m.jobs = append(m.jobs, job)
	return append([]Job{}, m.jobs...), nil
}

func (m *mockStore) Remove(id string) ([]Job, error) {
	filtered := make([]Job, 0, len(m.jobs))
	found := false
	for _, j := range m.jobs {
		if j.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, j)
	}
	if !found {
		return nil, fmt.Errorf("not found")
	}
	m.jobs = filtered
	return append([]Job{}, m.jobs...), nil
}

func (m *mockStore) Get(id string) (Job, error) {
	for _, j := range m.jobs {
		if j.ID == id {
			return j, nil
		}
	}
	return Job{}, fmt.Errorf("not found")
}

func (m *mockStore) Update(job Job) ([]Job, error) {
	found := false
	for i, j := range m.jobs {
		if j.ID == job.ID {
			m.jobs[i] = job
			found = true
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("not found")
	}
	return append([]Job{}, m.jobs...), nil
}

func TestManagerAddJob(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	job, err := m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "test")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}
	if job.ID == "" {
		t.Error("expected job id to be set")
	}
	if job.Type != JobTypeCron {
		t.Errorf("expected type cron, got %s", job.Type)
	}
	if job.Status != JobStatusActive {
		t.Errorf("expected status active, got %s", job.Status)
	}

	_, err = m.AddJob("", "cron", "0 9 * * *", "", 0, "")
	if err == nil {
		t.Error("expected error for empty command")
	}
}

func TestManagerListJobs(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	jobs, err := m.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	_, _ = m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "")

	jobs, err = m.ListJobs()
	if err != nil {
		t.Fatalf("ListJobs failed: %v", err)
	}
	if len(jobs) != 1 {
		t.Errorf("expected 1 job, got %d", len(jobs))
	}
}

func TestManagerRemoveJob(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	job, _ := m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "")

	err := m.RemoveJob(job.ID)
	if err != nil {
		t.Fatalf("RemoveJob failed: %v", err)
	}

	jobs, _ := m.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("expected 0 jobs, got %d", len(jobs))
	}

	err = m.RemoveJob("nonexistent")
	if err == nil {
		t.Error("expected error removing nonexistent job")
	}
}

func TestManagerRunDueJobs(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	_, err := m.AddJob("echo hello", "repeat", "", "1h", 0, "")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	// Set NextRun to a time in the past
	jobs, _ := m.ListJobs()
	jobs[0].NextRun = now.Add(-30 * time.Minute)
	jobs[0].IntervalSec = 3600
	_, _ = store.Update(jobs[0])

	err = m.RunDueJobs(now)
	if err != nil {
		t.Fatalf("RunDueJobs failed: %v", err)
	}

	if len(exec.commands) != 1 {
		t.Errorf("expected 1 command executed, got %d", len(exec.commands))
	}

	updated, _ := m.GetJob(jobs[0].ID)
	if updated.Executions != 1 {
		t.Errorf("expected 1 execution, got %d", updated.Executions)
	}
	if updated.NextRun != now.Add(time.Hour) {
		t.Errorf("expected next run in 1 hour, got %v", updated.NextRun)
	}
}

func TestManagerRunDueJobsCompleted(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	_, err := m.AddJob("echo hello", "repeat", "", "1h", 1, "")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	jobs, _ := m.ListJobs()
	jobs[0].NextRun = now.Add(-30 * time.Minute)
	jobs[0].IntervalSec = 3600
	_, _ = store.Update(jobs[0])

	err = m.RunDueJobs(now)
	if err != nil {
		t.Fatalf("RunDueJobs failed: %v", err)
	}

	updated, _ := m.GetJob(jobs[0].ID)
	if updated.Status != JobStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
	if !updated.NextRun.IsZero() {
		t.Errorf("expected NextRun to be zero for completed job, got %v", updated.NextRun)
	}
}

func TestManagerRunAllNow(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	_, _ = m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "")
	_, _ = m.AddJob("echo world", "cron", "0 10 * * *", "", 0, "")

	err := m.RunAllNow()
	if err != nil {
		t.Fatalf("RunAllNow failed: %v", err)
	}

	if len(exec.commands) != 2 {
		t.Errorf("expected 2 commands executed, got %d", len(exec.commands))
	}
}

func TestManagerRunAllNowSkipsPaused(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	job, _ := m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "")
	jobs, _ := m.ListJobs()
	jobs[0].Status = JobStatusPaused
	_, _ = store.Update(jobs[0])

	err := m.RunAllNow()
	if err != nil {
		t.Fatalf("RunAllNow failed: %v", err)
	}

	if len(exec.commands) != 0 {
		t.Errorf("expected 0 commands executed, got %d", len(exec.commands))
	}

	_ = job
}

func TestInstallCronJob(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	err := m.InstallCronJob(5)
	if err != nil {
		t.Fatalf("InstallCronJob failed: %v", err)
	}

	cronFile := filepath.Join(tmpDir, ".cockpit", "scheduler", "cron.txt")
	if _, err := os.Stat(cronFile); os.IsNotExist(err) {
		t.Errorf("expected cron file at %s", cronFile)
	}
}

func TestInstallSystemdTimer(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	err := m.InstallSystemdTimer(5)
	if err != nil {
		t.Fatalf("InstallSystemdTimer failed: %v", err)
	}

	serviceFile := filepath.Join(tmpDir, ".config", "systemd", "user", "aicockpit-scheduler.service")
	timerFile := filepath.Join(tmpDir, ".config", "systemd", "user", "aicockpit-scheduler.timer")

	if _, err := os.Stat(serviceFile); os.IsNotExist(err) {
		t.Errorf("expected service file at %s", serviceFile)
	}
	if _, err := os.Stat(timerFile); os.IsNotExist(err) {
		t.Errorf("expected timer file at %s", timerFile)
	}
}

func TestFormatJobList(t *testing.T) {
	jobs := []Job{
		{
			ID:        "sched_abc",
			Command:   "echo hello",
			Type:      JobTypeCron,
			CronExpr:  "0 9 * * *",
			Status:    JobStatusActive,
			NextRun:   time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC),
			CreatedAt: time.Now(),
		},
	}

	output := FormatJobList(jobs)
	if !strings.Contains(output, "sched_abc") {
		t.Error("expected output to contain job id")
	}
	if !strings.Contains(output, "cron") {
		t.Error("expected output to contain job type")
	}
	if !strings.Contains(output, "ativo") {
		t.Error("expected output to contain job status")
	}
}

func TestFormatJobListEmpty(t *testing.T) {
	output := FormatJobList([]Job{})
	if !strings.Contains(output, "Nenhum") {
		t.Error("expected empty list message")
	}
}

func TestNewManagerWithLogger(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	var logs []string
	logger := func(format string, args ...interface{}) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}

	m := NewManagerWithLogger(store, exec, logger)
	if m == nil {
		t.Fatal("expected manager to be non-nil")
	}

	_, err := m.AddJob("echo test", "cron", "0 9 * * *", "", 0, "")
	if err != nil {
		t.Fatalf("AddJob failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("expected no logs during add, got %d", len(logs))
	}
}

func TestManagerRunDueJobsNoJobs(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	var logs []string
	logger := func(format string, args ...interface{}) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}
	m := NewManagerWithLogger(store, exec, logger)

	now := time.Now()
	err := m.RunDueJobs(now)
	if err != nil {
		t.Fatalf("RunDueJobs failed: %v", err)
	}

	found := false
	for _, log := range logs {
		if strings.Contains(log, "no jobs due") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'no jobs due' log, got %v", logs)
	}
}

func TestManagerRunDueJobsStoreError(t *testing.T) {
	errStore := &errorStore{}
	exec := &mockExecutor{}
	m := NewManager(errStore, exec)

	err := m.RunDueJobs(time.Now())
	if err == nil {
		t.Error("expected error when store fails")
	}
}

// errorStore always returns errors for testing.
type errorStore struct{}

func (e *errorStore) Load() ([]Job, error) {
	return nil, fmt.Errorf("load error")
}
func (e *errorStore) Save(jobs []Job) error {
	return fmt.Errorf("save error")
}
func (e *errorStore) Add(job Job) ([]Job, error) {
	return nil, fmt.Errorf("add error")
}
func (e *errorStore) Remove(id string) ([]Job, error) {
	return nil, fmt.Errorf("remove error")
}
func (e *errorStore) Get(id string) (Job, error) {
	return Job{}, fmt.Errorf("get error")
}
func (e *errorStore) Update(job Job) ([]Job, error) {
	return nil, fmt.Errorf("update error")
}

func TestManagerRunDueJobsUpdateError(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	var logs []string
	logger := func(format string, args ...interface{}) {
		logs = append(logs, fmt.Sprintf(format, args...))
	}
	m := NewManagerWithLogger(store, exec, logger)

	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	job, _ := m.AddJob("echo hello", "repeat", "", "1h", 0, "")

	// Simulate a store that fails on update
	failingStore := &failingUpdateStore{inner: store, failID: job.ID}
	m.store = failingStore

	jobs, _ := failingStore.Load()
	jobs[0].NextRun = now.Add(-30 * time.Minute)
	jobs[0].IntervalSec = 3600
	_ = failingStore.Save(jobs)

	err := m.RunDueJobs(now)
	if err != nil {
		t.Fatalf("RunDueJobs failed: %v", err)
	}

	found := false
	for _, log := range logs {
		if strings.Contains(log, "failed to update job") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'failed to update job' log, got %v", logs)
	}
}

// failingUpdateStore fails updates for a specific job ID.
type failingUpdateStore struct {
	inner  *mockStore
	failID string
}

func (f *failingUpdateStore) Load() ([]Job, error) {
	return f.inner.Load()
}
func (f *failingUpdateStore) Save(jobs []Job) error {
	return f.inner.Save(jobs)
}
func (f *failingUpdateStore) Add(job Job) ([]Job, error) {
	return f.inner.Add(job)
}
func (f *failingUpdateStore) Remove(id string) ([]Job, error) {
	return f.inner.Remove(id)
}
func (f *failingUpdateStore) Get(id string) (Job, error) {
	return f.inner.Get(id)
}
func (f *failingUpdateStore) Update(job Job) ([]Job, error) {
	if job.ID == f.failID {
		return nil, fmt.Errorf("update failed for %s", job.ID)
	}
	return f.inner.Update(job)
}

func TestManagerRunAllNowStoreError(t *testing.T) {
	errStore := &errorStore{}
	exec := &mockExecutor{}
	m := NewManager(errStore, exec)

	err := m.RunAllNow()
	if err == nil {
		t.Error("expected error when store fails")
	}
}

func TestManagerRunDueJobsNotDue(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	job, _ := m.AddJob("echo hello", "repeat", "", "1h", 0, "")

	jobs, _ := store.Load()
	jobs[0].NextRun = now.Add(time.Hour)
	jobs[0].IntervalSec = 3600
	_, _ = store.Update(jobs[0])

	err := m.RunDueJobs(now)
	if err != nil {
		t.Fatalf("RunDueJobs failed: %v", err)
	}

	if len(exec.commands) != 0 {
		t.Errorf("expected 0 commands executed, got %d", len(exec.commands))
	}

	_ = job
}

func TestManagerAddJobStoreError(t *testing.T) {
	store := &errorStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	_, err := m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "")
	if err == nil {
		t.Error("expected error when store fails")
	}
}

func TestInstallCronJobInvalidInterval(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	m := NewManager(nil, nil)
	err := m.InstallCronJob(0)
	if err != nil {
		t.Fatalf("InstallCronJob failed: %v", err)
	}

	cronFile := filepath.Join(tmpDir, ".cockpit", "scheduler", "cron.txt")
	data, err := os.ReadFile(cronFile)
	if err != nil {
		t.Fatalf("failed to read cron file: %v", err)
	}
	if !strings.Contains(string(data), "*/5") {
		t.Errorf("expected default interval of 5 minutes, got %s", string(data))
	}
}

func TestInstallSystemdTimerInvalidInterval(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	m := NewManager(nil, nil)
	err := m.InstallSystemdTimer(0)
	if err != nil {
		t.Fatalf("InstallSystemdTimer failed: %v", err)
	}

	timerFile := filepath.Join(tmpDir, ".config", "systemd", "user", "aicockpit-scheduler.timer")
	data, err := os.ReadFile(timerFile)
	if err != nil {
		t.Fatalf("failed to read timer file: %v", err)
	}
	if !strings.Contains(string(data), "OnUnitActiveSec=5m") {
		t.Errorf("expected default interval of 5 minutes, got %s", string(data))
	}
}

func TestInstallSystemdTimerInvalidMode(t *testing.T) {
	m := NewManager(nil, nil)
	err := m.InstallSystemdTimer(-1)
	if err != nil {
		t.Fatalf("InstallSystemdTimer failed: %v", err)
	}
}

func TestFormatJobListLongCommand(t *testing.T) {
	jobs := []Job{
		{
			ID:        "sched_long",
			Command:   "echo 'this is a very long command that should be truncated in the list view'",
			Type:      JobTypeCron,
			CronExpr:  "0 9 * * *",
			Status:    JobStatusActive,
			CreatedAt: time.Now(),
		},
	}
	output := FormatJobList(jobs)
	if !strings.Contains(output, "sched_long") {
		t.Error("expected output to contain job id")
	}
	if strings.Contains(output, "this is a very long command that should be truncated in the list view") {
		t.Error("expected long command to be truncated")
	}
}

func TestManagerGetJob(t *testing.T) {
	store := &mockStore{}
	exec := &mockExecutor{}
	m := NewManager(store, exec)

	job, _ := m.AddJob("echo hello", "cron", "0 9 * * *", "", 0, "")

	got, err := m.GetJob(job.ID)
	if err != nil {
		t.Fatalf("GetJob failed: %v", err)
	}
	if got.ID != job.ID {
		t.Errorf("expected id %s, got %s", job.ID, got.ID)
	}

	_, err = m.GetJob("nonexistent")
	if err == nil {
		t.Error("expected error getting nonexistent job")
	}
}
