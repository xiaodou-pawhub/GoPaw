// Package db provides database utilities for the GoPaw platform.
package db

import (
	"database/sql"
	"fmt"
)

// SchemaInitializer is a function that initializes database schema.
type SchemaInitializer func(*sql.DB) error

// SchemaManager manages database schema initialization.
type SchemaManager struct {
	initializers map[string]SchemaInitializer
}

// NewSchemaManager creates a new schema manager.
func NewSchemaManager() *SchemaManager {
	return &SchemaManager{
		initializers: make(map[string]SchemaInitializer),
	}
}

// Register registers a schema initializer.
func (m *SchemaManager) Register(name string, initializer SchemaInitializer) {
	m.initializers[name] = initializer
}

// InitAll initializes all registered schemas.
func (m *SchemaManager) InitAll(db *sql.DB) error {
	for name, initializer := range m.initializers {
		if err := initializer(db); err != nil {
			return fmt.Errorf("failed to initialize %s schema: %w", name, err)
		}
	}
	return nil
}

// Init initializes a specific schema.
func (m *SchemaManager) Init(db *sql.DB, name string) error {
	initializer, ok := m.initializers[name]
	if !ok {
		return fmt.Errorf("schema initializer not found: %s", name)
	}
	return initializer(db)
}

// List returns all registered schema names.
func (m *SchemaManager) List() []string {
	names := make([]string, 0, len(m.initializers))
	for name := range m.initializers {
		names = append(names, name)
	}
	return names
}

// CommonSchema provides common schema definitions.
const CommonSchema = `
-- Common fields for all tables
-- created_at: DATETIME DEFAULT CURRENT_TIMESTAMP
-- updated_at: DATETIME DEFAULT CURRENT_TIMESTAMP
`

// InitCommonSchema initializes common schema elements.
func InitCommonSchema(db *sql.DB) error {
	// This is a placeholder for common schema initialization
	// In practice, each module should define its own schema
	return nil
}
