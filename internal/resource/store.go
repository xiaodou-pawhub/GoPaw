// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package resource manages resource packages and user grants for team mode.
package resource

import (
	"database/sql"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ResourceType defines the type of resource in a package.
type ResourceType string

const (
	ResourceTypeAgent     ResourceType = "agent"
	ResourceTypeSkill     ResourceType = "skill"
	ResourceTypeKnowledge ResourceType = "knowledge"
	ResourceTypeModel     ResourceType = "model"
)

// Package represents a resource package.
type Package struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"` // admin user id
	CreatedAt   time.Time `json:"created_at"`
	IsGlobal    bool      `json:"is_global"` // if true, available to all users
}

// Item represents a resource item in a package.
type Item struct {
	PackageID    string       `json:"package_id"`
	ResourceType ResourceType `json:"resource_type"`
	ResourceID   string       `json:"resource_id"`
}

// UserGrant represents a user's grant to use a package.
type UserGrant struct {
	UserID    string    `json:"user_id"`
	PackageID string    `json:"package_id"`
	GrantedBy string    `json:"granted_by"`
	GrantedAt time.Time `json:"granted_at"`
}

// Store provides database operations for resource packages.
type Store struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewStore creates a new resource package store.
func NewStore(db *sql.DB, logger *zap.Logger) (*Store, error) {
	s := &Store{
		db:     db,
		logger: logger.Named("resource"),
	}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	return s, nil
}

// initSchema creates the database tables.
func (s *Store) initSchema() error {
	schema := `
	-- Resource packages table
	CREATE TABLE IF NOT EXISTS resource_packages (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		description TEXT,
		created_by TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_global INTEGER DEFAULT 0
	);

	-- Resource package items table
	CREATE TABLE IF NOT EXISTS resource_package_items (
		package_id TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_id TEXT NOT NULL,
		PRIMARY KEY (package_id, resource_type, resource_id),
		FOREIGN KEY (package_id) REFERENCES resource_packages(id) ON DELETE CASCADE
	);

	-- User package grants table
	CREATE TABLE IF NOT EXISTS user_package_grants (
		user_id TEXT NOT NULL,
		package_id TEXT NOT NULL,
		granted_by TEXT NOT NULL,
		granted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, package_id),
		FOREIGN KEY (package_id) REFERENCES resource_packages(id) ON DELETE CASCADE,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	-- User agent permissions table (fine-grained control)
	CREATE TABLE IF NOT EXISTS user_agent_permissions (
		user_id TEXT NOT NULL,
		agent_id TEXT NOT NULL,
		can_use INTEGER DEFAULT 1,
		can_modify INTEGER DEFAULT 0,
		can_delete INTEGER DEFAULT 0,
		PRIMARY KEY (user_id, agent_id),
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
		FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE
	);

	-- Agent visibility table
	CREATE TABLE IF NOT EXISTS agent_visibility (
		agent_id TEXT PRIMARY KEY,
		visibility TEXT NOT NULL, -- 'global' | 'private' | 'shared'
		owner_id TEXT, -- NULL = system agent, non-NULL = user created
		FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
		FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE SET NULL
	);

	-- Create indexes for better query performance
	CREATE INDEX IF NOT EXISTS idx_resource_items_package ON resource_package_items(package_id);
	CREATE INDEX IF NOT EXISTS idx_user_grants_user ON user_package_grants(user_id);
	CREATE INDEX IF NOT EXISTS idx_agent_permissions_user ON user_agent_permissions(user_id);
	CREATE INDEX IF NOT EXISTS idx_agent_visibility_visibility ON agent_visibility(visibility);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("resource: failed to create schema: %w", err)
	}

	s.logger.Info("resource package schema initialized")
	return nil
}
