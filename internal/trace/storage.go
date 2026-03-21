// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trace

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGo)
)

// Storage handles database operations for traces.
type Storage struct {
	db     *sql.DB
	dbPath string
}

// NewStorage creates a new storage instance.
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open trace database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping trace database: %w", err)
	}

	s := &Storage{db: db, dbPath: dbPath}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to init trace schema: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Storage) Close() error {
	return s.db.Close()
}

// initSchema creates the database tables.
func (s *Storage) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS traces (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    status TEXT NOT NULL,
    error_message TEXT
);

CREATE TABLE IF NOT EXISTS trace_steps (
    id TEXT PRIMARY KEY,
    trace_id TEXT NOT NULL,
    step_number INTEGER NOT NULL,
    step_type TEXT NOT NULL,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    duration_ms INTEGER,
    input TEXT,
    output TEXT,
    metadata TEXT,
    FOREIGN KEY (trace_id) REFERENCES traces(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_traces_session ON traces(session_id);
CREATE INDEX IF NOT EXISTS idx_traces_started ON traces(started_at);
CREATE INDEX IF NOT EXISTS idx_steps_trace ON trace_steps(trace_id);
CREATE INDEX IF NOT EXISTS idx_steps_type ON trace_steps(step_type);
`
	_, err := s.db.Exec(schema)
	return err
}

// SaveTrace saves a trace to the database.
func (s *Storage) SaveTrace(t *Trace) error {
	_, err := s.db.Exec(
		`INSERT INTO traces (id, session_id, started_at, ended_at, status, error_message)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			ended_at = excluded.ended_at,
			status = excluded.status,
			error_message = excluded.error_message`,
		t.ID, t.SessionID, t.StartedAt, t.EndedAt, string(t.Status), t.ErrorMessage,
	)
	return err
}

// SaveStep saves a step to the database.
func (s *Storage) SaveStep(step *Step) error {
	duration := int(step.Duration().Milliseconds())
	_, err := s.db.Exec(
		`INSERT INTO trace_steps (id, trace_id, step_number, step_type, started_at, ended_at, duration_ms, input, output, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		step.ID, step.TraceID, step.StepNumber, string(step.StepType), step.StartedAt, step.EndedAt, duration, step.Input, step.Output, step.Metadata,
	)
	return err
}

// GetTrace retrieves a trace by ID.
func (s *Storage) GetTrace(id string) (*Trace, error) {
	row := s.db.QueryRow(
		`SELECT id, session_id, started_at, ended_at, status, error_message FROM traces WHERE id = ?`,
		id,
	)

	t := &Trace{}
	var endedAt sql.NullTime
	err := row.Scan(&t.ID, &t.SessionID, &t.StartedAt, &endedAt, &t.Status, &t.ErrorMessage)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if endedAt.Valid {
		t.EndedAt = endedAt.Time
	}

	// Load steps
	steps, err := s.GetSteps(id)
	if err != nil {
		return nil, err
	}
	t.Steps = steps

	return t, nil
}

// GetSteps retrieves all steps for a trace.
func (s *Storage) GetSteps(traceID string) ([]*Step, error) {
	rows, err := s.db.Query(
		`SELECT id, trace_id, step_number, step_type, started_at, ended_at, duration_ms, input, output, metadata
		FROM trace_steps WHERE trace_id = ? ORDER BY step_number`,
		traceID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*Step
	for rows.Next() {
		step := &Step{}
		var endedAt sql.NullTime
		var durationMs int
		err := rows.Scan(&step.ID, &step.TraceID, &step.StepNumber, &step.StepType, &step.StartedAt, &endedAt, &durationMs, &step.Input, &step.Output, &step.Metadata)
		if err != nil {
			return nil, err
		}
		if endedAt.Valid {
			step.EndedAt = endedAt.Time
		}
		steps = append(steps, step)
	}

	return steps, rows.Err()
}

// QueryTraces queries traces with filters.
func (s *Storage) QueryTraces(opts QueryOptions) ([]*Trace, error) {
	query := `SELECT id, session_id, started_at, ended_at, status, error_message FROM traces WHERE 1=1`
	args := []interface{}{}

	if opts.SessionID != "" {
		query += ` AND session_id = ?`
		args = append(args, opts.SessionID)
	}
	if opts.Status != "" {
		query += ` AND status = ?`
		args = append(args, opts.Status)
	}
	if !opts.StartedAfter.IsZero() {
		query += ` AND started_at >= ?`
		args = append(args, opts.StartedAfter)
	}
	if !opts.StartedBefore.IsZero() {
		query += ` AND started_at <= ?`
		args = append(args, opts.StartedBefore)
	}

	query += ` ORDER BY started_at DESC`

	if opts.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, opts.Limit)
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var traces []*Trace
	for rows.Next() {
		t := &Trace{}
		var endedAt sql.NullTime
		err := rows.Scan(&t.ID, &t.SessionID, &t.StartedAt, &endedAt, &t.Status, &t.ErrorMessage)
		if err != nil {
			return nil, err
		}
		if endedAt.Valid {
			t.EndedAt = endedAt.Time
		}
		traces = append(traces, t)
	}

	return traces, rows.Err()
}

// Cleanup deletes traces older than the given time.
func (s *Storage) Cleanup(before time.Time) (int64, error) {
	result, err := s.db.Exec(`DELETE FROM traces WHERE started_at < ?`, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// DBPath returns the database path.
func (s *Storage) DBPath() string {
	return s.dbPath
}

// QueryOptions represents options for querying traces.
type QueryOptions struct {
	SessionID      string
	Status         string
	StartedAfter   time.Time
	StartedBefore  time.Time
	Limit          int
}
