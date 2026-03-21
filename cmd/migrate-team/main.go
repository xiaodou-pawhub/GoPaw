// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure-Go SQLite driver (no CGo)
)

func main() {
	dataDir := "data"
	if len(os.Args) > 1 {
		dataDir = os.Args[1]
	}

	dbPath := filepath.Join(dataDir, "gopaw.db")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Printf("Database file not found: %s\n", dbPath)
		return
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("Running team migration...")

	// Migration 1: Add team_id to agents table
	fmt.Println("- Adding team_id to agents table...")
	if err := addColumnIfNotExists(db, "agents", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 2: Add team_id to workflows table
	fmt.Println("- Adding team_id to workflows table...")
	if err := addColumnIfNotExists(db, "workflows", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 3: Add team_id to knowledge_bases table
	fmt.Println("- Adding team_id to knowledge_bases table...")
	if err := addColumnIfNotExists(db, "knowledge_bases", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 4: Add team_id to triggers table
	fmt.Println("- Adding team_id to triggers table...")
	if err := addColumnIfNotExists(db, "triggers", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 5: Add team_id to orchestrations table
	fmt.Println("- Adding team_id to orchestrations table...")
	if err := addColumnIfNotExists(db, "orchestrations", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 6: Add team_id to mcp_servers table
	fmt.Println("- Adding team_id to mcp_servers table...")
	if err := addColumnIfNotExists(db, "mcp_servers", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 7: Add team_id to sessions table
	fmt.Println("- Adding team_id to sessions table...")
	if err := addColumnIfNotExists(db, "sessions", "team_id", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 8: Create indexes for team_id
	fmt.Println("- Creating indexes for team_id...")
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_agents_team ON agents(team_id)",
		"CREATE INDEX IF NOT EXISTS idx_workflows_team ON workflows(team_id)",
		"CREATE INDEX IF NOT EXISTS idx_knowledge_bases_team ON knowledge_bases(team_id)",
		"CREATE INDEX IF NOT EXISTS idx_triggers_team ON triggers(team_id)",
		"CREATE INDEX IF NOT EXISTS idx_orchestrations_team ON orchestrations(team_id)",
		"CREATE INDEX IF NOT EXISTS idx_mcp_servers_team ON mcp_servers(team_id)",
		"CREATE INDEX IF NOT EXISTS idx_sessions_team ON sessions(team_id)",
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	// Migration 9: Add visibility column to resources
	fmt.Println("- Adding visibility to agents table...")
	if err := addColumnIfNotExists(db, "agents", "visibility", "TEXT DEFAULT 'team'"); err != nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Println("- Adding visibility to workflows table...")
	if err := addColumnIfNotExists(db, "workflows", "visibility", "TEXT DEFAULT 'team'"); err != nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Println("- Adding visibility to knowledge_bases table...")
	if err := addColumnIfNotExists(db, "knowledge_bases", "visibility", "TEXT DEFAULT 'team'"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 10: Add created_by column to resources
	fmt.Println("- Adding created_by to agents table...")
	if err := addColumnIfNotExists(db, "agents", "created_by", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Println("- Adding created_by to workflows table...")
	if err := addColumnIfNotExists(db, "workflows", "created_by", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Println("- Adding created_by to knowledge_bases table...")
	if err := addColumnIfNotExists(db, "knowledge_bases", "created_by", "TEXT"); err != nil {
		log.Printf("Warning: %v", err)
	}

	fmt.Println("Team migration completed successfully!")
}

func addColumnIfNotExists(db *sql.DB, table, column, definition string) error {
	// Check if column exists
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?
	`, table, column).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to check column %s: %w", column, err)
	}

	if count > 0 {
		fmt.Printf("  ✓ Column %s already exists in %s\n", column, table)
		return nil
	}

	// Add column
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to add column %s to %s: %w", column, table, err)
	}

	fmt.Printf("  ✓ Added column %s to table %s\n", column, table)
	return nil
}