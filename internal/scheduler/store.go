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

// CronRun is the data model for a cron job execution.
// 中文：Cron 任务执行记录的数据模型
// English: Data model for cron job execution
type CronRun struct {
	ID           string
	JobID        string
	TriggeredAt  time.Time
	FinishedAt   time.Time
	Status       string // "running" | "success" | "error"
	Output       string
	ErrorMessage string
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

// CreateRun inserts a new CronRun record.
// 中文：创建新的 Cron 执行记录
// English: Insert a new CronRun record
func (s *JobStore) CreateRun(run *CronRun) (string, error) {
	if run.ID == "" {
		run.ID = uuid.New().String()
	}
	_, err := s.db.Exec(`
		INSERT INTO cron_runs (id, job_id, triggered_at, finished_at, status, output, error_msg)
		VALUES (?,?,?,?,?,?,?)`,
		run.ID, run.JobID, timeToUnix(run.TriggeredAt), timeToUnix(run.FinishedAt),
		run.Status, run.Output, run.ErrorMessage,
	)
	if err != nil {
		return "", fmt.Errorf("job store: create run: %w", err)
	}
	return run.ID, nil
}

// ListRuns returns the most recent N executions for a job.
// 中文：返回指定任务的最近 N 次执行记录（时间倒序）
// English: Return the most recent N executions for a job (ordered by triggered_at DESC)
func (s *JobStore) ListRuns(jobID string, limit int) ([]CronRun, error) {
	rows, err := s.db.Query(`
		SELECT id, job_id, triggered_at, finished_at, status, output, error_msg
		FROM cron_runs WHERE job_id = ?
		ORDER BY triggered_at DESC LIMIT ?`,
		jobID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("job store: list runs: %w", err)
	}
	defer rows.Close()

	var runs []CronRun
	for rows.Next() {
		r, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

// UpdateRun updates the status of a run record.
// 中文：更新执行记录的状态
// English: Update the status of a run record
func (s *JobStore) UpdateRun(id string, finishedAt time.Time, status, output, errMsg string) error {
	_, err := s.db.Exec(`
		UPDATE cron_runs
		SET finished_at=?, status=?, output=?, error_msg=?
		WHERE id=?`,
		timeToUnix(finishedAt), status, output, errMsg, id,
	)
	if err != nil {
		return fmt.Errorf("job store: update run: %w", err)
	}
	return nil
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

// scanRun scans a CronRun from a sql.Row or *sql.Rows.
// 中文：从数据库行扫描 CronRun 记录
// English: Scan a CronRun record from database row
func scanRun(row scanRow) (CronRun, error) {
	var r CronRun
	var triggeredMs, finishedMs int64
	err := row.Scan(
		&r.ID, &r.JobID, &triggeredMs, &finishedMs,
		&r.Status, &r.Output, &r.ErrorMessage,
	)
	if err != nil {
		return r, err
	}
	r.TriggeredAt = time.UnixMilli(triggeredMs)
	if finishedMs > 0 {
		r.FinishedAt = time.UnixMilli(finishedMs)
	}
	return r, nil
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
