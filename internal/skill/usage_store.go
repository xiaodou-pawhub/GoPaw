// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package skill

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

// UsageStore persists skill usage data to SQLite.
type UsageStore struct {
	db     *sql.DB
	logger *zap.Logger
}

// UsageRecord represents a single skill usage record.
type UsageRecord struct {
	SkillName string
	Count     int
	LastUsed  time.Time
}

// NewUsageStore creates a new usage store.
func NewUsageStore(dbPath string, logger *zap.Logger) (*UsageStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open usage store: %w", err)
	}

	store := &UsageStore{
		db:     db,
		logger: logger.Named("skill_usage_store"),
	}

	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

// initSchema creates the usage table if it doesn't exist.
func (s *UsageStore) initSchema() error {
	query := `
		CREATE TABLE IF NOT EXISTS skill_usage (
			skill_name TEXT PRIMARY KEY,
			count INTEGER NOT NULL DEFAULT 0,
			last_used TIMESTAMP NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_last_used ON skill_usage(last_used);
	`
	if _, err := s.db.Exec(query); err != nil {
		return fmt.Errorf("failed to init usage schema: %w", err)
	}
	return nil
}

// Save persists a usage record.
func (s *UsageStore) Save(record *UsageRecord) error {
	query := `
		INSERT INTO skill_usage (skill_name, count, last_used)
		VALUES (?, ?, ?)
		ON CONFLICT(skill_name) DO UPDATE SET
			count = excluded.count,
			last_used = excluded.last_used
	`
	_, err := s.db.Exec(query, record.SkillName, record.Count, record.LastUsed)
	if err != nil {
		return fmt.Errorf("failed to save usage record: %w", err)
	}
	return nil
}

// LoadAll loads all usage records.
func (s *UsageStore) LoadAll() ([]UsageRecord, error) {
	query := `SELECT skill_name, count, last_used FROM skill_usage`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to load usage records: %w", err)
	}
	defer rows.Close()

	var records []UsageRecord
	for rows.Next() {
		var r UsageRecord
		if err := rows.Scan(&r.SkillName, &r.Count, &r.LastUsed); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// Delete removes a usage record.
func (s *UsageStore) Delete(skillName string) error {
	query := `DELETE FROM skill_usage WHERE skill_name = ?`
	_, err := s.db.Exec(query, skillName)
	if err != nil {
		return fmt.Errorf("failed to delete usage record: %w", err)
	}
	return nil
}

// DeleteOlderThan removes records not used since the given time.
func (s *UsageStore) DeleteOlderThan(cutoff time.Time) (int64, error) {
	query := `DELETE FROM skill_usage WHERE last_used < ?`
	result, err := s.db.Exec(query, cutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old records: %w", err)
	}
	return result.RowsAffected()
}

// Close closes the database connection.
func (s *UsageStore) Close() error {
	return s.db.Close()
}
