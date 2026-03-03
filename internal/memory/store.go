// Package memory implements the conversation history and long-term memory storage layer.
package memory

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // SQLite driver (pure Go, no CGo)
)

// Store is the SQLite-backed persistence layer for conversation history and summaries.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at the given path and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_timeout=5000&_fk=true")
	if err != nil {
		return nil, fmt.Errorf("memory store: open db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is single-writer; one connection avoids SQLITE_BUSY

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("memory store: migrate: %w", err)
	}
	return s, nil
}

// Close shuts down the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// migrate creates all necessary tables, indexes and FTS5 triggers.
func (s *Store) migrate() error {
	// Enable WAL mode and foreign keys first.
	_, err := s.db.Exec(`
		PRAGMA journal_mode=WAL;
		PRAGMA foreign_keys=ON;
	`)
	if err != nil {
		return fmt.Errorf("memory store: enable WAL: %w", err)
	}

	ddl := `
-- Sessions table
CREATE TABLE IF NOT EXISTS sessions (
    id          TEXT PRIMARY KEY,
    name        TEXT DEFAULT '',
    user_id     TEXT NOT NULL,
    channel     TEXT NOT NULL,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);

-- Messages table
CREATE TABLE IF NOT EXISTS messages (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES sessions(id),
    role        TEXT NOT NULL CHECK(role IN ('user','assistant','system')),
    content     TEXT NOT NULL,
    token_count INTEGER DEFAULT 0,
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);

-- FTS5 virtual table for full-text search
CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    session_id UNINDEXED,
    content='messages',
    content_rowid='rowid'
);

-- Memory summaries table
CREATE TABLE IF NOT EXISTS memory_summaries (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL,
    summary     TEXT NOT NULL,
    msg_from    INTEGER,
    msg_to      INTEGER,
    created_at  INTEGER NOT NULL
);

-- Cron jobs table
CREATE TABLE IF NOT EXISTS cron_jobs (
    id           TEXT PRIMARY KEY,
    name         TEXT NOT NULL,
    description  TEXT,
    cron_expr    TEXT NOT NULL,
    channel      TEXT NOT NULL,
    session_id   TEXT,
    prompt       TEXT NOT NULL,
    enabled      INTEGER DEFAULT 1,
    active_from  TEXT,
    active_until TEXT,
    last_run     INTEGER,
    next_run     INTEGER,
    created_at   INTEGER NOT NULL,
    updated_at   INTEGER NOT NULL
);

-- Cron job execution history table
CREATE TABLE IF NOT EXISTS cron_runs (
    id           TEXT PRIMARY KEY,
    job_id       TEXT NOT NULL,
    triggered_at INTEGER NOT NULL,
    finished_at  INTEGER,
    status       TEXT NOT NULL,
    output       TEXT,
    error_msg    TEXT,
    FOREIGN KEY (job_id) REFERENCES cron_jobs(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_cron_runs_job_id ON cron_runs(job_id);
CREATE INDEX IF NOT EXISTS idx_cron_runs_triggered_at ON cron_runs(triggered_at DESC);

-- LLM providers table (configured via Web UI)
CREATE TABLE IF NOT EXISTS providers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    api_key     TEXT NOT NULL,
    model       TEXT NOT NULL,
    max_tokens  INTEGER DEFAULT 4096,
    timeout_sec INTEGER DEFAULT 60,
    is_active   INTEGER DEFAULT 0,
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

-- Channel plugin configuration table (secrets configured via Web UI)
CREATE TABLE IF NOT EXISTS channel_configs (
    channel     TEXT PRIMARY KEY,
    config_json TEXT NOT NULL DEFAULT '{}',
    updated_at  INTEGER NOT NULL
);

-- FTS5 Triggers: keep messages_fts in sync with messages table
CREATE TRIGGER IF NOT EXISTS messages_ai AFTER INSERT ON messages BEGIN
    INSERT INTO messages_fts(rowid, content, session_id) 
    VALUES (new.rowid, new.content, new.session_id);
END;

CREATE TRIGGER IF NOT EXISTS messages_ad AFTER DELETE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content, session_id) 
    VALUES('delete', old.rowid, old.content, old.session_id);
END;

CREATE TRIGGER IF NOT EXISTS messages_au AFTER UPDATE ON messages BEGIN
    INSERT INTO messages_fts(messages_fts, rowid, content, session_id) 
    VALUES('delete', old.rowid, old.content, old.session_id);
    INSERT INTO messages_fts(rowid, content, session_id) 
    VALUES (new.rowid, new.content, new.session_id);
END;
`
	_, err = s.db.Exec(ddl)
	if err != nil {
		return fmt.Errorf("memory store: exec DDL: %w", err)
	}

	// Verify FTS5 triggers are created.
	var count int
	err = s.db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='trigger' AND name LIKE 'messages_%'`).Scan(&count)
	if err != nil {
		return fmt.Errorf("memory store: verify triggers: %w", err)
	}
	if count < 3 {
		return fmt.Errorf("memory store: expected 3 FTS triggers, found %d", count)
	}

	// Migration: Add name column to sessions table if it doesn't exist
	var nameColExists int
	err = s.db.QueryRow(`SELECT count(*) FROM pragma_table_info('sessions') WHERE name='name'`).Scan(&nameColExists)
	if err != nil {
		return fmt.Errorf("memory store: check name column: %w", err)
	}
	if nameColExists == 0 {
		_, err = s.db.Exec(`ALTER TABLE sessions ADD COLUMN name TEXT DEFAULT ''`)
		if err != nil {
			return fmt.Errorf("memory store: add name column: %w", err)
		}
	}

	return nil
}

// EnsureSession creates the session row if it does not already exist.
func (s *Store) EnsureSession(id, userID, channel string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO sessions (id, user_id, channel, created_at, updated_at) VALUES (?,?,?,?,?)`,
		id, userID, channel, now, now,
	)
	return err
}

