// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package audit

import (
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages audit logs.
type Manager struct {
	db     *sql.DB
	logger *zap.Logger
	buffer chan *Log
	wg     sync.WaitGroup
	stop   chan struct{}
}

// NewManager creates a new audit manager.
func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		db:     db,
		logger: logger.Named("audit"),
		buffer: make(chan *Log, 1000),
		stop:   make(chan struct{}),
	}

	if err := m.initSchema(); err != nil {
		return nil, err
	}

	// Start async writer
	m.wg.Add(1)
	go m.asyncWriter()

	return m, nil
}

// Close closes the audit manager.
func (m *Manager) Close() error {
	close(m.stop)
	m.wg.Wait()
	return nil
}

// InitSchema creates the audit log database tables.
func InitSchema(db *sql.DB) error {
	schema := `
CREATE TABLE IF NOT EXISTS audit_logs (
    id TEXT PRIMARY KEY,
    timestamp TIMESTAMP NOT NULL,
    category TEXT NOT NULL,
    action TEXT NOT NULL,
    user_id TEXT,
    user_ip TEXT,
    resource_type TEXT,
    resource_id TEXT,
    status TEXT NOT NULL,
    details TEXT,
    error TEXT,
    duration INTEGER,
    request_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_category ON audit_logs(category);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_audit_status ON audit_logs(status);
CREATE INDEX IF NOT EXISTS idx_audit_request ON audit_logs(request_id);
CREATE INDEX IF NOT EXISTS idx_audit_time_cat ON audit_logs(timestamp, category);
`
	_, err := db.Exec(schema)
	return err
}

// initSchema creates the database tables (instance method).
func (m *Manager) initSchema() error {
	return InitSchema(m.db)
}

