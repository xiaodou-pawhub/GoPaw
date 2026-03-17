// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package queue

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages message queues.
type Manager struct {
	db      *sql.DB
	logger  *zap.Logger
	workers map[string]*Worker
	mu      sync.RWMutex
	stop    chan struct{}
}

// NewManager creates a new queue manager.
func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		db:      db,
		logger:  logger.Named("queue"),
		workers: make(map[string]*Worker),
		stop:    make(chan struct{}),
	}

	if err := m.initSchema(); err != nil {
		return nil, err
	}

	// Start delayed message processor
	go m.processDelayedMessages()

	return m, nil
}

// Close closes the queue manager.
func (m *Manager) Close() error {
	close(m.stop)

	// Stop all workers
	m.mu.RLock()
	for _, worker := range m.workers {
		worker.Stop()
	}
	m.mu.RUnlock()

	return nil
}

// initSchema creates the database tables.
func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    queue TEXT NOT NULL,
    type TEXT NOT NULL,
    payload TEXT NOT NULL,
    priority INTEGER DEFAULT 5,
    status TEXT DEFAULT 'pending',
    attempts INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    delay_until TIMESTAMP,
    processed_by TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT
);

CREATE INDEX IF NOT EXISTS idx_messages_queue_status ON messages(queue, status);
CREATE INDEX IF NOT EXISTS idx_messages_priority ON messages(priority);
CREATE INDEX IF NOT EXISTS idx_messages_delay ON messages(delay_until);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_type ON messages(type);

CREATE TABLE IF NOT EXISTS message_stats (
    queue TEXT PRIMARY KEY,
    pending_count INTEGER DEFAULT 0,
    processing_count INTEGER DEFAULT 0,
    completed_count INTEGER DEFAULT 0,
    failed_count INTEGER DEFAULT 0,
    delayed_count INTEGER DEFAULT 0,
    total_count INTEGER DEFAULT 0,
    updated_at TIMESTAMP
);
`
	_, err := m.db.Exec(schema)
	return err
}

// Publish publishes a message to a queue.
func (m *Manager) Publish(queue string, msgType string, payload map[string]interface{}, opts *PublishOptions) (*Message, error) {
	if opts == nil {
		defaultOpts := DefaultPublishOptions()
		opts = &defaultOpts
	}

	now := time.Now().UTC()
	msg := &Message{
		ID:         generateMessageID(),
		Queue:      queue,
		Type:       msgType,
		Payload:    payload,
		Priority:   opts.Priority,
		Status:     MessageStatusPending,
		MaxRetries: opts.MaxRetries,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if opts.Delay > 0 {
		delayUntil := now.Add(opts.Delay)
		msg.DelayUntil = &delayUntil
		msg.Status = MessageStatusDelayed
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = m.db.Exec(`
		INSERT INTO messages (id, queue, type, payload, priority, status, max_retries, delay_until, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.Queue, msg.Type, string(payloadJSON), msg.Priority, msg.Status, msg.MaxRetries, msg.DelayUntil, msg.CreatedAt, msg.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to insert message: %w", err)
	}

	m.logger.Debug("message published",
		zap.String("id", msg.ID),
		zap.String("queue", msg.Queue),
		zap.String("type", msg.Type))

	return msg, nil
}

// GetMessage retrieves a message by ID.
func (m *Manager) GetMessage(id string) (*Message, error) {
	msg := &Message{}
	var payloadJSON string

	err := m.db.QueryRow(`
		SELECT id, queue, type, payload, priority, status, attempts, max_retries, delay_until, processed_by,
		       created_at, updated_at, processed_at, completed_at, error
		FROM messages
		WHERE id = ?
	`, id).Scan(&msg.ID, &msg.Queue, &msg.Type, &payloadJSON, &msg.Priority, &msg.Status, &msg.Attempts, &msg.MaxRetries,
		&msg.DelayUntil, &msg.ProcessedBy, &msg.CreatedAt, &msg.UpdatedAt, &msg.ProcessedAt, &msg.CompletedAt, &msg.Error)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found: %s", id)
		}
		return nil, err
	}

	if payloadJSON != "" {
		json.Unmarshal([]byte(payloadJSON), &msg.Payload)
	}

	return msg, nil
}

