// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trigger

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages trigger configurations.
type Manager struct {
	db      *sql.DB
	logger  *zap.Logger
	triggers map[string]*Trigger
	mu       sync.RWMutex
}

// NewManager creates a new trigger manager.
func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		db:       db,
		logger:   logger.Named("trigger_manager"),
		triggers: make(map[string]*Trigger),
	}

	if err := m.initSchema(); err != nil {
		return nil, err
	}

	if err := m.LoadTriggers(); err != nil {
		return nil, err
	}

	return m, nil
}

// initSchema creates the database tables.
func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS triggers (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    type TEXT NOT NULL,
    config TEXT NOT NULL,
    reason TEXT,
    is_enabled BOOLEAN DEFAULT 1,
    last_fired_at TIMESTAMP,
    fire_count INTEGER DEFAULT 0,
    max_fires INTEGER,
    cooldown_seconds INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS trigger_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    trigger_id TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    fired_at TIMESTAMP NOT NULL,
    payload TEXT,
    success BOOLEAN DEFAULT 1,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_triggers_agent ON triggers(agent_id);
CREATE INDEX IF NOT EXISTS idx_triggers_enabled ON triggers(is_enabled);
CREATE INDEX IF NOT EXISTS idx_trigger_history_trigger ON trigger_history(trigger_id);
CREATE INDEX IF NOT EXISTS idx_trigger_history_fired ON trigger_history(fired_at);
`
	_, err := m.db.Exec(schema)
	return err
}

// LoadTriggers loads all triggers from database.
func (m *Manager) LoadTriggers() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.triggers = make(map[string]*Trigger)

	rows, err := m.db.Query(`
		SELECT id, agent_id, name, description, type, config, reason, is_enabled, 
		       last_fired_at, fire_count, max_fires, cooldown_seconds, created_at, updated_at
		FROM triggers
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		trigger := &Trigger{}
		var configJSON string
		var maxFires sql.NullInt64
		err := rows.Scan(
			&trigger.ID, &trigger.AgentID, &trigger.Name, &trigger.Description,
			&trigger.Type, &configJSON, &trigger.Reason, &trigger.IsEnabled,
			&trigger.LastFiredAt, &trigger.FireCount, &maxFires, &trigger.CooldownSec,
			&trigger.CreatedAt, &trigger.UpdatedAt,
		)
		if err != nil {
			m.logger.Warn("failed to scan trigger", zap.Error(err))
			continue
		}

		if maxFires.Valid {
			trigger.MaxFires = &[]int{int(maxFires.Int64)}[0]
		}

		// Parse config
		config, err := ParseConfig(trigger.Type, []byte(configJSON))
		if err != nil {
			m.logger.Warn("failed to parse trigger config", zap.String("trigger_id", trigger.ID), zap.Error(err))
			continue
		}
		trigger.Config = config

		m.triggers[trigger.ID] = trigger
	}

	m.logger.Info("triggers loaded", zap.Int("count", len(m.triggers)))
	return nil
}

// Get returns a trigger by ID.
func (m *Manager) Get(id string) (*Trigger, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	trigger, ok := m.triggers[id]
	if !ok {
		return nil, fmt.Errorf("trigger not found: %s", id)
	}

	return trigger, nil
}

// List returns all triggers.
func (m *Manager) List() []*Trigger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Trigger, 0, len(m.triggers))
	for _, trigger := range m.triggers {
		result = append(result, trigger)
	}
	return result
}

// ListEnabled returns all enabled triggers.
func (m *Manager) ListEnabled() []*Trigger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Trigger, 0)
	for _, trigger := range m.triggers {
		if trigger.IsEnabled {
			result = append(result, trigger)
		}
	}
	return result
}

// ListByAgent returns triggers for a specific agent.
func (m *Manager) ListByAgent(agentID string) []*Trigger {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Trigger, 0)
	for _, trigger := range m.triggers {
		if trigger.AgentID == agentID {
			result = append(result, trigger)
		}
	}
	return result
}

