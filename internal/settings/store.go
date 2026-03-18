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
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	BaseURL    string   `json:"base_url"`
	APIKey     string   `json:"api_key,omitempty"` // omitted in list responses for safety
	Model      string   `json:"model"`
	MaxTokens  int      `json:"max_tokens"`
	TimeoutSec int      `json:"timeout_sec"`
	
	// New fields for priority-based model management
	Priority   int      `json:"priority"`             // Priority order (0 = highest)
	Enabled    bool     `json:"enabled"`              // Whether provider is enabled
	Tags       []string `json:"tags"`                 // Capability tags like ["vision", "function_call"]
	
	// Legacy field for backward compatibility
	IsActive   bool     `json:"is_active"`            // Deprecated: use Enabled instead
	
	CreatedAt  int64    `json:"created_at"`
	UpdatedAt  int64    `json:"updated_at"`
}

// HasCapability checks if provider has a specific capability tag.
func (p *ProviderConfig) HasCapability(capability string) bool {
	for _, tag := range p.Tags {
		if tag == capability {
			return true
		}
	}
	return false
}

// IsVisionCapable returns true if provider supports vision.
func (p *ProviderConfig) IsVisionCapable() bool {
	return p.HasCapability("vision") || p.HasCapability("multimodal")
}

// Store manages runtime settings backed by the shared SQLite database.
type Store struct {
	db *sql.DB
}

// NewStore creates a Store backed by the given database connection.
func NewStore(db *sql.DB) *Store {
	store := &Store{db: db}
	
	// Run database migration for priority-based model management
	store.migrateProviderSchema()
	
	return store
}

// migrateProviderSchema adds new columns for priority-based model management.
func (s *Store) migrateProviderSchema() {
	// Add priority column if not exists
	_, _ = s.db.Exec(`
		ALTER TABLE providers ADD COLUMN priority INTEGER NOT NULL DEFAULT 0;
	`)
	
	// Add enabled column if not exists
	_, _ = s.db.Exec(`
		ALTER TABLE providers ADD COLUMN enabled INTEGER NOT NULL DEFAULT 1;
	`)
	
	// Migrate existing is_active to enabled for backward compatibility
	_, _ = s.db.Exec(`
		UPDATE providers SET enabled = is_active WHERE enabled = 0 AND is_active = 1;
	`)
	
	// Set priority based on creation order for existing providers
	_, _ = s.db.Exec(`
		UPDATE providers 
		SET priority = (
			SELECT COUNT(*) 
			FROM providers p2 
			WHERE p2.created_at < providers.created_at
		)
		WHERE priority = 0;
	`)
}

// ── LLM Providers ──────────────────────────────────────────────────────────

