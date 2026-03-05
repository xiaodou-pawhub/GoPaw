package cron

import (
	"time"
)

// CronJob represents a scheduled task.
// JSON tags are aligned with the frontend expectations (legacy scheduler compatibility).
type CronJob struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Schedule    string     `json:"schedule"`      // New field name
	CronExpr    string     `json:"cron_expr"`     // Alias for frontend compatibility
	Task        string     `json:"task"`          // New field name
	Prompt      string     `json:"prompt"`        // Alias for frontend compatibility
	Channel     string     `json:"channel"`       // Target channel: "feishu"
	TargetID    string     `json:"target_id"`     // Target ChatID (platform level)
	Enabled     bool       `json:"enabled"`       // Pause/Resume toggle
	CreatedAt   time.Time  `json:"created_at"`
	LastRunAt   *time.Time `json:"last_run_at,omitempty"`
	LastResult  string     `json:"last_result,omitempty"` // "success" or error message
	EntryID     int        `json:"-"`                     // Internal cron entry ID
}

// CronRun represents a single execution record of a job.
type CronRun struct {
	ID          string    `json:"id"`
	JobID       string    `json:"job_id"`
	TriggeredAt time.Time `json:"triggered_at"`
	FinishedAt  time.Time `json:"finished_at"`
	Status      string    `json:"status"` // "success", "error"
	Output      string    `json:"output"` // Final answer content
	Error       string    `json:"error,omitempty"`
}

// CronStore represents the persisted state of the cron system.
type CronStore struct {
	Version int       `json:"version"`
	Jobs    []CronJob `json:"jobs"`
}