// asyncWriter writes logs asynchronously.
func (m *Manager) asyncWriter() {
	defer m.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	batch := make([]*Log, 0, 100)

	for {
		select {
		case log := <-m.buffer:
			batch = append(batch, log)
			if len(batch) >= 100 {
				m.writeBatch(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				m.writeBatch(batch)
				batch = batch[:0]
			}

		case <-m.stop:
			// Flush remaining logs
			for len(m.buffer) > 0 {
				batch = append(batch, <-m.buffer)
			}
			if len(batch) > 0 {
				m.writeBatch(batch)
			}
			return
		}
	}
}

// writeBatch writes a batch of logs to the database.
func (m *Manager) writeBatch(logs []*Log) {
	tx, err := m.db.Begin()
	if err != nil {
		m.logger.Error("failed to begin transaction", zap.Error(err))
		return
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO audit_logs (id, timestamp, category, action, user_id, user_ip, resource_type, resource_id, status, details, error, duration, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		m.logger.Error("failed to prepare statement", zap.Error(err))
		return
	}
	defer stmt.Close()

	for _, log := range logs {
		detailsJSON, _ := json.Marshal(log.Details)
		_, err := stmt.Exec(
			log.ID,
			log.Timestamp,
			log.Category,
			log.Action,
			log.UserID,
			log.UserIP,
			log.ResourceType,
			log.ResourceID,
			log.Status,
			string(detailsJSON),
			log.Error,
			log.Duration,
			log.RequestID,
		)
		if err != nil {
			m.logger.Error("failed to insert audit log", zap.Error(err))
		}
	}

	if err := tx.Commit(); err != nil {
		m.logger.Error("failed to commit transaction", zap.Error(err))
	}
}

// Log logs an audit event synchronously.
func (m *Manager) Log(log *Log) error {
	if log.ID == "" {
		log.ID = generateLogID()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now().UTC()
	}

	detailsJSON, _ := json.Marshal(log.Details)
	_, err := m.db.Exec(`
		INSERT INTO audit_logs (id, timestamp, category, action, user_id, user_ip, resource_type, resource_id, status, details, error, duration, request_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.ID, log.Timestamp, log.Category, log.Action, log.UserID, log.UserIP, log.ResourceType, log.ResourceID, log.Status, string(detailsJSON), log.Error, log.Duration, log.RequestID)

	if err != nil {
		m.logger.Error("failed to log audit event", zap.Error(err))
		return err
	}

	return nil
}

// LogAsync logs an audit event asynchronously.
func (m *Manager) LogAsync(log *Log) {
	if log.ID == "" {
		log.ID = generateLogID()
	}
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now().UTC()
	}

	select {
	case m.buffer <- log:
	default:
		// Buffer full, drop the log
		m.logger.Warn("audit log buffer full, dropping log")
	}
}

// Query queries audit logs.
func (m *Manager) Query(opts QueryOptions) ([]*Log, error) {
	query := `SELECT id, timestamp, category, action, user_id, user_ip, resource_type, resource_id, status, details, error, duration, request_id FROM audit_logs WHERE 1=1`
	args := []interface{}{}

	if opts.Category != "" {
		query += " AND category = ?"
		args = append(args, opts.Category)
	}
	if opts.Action != "" {
		query += " AND action = ?"
		args = append(args, opts.Action)
	}
	if opts.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, opts.UserID)
	}
	if opts.ResourceType != "" {
		query += " AND resource_type = ?"
		args = append(args, opts.ResourceType)
	}
	if opts.ResourceID != "" {
		query += " AND resource_id = ?"
		args = append(args, opts.ResourceID)
	}
	if opts.Status != "" {
		query += " AND status = ?"
		args = append(args, opts.Status)
	}
	if opts.StartTime != nil {
		query += " AND timestamp >= ?"
		args = append(args, *opts.StartTime)
	}
	if opts.EndTime != nil {
		query += " AND timestamp <= ?"
		args = append(args, *opts.EndTime)
	}

	query += " ORDER BY timestamp DESC"

	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}
	if opts.Offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", opts.Offset)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*Log
	for rows.Next() {
		log := &Log{}
		var detailsJSON string
		err := rows.Scan(&log.ID, &log.Timestamp, &log.Category, &log.Action, &log.UserID, &log.UserIP, &log.ResourceType, &log.ResourceID, &log.Status, &detailsJSON, &log.Error, &log.Duration, &log.RequestID)
		if err != nil {
			m.logger.Warn("failed to scan audit log", zap.Error(err))
			continue
		}
		if detailsJSON != "" {
			json.Unmarshal([]byte(detailsJSON), &log.Details)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// GetStats returns audit log statistics.
func (m *Manager) GetStats(opts QueryOptions) (*Stats, error) {
	stats := &Stats{
		ByCategory: make(map[string]int64),
		ByAction:   make(map[string]int64),
		ByUser:     make(map[string]int64),
		ByDay:      make(map[string]int64),
	}

	// Total count
	err := m.db.QueryRow(`SELECT COUNT(*) FROM audit_logs`).Scan(&stats.TotalCount)
	if err != nil {
		return nil, err
	}

	// Success/Failed count
	m.db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE status = 'success'`).Scan(&stats.SuccessCount)
	m.db.QueryRow(`SELECT COUNT(*) FROM audit_logs WHERE status = 'failed'`).Scan(&stats.FailedCount)

	// By category
	rows, err := m.db.Query(`SELECT category, COUNT(*) FROM audit_logs GROUP BY category`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int64
			if err := rows.Scan(&category, &count); err != nil {
				m.logger.Warn("failed to scan category stat", zap.Error(err))
				continue
			}
			stats.ByCategory[category] = count
		}
		if err := rows.Err(); err != nil {
			m.logger.Warn("rows error in category stats", zap.Error(err))
		}
	}

	// By action
	rows, err = m.db.Query(`SELECT action, COUNT(*) FROM audit_logs GROUP BY action`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var action string
			var count int64
			if err := rows.Scan(&action, &count); err != nil {
				m.logger.Warn("failed to scan action stat", zap.Error(err))
				continue
			}
			stats.ByAction[action] = count
		}
		if err := rows.Err(); err != nil {
			m.logger.Warn("rows error in action stats", zap.Error(err))
		}
	}

	// By user
	rows, err = m.db.Query(`SELECT user_id, COUNT(*) FROM audit_logs WHERE user_id IS NOT NULL GROUP BY user_id`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var userID string
			var count int64
			if err := rows.Scan(&userID, &count); err != nil {
				m.logger.Warn("failed to scan user stat", zap.Error(err))
				continue
			}
			stats.ByUser[userID] = count
		}
		if err := rows.Err(); err != nil {
			m.logger.Warn("rows error in user stats", zap.Error(err))
		}
	}

	// By day (last 30 days)
	rows, err = m.db.Query(`SELECT date(timestamp), COUNT(*) FROM audit_logs WHERE timestamp >= datetime('now', '-30 days') GROUP BY date(timestamp)`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var day string
			var count int64
			if err := rows.Scan(&day, &count); err != nil {
				m.logger.Warn("failed to scan day stat", zap.Error(err))
				continue
			}
			stats.ByDay[day] = count
		}
		if err := rows.Err(); err != nil {
			m.logger.Warn("rows error in day stats", zap.Error(err))
		}
	}

	return stats, nil
}

