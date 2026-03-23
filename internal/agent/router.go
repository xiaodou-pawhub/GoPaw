// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package agent

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Router manages agent instances for sessions and handles agent switching.
type Router struct {
	manager      *Manager
	factory      *Factory
	instances    map[string]*ReActAgent // agent_id -> instance
	sessionAgent map[string]string      // session_id -> agent_id
	db           *sql.DB
	logger       *zap.Logger
	mu           sync.RWMutex
}

// NewRouter creates a new agent router.
func NewRouter(manager *Manager, factory *Factory, dbPath string, logger *zap.Logger) (*Router, error) {
	// Open database for session-agent bindings
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session-agent database: %w", err)
	}

	r := &Router{
		manager:      manager,
		factory:      factory,
		instances:    make(map[string]*ReActAgent),
		sessionAgent: make(map[string]string),
		db:           db,
		logger:       logger.Named("agent_router"),
	}

	if err := r.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	if err := r.loadSessionBindings(); err != nil {
		db.Close()
		return nil, err
	}

	return r, nil
}

// initSchema creates the database tables.
func (r *Router) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS session_agents (
    session_id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_session_agents_agent ON session_agents(agent_id);
`
	_, err := r.db.Exec(schema)
	return err
}

// loadSessionBindings loads existing session-agent bindings from database.
func (r *Router) loadSessionBindings() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rows, err := r.db.Query(`SELECT session_id, agent_id FROM session_agents`)
	if err != nil {
		return fmt.Errorf("failed to query session agents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var sessionID, agentID string
		if err := rows.Scan(&sessionID, &agentID); err != nil {
			r.logger.Warn("failed to scan session agent", zap.Error(err))
			continue
		}
		r.sessionAgent[sessionID] = agentID
	}

	r.logger.Info("session-agent bindings loaded", zap.Int("count", len(r.sessionAgent)))
	return nil
}

// GetAgentForSession returns the agent instance for a session.
// If no agent is assigned, assigns the default agent.
func (r *Router) GetAgentForSession(sessionID string) (*ReActAgent, string, error) {
	r.mu.RLock()
	agentID, exists := r.sessionAgent[sessionID]
	r.mu.RUnlock()

	if !exists {
		// Assign default agent
		defaultAgent, err := r.manager.GetDefault()
		if err != nil {
			return nil, "", fmt.Errorf("no default agent available: %w", err)
		}
		agentID = defaultAgent.ID
		if err := r.assignAgent(sessionID, agentID); err != nil {
			return nil, "", err
		}
	}

	// Get or create agent instance
	r.mu.RLock()
	instance, ok := r.instances[agentID]
	r.mu.RUnlock()

	if !ok {
		// Create new instance
		def, err := r.manager.Get(agentID)
		if err != nil {
			return nil, "", fmt.Errorf("agent not found: %w", err)
		}

		instance, err = r.factory.CreateAgent(def)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create agent instance: %w", err)
		}

		r.mu.Lock()
		r.instances[agentID] = instance
		r.mu.Unlock()

		r.logger.Info("agent instance created",
			zap.String("agent_id", agentID),
			zap.String("session_id", sessionID),
		)
	}

	return instance, agentID, nil
}

// GetCurrentAgentID returns the current agent ID for a session.
func (r *Router) GetCurrentAgentID(sessionID string) (string, error) {
	r.mu.RLock()
	agentID, exists := r.sessionAgent[sessionID]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("no agent assigned to session: %s", sessionID)
	}

	return agentID, nil
}

// GetAgentDefinition returns the agent definition by ID.
func (r *Router) GetAgentDefinition(agentID string) (*Definition, error) {
	return r.manager.Get(agentID)
}

// GetOrCreateAgent returns or creates an agent instance by ID.
func (r *Router) GetOrCreateAgent(agentID string) (*ReActAgent, error) {
	r.mu.RLock()
	instance, ok := r.instances[agentID]
	r.mu.RUnlock()

	if ok {
		return instance, nil
	}

	// Create new instance
	def, err := r.manager.Get(agentID)
	if err != nil {
		return nil, fmt.Errorf("agent not found: %w", err)
	}

	instance, err = r.factory.CreateAgent(def)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent instance: %w", err)
	}
	instance.id = agentID

	r.mu.Lock()
	r.instances[agentID] = instance
	r.mu.Unlock()

	r.logger.Info("agent instance created for trigger",
		zap.String("agent_id", agentID),
		zap.String("agent_name", def.Name))

	return instance, nil
}

// SwitchAgent switches the agent for a session.
func (r *Router) SwitchAgent(sessionID, agentID string) error {
	// Verify agent exists
	if _, err := r.manager.Get(agentID); err != nil {
		return fmt.Errorf("agent not found: %w", err)
	}

	// Check if already using this agent
	r.mu.RLock()
	currentID, exists := r.sessionAgent[sessionID]
	r.mu.RUnlock()

	if exists && currentID == agentID {
		return nil // Already using this agent
	}

	// Assign new agent
	if err := r.assignAgent(sessionID, agentID); err != nil {
		return err
	}

	r.logger.Info("agent switched",
		zap.String("session_id", sessionID),
		zap.String("from_agent", currentID),
		zap.String("to_agent", agentID),
	)

	return nil
}

// assignAgent assigns an agent to a session and persists the binding.
func (r *Router) assignAgent(sessionID, agentID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Update in-memory map
	r.sessionAgent[sessionID] = agentID

	// Persist to database
	_, err := r.db.Exec(`
		INSERT INTO session_agents (session_id, agent_id, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(session_id) DO UPDATE SET
			agent_id = excluded.agent_id,
			updated_at = excluded.updated_at
	`, sessionID, agentID, time.Now().UTC())

	if err != nil {
		return fmt.Errorf("failed to persist session-agent binding: %w", err)
	}

	return nil
}

// GetSessionCount returns the number of sessions using an agent.
func (r *Router) GetSessionCount(agentID string) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, id := range r.sessionAgent {
		if id == agentID {
			count++
		}
	}
	return count
}

// CleanupSession removes a session's agent binding.
func (r *Router) CleanupSession(sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessionAgent, sessionID)

	_, err := r.db.Exec(`DELETE FROM session_agents WHERE session_id = ?`, sessionID)
	if err != nil {
		return fmt.Errorf("failed to cleanup session: %w", err)
	}

	return nil
}

// Close closes the router and its database connection.
func (r *Router) Close() error {
	return r.db.Close()
}
