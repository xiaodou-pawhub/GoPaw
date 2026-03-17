// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package message

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages agent-to-agent messages.
type Manager struct {
	db       *sql.DB
	logger   *zap.Logger
	handlers map[string]MessageHandler // agent_id -> handler
	mu       sync.RWMutex
}

// MessageHandler is called when a message is received for an agent.
type MessageHandler func(msg *Message) error

// NewManager creates a new message manager.
func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		db:       db,
		logger:   logger.Named("agent_message_manager"),
		handlers: make(map[string]MessageHandler),
	}

	if err := m.initSchema(); err != nil {
		return nil, err
	}

	return m, nil
}

// initSchema creates the database tables.
func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS agent_messages (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    from_agent TEXT NOT NULL,
    to_agent TEXT NOT NULL,
    content TEXT NOT NULL,
    payload TEXT,
    parent_id TEXT,
    status TEXT DEFAULT 'pending',
    error TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    processed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_conversations (
    id TEXT PRIMARY KEY,
    agent_ids TEXT NOT NULL, -- JSON array
    title TEXT,
    message_count INTEGER DEFAULT 0,
    last_message_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_to_agent ON agent_messages(to_agent);
CREATE INDEX IF NOT EXISTS idx_messages_from_agent ON agent_messages(from_agent);
CREATE INDEX IF NOT EXISTS idx_messages_status ON agent_messages(status);
CREATE INDEX IF NOT EXISTS idx_messages_created ON agent_messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_parent ON agent_messages(parent_id);
CREATE INDEX IF NOT EXISTS idx_conversations_agents ON agent_conversations(agent_ids);
CREATE INDEX IF NOT EXISTS idx_conversations_updated ON agent_conversations(updated_at);
`
	_, err := m.db.Exec(schema)
	return err
}

// Send sends a message from one agent to another.
func (m *Manager) Send(msg *Message) error {
	// Insert message
	_, err := m.db.Exec(`
		INSERT INTO agent_messages (id, type, from_agent, to_agent, content, payload, parent_id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, msg.ID, msg.Type, msg.FromAgent, msg.ToAgent, msg.Content, msg.Payload, msg.ParentID, msg.Status, msg.CreatedAt, msg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert message: %w", err)
	}

	// Update or create conversation
	if err := m.updateConversation(msg); err != nil {
		m.logger.Warn("failed to update conversation", zap.Error(err))
	}

	m.logger.Info("message sent",
		zap.String("id", msg.ID),
		zap.String("type", string(msg.Type)),
		zap.String("from", msg.FromAgent),
		zap.String("to", msg.ToAgent))

	// Notify handler if registered
	m.mu.RLock()
	handler, ok := m.handlers[msg.ToAgent]
	m.mu.RUnlock()

	if ok {
		go func() {
			if err := handler(msg); err != nil {
				m.logger.Error("message handler failed", zap.Error(err))
				m.UpdateStatus(msg.ID, MessageStatusFailed, err.Error())
			}
		}()
	}

	return nil
}

// SendTask sends a task message.
func (m *Manager) SendTask(fromAgent, toAgent, description string, payload *TaskPayload) (*Message, error) {
	msg, err := NewMessage(MessageTypeTask, fromAgent, toAgent, description, payload)
	if err != nil {
		return nil, err
	}
	return msg, m.Send(msg)
}

// SendResponse sends a response message.
func (m *Manager) SendResponse(fromAgent, toAgent string, inReplyTo string, success bool, message string, data map[string]interface{}) (*Message, error) {
	payload := &ResponsePayload{
		InReplyTo: inReplyTo,
		Success:   success,
		Message:   message,
		Data:      data,
	}
	msg, err := NewMessage(MessageTypeResponse, fromAgent, toAgent, message, payload)
	if err != nil {
		return nil, err
	}
	msg.ParentID = inReplyTo
	return msg, m.Send(msg)
}

// SendNotify sends a notification message.
func (m *Manager) SendNotify(fromAgent, toAgent, event string, details map[string]interface{}) (*Message, error) {
	payload := &NotifyPayload{
		Event:   event,
		Details: details,
	}
	content := fmt.Sprintf("Event: %s", event)
	msg, err := NewMessage(MessageTypeNotify, fromAgent, toAgent, content, payload)
	if err != nil {
		return nil, err
	}
	return msg, m.Send(msg)
}

// SendQuery sends a query message.
func (m *Manager) SendQuery(fromAgent, toAgent, question string, context map[string]interface{}) (*Message, error) {
	payload := &QueryPayload{
		QueryID:  generateMessageID(),
		Question: question,
		Context:  context,
	}
	msg, err := NewMessage(MessageTypeQuery, fromAgent, toAgent, question, payload)
	if err != nil {
		return nil, err
	}
	return msg, m.Send(msg)
}

// Get retrieves a message by ID.
func (m *Manager) Get(id string) (*Message, error) {
	msg := &Message{}
	var payload sql.NullString
	err := m.db.QueryRow(`
		SELECT id, type, from_agent, to_agent, content, payload, parent_id, status, error, created_at, updated_at, processed_at
		FROM agent_messages
		WHERE id = ?
	`, id).Scan(&msg.ID, &msg.Type, &msg.FromAgent, &msg.ToAgent, &msg.Content, &payload, &msg.ParentID, &msg.Status, &msg.Error, &msg.CreatedAt, &msg.UpdatedAt, &msg.ProcessedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("message not found: %s", id)
		}
		return nil, err
	}
	if payload.Valid {
		msg.Payload = json.RawMessage(payload.String)
	}
	return msg, nil
}