// ListProvidersByPriority returns all enabled LLM providers ordered by priority.
func (s *Store) ListProvidersByPriority() ([]ProviderConfig, error) {
	rows, err := s.db.Query(`
		SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec,
		       is_active, tags, created_at, updated_at, priority, enabled
		FROM providers
		WHERE enabled = 1
		ORDER BY priority ASC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ProviderConfig
	for rows.Next() {
		p := &ProviderConfig{}
		var isActiveInt, enabledInt int
		var tagsJSON string
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
			&p.MaxTokens, &p.TimeoutSec, &isActiveInt, &tagsJSON, &p.CreatedAt, &p.UpdatedAt, &p.Priority, &enabledInt); err != nil {
			return nil, err
		}
		p.IsActive = isActiveInt == 1
		p.Enabled = enabledInt == 1
		_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
		if p.Tags == nil {
			p.Tags = []string{}
		}
		// Mask API key for safety
		if len(p.APIKey) > 8 {
			p.APIKey = p.APIKey[:4] + "****" + p.APIKey[len(p.APIKey)-4:]
		} else {
			p.APIKey = "****"
		}
		list = append(list, *p)
	}
	return list, rows.Err()
}

// SetProviderEnabled enables or disables a provider by ID.
func (s *Store) SetProviderEnabled(id string, enabled bool) error {
	enabledInt := 0
	if enabled {
		enabledInt = 1
	}
	now := time.Now().UnixMilli()
	_, err := s.db.Exec(`
		UPDATE providers 
		SET enabled = ?, updated_at = ? 
		WHERE id = ?
	`, enabledInt, now, id)
	return err
}

// ReorderProviders updates priorities based on new order.
func (s *Store) ReorderProviders(providerIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	now := time.Now().UnixMilli()
	for i, id := range providerIDs {
		if _, err := tx.Exec(`
			UPDATE providers 
			SET priority = ?, updated_at = ? 
			WHERE id = ?
		`, i, now, id); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// GetProvidersByCapability returns enabled providers that have the specified capability.
func (s *Store) GetProvidersByCapability(capability string) ([]ProviderConfig, error) {
	rows, err := s.db.Query(`
		SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, 
		       is_active, tags, priority, enabled, created_at, updated_at
		FROM providers
		WHERE enabled = 1
		ORDER BY priority ASC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []ProviderConfig
	for rows.Next() {
		p := &ProviderConfig{}
		var isActiveInt, enabledInt int
		var tagsJSON string
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
			&p.MaxTokens, &p.TimeoutSec, &isActiveInt, &tagsJSON, &p.Priority, &enabledInt,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.IsActive = isActiveInt == 1
		p.Enabled = enabledInt == 1
		_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
		if p.Tags == nil {
			p.Tags = []string{}
		}

		// Filter by capability
		if p.HasCapability(capability) {
			// Mask API key for safety
			if len(p.APIKey) > 8 {
				p.APIKey = p.APIKey[:4] + "****" + p.APIKey[len(p.APIKey)-4:]
			} else {
				p.APIKey = "****"
			}
			list = append(list, *p)
		}
	}
	return list, rows.Err()
}

// GetFirstVisionCapableProvider returns the first enabled vision-capable provider.
func (s *Store) GetFirstVisionCapableProvider() (*ProviderConfig, error) {
	providers, err := s.GetProvidersByCapability("vision")
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		providers, err = s.GetProvidersByCapability("multimodal")
		if err != nil {
			return nil, err
		}
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no vision-capable provider configured")
	}
	return &providers[0], nil
}

// GetActiveProvider returns the currently active LLM provider, or nil if none is set.
func (s *Store) GetActiveProvider() (*ProviderConfig, error) {
	row := s.db.QueryRow(
		`SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, tags, created_at, updated_at
		 FROM providers WHERE is_active = 1 LIMIT 1`,
	)
	p := &ProviderConfig{}
	var isActive int
	var tagsJSON string
	err := row.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
		&p.MaxTokens, &p.TimeoutSec, &isActive, &tagsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("settings: get active provider: %w", err)
	}
	p.IsActive = isActive == 1
	_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return p, nil
}

// GetProvider returns a single provider by ID with the real (unmasked) API key.
func (s *Store) GetProvider(id string) (*ProviderConfig, error) {
	row := s.db.QueryRow(
		`SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, tags, created_at, updated_at
		 FROM providers WHERE id = ?`, id,
	)
	p := &ProviderConfig{}
	var isActive int
	var tagsJSON string
	err := row.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
		&p.MaxTokens, &p.TimeoutSec, &isActive, &tagsJSON, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("settings: get provider: %w", err)
	}
	p.IsActive = isActive == 1
	_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
	if p.Tags == nil {
		p.Tags = []string{}
	}
	return p, nil
}

