// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package mcp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"go.uber.org/zap"
)

// Server represents an MCP server configuration.
type Server struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Command     string    `json:"command"`
	Args        []string  `json:"args"`
	Env         []string  `json:"env"`
	Transport   string    `json:"transport"` // "stdio" | "sse"
	URL         string    `json:"url,omitempty"` // for sse transport
	IsActive    bool      `json:"is_active"`
	IsBuiltin   bool      `json:"is_builtin"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Manager manages MCP server configurations and clients.
type Manager struct {
	db       *sql.DB
	logger   *zap.Logger
	clients  map[string]*tool.MCPClient
	servers  map[string]*Server
	mu       sync.RWMutex
}

// NewManager creates a new MCP manager.
func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		db:      db,
		logger:  logger.Named("mcp_manager"),
		clients: make(map[string]*tool.MCPClient),
		servers: make(map[string]*Server),
	}

	if err := m.initSchema(); err != nil {
		return nil, err
	}

	if err := m.LoadServers(); err != nil {
		return nil, err
	}

	return m, nil
}

// initSchema creates the database tables.
func (m *Manager) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS mcp_servers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    command TEXT,
    args TEXT, -- JSON array
    env TEXT, -- JSON array
    transport TEXT DEFAULT 'stdio',
    url TEXT,
    is_active BOOLEAN DEFAULT 0,
    is_builtin BOOLEAN DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_active ON mcp_servers(is_active);
CREATE INDEX IF NOT EXISTS idx_mcp_servers_builtin ON mcp_servers(is_builtin);
`
	_, err := m.db.Exec(schema)
	return err
}

// LoadServers loads all MCP servers from database.
func (m *Manager) LoadServers() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.servers = make(map[string]*Server)

	rows, err := m.db.Query(`
		SELECT id, name, description, command, args, env, transport, url, is_active, is_builtin, created_at, updated_at
		FROM mcp_servers
		ORDER BY created_at DESC
	`)
	if err != nil {
		return fmt.Errorf("failed to query mcp servers: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		server := &Server{}
		var argsJSON, envJSON string
		err := rows.Scan(
			&server.ID, &server.Name, &server.Description, &server.Command,
			&argsJSON, &envJSON, &server.Transport, &server.URL,
			&server.IsActive, &server.IsBuiltin, &server.CreatedAt, &server.UpdatedAt,
		)
		if err != nil {
			m.logger.Warn("failed to scan mcp server", zap.Error(err))
			continue
		}

		// Parse JSON arrays
		if argsJSON != "" {
			// Simple parsing - in production use json.Unmarshal
			server.Args = parseJSONStringArray(argsJSON)
		}
		if envJSON != "" {
			server.Env = parseJSONStringArray(envJSON)
		}

		m.servers[server.ID] = server
	}

	m.logger.Info("mcp servers loaded", zap.Int("count", len(m.servers)))
	return nil
}

// parseJSONStringArray parses a JSON string array.
func parseJSONStringArray(s string) []string {
	if s == "[]" || s == "" {
		return []string{}
	}
	var result []string
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return []string{}
	}
	return result
}

// Get returns an MCP server by ID.
func (m *Manager) Get(id string) (*Server, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	server, ok := m.servers[id]
	if !ok {
		return nil, fmt.Errorf("mcp server not found: %s", id)
	}

	return server, nil
}

// List returns all MCP servers.
func (m *Manager) List() []*Server {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Server, 0, len(m.servers))
	for _, server := range m.servers {
		result = append(result, server)
	}
	return result
}

