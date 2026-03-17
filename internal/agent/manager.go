// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package agent

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Manager manages multiple agent definitions.
type Manager struct {
	db           *sql.DB
	logger       *zap.Logger
	agentsDir    string
	agents       map[string]*Definition
	defaultAgent string
	mu           sync.RWMutex
}

// Definition represents an agent configuration.
type Definition struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Avatar      string    `json:"avatar"`
	ConfigPath  string    `json:"config_path"`
	IsActive    bool      `json:"is_active"`
	IsDefault   bool      `json:"is_default"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Config      *AgentConfig   `json:"config,omitempty"`
}

// NewManager creates a new agent manager.
func NewManager(dbPath string, agentsDir string, logger *zap.Logger) (*Manager, error) {
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agents directory: %w", err)
	}

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open agents database: %w", err)
	}

	m := &Manager{
		db:        db,
		logger:    logger.Named("agent_manager"),
		agentsDir: agentsDir,
		agents:    make(map[string]*Definition),
	}

	if err := m.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	if err := m.LoadAgents(); err != nil {
		db.Close()
		return nil, err
	}

	return m, nil
}

// Close closes the manager and its database connection.
func (m *Manager) Close() error {
	return m.db.Close()
}

// initSchema creates the database tables.
func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    avatar TEXT,
    config_path TEXT NOT NULL,
    is_active BOOLEAN DEFAULT 0,
    is_default BOOLEAN DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_agents_active ON agents(is_active);
CREATE INDEX IF NOT EXISTS idx_agents_default ON agents(is_default);
`
	_, err := m.db.Exec(schema)
	return err
}

// LoadAgents loads all agents from database and config files.
func (m *Manager) LoadAgents() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear existing agents
	m.agents = make(map[string]*Definition)
	m.defaultAgent = ""

	// Load from database
	rows, err := m.db.Query(`
		SELECT id, name, description, avatar, config_path, is_active, is_default, created_at, updated_at
		FROM agents
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query agents: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		def := &Definition{}
		err := rows.Scan(
			&def.ID, &def.Name, &def.Description, &def.Avatar,
			&def.ConfigPath, &def.IsActive, &def.IsDefault,
			&def.CreatedAt, &def.UpdatedAt,
		)
		if err != nil {
			m.logger.Warn("failed to scan agent", zap.Error(err))
			continue
		}

		// Load config from file
		if err := m.loadConfig(def); err != nil {
			m.logger.Warn("failed to load agent config",
				zap.String("agent_id", def.ID),
				zap.Error(err),
			)
		}

		m.agents[def.ID] = def
		if def.IsDefault {
			m.defaultAgent = def.ID
		}
	}

	// Ensure at least one default agent
	if m.defaultAgent == "" && len(m.agents) > 0 {
		// Set the first agent as default
		for id := range m.agents {
			m.defaultAgent = id
			m.setDefaultInDB(id)
			break
		}
	}

	m.logger.Info("agents loaded", zap.Int("count", len(m.agents)))
	return nil
}

// loadConfig loads the YAML config for an agent.
func (m *Manager) loadConfig(def *Definition) error {
	if def.ConfigPath == "" {
		return nil
	}

	data, err := os.ReadFile(def.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	config := &AgentConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	def.Config = config
	return nil
}

// Get returns an agent by ID.
func (m *Manager) Get(id string) (*Definition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	def, ok := m.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}

	return def, nil
}

// GetDefault returns the default agent.
func (m *Manager) GetDefault() (*Definition, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultAgent == "" {
		return nil, fmt.Errorf("no default agent configured")
	}

	def, ok := m.agents[m.defaultAgent]
	if !ok {
		return nil, fmt.Errorf("default agent not found: %s", m.defaultAgent)
	}

	return def, nil
}

// List returns all agents.
func (m *Manager) List() []*Definition {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Definition, 0, len(m.agents))
	for _, def := range m.agents {
		result = append(result, def)
	}
	return result
}

// Create creates a new agent.
func (m *Manager) Create(def *Definition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.agents[def.ID]; exists {
		return fmt.Errorf("agent already exists: %s", def.ID)
	}

	now := time.Now().UTC()
	def.CreatedAt = now
	def.UpdatedAt = now

	// Set config path if not provided
	if def.ConfigPath == "" {
		def.ConfigPath = filepath.Join(m.agentsDir, def.ID+".yaml")
	}

	// Save config to file
	if def.Config != nil {
		if err := m.saveConfig(def); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Insert into database
	_, err := m.db.Exec(`
		INSERT INTO agents (id, name, description, avatar, config_path, is_active, is_default, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, def.ID, def.Name, def.Description, def.Avatar, def.ConfigPath,
		def.IsActive, def.IsDefault, def.CreatedAt, def.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert agent: %w", err)
	}

	m.agents[def.ID] = def

	// If this is the first agent, make it default
	if len(m.agents) == 1 {
		m.defaultAgent = def.ID
		m.setDefaultInDB(def.ID)
	}

	m.logger.Info("agent created",
		zap.String("id", def.ID),
		zap.String("name", def.Name),
	)

	return nil
}