// ListProviders returns all configured LLM providers. APIKey is masked for safety.
func (s *Store) ListProviders() ([]ProviderConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, tags, created_at, updated_at, priority, enabled
		 FROM providers ORDER BY created_at`,
	)
	if err != nil {
		return nil, fmt.Errorf("settings: list providers: %w", err)
	}
	defer rows.Close()

	var list []ProviderConfig
	for rows.Next() {
		p := ProviderConfig{}
		var isActiveInt, enabledInt int
		var tagsJSON string
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
			&p.MaxTokens, &p.TimeoutSec, &isActiveInt, &tagsJSON, &p.CreatedAt, &p.UpdatedAt, &p.Priority, &enabledInt); err != nil {
			return nil, fmt.Errorf("settings: scan provider: %w", err)
		}
		p.IsActive = isActiveInt == 1
		p.Enabled = enabledInt == 1
		_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
		if p.Tags == nil {
			p.Tags = []string{}
		}
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
// If p.APIKey is empty or masked ("****"), the old API key is preserved.
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

	// 中文：如果 API Key 为空或脱敏值，保留旧值
	// English: If API Key is empty or masked, preserve the old value
	if p.APIKey == "" || p.APIKey == "****" || (len(p.APIKey) == 8 && p.APIKey[4:] == "****") {
		old, err := s.GetProvider(p.ID)
		if err == nil && old != nil {
			p.APIKey = old.APIKey
		}
	}

	// Support both legacy IsActive and new Enabled fields
	isActive := 0
	if p.IsActive {
		isActive = 1
	}
	
	// Default to enabled for new providers
	enabledInt := 1
	if p.Enabled {
		enabledInt = 1
	} else if p.ID != "" {
		// For existing providers, preserve current enabled state
		old, err := s.GetProvider(p.ID)
		if err == nil && old != nil {
			if !old.Enabled {
				enabledInt = 0
			}
		}
	}

	// Keep IsActive and Enabled in sync for backward compatibility
	if p.IsActive && !p.Enabled {
		enabledInt = 1
	}
	if !p.IsActive && p.Enabled {
		isActive = 1
	}

	tagsData, _ := json.Marshal(p.Tags)
	if len(p.Tags) == 0 {
		tagsData = []byte("[]")
	}

	_, err := s.db.Exec(
		`INSERT INTO providers (id, name, base_url, api_key, model, max_tokens, timeout_sec,
		                        is_active, tags, created_at, updated_at, priority, enabled)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   name=excluded.name, base_url=excluded.base_url,
		   model=excluded.model, max_tokens=excluded.max_tokens, timeout_sec=excluded.timeout_sec,
		   is_active=excluded.is_active, tags=excluded.tags,
		   created_at=excluded.created_at, updated_at=excluded.updated_at,
		   priority=excluded.priority, enabled=excluded.enabled`,
		p.ID, p.Name, p.BaseURL, p.APIKey, p.Model,
		p.MaxTokens, p.TimeoutSec, isActive, string(tagsData), p.CreatedAt, p.UpdatedAt, p.Priority, enabledInt,
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

// GetProvidersByPriority returns all configured providers ordered by priority:
// the active provider first, then others ordered by creation time.
// Unlike ListProviders, the API key is NOT masked, so callers can use the configs directly.
func (s *Store) GetProvidersByPriority() ([]ProviderConfig, error) {
	rows, err := s.db.Query(
		`SELECT id, name, base_url, api_key, model, max_tokens, timeout_sec, is_active, tags, created_at, updated_at
		 FROM providers ORDER BY is_active DESC, created_at ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("settings: list providers by priority: %w", err)
	}
	defer rows.Close()

	var list []ProviderConfig
	for rows.Next() {
		p := ProviderConfig{}
		var isActive int
		var tagsJSON string
		if err := rows.Scan(&p.ID, &p.Name, &p.BaseURL, &p.APIKey, &p.Model,
			&p.MaxTokens, &p.TimeoutSec, &isActive, &tagsJSON, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("settings: scan provider: %w", err)
		}
		p.IsActive = isActive == 1
		_ = json.Unmarshal([]byte(tagsJSON), &p.Tags)
		if p.Tags == nil {
			p.Tags = []string{}
		}
		list = append(list, p)
	}
	return list, rows.Err()
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