// Create creates a new MCP server.
func (m *Manager) Create(server *Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.servers[server.ID]; exists {
		return fmt.Errorf("mcp server already exists: %s", server.ID)
	}

	now := time.Now().UTC()
	server.CreatedAt = now
	server.UpdatedAt = now

	// Convert arrays to JSON
	argsJSON, _ := json.Marshal(server.Args)
	envJSON, _ := json.Marshal(server.Env)

	_, err := m.db.Exec(`
		INSERT INTO mcp_servers (id, name, description, command, args, env, transport, url, is_active, is_builtin, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, server.ID, server.Name, server.Description, server.Command,
		string(argsJSON), string(envJSON), server.Transport, server.URL,
		server.IsActive, server.IsBuiltin, server.CreatedAt, server.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert mcp server: %w", err)
	}

	m.servers[server.ID] = server

	// Start the client if active
	if server.IsActive {
		if err := m.startClient(server); err != nil {
			m.logger.Warn("failed to start mcp client", zap.String("server_id", server.ID), zap.Error(err))
		}
	}

	m.logger.Info("mcp server created", zap.String("id", server.ID), zap.String("name", server.Name))
	return nil
}

// Update updates an existing MCP server.
func (m *Manager) Update(id string, server *Server) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, ok := m.servers[id]
	if !ok {
		return fmt.Errorf("mcp server not found: %s", id)
	}

	server.ID = id
	server.CreatedAt = existing.CreatedAt
	server.UpdatedAt = time.Now().UTC()

	// Stop existing client if running
	if existing.IsActive {
		m.stopClient(id)
	}

	// Convert arrays to JSON
	argsJSON, _ := json.Marshal(server.Args)
	envJSON, _ := json.Marshal(server.Env)

	_, err := m.db.Exec(`
		UPDATE mcp_servers
		SET name = ?, description = ?, command = ?, args = ?, env = ?, transport = ?, url = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`, server.Name, server.Description, server.Command, string(argsJSON), string(envJSON),
		server.Transport, server.URL, server.IsActive, server.UpdatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update mcp server: %w", err)
	}

	m.servers[id] = server

	// Start the client if active
	if server.IsActive {
		if err := m.startClient(server); err != nil {
			m.logger.Warn("failed to start mcp client", zap.String("server_id", server.ID), zap.Error(err))
		}
	}

	m.logger.Info("mcp server updated", zap.String("id", id), zap.String("name", server.Name))
	return nil
}

// Delete deletes an MCP server.
func (m *Manager) Delete(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return fmt.Errorf("mcp server not found: %s", id)
	}

	// Stop client if running
	if server.IsActive {
		m.stopClient(id)
	}

	_, err := m.db.Exec(`DELETE FROM mcp_servers WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete mcp server: %w", err)
	}

	delete(m.servers, id)
	delete(m.clients, id)

	m.logger.Info("mcp server deleted", zap.String("id", id))
	return nil
}

// SetActive activates or deactivates an MCP server.
func (m *Manager) SetActive(id string, active bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	server, ok := m.servers[id]
	if !ok {
		return fmt.Errorf("mcp server not found: %s", id)
	}

	server.IsActive = active
	server.UpdatedAt = time.Now().UTC()

	_, err := m.db.Exec(`UPDATE mcp_servers SET is_active = ?, updated_at = ? WHERE id = ?`, active, server.UpdatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update mcp server status: %w", err)
	}

	if active {
		if err := m.startClient(server); err != nil {
			return fmt.Errorf("failed to start mcp client: %w", err)
		}
	} else {
		m.stopClient(id)
	}

	m.logger.Info("mcp server status changed", zap.String("id", id), zap.Bool("active", active))
	return nil
}

// startClient starts an MCP client for a server.
func (m *Manager) startClient(server *Server) error {
	if server.Transport != "stdio" {
		// TODO: Support SSE transport
		return fmt.Errorf("transport %s not supported yet", server.Transport)
	}

	client := tool.NewMCPClient(server.Name, server.Command, server.Args)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := client.Start(ctx); err != nil {
		return err
	}

	m.clients[server.ID] = client
	return nil
}

// stopClient stops an MCP client.
func (m *Manager) stopClient(id string) {
	if _, ok := m.clients[id]; ok {
		// TODO: Implement proper shutdown
		delete(m.clients, id)
	}
}

// GetClient returns the MCP client for a server.
func (m *Manager) GetClient(id string) (*tool.MCPClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]
	if !ok {
		return nil, fmt.Errorf("mcp client not found or not active: %s", id)
	}

	return client, nil
}

// GetTools returns all tools from all active MCP servers.
func (m *Manager) GetTools() []tool.MCPToolInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var allTools []tool.MCPToolInfo
	for id, client := range m.clients {
		tools := client.GetTools()
		// Prefix tool names with server ID to avoid conflicts
		for i := range tools {
			tools[i].Name = fmt.Sprintf("%s_%s", id, tools[i].Name)
		}
		allTools = append(allTools, tools...)
	}
	return allTools
}

// CreateBuiltinServers creates built-in MCP servers.
func (m *Manager) CreateBuiltinServers(workspaceRoot string) error {
	// Create filesystem MCP server
	fsServer := &Server{
		ID:          "filesystem",
		Name:        "文件系统",
		Description: "访问本地文件系统",
		Command:     "npx",
		Args:        []string{"-y", "@modelcontextprotocol/server-filesystem", workspaceRoot},
		Transport:   "stdio",
		IsBuiltin:   true,
		IsActive:    true,
	}

	if _, err := m.Get(fsServer.ID); err != nil {
		// Server doesn't exist, create it
		if err := m.Create(fsServer); err != nil {
			m.logger.Warn("failed to create builtin filesystem mcp server", zap.Error(err))
		}
	}

	return nil
}
