package config

import (
	"os"
	"path/filepath"
	"testing"
)

// TestConfig_Validate tests configuration validation.
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: Config{
				Server: ServerConfig{
					Host: "0.0.0.0",
					Port: 8088,
				},
				Workspace: WorkspaceConfig{
					Dir: "~/.gopaw",
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port too low",
			cfg: Config{
				Server: ServerConfig{
					Port: 0,
				},
				Workspace: WorkspaceConfig{
					Dir: "~/.gopaw",
				},
			},
			wantErr: true,
			errMsg:  "server.port must be between 1 and 65535",
		},
		{
			name: "invalid port too high",
			cfg: Config{
				Server: ServerConfig{
					Port: 70000,
				},
				Workspace: WorkspaceConfig{
					Dir: "~/.gopaw",
				},
			},
			wantErr: true,
			errMsg:  "server.port must be between 1 and 65535",
		},
		{
			name: "missing workspace dir",
			cfg: Config{
				Server: ServerConfig{
					Port: 8088,
				},
				Workspace: WorkspaceConfig{
					Dir: "",
				},
			},
			wantErr: true,
			errMsg:  "workspace.dir is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if tt.errMsg != "" && err.Error()[:len(tt.errMsg)] != tt.errMsg {
					t.Fatalf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestExpandEnvVars tests environment variable expansion.
func TestExpandEnvVars(t *testing.T) {
	// Set a test environment variable
	os.Setenv("GOPAW_TEST_VAR", "test-value")
	defer os.Unsetenv("GOPAW_TEST_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no env var",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single env var",
			input:    "prefix ${GOPAW_TEST_VAR} suffix",
			expected: "prefix test-value suffix",
		},
		{
			name:     "multiple env vars",
			input:    "${GOPAW_TEST_VAR}-${GOPAW_TEST_VAR}",
			expected: "test-value-test-value",
		},
		{
			name:     "undefined env var",
			input:    "prefix ${UNDEFINED_VAR} suffix",
			expected: "prefix ${UNDEFINED_VAR} suffix",
		},
		{
			name:     "nested braces",
			input:    "${GOPAW_TEST_VAR}",
			expected: "test-value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandEnvVars(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestLoadConfig tests loading configuration from a file.
func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write test config
	configContent := `
server:
  host: "0.0.0.0"
  port: 9000
workspace:
  dir: "/tmp/gopaw"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	m, err := NewManager(configPath, nil)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify values
	cfg := m.Get()
	if cfg.Server.Port != 9000 {
		t.Fatalf("expected port 9000, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("expected host 0.0.0.0, got %s", cfg.Server.Host)
	}
	if cfg.Workspace.Dir != "/tmp/gopaw" {
		t.Fatalf("expected workspace dir /tmp/gopaw, got %s", cfg.Workspace.Dir)
	}
}

// TestLoadConfig_WithEnvVar tests loading config with environment variable expansion.
func TestLoadConfig_WithEnvVar(t *testing.T) {
	// Set environment variable
	tmpDir := t.TempDir()
	os.Setenv("GOPAW_TEST_DIR", tmpDir)
	defer os.Unsetenv("GOPAW_TEST_DIR")

	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write config with env var
	configContent := `
server:
  port: 8088
workspace:
  dir: "${GOPAW_TEST_DIR}/gopaw"
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	m, err := NewManager(configPath, nil)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// Verify env var was expanded
	cfg := m.Get()
	expectedDir := filepath.Join(tmpDir, "gopaw")
	if cfg.Workspace.Dir != expectedDir {
		t.Fatalf("expected dir %s, got %s", expectedDir, cfg.Workspace.Dir)
	}
}

// TestConfigDefaults tests that default values are applied.
func TestConfigDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write minimal config
	configContent := `
server:
  port: 8088
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	// Load config
	m, err := NewManager(configPath, nil)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	cfg := m.Get()

	// Check defaults
	if cfg.App.Name != "GoPaw" {
		t.Fatalf("expected app name GoPaw, got %s", cfg.App.Name)
	}
	if cfg.Agent.MaxSteps != 20 {
		t.Fatalf("expected max steps 20, got %d", cfg.Agent.MaxSteps)
	}
	if cfg.Agent.Memory.ContextLimit != 4000 {
		t.Fatalf("expected context limit 4000, got %d", cfg.Agent.Memory.ContextLimit)
	}
	if cfg.Workspace.Dir != "~/.gopaw" {
		t.Fatalf("expected workspace dir ~/.gopaw, got %s", cfg.Workspace.Dir)
	}
}