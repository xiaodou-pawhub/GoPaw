package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
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

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	fmt.Println("Running database migrations...")

	// Migration 1: Add queue column to messages table if not exists
	fmt.Println("- Checking messages table...")
	if err := addColumnIfNotExists(db, "messages", "queue", "TEXT NOT NULL DEFAULT ''"); err != nil {
		log.Printf("Warning: %v", err)
	}

	// Migration 2: Ensure audit_logs table exists
	fmt.Println("- Checking audit_logs table...")
	if err := createTableIfNotExists(db, "audit_logs", `
		CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			timestamp TIMESTAMP NOT NULL,
			category TEXT NOT NULL,
			action TEXT NOT NULL,
			user_id TEXT,
			user_ip TEXT,
			resource_type TEXT,
			resource_id TEXT,
			status TEXT NOT NULL,
			details TEXT,
			error TEXT,
			duration INTEGER,
			request_id TEXT
		)
	`); err != nil {
		log.Fatalf("Failed to create audit_logs table: %v", err)
	}

	// Migration 3: Ensure message_stats table exists
	fmt.Println("- Checking message_stats table...")
	if err := createTableIfNotExists(db, "message_stats", `
		CREATE TABLE IF NOT EXISTS message_stats (
			queue TEXT PRIMARY KEY,
			pending_count INTEGER DEFAULT 0,
			processing_count INTEGER DEFAULT 0,
			completed_count INTEGER DEFAULT 0,
			failed_count INTEGER DEFAULT 0,
			delayed_count INTEGER DEFAULT 0,
			total_count INTEGER DEFAULT 0,
			updated_at TIMESTAMP
		)
	`); err != nil {
		log.Fatalf("Failed to create message_stats table: %v", err)
	}

	fmt.Println("Database migrations completed successfully!")
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
		fmt.Printf("  ✓ Column %s already exists\n", column)
		return nil
	}

	// Add column
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
	_, err = db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to add column %s: %w", column, err)
	}

	fmt.Printf("  ✓ Added column %s to table %s\n", column, table)
	return nil
}

func createTableIfNotExists(db *sql.DB, table, createSQL string) error {
	_, err := db.Exec(createSQL)
	if err != nil {
		return err
	}
	fmt.Printf("  ✓ Table %s ready\n", table)
	return nil
}
