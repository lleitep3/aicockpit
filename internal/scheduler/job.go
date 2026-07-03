package scheduler

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// JobType represents the scheduling type of a job.
type JobType string

const (
	// JobTypeCron schedules a job using cron expression.
	JobTypeCron JobType = "cron"
	// JobTypeRepeat schedules a job with a fixed interval and optional max repetitions.
	JobTypeRepeat JobType = "repeat"
)

// JobStatus represents the current status of a job.
type JobStatus string

const (
	// JobStatusActive means the job is enabled and scheduled.
	JobStatusActive JobStatus = "ativo"
	// JobStatusPaused means the job is temporarily disabled.
	JobStatusPaused JobStatus = "pausado"
	// JobStatusCompleted means the job reached its max executions.
	JobStatusCompleted JobStatus = "concluido"
)

// Job represents a scheduled command or script.
type Job struct {
	ID            string    `json:"id"`
	Command       string    `json:"command"`
	Type          JobType   `json:"type"`
	CronExpr      string    `json:"cron_expr,omitempty"`
	Interval      string    `json:"interval,omitempty"`
	IntervalSec   int64     `json:"interval_sec,omitempty"`
	MaxExecutions int       `json:"max_executions,omitempty"`
	Executions    int       `json:"executions"`
	LastRun       time.Time `json:"last_run,omitempty"`
	NextRun       time.Time `json:"next_run,omitempty"`
	Status        JobStatus `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	Description   string    `json:"description,omitempty"`
}

// Common interval aliases used by the scheduler.
var intervalAliases = map[string]int64{
	"1h":  3600,
	"30m": 1800,
	"15m": 900,
	"5m":  300,
	"1m":  60,
	"1d":  86400,
	"1w":  604800,
}

// GenerateID creates a new scheduler job identifier.
func GenerateID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "sched_" + hex.EncodeToString(b)
}

// Validate checks whether the job configuration is valid.
func (j *Job) Validate() error {
	if strings.TrimSpace(j.Command) == "" {
		return errors.New("comando nao pode ser vazio")
	}

	switch j.Type {
	case JobTypeCron:
		if strings.TrimSpace(j.CronExpr) == "" {
			return errors.New("expressao cron nao pode ser vazia")
		}
		if _, err := NormalizeCron(j.CronExpr); err != nil {
			return err
		}
	case JobTypeRepeat:
		if strings.TrimSpace(j.Interval) == "" {
			return errors.New("intervalo nao pode ser vazio")
		}
		sec, err := ParseInterval(j.Interval)
		if err != nil {
			return err
		}
		j.IntervalSec = sec
		if j.MaxExecutions < 0 {
			return errors.New("repeticoes nao podem ser negativas")
		}
	default:
		return fmt.Errorf("tipo de agendamento invalido: %s", j.Type)
	}

	return nil
}

// ComputeNextRun updates the NextRun field based on the job type and last run.
func (j *Job) ComputeNextRun(from time.Time) {
	if j.Status == JobStatusPaused || j.Status == JobStatusCompleted {
		return
	}

	switch j.Type {
	case JobTypeCron:
		next, _ := NextCronRun(j.CronExpr, from)
		j.NextRun = next
	case JobTypeRepeat:
		if j.IntervalSec == 0 {
			j.IntervalSec, _ = ParseInterval(j.Interval)
		}
		j.NextRun = from.Add(time.Duration(j.IntervalSec) * time.Second)
	}
}

// ShouldExecute returns true if the job is due to run at the given time.
func (j *Job) ShouldExecute(now time.Time) bool {
	if j.Status == JobStatusPaused || j.Status == JobStatusCompleted {
		return false
	}
	if j.NextRun.IsZero() {
		return true
	}
	return !now.Before(j.NextRun)
}

// MarkExecuted updates the job after a successful execution.
func (j *Job) MarkExecuted(now time.Time) {
	j.LastRun = now
	j.Executions++

	if j.Type == JobTypeRepeat && j.MaxExecutions > 0 && j.Executions >= j.MaxExecutions {
		j.Status = JobStatusCompleted
		j.NextRun = time.Time{}
		return
	}

	j.ComputeNextRun(now)
}

// FormatPattern returns a human readable description of the schedule pattern.
func (j *Job) FormatPattern() string {
	switch j.Type {
	case JobTypeCron:
		return FormatCronDescription(j.CronExpr)
	case JobTypeRepeat:
		if j.MaxExecutions > 0 {
			return fmt.Sprintf("a cada %s, max %d execucoes", j.Interval, j.MaxExecutions)
		}
		return fmt.Sprintf("a cada %s", j.Interval)
	}
	return string(j.Type)
}

// ParseInterval converts an interval string (e.g., "1h", "30m") to seconds.
func ParseInterval(input string) (int64, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	if alias, ok := intervalAliases[input]; ok {
		return alias, nil
	}

	re := regexp.MustCompile(`^(\d+)\s*(h|m|s|d|w)$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) != 3 {
		return 0, fmt.Errorf("intervalo invalido: %s (ex: 1h, 30m, 2d)", input)
	}

	value, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("intervalo invalido: %s", input)
	}

	unit := matches[2]
	var multiplier int64
	switch unit {
	case "s":
		multiplier = 1
	case "m":
		multiplier = 60
	case "h":
		multiplier = 3600
	case "d":
		multiplier = 86400
	case "w":
		multiplier = 604800
	}

	return value * multiplier, nil
}