// Create creates a new trigger.
func (m *Manager) Create(trigger *Trigger) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.triggers[trigger.ID]; exists {
		return fmt.Errorf("trigger already exists: %s", trigger.ID)
	}

	now := time.Now().UTC()
	trigger.CreatedAt = now
	trigger.UpdatedAt = now
	trigger.FireCount = 0

	// Marshal config
	configJSON, err := MarshalConfig(trigger.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var maxFires interface{}
	if trigger.MaxFires != nil {
		maxFires = *trigger.MaxFires
	} else {
		maxFires = nil
	}

	_, err = m.db.Exec(`
		INSERT INTO triggers (id, agent_id, name, description, type, config, reason, is_enabled, 
		                      fire_count, max_fires, cooldown_seconds, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, trigger.ID, trigger.AgentID, trigger.Name, trigger.Description,
		trigger.Type, string(configJSON), trigger.Reason, trigger.IsEnabled,
		trigger.FireCount, maxFires, trigger.CooldownSec, trigger.CreatedAt, trigger.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert trigger: %w", err)
	}

	m.triggers[trigger.ID] = trigger

	m.logger.Info("trigger created",
		zap.String("id", trigger.ID),
		zap.String("name", trigger.Name),
		zap.String("type", trigger.Type),
	)
	return nil
}

// Update updates an existing trigger.
func (m *Manager) Update(id string, trigger *Trigger) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.triggers[id]
	if !ok {
		return fmt.Errorf("trigger not found: %s", id)
	}

	trigger.ID = id
	trigger.CreatedAt = existing.CreatedAt
	trigger.UpdatedAt = time.Now().UTC()
	trigger.FireCount = existing.FireCount
	trigger.LastFiredAt = existing.LastFiredAt

	// Marshal config
	configJSON, err := MarshalConfig(trigger.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	var maxFires interface{}
	if trigger.MaxFires != nil {
		maxFires = *trigger.MaxFires
	} else {
		maxFires = nil
	}

	_, err = m.db.Exec(`
		UPDATE triggers
		SET name = ?, description = ?, type = ?, config = ?, reason = ?, is_enabled = ?,
		    max_fires = ?, cooldown_seconds = ?, updated_at = ?
		WHERE id = ?
	`, trigger.Name, trigger.Description, trigger.Type, string(configJSON),
		trigger.Reason, trigger.IsEnabled, maxFires, trigger.CooldownSec,
		trigger.UpdatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update trigger: %w", err)
	}

	m.triggers[id] = trigger

	m.logger.Info("trigger updated", zap.String("id", id))
	return nil
}

// Delete deletes a trigger.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.triggers[id]; !ok {
		return fmt.Errorf("trigger not found: %s", id)
	}

	_, err := m.db.Exec(`DELETE FROM triggers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete trigger: %w", err)
	}

	delete(m.triggers, id)

	m.logger.Info("trigger deleted", zap.String("id", id))
	return nil
}

// SetEnabled enables or disables a trigger.
func (m *Manager) SetEnabled(id string, enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	trigger, ok := m.triggers[id]
	if !ok {
		return fmt.Errorf("trigger not found: %s", id)
	}

	trigger.IsEnabled = enabled
	trigger.UpdatedAt = time.Now().UTC()

	_, err := m.db.Exec(`UPDATE triggers SET is_enabled = ?, updated_at = ? WHERE id = ?`,
		enabled, trigger.UpdatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update trigger: %w", err)
	}

	m.logger.Info("trigger status changed", zap.String("id", id), zap.Bool("enabled", enabled))
	return nil
}

// RecordFire records a trigger fire event.
func (m *Manager) RecordFire(triggerID, agentID string, payload map[string]interface{}, success bool, errorMsg string) error {
	payloadJSON, _ := json.Marshal(payload)

	_, err := m.db.Exec(`
		INSERT INTO trigger_history (trigger_id, agent_id, fired_at, payload, success, error_message)
		VALUES (?, ?, ?, ?, ?, ?)
	`, triggerID, agentID, time.Now().UTC(), string(payloadJSON), success, errorMsg)
	if err != nil {
		return fmt.Errorf("failed to record trigger fire: %w", err)
	}

	// Update trigger stats
	_, err = m.db.Exec(`
		UPDATE triggers 
		SET fire_count = fire_count + 1, last_fired_at = ?
		WHERE id = ?
	`, time.Now().UTC(), triggerID)
	if err != nil {
		m.logger.Warn("failed to update trigger stats", zap.Error(err))
	}

	return nil
}

// GetHistory returns trigger fire history.
func (m *Manager) GetHistory(triggerID string, limit int) ([]*History, error) {
	rows, err := m.db.Query(`
		SELECT id, trigger_id, agent_id, fired_at, payload, success, error_message
		FROM trigger_history
		WHERE trigger_id = ?
		ORDER BY fired_at DESC
		LIMIT ?
	`, triggerID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query trigger history: %w", err)
	}
	defer rows.Close()

	var history []*History
	for rows.Next() {
		h := &History{}
		err := rows.Scan(&h.ID, &h.TriggerID, &h.AgentID, &h.FiredAt, &h.Payload, &h.Success, &h.ErrorMessage)
		if err != nil {
			m.logger.Warn("failed to scan history", zap.Error(err))
			continue
		}
		history = append(history, h)
	}

	return history, nil
}

// UpdateLastFired updates the last fired time for a trigger.
func (m *Manager) UpdateLastFired(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	trigger, ok := m.triggers[id]
	if !ok {
		return fmt.Errorf("trigger not found: %s", id)
	}

	now := time.Now().UTC()
	trigger.LastFiredAt = &now
	trigger.FireCount++

	_, err := m.db.Exec(`
		UPDATE triggers SET last_fired_at = ?, fire_count = fire_count + 1 WHERE id = ?
	`, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last fired: %w", err)
	}

	return nil
}
