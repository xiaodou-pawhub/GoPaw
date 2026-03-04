package memory

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Category represents the classification of a long-term memory entry.
type Category string

const (
	CategoryCore         Category = "core"         // 永久偏好/重要事实，不衰减
	CategoryDaily        Category = "daily"        // 今日笔记，7天半衰期
	CategoryConversation Category = "conversation" // 会话片段，14天半衰期
)

// MemoryEntry is one record in the long-term memory store.
type MemoryEntry struct {
	ID        string
	Key       string
	Content   string
	Category  Category
	CreatedAt time.Time
	UpdatedAt time.Time
	Score     float64 // relevance score after search + decay
}

// LTMStore is the SQLite-backed long-term structured memory store (memories.db).
type LTMStore struct {
	db *sql.DB
}

// NewLTMStore opens (or creates) memories.db and runs schema migration.
func NewLTMStore(dbPath string) (*LTMStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("ltm: open db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &LTMStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ltm: migrate: %w", err)
	}
	return s, nil
}

// Close shuts down the database.
func (s *LTMStore) Close() error {
	return s.db.Close()
}

func (s *LTMStore) migrate() error {
	_, err := s.db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`)
	if err != nil {
		return fmt.Errorf("ltm: pragma: %w", err)
	}

	ddl := `
CREATE TABLE IF NOT EXISTS memories (
    id          TEXT PRIMARY KEY,
    key         TEXT NOT NULL UNIQUE,
    content     TEXT NOT NULL,
    category    TEXT NOT NULL DEFAULT 'core',
    created_at  INTEGER NOT NULL,
    updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_memories_category ON memories(category);
CREATE INDEX IF NOT EXISTS idx_memories_updated ON memories(updated_at DESC);

CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
    key, content,
    content='memories',
    content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
    INSERT INTO memories_fts(rowid, key, content) VALUES (new.rowid, new.key, new.content);
END;

CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, key, content)
    VALUES ('delete', old.rowid, old.key, old.content);
END;

CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
    INSERT INTO memories_fts(memories_fts, rowid, key, content)
    VALUES ('delete', old.rowid, old.key, old.content);
    INSERT INTO memories_fts(rowid, key, content) VALUES (new.rowid, new.key, new.content);
END;
`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("ltm: ddl: %w", err)
	}
	return nil
}

// Store inserts or updates a memory entry by key.
func (s *LTMStore) Store(key, content string, category Category) error {
	if key == "" || content == "" {
		return fmt.Errorf("ltm: key and content must not be empty")
	}
	now := time.Now().UnixMilli()
	id := uuid.New().String()

	_, err := s.db.Exec(`
		INSERT INTO memories (id, key, content, category, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET
			content    = excluded.content,
			category   = excluded.category,
			updated_at = excluded.updated_at
	`, id, key, content, string(category), now, now)
	if err != nil {
		return fmt.Errorf("ltm: store %q: %w", key, err)
	}
	return nil
}

// Forget deletes the memory entry with the given key.
// Returns false (no error) if the key was not found.
func (s *LTMStore) Forget(key string) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM memories WHERE key = ?`, key)
	if err != nil {
		return false, fmt.Errorf("ltm: forget %q: %w", key, err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// Get retrieves a single memory by key.
func (s *LTMStore) Get(key string) (*MemoryEntry, error) {
	row := s.db.QueryRow(`
		SELECT id, key, content, category, created_at, updated_at
		FROM memories WHERE key = ?`, key)
	return scanEntry(row)
}

// List returns memories filtered by optional category, ordered by updated_at DESC.
func (s *LTMStore) List(category Category, limit int) ([]MemoryEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error
	if category == "" {
		rows, err = s.db.Query(`
			SELECT id, key, content, category, created_at, updated_at
			FROM memories ORDER BY updated_at DESC LIMIT ?`, limit)
	} else {
		rows, err = s.db.Query(`
			SELECT id, key, content, category, created_at, updated_at
			FROM memories WHERE category = ? ORDER BY updated_at DESC LIMIT ?`,
			string(category), limit)
	}
	if err != nil {
		return nil, fmt.Errorf("ltm: list: %w", err)
	}
	defer rows.Close()
	return scanEntries(rows)
}

// Recall performs FTS5 full-text search and returns results with relevance scores.
// An empty query returns the most recently updated entries.
func (s *LTMStore) Recall(query string, limit int, category Category) ([]MemoryEntry, error) {
	if limit <= 0 {
		limit = 5
	}

	if strings.TrimSpace(query) == "" {
		return s.List(category, limit)
	}

	// Build FTS5 query: escape and OR-join each token
	tokens := strings.Fields(query)
	ftsTerms := make([]string, 0, len(tokens))
	for _, t := range tokens {
		t = strings.ReplaceAll(t, `"`, `""`)
		ftsTerms = append(ftsTerms, `"`+t+`"`)
	}
	ftsQuery := strings.Join(ftsTerms, " OR ")

	var catFilter string
	var args []any
	args = append(args, ftsQuery)
	if category != "" {
		catFilter = " AND m.category = ?"
		args = append(args, string(category))
	}
	args = append(args, limit)

	rows, err := s.db.Query(fmt.Sprintf(`
		SELECT m.id, m.key, m.content, m.category, m.created_at, m.updated_at,
		       (-bm25(memories_fts)) AS score
		FROM memories_fts f
		JOIN memories m ON m.rowid = f.rowid
		WHERE memories_fts MATCH ?%s
		ORDER BY score DESC
		LIMIT ?`, catFilter), args...)
	if err != nil {
		return nil, fmt.Errorf("ltm: recall fts: %w", err)
	}
	defer rows.Close()

	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var createdMs, updatedMs int64
		var catStr string
		if err := rows.Scan(&e.ID, &e.Key, &e.Content, &catStr, &createdMs, &updatedMs, &e.Score); err != nil {
			return nil, fmt.Errorf("ltm: scan recall: %w", err)
		}
		e.Category = Category(catStr)
		e.CreatedAt = time.UnixMilli(createdMs)
		e.UpdatedAt = time.UnixMilli(updatedMs)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

// ApplyTimeDecay adjusts entry scores based on age.
// Core entries are never decayed; instead they receive a +0.3 boost capped at 1.0.
// Other categories decay with a configurable half-life (days).
func ApplyTimeDecay(entries []MemoryEntry, halfLifeDays float64) []MemoryEntry {
	if halfLifeDays <= 0 {
		halfLifeDays = 30
	}
	now := time.Now()
	for i := range entries {
		e := &entries[i]
		if e.Category == CategoryCore {
			e.Score = math.Min(e.Score+0.3, 1.0)
			continue
		}
		ageDays := now.Sub(e.UpdatedAt).Hours() / 24
		factor := math.Exp(-ageDays / halfLifeDays * math.Log(2))
		e.Score *= factor
	}
	return entries
}

// DeleteByCategory removes all entries of the given category older than the cutoff time.
func (s *LTMStore) DeleteByCategory(category Category, olderThan time.Time) (int64, error) {
	res, err := s.db.Exec(
		`DELETE FROM memories WHERE category = ? AND updated_at < ?`,
		string(category), olderThan.UnixMilli(),
	)
	if err != nil {
		return 0, fmt.Errorf("ltm: delete by category: %w", err)
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// DB returns the raw *sql.DB for direct access when needed.
func (s *LTMStore) DB() *sql.DB {
	return s.db
}

// --- helpers ---

func scanEntry(row *sql.Row) (*MemoryEntry, error) {
	var e MemoryEntry
	var createdMs, updatedMs int64
	var catStr string
	if err := row.Scan(&e.ID, &e.Key, &e.Content, &catStr, &createdMs, &updatedMs); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("ltm: scan entry: %w", err)
	}
	e.Category = Category(catStr)
	e.CreatedAt = time.UnixMilli(createdMs)
	e.UpdatedAt = time.UnixMilli(updatedMs)
	return &e, nil
}

func scanEntries(rows *sql.Rows) ([]MemoryEntry, error) {
	var entries []MemoryEntry
	for rows.Next() {
		var e MemoryEntry
		var createdMs, updatedMs int64
		var catStr string
		if err := rows.Scan(&e.ID, &e.Key, &e.Content, &catStr, &createdMs, &updatedMs); err != nil {
			return nil, fmt.Errorf("ltm: scan entries: %w", err)
		}
		e.Category = Category(catStr)
		e.CreatedAt = time.UnixMilli(createdMs)
		e.UpdatedAt = time.UnixMilli(updatedMs)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