// Dequeue retrieves the next available message from a queue.
func (m *Manager) Dequeue(queue string, workerID string) (*Message, error) {
	now := time.Now().UTC()

	// Start transaction
	tx, err := m.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Get the next pending message (ordered by priority, then created_at)
	var msg Message
	var payloadJSON string

	err = tx.QueryRow(`
		SELECT id, queue, type, payload, priority, status, attempts, max_retries, delay_until
		FROM messages
		WHERE queue = ? AND status = 'pending' AND (delay_until IS NULL OR delay_until <= ?)
		ORDER BY priority ASC, created_at ASC
		LIMIT 1
	`, queue, now).Scan(&msg.ID, &msg.Queue, &msg.Type, &payloadJSON, &msg.Priority, &msg.Status, &msg.Attempts, &msg.MaxRetries, &msg.DelayUntil)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No message available
		}
		return nil, err
	}

	// Update status to processing
	processedAt := now
	_, err = tx.Exec(`
		UPDATE messages
		SET status = 'processing', processed_by = ?, processed_at = ?, updated_at = ?, attempts = attempts + 1
		WHERE id = ?
	`, workerID, processedAt, now, msg.ID)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	msg.Status = MessageStatusProcessing
	msg.ProcessedBy = workerID
	msg.ProcessedAt = &processedAt
	msg.Attempts++

	if payloadJSON != "" {
		json.Unmarshal([]byte(payloadJSON), &msg.Payload)
	}

	return &msg, nil
}

// Complete marks a message as completed.
func (m *Manager) Complete(msgID string) error {
	now := time.Now().UTC()
	_, err := m.db.Exec(`
		UPDATE messages
		SET status = 'completed', completed_at = ?, updated_at = ?
		WHERE id = ?
	`, now, now, msgID)
	return err
}

// Fail marks a message as failed.
func (m *Manager) Fail(msgID string, errMsg string) error {
	now := time.Now().UTC()
	_, err := m.db.Exec(`
		UPDATE messages
		SET status = 'failed', error = ?, updated_at = ?
		WHERE id = ?
	`, errMsg, now, msgID)
	return err
}

// Retry marks a message for retry.
func (m *Manager) Retry(msgID string) error {
	now := time.Now().UTC()
	_, err := m.db.Exec(`
		UPDATE messages
		SET status = 'pending', error = '', updated_at = ?
		WHERE id = ?
	`, now, msgID)
	return err
}

// Delete deletes a message.
func (m *Manager) Delete(msgID string) error {
	_, err := m.db.Exec(`DELETE FROM messages WHERE id = ?`, msgID)
	return err
}

// ListMessages lists messages in a queue.
func (m *Manager) ListMessages(queue string, status string, limit int) ([]*Message, error) {
	query := `SELECT id, queue, type, payload, priority, status, attempts, max_retries, delay_until, processed_by,
	          created_at, updated_at, processed_at, completed_at, error
	          FROM messages WHERE queue = ?`
	args := []interface{}{queue}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY priority ASC, created_at ASC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var payloadJSON string

		err := rows.Scan(&msg.ID, &msg.Queue, &msg.Type, &payloadJSON, &msg.Priority, &msg.Status, &msg.Attempts, &msg.MaxRetries,
			&msg.DelayUntil, &msg.ProcessedBy, &msg.CreatedAt, &msg.UpdatedAt, &msg.ProcessedAt, &msg.CompletedAt, &msg.Error)
		if err != nil {
			m.logger.Warn("failed to scan message", zap.Error(err))
			continue
		}

		if payloadJSON != "" {
			json.Unmarshal([]byte(payloadJSON), &msg.Payload)
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

// GetStats returns statistics for a queue.
func (m *Manager) GetStats(queue string) (*Stats, error) {
	stats := &Stats{Queue: queue}

	// Get counts by status
	rows, err := m.db.Query(`
		SELECT status, COUNT(*) FROM messages WHERE queue = ? GROUP BY status
	`, queue)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int64
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		switch MessageStatus(status) {
		case MessageStatusPending:
			stats.PendingCount = count
		case MessageStatusProcessing:
			stats.ProcessingCount = count
		case MessageStatusCompleted:
			stats.CompletedCount = count
		case MessageStatusFailed:
			stats.FailedCount = count
		case MessageStatusDelayed:
			stats.DelayedCount = count
		}
		stats.TotalCount += count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats.UpdatedAt = time.Now().UTC()
	return stats, nil
}

// ListQueues returns all queue names.
func (m *Manager) ListQueues() ([]string, error) {
	rows, err := m.db.Query(`SELECT DISTINCT queue FROM messages`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var queues []string
	for rows.Next() {
		var queue string
		if err := rows.Scan(&queue); err != nil {
			continue
		}
		queues = append(queues, queue)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return queues, nil
}

// processDelayedMessages periodically checks and moves delayed messages to pending.
func (m *Manager) processDelayedMessages() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.moveDelayedMessages()
		case <-m.stop:
			return
		}
	}
}

// moveDelayedMessages moves expired delayed messages to pending.
func (m *Manager) moveDelayedMessages() {
	now := time.Now().UTC()
	_, err := m.db.Exec(`
		UPDATE messages
		SET status = 'pending', delay_until = NULL, updated_at = ?
		WHERE status = 'delayed' AND delay_until <= ?
	`, now, now)

	if err != nil {
		m.logger.Warn("failed to move delayed messages", zap.Error(err))
	}
}

// generateMessageID generates a unique message ID.
func generateMessageID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return "msg_" + time.Now().Format("20060102150405") + "_" + hex.EncodeToString(b)[:8]
}
