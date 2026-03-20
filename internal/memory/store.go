// Package memory implements the conversation history and long-term memory storage layer.
package memory

import (
	"bytes"
	"database/sql"
	"encoding/binary"
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
	db.SetMaxOpenConns(1)

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

func (s *Store) migrate() error {
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
    embedding   BLOB,
    created_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);

-- FTS5 virtual table
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

-- LLM providers table
CREATE TABLE IF NOT EXISTS providers (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    base_url    TEXT NOT NULL,
    api_key     TEXT NOT NULL,
    model       TEXT NOT NULL,
    max_tokens  INTEGER DEFAULT 4096,
    timeout_sec INTEGER DEFAULT 60,
    is_active   INTEGER DEFAULT 0,
    tags        TEXT DEFAULT '[]',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);

-- Channel plugin configuration table
CREATE TABLE IF NOT EXISTS channel_configs (
    channel     TEXT PRIMARY KEY,
    config_json TEXT NOT NULL DEFAULT '{}',
    updated_at  INTEGER NOT NULL
);

-- Triggers
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

	var embedColExists int
	_ = s.db.QueryRow(`SELECT count(*) FROM pragma_table_info('messages') WHERE name='embedding'`).Scan(&embedColExists)
	if embedColExists == 0 {
		_, _ = s.db.Exec(`ALTER TABLE messages ADD COLUMN embedding BLOB`)
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
	Embedding  []float32
	CreatedAt  int64
}

// AddMessage inserts a message row.
func (s *Store) AddMessage(msg StoredMessage) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	embedBlob := encodeEmbedding(msg.Embedding)
	if _, err := tx.Exec(
		`INSERT INTO messages (id, session_id, role, content, token_count, embedding, created_at) VALUES (?,?,?,?,?,?,?)`,
		msg.ID, msg.SessionID, msg.Role, msg.Content, msg.TokenCount, embedBlob, msg.CreatedAt,
	); err != nil {
		return err
	}
	return tx.Commit()
}

// GetRecentMessages returns messages in chronological order (oldest first).
// Returns the most recent messages up to limit, ordered from oldest to newest.
func (s *Store) GetRecentMessages(sessionID string, limit int) ([]StoredMessage, error) {
	// First get the most recent messages (DESC), then reverse to ASC order
	rows, err := s.db.Query(
		`SELECT id, session_id, role, content, token_count, embedding, created_at
         FROM messages WHERE session_id = ?
         ORDER BY created_at DESC LIMIT ?`,
		sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		var embedBlob []byte
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokenCount, &embedBlob, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Embedding = decodeEmbedding(embedBlob)
		msgs = append([]StoredMessage{m}, msgs...) // Prepend to reverse order
	}
	
	// Debug: verify order (first should have smallest created_at)
	if len(msgs) > 1 {
		fmt.Printf("[DEBUG] GetRecentMessages: session=%s, count=%d, first_created_at=%d, last_created_at=%d\n",
			sessionID, len(msgs), msgs[0].CreatedAt, msgs[len(msgs)-1].CreatedAt)
	}
	
	return msgs, nil
}

// SearchMessages uses FTS5.
func (s *Store) SearchMessages(sessionID, query string, limit int) ([]StoredMessage, error) {
	rows, err := s.db.Query(
		`SELECT m.id, m.session_id, m.role, m.content, m.token_count, m.embedding, m.created_at
         FROM messages_fts f
         JOIN messages m ON m.rowid = f.rowid
         WHERE messages_fts MATCH ? AND f.session_id = ?
         ORDER BY rank LIMIT ?`,
		query, sessionID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []StoredMessage
	for rows.Next() {
		var m StoredMessage
		var embedBlob []byte
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.TokenCount, &embedBlob, &m.CreatedAt); err != nil {
			return nil, err
		}
		m.Embedding = decodeEmbedding(embedBlob)
		msgs = append(msgs, m)
	}
	return msgs, nil
}

// ListSessions returns all sessions.
func (s *Store) ListSessions() ([]SessionInfo, error) {
	rows, err := s.db.Query(`SELECT id, name, user_id, channel, created_at, updated_at FROM sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions []SessionInfo
	for rows.Next() {
		var sess SessionInfo
		var created, updated int64
		if err := rows.Scan(&sess.ID, &sess.Name, &sess.UserID, &sess.Channel, &created, &updated); err != nil {
			return nil, err
		}
		sess.CreatedAt = time.UnixMilli(created)
		sess.UpdatedAt = time.UnixMilli(updated)
		sessions = append(sessions, sess)
	}
	return sessions, nil
}

// SessionInfo represents session metadata.
type SessionInfo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UserID    string    `json:"user_id"`
	Channel   string    `json:"channel"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// DeleteSession removes session and all its associated messages and summaries.
func (s *Store) DeleteSession(sessionID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete messages (and their FTS5 entries via trigger)
	if _, err := tx.Exec(`DELETE FROM messages WHERE session_id = ?`, sessionID); err != nil {
		return err
	}

	// Delete memory summaries
	if _, err := tx.Exec(`DELETE FROM memory_summaries WHERE session_id = ?`, sessionID); err != nil {
		return err
	}

	// Delete the session itself
	if _, err := tx.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID); err != nil {
		return err
	}

	return tx.Commit()
}

// GetLatestSummary retrieves the last summary.
func (s *Store) GetLatestSummary(sessionID string) (string, error) {
	var summary string
	err := s.db.QueryRow(`SELECT summary FROM memory_summaries WHERE session_id = ? ORDER BY created_at DESC LIMIT 1`, sessionID).Scan(&summary)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return summary, err
}

// StoreSummary persists a summary.
func (s *Store) StoreSummary(id, sessionID, summary string, from, to int64) error {
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`INSERT INTO memory_summaries (id, session_id, summary, msg_from, msg_to, created_at) VALUES (?,?,?,?,?,?)`, id, sessionID, summary, from, to, now)
	return err
}

// DeleteInactiveSessions handles hygiene.
func (s *Store) DeleteInactiveSessions(olderThan time.Time) (int64, error) {
	cutoff := olderThan.UnixMilli()
	res, err := s.db.Exec(`DELETE FROM sessions WHERE updated_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GetSessionStats for health reporting.
func (s *Store) GetSessionStats(sessionID string) (count, total, user, assist int, err error) {
	err = s.db.QueryRow(`
		SELECT COUNT(*), IFNULL(SUM(token_count), 0),
		IFNULL(SUM(CASE WHEN role='user' THEN token_count ELSE 0 END), 0),
		IFNULL(SUM(CASE WHEN role='assistant' THEN token_count ELSE 0 END), 0)
		FROM messages WHERE session_id = ?`, sessionID).Scan(&count, &total, &user, &assist)
	return
}

func (s *Store) DB() *sql.DB { return s.db }

// ── Embedding Helpers ──────────────────────────────────────────────────────

func encodeEmbedding(vec []float32) []byte {
	if len(vec) == 0 { return nil }
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, vec)
	return buf.Bytes()
}

func decodeEmbedding(b []byte) []float32 {
	if len(b) == 0 { return nil }
	vec := make([]float32, len(b)/4)
	binary.Read(bytes.NewReader(b), binary.LittleEndian, &vec)
	return vec
}