// Update updates an existing agent.
func (m *Manager) Update(id string, def *Definition) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	def.ID = id
	def.CreatedAt = existing.CreatedAt
	def.UpdatedAt = time.Now().UTC()

	// Update config file if changed
	if def.Config != nil {
		if err := m.saveConfig(def); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
	}

	// Update database
	_, err := m.db.Exec(`
		UPDATE agents
		SET name = ?, description = ?, avatar = ?, config_path = ?, is_active = ?, is_default = ?, updated_at = ?
		WHERE id = ?
	`, def.Name, def.Description, def.Avatar, def.ConfigPath,
		def.IsActive, def.IsDefault, def.UpdatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	// Update default agent tracking
	if def.IsDefault && m.defaultAgent != id {
		m.setDefaultInDB(id)
		// Unset previous default
		if m.defaultAgent != "" {
			if prev, ok := m.agents[m.defaultAgent]; ok {
				prev.IsDefault = false
			}
		}
		m.defaultAgent = id
	}

	m.agents[id] = def

	m.logger.Info("agent updated",
		zap.String("id", id),
		zap.String("name", def.Name),
	)

	return nil
}

// Delete deletes an agent.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	def, ok := m.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	// Delete from database
	_, err := m.db.Exec(`DELETE FROM agents WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete agent: %w", err)
	}

	// Delete config file
	if def.ConfigPath != "" {
		if err := os.Remove(def.ConfigPath); err != nil && !os.IsNotExist(err) {
			m.logger.Warn("failed to delete config file",
				zap.String("path", def.ConfigPath),
				zap.Error(err),
			)
		}
	}

	delete(m.agents, id)

	// Update default if needed
	if m.defaultAgent == id {
		m.defaultAgent = ""
		for otherID := range m.agents {
			m.defaultAgent = otherID
			m.setDefaultInDB(otherID)
			break
		}
	}

	m.logger.Info("agent deleted", zap.String("id", id))
	return nil
}

// SetDefault sets the default agent.
func (m *Manager) SetDefault(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.agents[id]; !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	return m.setDefaultInDB(id)
}

// setDefaultInDB updates the default flag in database.
func (m *Manager) setDefaultInDB(id string) error {
	// Unset all defaults
	_, err := m.db.Exec(`UPDATE agents SET is_default = 0`)
	if err != nil {
		return err
	}

	// Set new default
	_, err = m.db.Exec(`UPDATE agents SET is_default = 1 WHERE id = ?`, id)
	if err != nil {
		return err
	}

	// Update in-memory state
	for _, def := range m.agents {
		def.IsDefault = (def.ID == id)
	}
	m.defaultAgent = id

	return nil
}

// saveConfig saves the agent config to YAML file.
func (m *Manager) saveConfig(def *Definition) error {
	if def.Config == nil {
		return nil
	}

	data, err := yaml.Marshal(def.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(def.ConfigPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetAgentsDir returns the agents directory path.
func (m *Manager) GetAgentsDir() string {
	return m.agentsDir
}
