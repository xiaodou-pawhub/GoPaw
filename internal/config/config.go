// Package config handles configuration loading, validation and access throughout GoPaw.
// It uses Viper for multi-format support, environment-variable substitution and hot-reload.
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
	Name     string `mapstructure:"name"`
	Language string `mapstructure:"language"`
	Timezone string `mapstructure:"timezone"`
	Debug    bool   `mapstructure:"debug"`
}

// ServerConfig holds HTTP server bind settings.
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// StorageConfig selects and configures the persistence backend.
type StorageConfig struct {
	Type string `mapstructure:"type"`
	Path string `mapstructure:"path"`
}

// LLMConfig describes how to reach the language model provider.
type LLMConfig struct {
	Provider        string `mapstructure:"provider"`
	BaseURL         string `mapstructure:"base_url"`
	APIKey          string `mapstructure:"api_key"`
	Model           string `mapstructure:"model"`
	TimeoutSeconds  int    `mapstructure:"timeout"`
	MaxTokens       int    `mapstructure:"max_tokens"`
	// CustomEndpoint is used when provider == "custom".
	CustomEndpoint  string `mapstructure:"endpoint"`
	// ResponsePath is a dot-path into the JSON response body for custom providers.
	ResponsePath    string `mapstructure:"response_path"`
}

// MemoryConfig tunes the in-session context window and persistence.
type MemoryConfig struct {
	ContextLimit int `mapstructure:"context_limit"`
	HistoryLimit int `mapstructure:"history_limit"`
}

// AgentConfig controls agent behaviour.
type AgentConfig struct {
	SystemPrompt string       `mapstructure:"system_prompt"`
	MaxSteps     int          `mapstructure:"max_steps"`
	Memory       MemoryConfig `mapstructure:"memory"`
}

// PluginsConfig lists which channel plugins are enabled.
type PluginsConfig struct {
	Enabled []string `mapstructure:"enabled"`
}

// FeishuPluginConfig holds Feishu (Lark) Open Platform credentials.
type FeishuPluginConfig struct {
	AppID             string `mapstructure:"app_id"`
	AppSecret         string `mapstructure:"app_secret"`
	VerificationToken string `mapstructure:"verification_token"`
	EncryptKey        string `mapstructure:"encrypt_key"`
}

// DingTalkPluginConfig holds DingTalk credentials.
type DingTalkPluginConfig struct {
	ClientID     string `mapstructure:"client_id"`
	ClientSecret string `mapstructure:"client_secret"`
}

// WebhookPluginConfig holds settings for the generic Webhook channel.
type WebhookPluginConfig struct {
	Token       string `mapstructure:"token"`
	CallbackURL string `mapstructure:"callback_url"`
}

// PluginConfig aggregates per-plugin configurations.
type PluginConfig struct {
	Feishu   FeishuPluginConfig   `mapstructure:"feishu"`
	DingTalk DingTalkPluginConfig `mapstructure:"dingtalk"`
	Webhook  WebhookPluginConfig  `mapstructure:"webhook"`
}

// SkillsConfig controls which skills are loaded and their per-skill settings.
type SkillsConfig struct {
	Dir     string                 `mapstructure:"dir"`
	Enabled []string               `mapstructure:"enabled"`
	Config  map[string]interface{} `mapstructure:"config"`
}

// LogConfig controls structured log output.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
	Output string `mapstructure:"output"`
	File   string `mapstructure:"file"`
}

// Config is the root configuration structure for the entire application.
type Config struct {
	App     AppConfig     `mapstructure:"app"`
	Server  ServerConfig  `mapstructure:"server"`
	Storage StorageConfig `mapstructure:"storage"`
	LLM     LLMConfig     `mapstructure:"llm"`
	Agent   AgentConfig   `mapstructure:"agent"`
	Plugins PluginsConfig `mapstructure:"plugins"`
	Plugin  PluginConfig  `mapstructure:"plugin"`
	Skills  SkillsConfig  `mapstructure:"skills"`
	Log     LogConfig     `mapstructure:"log"`
}

var envVarRe = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// expandEnvVars replaces ${ENV_VAR} placeholders with the actual environment variable values.
func expandEnvVars(s string) string {
	return envVarRe.ReplaceAllStringFunc(s, func(match string) string {
		key := envVarRe.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(key); ok {
			return val
		}
		return match // leave unreplaced if the variable is not set
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
	v.SetDefault("app.name", "GoPaw")
	v.SetDefault("app.language", "zh")
	v.SetDefault("app.timezone", "Asia/Shanghai")
	v.SetDefault("app.debug", false)

	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8088)

	v.SetDefault("storage.type", "sqlite")
	v.SetDefault("storage.path", "data/gopaw.db")

	v.SetDefault("llm.provider", "openai_compatible")
	v.SetDefault("llm.base_url", "https://api.openai.com/v1")
	v.SetDefault("llm.model", "gpt-4o-mini")
	v.SetDefault("llm.timeout", 60)
	v.SetDefault("llm.max_tokens", 4096)

	v.SetDefault("agent.max_steps", 20)
	v.SetDefault("agent.memory.context_limit", 4000)
	v.SetDefault("agent.memory.history_limit", 50)

	v.SetDefault("skills.dir", "skills/")

	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
}

// unmarshal deserialises Viper settings into the typed Config, expanding env vars.
func (m *Manager) unmarshal() error {
	// Write back the raw map so that string values get env-var expansion.
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