// UpdateSessionName updates the session's display name.
func (s *Store) UpdateSessionName(sessionID, name string) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`UPDATE sessions SET name = ?, updated_at = ? WHERE id = ?`,
		name, now, sessionID,
	)
	return err
}

// StoredMessage mirrors the messages table row.
type StoredMessage struct {
	ID         string
	SessionID  string
	Role       string
	Content    string
	TokenCount int
	CreatedAt  int64
}

// AddMessage inserts a message row and updates the FTS index.
func (s *Store) AddMessage(msg StoredMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("store: begin tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(
		`INSERT INTO messages (id, session_id, role, content, token_count, created_at) VALUES (?,?,?,?,?,?)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.TokenCount, msg.CreatedAt,
	); err != nil {
		return fmt.Errorf("store: insert message: %w", err)
	}

	// Keep the FTS table in sync.
	if _, err := tx.Exec(
		`INSERT INTO messages_fts (rowid, content, session_id)
         SELECT rowid, content, session_id FROM messages WHERE id = ?`, msg.ID,
	); err != nil {
		return fmt.Errorf("store: update fts: %w", err)
	}

	return tx.Commit()
}

// GetRecentMessages returns the most recent limit messages for the given session.
func (s *Store) GetRecentMessages(sessionID string, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, token_count, created_at
         FROM messages WHERE session_id = ?
         ORDER BY created_at DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query messages: %w", err)
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokenCount, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	// Reverse so oldest-first order is preserved.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs, rows.Err()
}

// SearchMessages uses FTS5 to full-text search within a session.
func (s *Store) SearchMessages(sessionID, query string, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.session_id, m.role, m.content, m.token_count, m.created_at
         FROM messages_fts f
         JOIN messages m ON m.rowid = f.rowid
         WHERE messages_fts MATCH ? AND f.session_id = ?
         ORDER BY rank LIMIT ?`,
		query, sessionID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: fts search: %w", err)
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokenCount, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan search result: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

// DeleteOldMessages removes messages with rowids in the given range (inclusive).
func (s *Store) DeleteOldMessages(sessionID string, fromRowID, toRowID int64) error {
	_, err := s.db.Exec(
		`DELETE FROM messages WHERE session_id = ? AND rowid BETWEEN ? AND ?`,
		sessionID, fromRowID, toRowID,
	)
	return err
}

// StoreSummary persists a compression summary for a session.
func (s *Store) StoreSummary(id, sessionID, summary string, msgFrom, msgTo int64) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`INSERT INTO memory_summaries (id, session_id, summary, msg_from, msg_to, created_at) VALUES (?,?,?,?,?,?)`,
		id, sessionID, summary, msgFrom, msgTo, now,
	)
	return err
}

// GetLatestSummary retrieves the most recent summary for a session (if any).
func (s *Store) GetLatestSummary(sessionID string) (string, error) {
	var summary string
	err := s.db.QueryRow(
		`SELECT summary FROM memory_summaries WHERE session_id = ? ORDER BY created_at DESC LIMIT 1`,
		sessionID,
	).Scan(&summary)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: get summary: %w", err)
	}
	return summary, nil
}

// DeleteSession removes all data associated with a session.
func (s *Store) DeleteSession(sessionID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	if _, err := tx.Exec(`DELETE FROM messages WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM memory_summaries WHERE session_id = ?`, sessionID); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID); err != nil {
		return err
	}
	return tx.Commit()
}

// GetSessionStats calculates message count and token usage for a session.
func (s *Store) GetSessionStats(sessionID string) (count, total, user, assist int, err error) {
	err = s.db.QueryRow(`
		SELECT 
			COUNT(*), 
			IFNULL(SUM(token_count), 0),
			IFNULL(SUM(CASE WHEN role = 'user' THEN token_count ELSE 0 END), 0),
			IFNULL(SUM(CASE WHEN role = 'assistant' THEN token_count ELSE 0 END), 0)
		FROM messages WHERE session_id = ?`, sessionID).Scan(&count, &total, &user, &assist)
	if err == sql.ErrNoRows {
		return 0, 0, 0, 0, nil
	}
	return
}

// DB returns the raw *sql.DB for use by other storage layers (e.g. scheduler).
func (s *Store) DB() *sql.DB {
	return s.db
}

// ListSessions returns all sessions ordered by updated_at DESC.
func (s *Store) ListSessions() ([]SessionInfo, error) {
	rows, err := s.db.Query(`
		SELECT id, name, user_id, channel, created_at, updated_at
		FROM sessions
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var sess SessionInfo
		var createdAtMs, updatedAtMs int64
		err := rows.Scan(&sess.ID, &sess.Name, &sess.UserID, &sess.Channel, &createdAtMs, &updatedAtMs)
		if err != nil {
			return nil, err
		}
		sess.CreatedAt = time.UnixMilli(createdAtMs)
		sess.UpdatedAt = time.UnixMilli(updatedAtMs)
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// SessionInfo represents session metadata for API responses.
type SessionInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UserID    string    `json:"user_id"`
	Channel   string    `json:"channel"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
