// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package agent

import "fmt"

// AgentConfig represents the complete agent configuration for multi-agent mode.
type AgentConfig struct {
	// LLM configuration
	LLM LLMConfig `yaml:"llm" json:"llm"`

	// System prompt for the agent
	SystemPrompt string `yaml:"system_prompt" json:"system_prompt"`

	// Tools configuration
	Tools ToolsConfig `yaml:"tools" json:"tools"`

	// Skills to enable
	Skills []string `yaml:"skills" json:"skills"`

	// Autonomy levels for tools
	Autonomy map[string]string `yaml:"autonomy" json:"autonomy"`

	// Workspace configuration
	Workspace WorkspaceConfig `yaml:"workspace" json:"workspace"`

	// Memory configuration
	Memory MemoryConfig `yaml:"memory" json:"memory"`

	// Max steps for tool execution
	MaxSteps int `yaml:"max_steps" json:"max_steps"`
}

// LLMConfig represents LLM configuration.
type LLMConfig struct {
	// Model name (e.g., "gpt-4", "claude-3-opus")
	Model string `yaml:"model" json:"model"`

	// Temperature (0-2)
	Temperature float32 `yaml:"temperature" json:"temperature"`

	// Max tokens for response
	MaxTokens int `yaml:"max_tokens" json:"max_tokens"`

	// Top P sampling
	TopP float32 `yaml:"top_p" json:"top_p"`

	// Presence penalty
	PresencePenalty float32 `yaml:"presence_penalty" json:"presence_penalty"`

	// Frequency penalty
	FrequencyPenalty float32 `yaml:"frequency_penalty" json:"frequency_penalty"`
}

// ToolsConfig represents tools configuration.
type ToolsConfig struct {
	// Enabled tools (if empty, all tools are enabled)
	Enabled []string `yaml:"enabled" json:"enabled"`

	// Disabled tools
	Disabled []string `yaml:"disabled" json:"disabled"`
}

// WorkspaceConfig represents workspace configuration.
type WorkspaceConfig struct {
	// Root directory for agent workspace (relative to agents dir)
	Root string `yaml:"root" json:"root"`

	// Allowed paths within workspace
	AllowedPaths []string `yaml:"allowed_paths" json:"allowed_paths"`
}

// MemoryConfig represents memory configuration.
type MemoryConfig struct {
	// Whether memory is enabled
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Namespace for memory isolation
	Namespace string `yaml:"namespace" json:"namespace"`
}

// DefaultAgentConfig returns a default agent configuration.
func DefaultAgentConfig() *AgentConfig {
	return &AgentConfig{
		LLM: LLMConfig{
			Model:       "gpt-4",
			Temperature: 0.7,
			MaxTokens:   4000,
			TopP:        1.0,
		},
		SystemPrompt: "You are a helpful AI assistant.",
		Tools: ToolsConfig{
			Enabled:  []string{},
			Disabled: []string{},
		},
		Skills:   []string{},
		Autonomy: DefaultAutonomy(),
		Workspace: WorkspaceConfig{
			Root:         "",
			AllowedPaths: []string{},
		},
		Memory: MemoryConfig{
			Enabled:   true,
			Namespace: "default",
		},
		MaxSteps: 20,
	}
}

// DefaultAutonomy returns default autonomy levels.
func DefaultAutonomy() map[string]string {
	return map[string]string{
		"read_file":     "L1",
		"write_file":    "L2",
		"file_edit":     "L2",
		"file_manage":   "L2",
		"file_search":   "L1",
		"shell":         "L3",
		"web_search":    "L1",
		"http_get":      "L1",
		"http_post":     "L2",
		"memory_store":  "L1",
		"memory_recall": "L1",
		"memory_search": "L1",
		"send_to_user":  "L2",
	}
}

// IsToolEnabled checks if a tool is enabled for this agent.
func (c *AgentConfig) IsToolEnabled(toolName string) bool {
	// If enabled list is specified, tool must be in it
	if len(c.Tools.Enabled) > 0 {
		for _, t := range c.Tools.Enabled {
			if t == toolName {
				return true
			}
		}
		return false
	}

	// Otherwise, check if tool is in disabled list
	for _, t := range c.Tools.Disabled {
		if t == toolName {
			return false
		}
	}

	return true
}

// GetAutonomyLevel returns the autonomy level for a tool.
func (c *AgentConfig) GetAutonomyLevel(toolName string) string {
	if level, ok := c.Autonomy[toolName]; ok {
		return level
	}
	return "L2" // Default to L2
}

// MergeWithDefault merges the config with default values.
func (c *AgentConfig) MergeWithDefault() {
	defaultCfg := DefaultAgentConfig()

	if c.LLM.Model == "" {
		c.LLM.Model = defaultCfg.LLM.Model
	}
	if c.LLM.Temperature == 0 {
		c.LLM.Temperature = defaultCfg.LLM.Temperature
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = defaultCfg.LLM.MaxTokens
	}
	if c.SystemPrompt == "" {
		c.SystemPrompt = defaultCfg.SystemPrompt
	}
	if c.MaxSteps == 0 {
		c.MaxSteps = defaultCfg.MaxSteps
	}
	if c.Autonomy == nil {
		c.Autonomy = defaultCfg.Autonomy
	}
	if c.Memory.Namespace == "" {
		c.Memory.Namespace = defaultCfg.Memory.Namespace
	}
}

// Validate validates the configuration.
func (c *AgentConfig) Validate() error {
	if c.LLM.Model == "" {
		return fmt.Errorf("llm.model is required")
	}
	if c.SystemPrompt == "" {
		return fmt.Errorf("system_prompt is required")
	}
	if c.MaxSteps < 1 || c.MaxSteps > 100 {
		return fmt.Errorf("max_steps must be between 1 and 100")
	}
	return nil
}
