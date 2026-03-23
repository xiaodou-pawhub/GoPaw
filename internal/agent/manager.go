// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGo)
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
	db, err := sql.Open("sqlite", dbPath)
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

CREATE TABLE IF NOT EXISTS agent_versions (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    name TEXT,
    description TEXT,
    config TEXT,
    created_at TIMESTAMP NOT NULL,
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_agent_versions_agent ON agent_versions(agent_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_versions_unique ON agent_versions(agent_id, version);
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

// ========== 版本管理 ==========

// AgentVersion Agent 版本
type AgentVersion struct {
	ID          string       `json:"id" db:"id"`
	AgentID     string       `json:"agent_id" db:"agent_id"`
	Version     int          `json:"version" db:"version"`
	Name        string       `json:"name" db:"name"`
	Description string       `json:"description" db:"description"`
	Config      *AgentConfig `json:"config" db:"config"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	CreatedBy   string       `json:"created_by" db:"created_by"`
}

// CreateVersion 创建版本
func (m *Manager) CreateVersion(agentID, name, createdBy string) (*AgentVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	def, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 获取下一个版本号
	var maxVersion int
	err := m.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM agent_versions WHERE agent_id = ?`, agentID).Scan(&maxVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get max version: %w", err)
	}

	nextVersion := maxVersion + 1
	versionID := fmt.Sprintf("av_%s_%d", agentID, nextVersion)

	configJSON, _ := json.Marshal(def.Config)

	version := &AgentVersion{
		ID:          versionID,
		AgentID:     agentID,
		Version:     nextVersion,
		Name:        name,
		Description: def.Description,
		Config:      def.Config,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}

	_, err = m.db.Exec(`
		INSERT INTO agent_versions (id, agent_id, version, name, description, config, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, version.ID, version.AgentID, version.Version, version.Name, version.Description, string(configJSON), version.CreatedAt, version.CreatedBy)

	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return version, nil
}

// ListVersions 列出版本
func (m *Manager) ListVersions(agentID string) ([]*AgentVersion, error) {
	rows, err := m.db.Query(`
		SELECT id, agent_id, version, name, description, config, created_at, created_by
		FROM agent_versions
		WHERE agent_id = ?
		ORDER BY version DESC
	`, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []*AgentVersion
	for rows.Next() {
		v := &AgentVersion{}
		var configJSON sql.NullString
		err := rows.Scan(&v.ID, &v.AgentID, &v.Version, &v.Name, &v.Description, &configJSON, &v.CreatedAt, &v.CreatedBy)
		if err != nil {
			continue
		}
		if configJSON.Valid && configJSON.String != "" {
			var config AgentConfig
			if err := json.Unmarshal([]byte(configJSON.String), &config); err == nil {
				v.Config = &config
			}
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// GetVersion 获取版本
func (m *Manager) GetVersion(agentID string, version int) (*AgentVersion, error) {
	v := &AgentVersion{}
	var configJSON sql.NullString

	err := m.db.QueryRow(`
		SELECT id, agent_id, version, name, description, config, created_at, created_by
		FROM agent_versions
		WHERE agent_id = ? AND version = ?
	`, agentID, version).Scan(&v.ID, &v.AgentID, &v.Version, &v.Name, &v.Description, &configJSON, &v.CreatedAt, &v.CreatedBy)

	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	if configJSON.Valid && configJSON.String != "" {
		var config AgentConfig
		if err := json.Unmarshal([]byte(configJSON.String), &config); err == nil {
			v.Config = &config
		}
	}

	return v, nil
}

// RollbackVersion 回滚到指定版本
func (m *Manager) RollbackVersion(agentID string, version int) (*Definition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	v, err := m.GetVersion(agentID, version)
	if err != nil {
		return nil, err
	}

	def, ok := m.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	// 更新配置
	if v.Config != nil {
		def.Config = v.Config
		def.Name = v.Name
		def.Description = v.Description
		def.UpdatedAt = time.Now()

		// 保存配置文件
		if err := m.saveConfig(def); err != nil {
			return nil, fmt.Errorf("failed to save config: %w", err)
		}

		// 更新数据库
		_, err = m.db.Exec(`
			UPDATE agents SET name = ?, description = ?, updated_at = ?
			WHERE id = ?
		`, def.Name, def.Description, def.UpdatedAt, def.ID)

		if err != nil {
			return nil, fmt.Errorf("failed to update agent: %w", err)
		}
	}

	return def, nil
}

// DeleteVersion 删除版本
func (m *Manager) DeleteVersion(agentID string, version int) error {
	_, err := m.db.Exec(`DELETE FROM agent_versions WHERE agent_id = ? AND version = ?`, agentID, version)
	return err
}

// GetVersionStats 获取版本统计
func (m *Manager) GetVersionStats(agentID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var count int
	err := m.db.QueryRow(`SELECT COUNT(*) FROM agent_versions WHERE agent_id = ?`, agentID).Scan(&count)
	if err != nil {
		return nil, err
	}
	stats["total_versions"] = count

	var latestVersion int
	err = m.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM agent_versions WHERE agent_id = ?`, agentID).Scan(&latestVersion)
	if err != nil {
		return nil, err
	}
	stats["latest_version"] = latestVersion

	return stats, nil
}

// ========== 性能分析 ==========

// AgentStats Agent 统计
type AgentStats struct {
	AgentID          string  `json:"agent_id"`
	TotalExecutions  int     `json:"total_executions"`
	SuccessCount     int     `json:"success_count"`
	FailureCount     int     `json:"failure_count"`
	TotalTokens      int64   `json:"total_tokens"`
	TotalInputTokens int64   `json:"total_input_tokens"`
	TotalOutputTokens int64  `json:"total_output_tokens"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
	MaxLatencyMs     int64   `json:"max_latency_ms"`
	MinLatencyMs     int64   `json:"min_latency_ms"`
	LastExecutedAt   *time.Time `json:"last_executed_at,omitempty"`
}

// GetAgentStats 获取 Agent 统计
func (m *Manager) GetAgentStats(agentID string) (*AgentStats, error) {
	stats := &AgentStats{AgentID: agentID}

	// 从 traces 表获取统计（如果存在）
	err := m.db.QueryRow(`
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0) as success,
			COALESCE(SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END), 0) as failure,
			COALESCE(SUM(total_tokens), 0) as tokens,
			COALESCE(SUM(input_tokens), 0) as input_tokens,
			COALESCE(SUM(output_tokens), 0) as output_tokens,
			COALESCE(AVG(duration_ms), 0) as avg_latency,
			COALESCE(MAX(duration_ms), 0) as max_latency,
			COALESCE(MIN(duration_ms), 0) as min_latency,
			MAX(created_at) as last_executed
		FROM traces
		WHERE agent_id = ?
	`, agentID).Scan(
		&stats.TotalExecutions, &stats.SuccessCount, &stats.FailureCount,
		&stats.TotalTokens, &stats.TotalInputTokens, &stats.TotalOutputTokens,
		&stats.AvgLatencyMs, &stats.MaxLatencyMs, &stats.MinLatencyMs,
		&stats.LastExecutedAt,
	)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get agent stats: %w", err)
	}

	return stats, nil
}

// GetAgentDailyStats 获取 Agent 每日统计
func (m *Manager) GetAgentDailyStats(agentID string, days int) ([]map[string]interface{}, error) {
	rows, err := m.db.Query(`
		SELECT
			date(created_at) as date,
			COUNT(*) as executions,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success,
			SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END) as failure,
			SUM(total_tokens) as tokens,
			AVG(duration_ms) as avg_latency
		FROM traces
		WHERE agent_id = ? AND created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY date(created_at)
		ORDER BY date DESC
	`, agentID, days)

	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var date string
		var executions, success, failure int
		var tokens int64
		var avgLatency float64

		err := rows.Scan(&date, &executions, &success, &failure, &tokens, &avgLatency)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"date":        date,
			"executions":  executions,
			"success":     success,
			"failure":     failure,
			"tokens":      tokens,
			"avg_latency": avgLatency,
		})
	}

	return results, nil
}

// GetAgentErrorStats 获取 Agent 错误统计
func (m *Manager) GetAgentErrorStats(agentID string, limit int) ([]map[string]interface{}, error) {
	rows, err := m.db.Query(`
		SELECT
			error_message,
			COUNT(*) as count,
			MAX(created_at) as last_occurred
		FROM traces
		WHERE agent_id = ? AND status = 'error' AND error_message IS NOT NULL
		GROUP BY error_message
		ORDER BY count DESC
		LIMIT ?
	`, agentID, limit)

	if err != nil {
		return nil, fmt.Errorf("failed to get error stats: %w", err)
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var errorMsg string
		var count int
		var lastOccurred time.Time

		err := rows.Scan(&errorMsg, &count, &lastOccurred)
		if err != nil {
			continue
		}

		results = append(results, map[string]interface{}{
			"error_message": errorMsg,
			"count":         count,
			"last_occurred": lastOccurred,
		})
	}

	return results, nil
}

// GetAllAgentsStats 获取所有 Agent 统计
func (m *Manager) GetAllAgentsStats() ([]*AgentStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*AgentStats
	for agentID := range m.agents {
		stats, err := m.GetAgentStats(agentID)
		if err != nil {
			continue
		}
		results = append(results, stats)
	}

	return results, nil
}
