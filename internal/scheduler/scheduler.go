package scheduler

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Manager coordinates scheduled jobs.
type Manager struct {
	store    Store
	executor Executor
	logger   func(format string, args ...interface{})
}

// NewManager creates a new scheduler manager.
func NewManager(store Store, executor Executor) *Manager {
	if store == nil {
		store = NewJSONStore(nil)
	}
	if executor == nil {
		executor = NewShellExecutor()
	}

	return &Manager{
		store:    store,
		executor: executor,
		logger: func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		},
	}
}

// NewManagerWithLogger creates a manager with a custom logger.
func NewManagerWithLogger(store Store, executor Executor, logger func(format string, args ...interface{})) *Manager {
	m := NewManager(store, executor)
	if logger != nil {
		m.logger = logger
	}
	return m
}

// AddJob creates a new scheduled job.
func (m *Manager) AddJob(command, jobType, cronExpr, interval string, maxExecutions int, description string) (Job, error) {
	job := Job{
		ID:            GenerateID(),
		Command:       command,
		Type:          JobType(jobType),
		CronExpr:      cronExpr,
		Interval:      interval,
		MaxExecutions: maxExecutions,
		Status:        JobStatusActive,
		CreatedAt:     Now(),
		Description:   description,
	}

	if err := job.Validate(); err != nil {
		return Job{}, err
	}

	job.ComputeNextRun(Now())

	_, err := m.store.Add(job)
	if err != nil {
		return Job{}, fmt.Errorf("failed to add job: %w", err)
	}

	return job, nil
}

// ListJobs returns all scheduled jobs.
func (m *Manager) ListJobs() ([]Job, error) {
	return m.store.Load()
}

// RemoveJob removes a scheduled job by id.
func (m *Manager) RemoveJob(id string) error {
	_, err := m.store.Remove(id)
	if err != nil {
		return fmt.Errorf("failed to remove job: %w", err)
	}
	return nil
}

// GetJob retrieves a single job by id.
func (m *Manager) GetJob(id string) (Job, error) {
	return m.store.Get(id)
}

// RunDueJobs executes all jobs that are due at the given time.
func (m *Manager) RunDueJobs(now time.Time) error {
	jobs, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	ran := false
	for i, job := range jobs {
		if !job.ShouldExecute(now) {
			continue
		}

		ran = true
		m.logger("[scheduler] executing job %s: %s", job.ID, job.Command)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		result := m.executor.Execute(ctx, job.Command)
		cancel()

		job.MarkExecuted(now)
		if _, err := m.store.Update(job); err != nil {
			m.logger("[scheduler] failed to update job %s: %v", job.ID, err)
		}

		if result.Error != nil {
			m.logger("[scheduler] job %s failed: %v", job.ID, result.Error)
		} else {
			m.logger("[scheduler] job %s completed successfully", job.ID)
		}

		jobs[i] = job
	}

	if !ran {
		m.logger("[scheduler] no jobs due at %s", now.Format(time.RFC3339))
	}

	return nil
}

// RunAllNow executes all active jobs immediately, ignoring their schedule.
func (m *Manager) RunAllNow() error {
	jobs, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	for i, job := range jobs {
		if job.Status != JobStatusActive {
			continue
		}

		m.logger("[scheduler] executing job %s immediately", job.ID)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		result := m.executor.Execute(ctx, job.Command)
		cancel()

		if result.Error != nil {
			m.logger("[scheduler] job %s failed: %v", job.ID, result.Error)
		} else {
			m.logger("[scheduler] job %s completed successfully", job.ID)
		}

		job.LastRun = Now()
		job.Executions++
		if _, err := m.store.Update(job); err != nil {
			m.logger("[scheduler] failed to update job %s: %v", job.ID, err)
		}
		jobs[i] = job
	}

	return nil
}

// InstallCronJob creates a system crontab entry to run `cockpit scheduler run` periodically.
func (m *Manager) InstallCronJob(intervalMinutes int) error {
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}

	binary, err := os.Executable()
	if err != nil {
		binary = "cockpit"
	}

	cronLine := fmt.Sprintf("*/%d * * * * %s scheduler run >> %s 2>&1",
		intervalMinutes,
		binary,
		filepath.Join(os.Getenv("HOME"), ".cockpit", "logs", "scheduler-cron.log"),
	)

	cronFile := filepath.Join(os.Getenv("HOME"), ".cockpit", "scheduler", "cron.txt")
	if err := os.MkdirAll(filepath.Dir(cronFile), 0o755); err != nil {
		return fmt.Errorf("failed to create scheduler directory: %w", err)
	}
	if err := os.WriteFile(cronFile, []byte(cronLine+"\n"), 0o644); err != nil {
		return fmt.Errorf("failed to write cron file: %w", err)
	}

	m.logger("[scheduler] cron entry saved to %s", cronFile)
	m.logger("[scheduler] to activate, run: crontab %s", cronFile)

	return nil
}

// InstallSystemdTimer creates a systemd user timer for the scheduler.
func (m *Manager) InstallSystemdTimer(intervalMinutes int) error {
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}

	binary, err := os.Executable()
	if err != nil {
		binary = "cockpit"
	}

	userDir := filepath.Join(os.Getenv("HOME"), ".config", "systemd", "user")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		return fmt.Errorf("failed to create systemd user directory: %w", err)
	}

	service := fmt.Sprintf(`[Unit]
Description=AICockpit Scheduler

[Service]
Type=oneshot
ExecStart=%s scheduler run
`, binary)

	timer := fmt.Sprintf(`[Unit]
Description=AICockpit Scheduler Timer

[Timer]
OnBootSec=1min
OnUnitActiveSec=%dm

[Install]
WantedBy=timers.target
`, intervalMinutes)

	if err := os.WriteFile(filepath.Join(userDir, "aicockpit-scheduler.service"), []byte(service), 0o644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "aicockpit-scheduler.timer"), []byte(timer), 0o644); err != nil {
		return fmt.Errorf("failed to write timer file: %w", err)
	}

	m.logger("[scheduler] systemd user timer installed at %s", userDir)
	m.logger("[scheduler] to activate, run: systemctl --user daemon-reload && systemctl --user enable --now aicockpit-scheduler.timer")

	return nil
}

// FormatJobList returns a formatted string for listing jobs.
func FormatJobList(jobs []Job) string {
	if len(jobs) == 0 {
		return "Nenhum agendamento encontrado."
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-15s | %-25s | %-10s | %-10s | %-20s\n", "ID", "COMANDO", "TIPO", "STATUS", "PROXIMA EXECUCAO"))
	b.WriteString(strings.Repeat("-", 95) + "\n")

	for _, job := range jobs {
		nextRun := "-"
		if !job.NextRun.IsZero() {
			nextRun = job.NextRun.Format("2006-01-02 15:04:05")
		}
		cmd := job.Command
		if len(cmd) > 25 {
			cmd = cmd[:22] + "..."
		}
		b.WriteString(fmt.Sprintf("%-15s | %-25s | %-10s | %-10s | %-20s\n", job.ID, cmd, job.Type, job.Status, nextRun))
	}

	return b.String()
}