// NormalizeCron converts common aliases into a standard cron expression.
func NormalizeCron(expr string) (string, error) {
	expr = strings.TrimSpace(strings.ToLower(expr))

	aliases := map[string]string{
		"@yearly":   "0 0 1 1 *",
		"@annually": "0 0 1 1 *",
		"@monthly":  "0 0 1 * *",
		"@weekly":   "0 0 * * 0",
		"@daily":    "0 0 * * *",
		"@midnight": "0 0 * * *",
		"@hourly":   "0 * * * *",
		"weekdays":  "0 0 * * 1-5",
		"weekends":  "0 0 * * 0,6",
		"daily":     "0 0 * * *",
		"hourly":    "0 * * * *",
	}

	if alias, ok := aliases[expr]; ok {
		return alias, nil
	}

	parts := strings.Fields(expr)
	if len(parts) == 5 {
		return expr, nil
	}

	return "", fmt.Errorf("expressao cron invalida: %s", expr)
}

// FormatCronDescription returns a human readable description for a cron expression.
func FormatCronDescription(expr string) string {
	expr = strings.TrimSpace(strings.ToLower(expr))
	aliases := map[string]string{
		"@yearly":   "Anualmente",
		"@annually": "Anualmente",
		"@monthly":  "Mensalmente",
		"@weekly":   "Semanalmente",
		"@daily":    "Diariamente",
		"@midnight": "Diariamente a meia-noite",
		"@hourly":   "A cada hora",
		"weekdays":  "Dias de semana",
		"weekends":  "Fins de semana",
		"daily":     "Diariamente",
		"hourly":    "A cada hora",
	}

	if alias, ok := aliases[expr]; ok {
		return alias
	}

	parts := strings.Fields(expr)
	if len(parts) == 5 {
		return fmt.Sprintf("Cron: %s", expr)
	}

	return expr
}

// NextCronRun calculates the next execution time for a cron expression.
// It supports a subset of cron: minute, hour, day of month, month, day of week.
func NextCronRun(expr string, from time.Time) (time.Time, error) {
	normalized, err := NormalizeCron(expr)
	if err != nil {
		return time.Time{}, err
	}

	parts := strings.Fields(normalized)
	if len(parts) != 5 {
		return time.Time{}, fmt.Errorf("expressao cron invalida: %s", normalized)
	}

	minuteField := parts[0]
	hourField := parts[1]
	dayMonthField := parts[2]
	monthField := parts[3]
	dayWeekField := parts[4]

	candidate := from.Truncate(time.Minute).Add(time.Minute)
	maxIterations := 366 * 24 * 60

	for i := 0; i < maxIterations; i++ {
		if matchField(minuteField, candidate.Minute()) &&
			matchField(hourField, candidate.Hour()) &&
			matchField(dayMonthField, candidate.Day()) &&
			matchField(monthField, int(candidate.Month())) &&
			matchDayOfWeek(dayWeekField, candidate.Weekday()) {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}

	return time.Time{}, fmt.Errorf("nao foi possivel calcular proxima execucao para: %s", normalized)
}

func matchField(field string, value int) bool {
	field = strings.TrimSpace(field)

	if field == "*" {
		return true
	}

	// Handle ranges like 1-5
	if strings.Contains(field, "-") && !strings.Contains(field, "/") {
		parts := strings.SplitN(field, "-", 2)
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			return value >= start && value <= end
		}
	}

	// Handle lists like 1,3,5
	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			part = strings.TrimSpace(part)
			if matchField(part, value) {
				return true
			}
		}
		return false
	}

	// Handle step like */5
	if strings.HasPrefix(field, "*/") {
		step, err := strconv.Atoi(field[2:])
		if err == nil && step > 0 {
			return value%step == 0
		}
	}

	// Handle exact value
	v, err := strconv.Atoi(field)
	if err == nil {
		return value == v
	}

	return false
}

func matchDayOfWeek(field string, weekday time.Weekday) bool {
	field = strings.TrimSpace(field)
	if field == "*" {
		return true
	}

	// Handle range like 1-5
	if strings.Contains(field, "-") && !strings.Contains(field, ",") {
		parts := strings.SplitN(field, "-", 2)
		start, err1 := strconv.Atoi(parts[0])
		end, err2 := strconv.Atoi(parts[1])
		if err1 == nil && err2 == nil {
			w := int(weekday)
			if w == 0 {
				w = 7
			}
			return w >= start && w <= end
		}
	}

	// Handle list like 0,6 or 1,3
	if strings.Contains(field, ",") {
		for _, part := range strings.Split(field, ",") {
			part = strings.TrimSpace(part)
			if matchDayOfWeek(part, weekday) {
				return true
			}
		}
		return false
	}

	v, err := strconv.Atoi(field)
	if err != nil {
		return false
	}

	w := int(weekday)
	if w == 0 {
		w = 7
	}

	// Both 0 and 7 represent Sunday in cron notation.
	if v == 0 {
		return w == 7
	}
	return w == v
}
