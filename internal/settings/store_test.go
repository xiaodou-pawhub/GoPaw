package settings

import (
	"database/sql"
	"os"
	"testing"

	_ "modernc.org/sqlite"
)

// newTestDB creates an in-memory SQLite database for testing.
func newTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	// Create the required tables
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS providers (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			base_url TEXT NOT NULL,
			api_key TEXT NOT NULL,
			model TEXT NOT NULL,
			max_tokens INTEGER DEFAULT 4096,
			timeout_sec INTEGER DEFAULT 60,
			is_active INTEGER DEFAULT 0,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);
		CREATE TABLE IF NOT EXISTS channel_configs (
			channel TEXT PRIMARY KEY,
			config_json TEXT NOT NULL DEFAULT '{}',
			updated_at INTEGER NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("failed to create tables: %v", err)
	}
	return db
}

// TestProvider_CRUD tests basic CRUD operations for LLM providers.
func TestProvider_CRUD(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Test SaveProvider (create)
	provider := &ProviderConfig{
		Name:    "OpenAI",
		BaseURL: "https://api.openai.com/v1",
		APIKey:  "sk-test12345678",
		Model:   "gpt-4o-mini",
	}
	err := store.SaveProvider(provider)
	if err != nil {
		t.Fatalf("SaveProvider failed: %v", err)
	}
	if provider.ID == "" {
		t.Fatal("expected provider ID to be set")
	}

	// Test ListProviders
	providers, err := store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	// APIKey should be masked
	if providers[0].APIKey == "sk-test12345678" {
		t.Fatal("expected APIKey to be masked")
	}

	// Test SetActiveProvider
	err = store.SetActiveProvider(provider.ID)
	if err != nil {
		t.Fatalf("SetActiveProvider failed: %v", err)
	}

	// Test GetActiveProvider
	active, err := store.GetActiveProvider()
	if err != nil {
		t.Fatalf("GetActiveProvider failed: %v", err)
	}
	if active == nil {
		t.Fatal("expected active provider, got nil")
	}
	if active.Name != "OpenAI" {
		t.Fatalf("expected OpenAI, got %s", active.Name)
	}

	// Test DeleteProvider
	err = store.DeleteProvider(provider.ID)
	if err != nil {
		t.Fatalf("DeleteProvider failed: %v", err)
	}

	// Verify deletion
	providers, err = store.ListProviders()
	if err != nil {
		t.Fatalf("ListProviders after delete failed: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("expected 0 providers after delete, got %d", len(providers))
	}
}

// TestProvider_MultipleProviders tests switching between multiple providers.
func TestProvider_MultipleProviders(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Create two providers
	p1 := &ProviderConfig{Name: "OpenAI", BaseURL: "https://api.openai.com/v1", APIKey: "sk-p1", Model: "gpt-4"}
	p2 := &ProviderConfig{Name: "Anthropic", BaseURL: "https://api.anthropic.com", APIKey: "sk-p2", Model: "claude-3"}

	store.SaveProvider(p1) //nolint:errcheck
	store.SaveProvider(p2)  //nolint:errcheck

	// Activate p1
	store.SetActiveProvider(p1.ID) //nolint:errcheck
	active, _ := store.GetActiveProvider()
	if active.Name != "OpenAI" {
		t.Fatalf("expected OpenAI active, got %s", active.Name)
	}

	// Switch to p2
	store.SetActiveProvider(p2.ID) //nolint:errcheck
	active, _ = store.GetActiveProvider()
	if active.Name != "Anthropic" {
		t.Fatalf("expected Anthropic active, got %s", active.Name)
	}
}

// TestChannelConfig tests channel configuration storage.
func TestChannelConfig(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Test GetChannelConfig (non-existent)
	cfg, err := store.GetChannelConfig("feishu")
	if err != nil {
		t.Fatalf("GetChannelConfig failed: %v", err)
	}
	if cfg != "{}" {
		t.Fatalf("expected empty config {}, got %s", cfg)
	}

	// Test SetChannelConfig
	jsonCfg := `{"app_id":"test-app","app_secret":"test-secret"}`
	err = store.SetChannelConfig("feishu", jsonCfg)
	if err != nil {
		t.Fatalf("SetChannelConfig failed: %v", err)
	}

	// Test GetChannelConfig (existing)
	cfg, err = store.GetChannelConfig("feishu")
	if err != nil {
		t.Fatalf("GetChannelConfig failed: %v", err)
	}
	if cfg != jsonCfg {
		t.Fatalf("expected %s, got %s", jsonCfg, cfg)
	}
}

// TestChannelConfig_InvalidJSON tests that invalid JSON is rejected.
func TestChannelConfig_InvalidJSON(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	store := NewStore(db)

	err := store.SetChannelConfig("feishu", "not valid json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

// TestIsSetupRequired tests the setup requirement check.
func TestIsSetupRequired(t *testing.T) {
	db := newTestDB(t)
	defer db.Close()

	store := NewStore(db)

	// Should be required initially (no active provider)
	if !store.IsSetupRequired() {
		t.Fatal("expected setup required initially")
	}

	// Add and activate a provider
	p := &ProviderConfig{Name: "Test", BaseURL: "http://test.com", APIKey: "test", Model: "test"}
	store.SaveProvider(p)   //nolint:errcheck
	store.SetActiveProvider(p.ID) //nolint:errcheck

	// Should not be required now
	if store.IsSetupRequired() {
		t.Fatal("expected setup not required after adding provider")
	}
}