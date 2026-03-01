// Package scheduler manages cron-based scheduled tasks.
package scheduler

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CronJob is the data model for a scheduled task.
type CronJob struct {
	ID          string
	Name        string
	Description string
	CronExpr    string
	Channel     string
	SessionID   string
	Prompt      string
	Enabled     bool
	ActiveFrom  string // "HH:MM", empty means no restriction
	ActiveUntil string // "HH:MM", empty means no restriction
	LastRun     time.Time
	NextRun     time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// JobStore handles persistence for CronJob records.
type JobStore struct {
	db *sql.DB
}

// NewJobStore creates a JobStore using the provided database connection.
// The database must already have the cron_jobs table created by memory.Store.migrate().
func NewJobStore(db *sql.DB) *JobStore {
	return &JobStore{db: db}
}

// Create inserts a new CronJob and returns its generated ID.
func (s *JobStore) Create(job *CronJob) (string, error) {
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	now := time.Now()
	job.CreatedAt = now
	job.UpdatedAt = now

	_, err := s.db.Exec(`
		INSERT INTO cron_jobs
			(id, name, description, cron_expr, channel, session_id, prompt,
			 enabled, active_from, active_until, last_run, next_run, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		job.ID, job.Name, job.Description, job.CronExpr, job.Channel, job.SessionID,
		job.Prompt, boolToInt(job.Enabled), job.ActiveFrom, job.ActiveUntil,
		timeToUnix(job.LastRun), timeToUnix(job.NextRun),
		now.UnixMilli(), now.UnixMilli(),
	)
	if err != nil {
		return "", fmt.Errorf("job store: create: %w", err)
	}
	return job.ID, nil
}

// List returns all CronJob records.
func (s *JobStore) List() ([]CronJob, error) {
	rows, err := s.db.Query(`
		SELECT id, name, description, cron_expr, channel, session_id, prompt,
		       enabled, active_from, active_until, last_run, next_run, created_at, updated_at
		FROM cron_jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("job store: list: %w", err)
	}
	defer rows.Close()

	var jobs []CronJob
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// Get retrieves a CronJob by ID.
func (s *JobStore) Get(id string) (*CronJob, error) {
	row := s.db.QueryRow(`
		SELECT id, name, description, cron_expr, channel, session_id, prompt,
		       enabled, active_from, active_until, last_run, next_run, created_at, updated_at
		FROM cron_jobs WHERE id = ?`, id)

	j, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job store: job %q not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("job store: get: %w", err)
	}
	return &j, nil
}

// Update saves changes to an existing CronJob.
func (s *JobStore) Update(job *CronJob) error {
	job.UpdatedAt = time.Now()
	_, err := s.db.Exec(`
		UPDATE cron_jobs
		SET name=?, description=?, cron_expr=?, channel=?, session_id=?, prompt=?,
		    enabled=?, active_from=?, active_until=?, last_run=?, next_run=?, updated_at=?
		WHERE id=?`,
		job.Name, job.Description, job.CronExpr, job.Channel, job.SessionID, job.Prompt,
		boolToInt(job.Enabled), job.ActiveFrom, job.ActiveUntil,
		timeToUnix(job.LastRun), timeToUnix(job.NextRun), job.UpdatedAt.UnixMilli(),
		job.ID,
	)
	if err != nil {
		return fmt.Errorf("job store: update: %w", err)
	}
	return nil
}

// Delete removes a CronJob by ID.
func (s *JobStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM cron_jobs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("job store: delete: %w", err)
	}
	return nil
}

// UpdateLastRun records the last execution time for a job.
func (s *JobStore) UpdateLastRun(id string, lastRun, nextRun time.Time) error {
	_, err := s.db.Exec(
		`UPDATE cron_jobs SET last_run=?, next_run=?, updated_at=? WHERE id=?`,
		lastRun.UnixMilli(), nextRun.UnixMilli(), time.Now().UnixMilli(), id,
	)
	return err
}

// scanRow is a common interface for sql.Row and *sql.Rows.
type scanRow interface {
	Scan(dest ...interface{}) error
}

func scanJob(row scanRow) (CronJob, error) {
	var j CronJob
	var enabledInt int
	var lastRunMs, nextRunMs, createdMs, updatedMs int64
	err := row.Scan(
		&j.ID, &j.Name, &j.Description, &j.CronExpr, &j.Channel, &j.SessionID,
		&j.Prompt, &enabledInt, &j.ActiveFrom, &j.ActiveUntil,
		&lastRunMs, &nextRunMs, &createdMs, &updatedMs,
	)
	if err != nil {
		return j, err
	}
	j.Enabled = enabledInt != 0
	j.LastRun = time.UnixMilli(lastRunMs)
	j.NextRun = time.UnixMilli(nextRunMs)
	j.CreatedAt = time.UnixMilli(createdMs)
	j.UpdatedAt = time.UnixMilli(updatedMs)
	return j, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func timeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
