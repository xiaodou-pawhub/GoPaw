// Package config handles configuration loading, validation and access throughout GoPaw.
// config.yaml contains only static startup settings (server, storage, log, plugin list).
// Runtime settings (LLM providers, channel secrets, agent persona) are managed via the Web UI
// and stored in SQLite — see internal/settings.
package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// AppConfig holds top-level application identity settings.
type AppConfig struct {
	Name       string `mapstructure:"name"`
	Language   string `mapstructure:"language"`
	Timezone   string `mapstructure:"timezone"`
	AdminToken string `mapstructure:"admin_token"`
}

// ServerConfig holds HTTP server bind settings.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// WorkspaceConfig holds the root directory for all runtime data.
// All paths (DB, logs, agent files) are derived from this directory.
type WorkspaceConfig struct {
	Dir string `mapstructure:"dir"`
}

// MemoryConfig tunes the in-session context window and persistence.
type MemoryConfig struct {
	ContextLimit int `mapstructure:"context_limit"`
	HistoryLimit int `mapstructure:"history_limit"`
}

// AgentConfig controls agent behaviour. LLM provider and system prompt are
// managed separately (Web UI → SQLite for provider, data/AGENT.md for prompt).
type AgentConfig struct {
	MaxSteps int          `mapstructure:"max_steps"`
	Memory   MemoryConfig `mapstructure:"memory"`
}

// SkillsConfig controls which skills are initially enabled.
// Skills are always loaded from {workspace.dir}/skills/ — no separate dir config needed.
type SkillsConfig struct {
	Enabled []string `mapstructure:"enabled"`
}

// LogConfig controls structured log output.
// Output: "stdout" (console only) | "file" (workspace log file only) | "both" (console + file)
// Log file path is always {workspace}/logs/gopaw.log — not configurable.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
}

// Config is the root configuration structure for the application startup settings.
type Config struct {
	Workspace  WorkspaceConfig   `mapstructure:"workspace"`
	App        AppConfig         `mapstructure:"app"`
	Server     ServerConfig      `mapstructure:"server"`
	Agent      AgentConfig       `mapstructure:"agent"`
	Skills     SkillsConfig      `mapstructure:"skills"`
	MCPServers []MCPServerConfig `mapstructure:"mcp_servers"`
	Log        LogConfig         `mapstructure:"log"`
}

// MCPServerConfig defines how to connect to an MCP server.
type MCPServerConfig struct {
	Name    string   `mapstructure:"name"`
	Command string   `mapstructure:"command"`
	Args    []string `mapstructure:"args"`
}

// Validate checks required configuration fields.
func (c *Config) Validate() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}
	if c.Workspace.Dir == "" {
		return fmt.Errorf("workspace.dir is required")
	}
	return nil
}

var envVarRe = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// expandEnvVars replaces ${ENV_VAR} placeholders with the actual environment variable values.
func expandEnvVars(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		key := envVarRe.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return match
	})
}

// Manager wraps Viper and exposes the typed Config struct.
type Manager struct {
	v      *viper.Viper
	cfg    *Config
	logger *zap.Logger

	// onChange is called whenever a live config reload succeeds.
	onChange []func(*Config)
}

// NewManager creates a Manager that loads configuration from the given file path.
func NewManager(cfgFile string, logger *zap.Logger) (*Manager, error) {
	v := viper.New()

	setDefaults(v)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("$HOME/.gopaw")
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("config: read config file: %w", err)
	}

	m := &Manager{v: v, logger: logger}
	if err := m.unmarshal(); err != nil {
		return nil, err
	}

	return m, nil
}

// setDefaults applies safe default values.
func setDefaults(v *viper.Viper) {
	v.SetDefault("workspace.dir", "~/.gopaw")

	v.SetDefault("app.name", "GoPaw")
	v.SetDefault("app.language", "zh")
	v.SetDefault("app.timezone", "Asia/Shanghai")

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8088)

	v.SetDefault("agent.max_steps", 20)
	v.SetDefault("agent.memory.context_limit", 4000)
	v.SetDefault("agent.memory.history_limit", 50)

	// skills dir is always derived from workspace.dir — no separate default needed

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
}

// unmarshal deserialises Viper settings into the typed Config, expanding env vars.
func (m *Manager) unmarshal() error {
	raw := m.v.AllSettings()
	expandMap(raw)
	for k, v := range flatten(raw, "") {
		m.v.Set(k, v)
	}

	cfg := &Config{}
	if err := m.v.Unmarshal(cfg); err != nil {
		return fmt.Errorf("config: unmarshal: %w", err)
	}
	m.cfg = cfg

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("config: validate: %w", err)
	}

	return nil
}

// expandMap recursively applies env-var expansion to all string leaf values.
func expandMap(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = expandEnvVars(val)
		case map[string]interface{}:
			expandMap(val)
		}
	}
}

// flatten converts a nested map to dot-delimited keys for Viper.Set.
func flatten(m map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			for fk, fv := range flatten(val, key) {
				result[fk] = fv
			}
		default:
			result[key] = val
		}
	}
	return result
}

// Get returns the current parsed configuration (immutable snapshot).
func (m *Manager) Get() *Config {
	return m.cfg
}

// OnChange registers a callback that fires after every successful hot-reload.
func (m *Manager) OnChange(fn func(*Config)) {
	m.onChange = append(m.onChange, fn)
}

// WatchConfig starts the Viper filesystem watcher for hot-reload.
func (m *Manager) WatchConfig() {
	m.v.WatchConfig()
	m.v.OnConfigChange(func(e fsnotify.Event) {
		m.logger.Info("config file changed, reloading", zap.String("file", e.Name))
		if err := m.unmarshal(); err != nil {
			m.logger.Error("config: reload failed", zap.Error(err))
			return
		}
		for _, fn := range m.onChange {
			fn(m.cfg)
		}
	})
}
