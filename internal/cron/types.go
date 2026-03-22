package cron

import (
	"time"
)

// CronJob represents a scheduled task.
type CronJob struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Schedule   string     `json:"schedule"`
	Task       string     `json:"task"`
	Channel    string     `json:"channel"`
	TargetID   string     `json:"target_id"`
	Enabled    bool       `json:"enabled"`
	CreatedAt  time.Time  `json:"created_at"`
	LastRunAt  *time.Time `json:"last_run_at,omitempty"`
	LastResult string     `json:"last_result,omitempty"`
	EntryID    int        `json:"-"`
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
