package scheduler

import (
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	id := GenerateID()
	if id == "" {
		t.Fatal("GenerateID should not return empty string")
	}
	if !hasPrefix(id, "sched_") {
		t.Fatalf("expected id to start with sched_, got %s", id)
	}
	if len(id) <= len("sched_") {
		t.Fatalf("expected id to be longer than sched_, got %s", id)
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func TestParseInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1h", 3600, false},
		{"30m", 1800, false},
		{"15m", 900, false},
		{"5m", 300, false},
		{"1m", 60, false},
		{"1d", 86400, false},
		{"1w", 604800, false},
		{"2h", 7200, false},
		{"10s", 10, false},
		{"", 0, true},
		{"invalid", 0, true},
		{"1x", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseInterval(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseInterval(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("ParseInterval(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizeCron(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"@daily", "0 0 * * *", false},
		{"@hourly", "0 * * * *", false},
		{"daily", "0 0 * * *", false},
		{"weekdays", "0 0 * * 1-5", false},
		{"weekends", "0 0 * * 0,6", false},
		{"0 9 * * *", "0 9 * * *", false},
		{"0 9 * * 1,3", "0 9 * * 1,3", false},
		{"", "", true},
		{"invalid", "", true},
		{"0 9 * *", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := NormalizeCron(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NormalizeCron(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if got != tt.expected {
				t.Errorf("NormalizeCron(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNextCronRun(t *testing.T) {
	base := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		expr     string
		expected time.Time
	}{
		{"0 9 * * *", time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)},
		{"0 * * * *", time.Date(2026, 7, 3, 13, 0, 0, 0, time.UTC)},
		{"daily", time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got, err := NextCronRun(tt.expr, base)
			if err != nil {
				t.Fatalf("NextCronRun(%q) error = %v", tt.expr, err)
			}
			if !got.Equal(tt.expected) {
				t.Errorf("NextCronRun(%q) = %v, want %v", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestJobValidate(t *testing.T) {
	tests := []struct {
		name    string
		job     Job
		wantErr bool
	}{
		{
			name: "valid cron",
			job: Job{
				Command:  "echo hello",
				Type:     JobTypeCron,
				CronExpr: "0 9 * * *",
			},
			wantErr: false,
		},
		{
			name: "valid repeat",
			job: Job{
				Command:       "scripts/backup.sh",
				Type:          JobTypeRepeat,
				Interval:      "1h",
				MaxExecutions: 3,
			},
			wantErr: false,
		},
		{
			name: "empty command",
			job: Job{
				Command:  "",
				Type:     JobTypeCron,
				CronExpr: "0 9 * * *",
			},
			wantErr: true,
		},
		{
			name: "empty cron",
			job: Job{
				Command:  "echo hello",
				Type:     JobTypeCron,
				CronExpr: "",
			},
			wantErr: true,
		},
		{
			name: "invalid interval",
			job: Job{
				Command:  "echo hello",
				Type:     JobTypeRepeat,
				Interval: "invalid",
			},
			wantErr: true,
		},
		{
			name: "invalid type",
			job: Job{
				Command: "echo hello",
				Type:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.job.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestJobShouldExecute(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	job := Job{
		ID:       "sched_test",
		Command:  "echo hello",
		Type:     JobTypeCron,
		CronExpr: "0 9 * * *",
		Status:   JobStatusActive,
		NextRun:  time.Date(2026, 7, 3, 11, 0, 0, 0, time.UTC),
	}

	if !job.ShouldExecute(now) {
		t.Error("expected job to be due for execution")
	}

	job.Status = JobStatusPaused
	if job.ShouldExecute(now) {
		t.Error("paused job should not execute")
	}

	job.Status = JobStatusCompleted
	if job.ShouldExecute(now) {
		t.Error("completed job should not execute")
	}
}

func TestJobMarkExecuted(t *testing.T) {
	now := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	job := Job{
		ID:            "sched_test",
		Command:       "echo hello",
		Type:          JobTypeRepeat,
		Interval:      "1h",
		IntervalSec:   3600,
		MaxExecutions: 2,
		Status:        JobStatusActive,
		Executions:    0,
	}

	job.MarkExecuted(now)
	if job.Executions != 1 {
		t.Errorf("expected executions = 1, got %d", job.Executions)
	}
	if job.Status != JobStatusActive {
		t.Errorf("expected status active, got %s", job.Status)
	}

	job.MarkExecuted(now)
	if job.Executions != 2 {
		t.Errorf("expected executions = 2, got %d", job.Executions)
	}
	if job.Status != JobStatusCompleted {
		t.Errorf("expected status completed, got %s", job.Status)
	}
	if !job.NextRun.IsZero() {
		t.Error("expected NextRun to be zero after completion")
	}
}

func TestJobFormatPattern(t *testing.T) {
	cronJob := Job{
		Type:     JobTypeCron,
		CronExpr: "daily",
	}
	if cronJob.FormatPattern() != "Diariamente" {
		t.Errorf("expected 'Diariamente', got %s", cronJob.FormatPattern())
	}

	repeatJob := Job{
		Type:          JobTypeRepeat,
		Interval:      "1h",
		MaxExecutions: 3,
	}
	expected := "a cada 1h, max 3 execucoes"
	if repeatJob.FormatPattern() != expected {
		t.Errorf("expected %q, got %q", expected, repeatJob.FormatPattern())
	}
}

func TestMatchField(t *testing.T) {
	tests := []struct {
		field  string
		value  int
		expect bool
	}{
		{"*", 5, true},
		{"5", 5, true},
		{"5", 4, false},
		{"1-5", 3, true},
		{"1-5", 6, false},
		{"1,3,5", 3, true},
		{"1,3,5", 4, false},
		{"*/5", 10, true},
		{"*/5", 7, false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := matchField(tt.field, tt.value)
			if got != tt.expect {
				t.Errorf("matchField(%q, %d) = %v, want %v", tt.field, tt.value, got, tt.expect)
			}
		})
	}
}

func TestMatchDayOfWeek(t *testing.T) {
	tests := []struct {
		field   string
		weekday time.Weekday
		expect  bool
	}{
		{"*", time.Monday, true},
		{"1", time.Monday, true},
		{"1", time.Tuesday, false},
		{"1-5", time.Friday, true},
		{"1-5", time.Saturday, false},
		{"0,6", time.Sunday, true},
		{"0,6", time.Monday, false},
		{"7", time.Sunday, true},
		{"invalid", time.Monday, false},
	}

	for _, tt := range tests {
		t.Run(tt.field, func(t *testing.T) {
			got := matchDayOfWeek(tt.field, tt.weekday)
			if got != tt.expect {
				t.Errorf("matchDayOfWeek(%q, %v) = %v, want %v", tt.field, tt.weekday, got, tt.expect)
			}
		})
	}
}

func TestComputeNextRun(t *testing.T) {
	base := time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)

	cronJob := Job{
		Type:     JobTypeCron,
		CronExpr: "0 9 * * *",
		Status:   JobStatusActive,
	}
	cronJob.ComputeNextRun(base)
	expected := time.Date(2026, 7, 4, 9, 0, 0, 0, time.UTC)
	if !cronJob.NextRun.Equal(expected) {
		t.Errorf("expected next run %v, got %v", expected, cronJob.NextRun)
	}

	repeatJob := Job{
		Type:        JobTypeRepeat,
		Interval:    "1h",
		IntervalSec: 3600,
		Status:      JobStatusActive,
	}
	repeatJob.ComputeNextRun(base)
	if !repeatJob.NextRun.Equal(base.Add(time.Hour)) {
		t.Errorf("expected next run %v, got %v", base.Add(time.Hour), repeatJob.NextRun)
	}

	pausedJob := Job{
		Type:     JobTypeCron,
		CronExpr: "0 9 * * *",
		Status:   JobStatusPaused,
	}
	pausedJob.ComputeNextRun(base)
	if !pausedJob.NextRun.IsZero() {
		t.Error("paused job should have zero NextRun")
	}
}

func TestFormatCronDescription(t *testing.T) {
	tests := []struct {
		expr     string
		expected string
	}{
		{"@daily", "Diariamente"},
		{"@hourly", "A cada hora"},
		{"weekdays", "Dias de semana"},
		{"weekends", "Fins de semana"},
		{"0 9 * * *", "Cron: 0 9 * * *"},
		{"invalid", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.expr, func(t *testing.T) {
			got := FormatCronDescription(tt.expr)
			if got != tt.expected {
				t.Errorf("FormatCronDescription(%q) = %q, want %q", tt.expr, got, tt.expected)
			}
		})
	}
}

func TestShouldExecuteZeroNextRun(t *testing.T) {
	job := Job{
		ID:      "sched_zero",
		Command: "echo hello",
		Type:    JobTypeCron,
		Status:  JobStatusActive,
	}
	if !job.ShouldExecute(time.Now()) {
		t.Error("job with zero NextRun should execute")
	}
}

func TestFormatPatternRepeatUnlimited(t *testing.T) {
	job := Job{
		Type:     JobTypeRepeat,
		Interval: "30m",
	}
	expected := "a cada 30m"
	if job.FormatPattern() != expected {
		t.Errorf("expected %q, got %q", expected, job.FormatPattern())
	}
}

func TestValidateRepeatNegative(t *testing.T) {
	job := Job{
		Command:       "echo hello",
		Type:          JobTypeRepeat,
		Interval:      "1h",
		MaxExecutions: -1,
	}
	if err := job.Validate(); err == nil {
		t.Error("expected error for negative max executions")
	}
}

func TestParseIntervalInvalidNumber(t *testing.T) {
	_, err := ParseInterval("abc")
	if err == nil {
		t.Error("expected error for invalid interval")
	}
}
