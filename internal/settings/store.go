// Package settings manages runtime settings stored in SQLite:
// LLM provider configurations, channel plugin secrets, and agent persona (AGENT.md).
// These are configured via the Web UI after startup, separate from the static config.yaml.
package settings

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// ProviderConfig holds the configuration for a single LLM provider.
type ProviderConfig struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	BaseURL    string `json:"base_url"`
	APIKey     string `json:"api_key,omitempty"` // omitted in list responses for safety
	Model      string `json:"model"`
	MaxTokens  int    `json:"max_tokens"`
	TimeoutSec int    `json:"timeout_sec"`
	IsActive   bool   `json:"is_active"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// Store manages runtime settings backed by the shared SQLite database.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// ── LLM Providers ──────────────────────────────────────────────────────────

// GetActiveProvider returns the currently active LLM provider, or nil if none is set.
func (s *Store) GetActiveProvider() (*ProviderConfig, error) {
	row := s.db.QueryRow(
		`SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, created_at, updated_at
		 FROM providers WHERE is_active = 1 LIMIT 1`,
	)
	p := &ProviderConfig{}
	var isActive int
	err := row.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
		&p.MaxTokens, &p.TimeoutSec, &isActive, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("settings: get active provider: %w", err)
	}
	p.IsActive = isActive == 1
	return p, nil
}

// ListProviders returns all configured LLM providers. APIKey is masked for safety.
func (s *Store) ListProviders() ([]ProviderConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, created_at, updated_at
		 FROM providers ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("settings: list providers: %w", err)
	}
	defer rows.Close()

	var list []ProviderConfig
	for rows.Next() {
		p := ProviderConfig{}
		var isActive int
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
			&p.MaxTokens, &p.TimeoutSec, &isActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("settings: scan provider: %w", err)
		}
		p.IsActive = isActive == 1
		// Mask API key in list view
		if len(p.APIKey) > 8 {
			p.APIKey = p.APIKey[:4] + "****" + p.APIKey[len(p.APIKey)-4:]
		} else {
			p.APIKey = "****"
		}
		list = append(list, p)
	}
	return list, rows.Err()
}

// SaveProvider inserts or updates an LLM provider (upsert by ID).
// If p.ID is empty, a new UUID is assigned.
func (s *Store) SaveProvider(p *ProviderConfig) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	now := time.Now().UnixMilli()
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.MaxTokens == 0 {
		p.MaxTokens = 4096
	}
	if p.TimeoutSec == 0 {
		p.TimeoutSec = 60
	}
	isActive := 0
	if p.IsActive {
		isActive = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO providers (id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name, base_url=excluded.base_url, api_key=excluded.api_key,
		   model=excluded.model, max_tokens=excluded.max_tokens, timeout_sec=excluded.timeout_sec,
		   is_active=excluded.is_active, updated_at=excluded.updated_at`,
		p.ID, p.Name, p.BaseURL, p.APIKey, p.Model,
		p.MaxTokens, p.TimeoutSec, isActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("settings: save provider: %w", err)
	}
	return nil
}

// SetActiveProvider marks the given provider as active, deactivating all others.
func (s *Store) SetActiveProvider(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck

	now := time.Now().UnixMilli()
	if _, err := tx.Exec(`UPDATE providers SET is_active = 0, updated_at = ?`, now); err != nil {
		return fmt.Errorf("settings: deactivate providers: %w", err)
	}
	res, err := tx.Exec(`UPDATE providers SET is_active = 1, updated_at = ? WHERE id = ?`, now, id)
	if err != nil {
		return fmt.Errorf("settings: activate provider: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("settings: provider %q not found", id)
	}
	return tx.Commit()
}

// DeleteProvider removes a provider by ID.
func (s *Store) DeleteProvider(id string) error {
	_, err := s.db.Exec(`DELETE FROM providers WHERE id = ?`, id)
	return err
}

// ── Channel Configs ────────────────────────────────────────────────────────

// GetChannelConfig returns the raw JSON config for the given channel plugin.
// Returns "{}" if no config has been saved yet.
func (s *Store) GetChannelConfig(channelName string) (string, error) {
	var cfg string
	err := s.db.QueryRow(
		`SELECT config_json FROM channel_configs WHERE channel = ?`, channelName,
	).Scan(&cfg)
	if err == sql.ErrNoRows {
		return "{}", nil
	}
	if err != nil {
		return "", fmt.Errorf("settings: get channel config: %w", err)
	}
	return cfg, nil
}

// SetChannelConfig upserts the JSON config for a channel plugin.
func (s *Store) SetChannelConfig(channelName, jsonCfg string) error {
	if !json.Valid([]byte(jsonCfg)) {
		return fmt.Errorf("settings: invalid JSON for channel %q config", channelName)
	}
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(
		`INSERT INTO channel_configs (channel, config_json, updated_at) VALUES (?, ?, ?)
		 ON CONFLICT(channel) DO UPDATE SET config_json=excluded.config_json, updated_at=excluded.updated_at`,
		channelName, jsonCfg, now,
	)
	if err != nil {
		return fmt.Errorf("settings: set channel config: %w", err)
	}
	return nil
}

// ── AGENT.md ───────────────────────────────────────────────────────────────

// DefaultAgentPrompt is used when data/AGENT.md does not exist yet.
const DefaultAgentPrompt = `# GoPaw Agent 设定

你是一个智能助理，名字叫 GoPaw。
你会帮助用户完成各种任务，回答问题，处理文件。
请用中文回复，语气友好自然。
当你需要调用工具时，严格按照指定格式输出。`

// ReadAgentMD reads the agent system prompt from the given file path.
// Returns DefaultAgentPrompt if the file does not exist yet.
func ReadAgentMD(path string) (string, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return DefaultAgentPrompt, nil
	}
	if err != nil {
		return "", fmt.Errorf("settings: read AGENT.md: %w", err)
	}
	return string(data), nil
}

// WriteAgentMD writes the agent system prompt to the given file path,
// creating parent directories as needed.
func WriteAgentMD(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("settings: create AGENT.md dir: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

// IsSetupRequired returns true if no active LLM provider has been configured yet.
func (s *Store) IsSetupRequired() bool {
	p, err := s.GetActiveProvider()
	return err != nil || p == nil
}