// ListMessages lists messages for an agent.
func (m *Manager) ListMessages(agentID string, limit int, status string) ([]*Message, error) {
	query := `
		SELECT id, type, from_agent, to_agent, content, payload, parent_id, status, error, created_at, updated_at, processed_at
		FROM agent_messages
		WHERE to_agent = ?
	`
	args := []interface{}{agentID}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := m.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var payload sql.NullString
		err := rows.Scan(&msg.ID, &msg.Type, &msg.FromAgent, &msg.ToAgent, &msg.Content, &payload, &msg.ParentID, &msg.Status, &msg.Error, &msg.CreatedAt, &msg.UpdatedAt, &msg.ProcessedAt)
		if err != nil {
			m.logger.Warn("failed to scan message", zap.Error(err))
			continue
		}
		if payload.Valid {
			msg.Payload = json.RawMessage(payload.String)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ListSentMessages lists messages sent by an agent.
func (m *Manager) ListSentMessages(agentID string, limit int) ([]*Message, error) {
	rows, err := m.db.Query(`
		SELECT id, type, from_agent, to_agent, content, payload, parent_id, status, error, created_at, updated_at, processed_at
		FROM agent_messages
		WHERE from_agent = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, agentID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var payload sql.NullString
		err := rows.Scan(&msg.ID, &msg.Type, &msg.FromAgent, &msg.ToAgent, &msg.Content, &payload, &msg.ParentID, &msg.Status, &msg.Error, &msg.CreatedAt, &msg.UpdatedAt, &msg.ProcessedAt)
		if err != nil {
			m.logger.Warn("failed to scan message", zap.Error(err))
			continue
		}
		if payload.Valid {
			msg.Payload = json.RawMessage(payload.String)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ListConversation lists messages in a conversation thread.
func (m *Manager) ListConversation(parentID string) ([]*Message, error) {
	rows, err := m.db.Query(`
		SELECT id, type, from_agent, to_agent, content, payload, parent_id, status, error, created_at, updated_at, processed_at
		FROM agent_messages
		WHERE parent_id = ? OR id = ?
		ORDER BY created_at ASC
	`, parentID, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var payload sql.NullString
		err := rows.Scan(&msg.ID, &msg.Type, &msg.FromAgent, &msg.ToAgent, &msg.Content, &payload, &msg.ParentID, &msg.Status, &msg.Error, &msg.CreatedAt, &msg.UpdatedAt, &msg.ProcessedAt)
		if err != nil {
			m.logger.Warn("failed to scan message", zap.Error(err))
			continue
		}
		if payload.Valid {
			msg.Payload = json.RawMessage(payload.String)
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// UpdateStatus updates the status of a message.
func (m *Manager) UpdateStatus(id string, status MessageStatus, error string) error {
	now := time.Now().UTC()
	var processedAt interface{}
	if status == MessageStatusCompleted || status == MessageStatusFailed {
		processedAt = now
	}

	_, err := m.db.Exec(`
		UPDATE agent_messages
		SET status = ?, error = ?, updated_at = ?, processed_at = ?
		WHERE id = ?
	`, status, error, now, processedAt, id)
	return err
}

// GetPendingMessages gets pending messages for an agent.
func (m *Manager) GetPendingMessages(agentID string, limit int) ([]*Message, error) {
	return m.ListMessages(agentID, limit, string(MessageStatusPending))
}

// RegisterHandler registers a handler for incoming messages.
func (m *Manager) RegisterHandler(agentID string, handler MessageHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[agentID] = handler
}

// UnregisterHandler unregisters a handler.
func (m *Manager) UnregisterHandler(agentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.handlers, agentID)
}

// GetStats gets message statistics for an agent.
func (m *Manager) GetStats(agentID string) (*MessageStats, error) {
	stats := &MessageStats{}

	// Total sent
	err := m.db.QueryRow(`SELECT COUNT(*) FROM agent_messages WHERE from_agent = ?`, agentID).Scan(&stats.TotalSent)
	if err != nil {
		return nil, err
	}

	// Total received
	err = m.db.QueryRow(`SELECT COUNT(*) FROM agent_messages WHERE to_agent = ?`, agentID).Scan(&stats.TotalReceived)
	if err != nil {
		return nil, err
	}

	// Pending count
	err = m.db.QueryRow(`SELECT COUNT(*) FROM agent_messages WHERE to_agent = ? AND status = 'pending'`, agentID).Scan(&stats.PendingCount)
	if err != nil {
		return nil, err
	}

	// Failed count
	err = m.db.QueryRow(`SELECT COUNT(*) FROM agent_messages WHERE to_agent = ? AND status = 'failed'`, agentID).Scan(&stats.FailedCount)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// updateConversation updates or creates a conversation.
func (m *Manager) updateConversation(msg *Message) error {
	// Generate conversation ID from sorted agent IDs
	agentIDs := []string{msg.FromAgent, msg.ToAgent}
	if agentIDs[0] > agentIDs[1] {
		agentIDs[0], agentIDs[1] = agentIDs[1], agentIDs[0]
	}
	convID := fmt.Sprintf("conv_%s_%s", agentIDs[0], agentIDs[1])

	agentIDsJSON, _ := json.Marshal(agentIDs)
	now := time.Now().UTC()

	// Try to update existing conversation
	result, err := m.db.Exec(`
		UPDATE agent_conversations
		SET message_count = message_count + 1, last_message_at = ?, updated_at = ?
		WHERE id = ?
	`, now, now, convID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		// Create new conversation
		title := fmt.Sprintf("%s ↔ %s", msg.FromAgent, msg.ToAgent)
		_, err = m.db.Exec(`
			INSERT INTO agent_conversations (id, agent_ids, title, message_count, last_message_at, created_at, updated_at)
			VALUES (?, ?, ?, 1, ?, ?, ?)
		`, convID, string(agentIDsJSON), title, now, now, now)
		if err != nil {
			return err
		}
	}

	return nil
}

// ListConversations lists conversations for an agent.
func (m *Manager) ListConversations(agentID string, limit int) ([]*Conversation, error) {
	rows, err := m.db.Query(`
		SELECT id, agent_ids, title, message_count, last_message_at, created_at, updated_at
		FROM agent_conversations
		WHERE agent_ids LIKE ?
		ORDER BY last_message_at DESC
		LIMIT ?
	`, fmt.Sprintf("%%%s%%", agentID), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversations []*Conversation
	for rows.Next() {
		conv := &Conversation{}
		var agentIDsJSON string
		err := rows.Scan(&conv.ID, &agentIDsJSON, &conv.Title, &conv.MessageCount, &conv.LastMessageAt, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			m.logger.Warn("failed to scan conversation", zap.Error(err))
			continue
		}
		json.Unmarshal([]byte(agentIDsJSON), &conv.AgentIDs)
		conversations = append(conversations, conv)
	}

	return conversations, nil
}