// Export exports audit logs to a file.
func (m *Manager) Export(opts ExportOptions) (string, error) {
	queryOpts := QueryOptions{
		Category:  opts.Category,
		UserID:    opts.UserID,
		StartTime: opts.StartTime,
		EndTime:   opts.EndTime,
		Limit:     10000, // Max 10000 records
	}

	logs, err := m.Query(queryOpts)
	if err != nil {
		return "", err
	}

	filename := fmt.Sprintf("audit_logs_%s.%s", time.Now().Format("20060102_150405"), opts.Format)
	filepath := filepath.Join(os.TempDir(), filename)

	switch opts.Format {
	case "csv":
		err = m.exportCSV(filepath, logs)
	case "json":
		err = m.exportJSON(filepath, logs)
	default:
		return "", fmt.Errorf("unsupported format: %s", opts.Format)
	}

	if err != nil {
		return "", err
	}

	return filepath, nil
}

// exportCSV exports logs to CSV format.
func (m *Manager) exportCSV(filepath string, logs []*Log) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Header
	writer.Write([]string{"ID", "Timestamp", "Category", "Action", "UserID", "UserIP", "ResourceType", "ResourceID", "Status", "Error", "Duration", "RequestID"})

	// Data
	for _, log := range logs {
		details, _ := json.Marshal(log.Details)
		writer.Write([]string{
			log.ID,
			log.Timestamp.Format(time.RFC3339),
			string(log.Category),
			string(log.Action),
			log.UserID,
			log.UserIP,
			log.ResourceType,
			log.ResourceID,
			string(log.Status),
			log.Error,
			fmt.Sprintf("%d", log.Duration),
			log.RequestID,
			string(details),
		})
	}

	return nil
}

// exportJSON exports logs to JSON format.
func (m *Manager) exportJSON(filepath string, logs []*Log) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(logs)
}

// Cleanup deletes old audit logs.
func (m *Manager) Cleanup(olderThan time.Duration) error {
	cutoff := time.Now().UTC().Add(-olderThan)
	_, err := m.db.Exec(`DELETE FROM audit_logs WHERE timestamp < ?`, cutoff)
	if err != nil {
		m.logger.Error("failed to cleanup audit logs", zap.Error(err))
		return err
	}
	return nil
}

// Convenience methods for common audit events

// LogAuth logs an authentication event.
func (m *Manager) LogAuth(userID, userIP string, action Action, success bool, details map[string]interface{}) {
	status := StatusSuccess
	if !success {
		status = StatusFailed
	}
	m.LogAsync(&Log{
		Category: CategoryAuth,
		Action:   action,
		UserID:   userID,
		UserIP:   userIP,
		Status:   status,
		Details:  details,
	})
}

// LogAgent logs an agent operation.
func (m *Manager) LogAgent(userID, userIP string, action Action, agentID string, success bool, details map[string]interface{}) {
	status := StatusSuccess
	if !success {
		status = StatusFailed
	}
	m.LogAsync(&Log{
		Category:     CategoryAgent,
		Action:       action,
		UserID:       userID,
		UserIP:       userIP,
		ResourceType: "agent",
		ResourceID:   agentID,
		Status:       status,
		Details:      details,
	})
}

// LogWorkflow logs a workflow operation.
func (m *Manager) LogWorkflow(userID, userIP string, action Action, workflowID string, success bool, details map[string]interface{}) {
	status := StatusSuccess
	if !success {
		status = StatusFailed
	}
	m.LogAsync(&Log{
		Category:     CategoryWorkflow,
		Action:       action,
		UserID:       userID,
		UserIP:       userIP,
		ResourceType: "workflow",
		ResourceID:   workflowID,
		Status:       status,
		Details:      details,
	})
}

// LogTrigger logs a trigger operation.
func (m *Manager) LogTrigger(userID, userIP string, action Action, triggerID string, success bool, details map[string]interface{}) {
	status := StatusSuccess
	if !success {
		status = StatusFailed
	}
	m.LogAsync(&Log{
		Category:     CategoryTrigger,
		Action:       action,
		UserID:       userID,
		UserIP:       userIP,
		ResourceType: "trigger",
		ResourceID:   triggerID,
		Status:       status,
		Details:      details,
	})
}

// LogMCP logs an MCP operation.
func (m *Manager) LogMCP(userID, userIP string, action Action, mcpID string, success bool, details map[string]interface{}) {
	status := StatusSuccess
	if !success {
		status = StatusFailed
	}
	m.LogAsync(&Log{
		Category:     CategoryMCP,
		Action:       action,
		UserID:       userID,
		UserIP:       userIP,
		ResourceType: "mcp",
		ResourceID:   mcpID,
		Status:       status,
		Details:      details,
	})
}

// LogSystem logs a system event.
func (m *Manager) LogSystem(action Action, details map[string]interface{}, err string) {
	status := StatusSuccess
	if err != "" {
		status = StatusFailed
	}
	m.LogAsync(&Log{
		Category: CategorySystem,
		Action:   action,
		Status:   status,
		Details:  details,
		Error:    err,
	})
}

// generateLogID generates a unique log ID.
func generateLogID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "audit_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(b)[:8]
}
