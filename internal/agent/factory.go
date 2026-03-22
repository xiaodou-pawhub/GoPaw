// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/sandbox"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/internal/trace"
	"go.uber.org/zap"
)

// Factory creates agent instances from definitions.
type Factory struct {
	// Shared components
	llmClient         llm.Client
	toolRegistry      *tool.Registry
	skillManager      *skill.Manager
	memoryManager     *memory.Manager
	ltmStore          *memory.LTMStore
	sandboxManager    *sandbox.Manager
	traceManager      *trace.Manager
	globalKnowledgeFn func(ctx context.Context) (string, error)

	// Configuration
	workspaceRoot string
	logger        *zap.Logger
}

// FactoryConfig holds configuration for the factory.
type FactoryConfig struct {
	LLMClient         llm.Client
	ToolRegistry      *tool.Registry
	SkillManager      *skill.Manager
	MemoryManager     *memory.Manager
	LTMStore          *memory.LTMStore
	SandboxManager    *sandbox.Manager
	TraceManager      *trace.Manager
	WorkspaceRoot     string
	Logger            *zap.Logger
	GlobalKnowledgeFn func(ctx context.Context) (string, error)
}

// NewFactory creates a new agent factory.
func NewFactory(cfg FactoryConfig) *Factory {
	return &Factory{
		llmClient:         cfg.LLMClient,
		toolRegistry:      cfg.ToolRegistry,
		skillManager:      cfg.SkillManager,
		memoryManager:     cfg.MemoryManager,
		ltmStore:          cfg.LTMStore,
		sandboxManager:    cfg.SandboxManager,
		traceManager:      cfg.TraceManager,
		workspaceRoot:     cfg.WorkspaceRoot,
		globalKnowledgeFn: cfg.GlobalKnowledgeFn,
		logger:            cfg.Logger.Named("agent_factory"),
	}
}

// CreateAgent creates an agent instance from a definition.
func (f *Factory) CreateAgent(def *Definition) (*ReActAgent, error) {
	if def.Config == nil {
		return nil, fmt.Errorf("agent config is nil")
	}

	// Merge with defaults
	config := *def.Config
	config.MergeWithDefault()

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid agent config: %w", err)
	}

	// Create agent workspace if needed
	agentWorkspace := f.getAgentWorkspace(def)
	if err := os.MkdirAll(agentWorkspace, 0755); err != nil {
		return nil, fmt.Errorf("failed to create agent workspace: %w", err)
	}

	// Create agent-specific directories
	dirs := []string{
		filepath.Join(agentWorkspace, "data"),
		filepath.Join(agentWorkspace, "skills"),
		filepath.Join(agentWorkspace, "temp"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Build agent config
	agentCfg := Config{
		DefaultPrompt:       config.SystemPrompt,
		AgentMDPath:         "", // Not used in multi-agent mode
		LTMStore:            f.ltmStore,
		MemoryNotesDir:      filepath.Join(agentWorkspace, "memory"),
		MaxSteps:            config.MaxSteps,
		Hooks: Hooks{
			PreReasoning: []HookPreReasoning{InjectCurrentTime()},
			PostTool:     []HookPostTool{},
		},
		ConvLog:           nil, // Will be set by caller if needed
		FocusManager:      nil, // Will be set by caller if needed
		TraceManager:      f.traceManager,
		SandboxManager:    f.sandboxManager,
		GlobalKnowledgeFunc: f.globalKnowledgeFn,
	}

	// Create the agent
	agent := New(
		f.llmClient,
		f.toolRegistry,
		f.skillManager,
		f.memoryManager,
		agentCfg,
		f.logger,
	)

	// Store agent metadata in the struct for later reference
	agent.defaultPrompt = config.SystemPrompt

	f.logger.Info("agent created",
		zap.String("id", def.ID),
		zap.String("name", def.Name),
		zap.String("workspace", agentWorkspace),
	)

	return agent, nil
}

// getAgentWorkspace returns the workspace path for an agent.
func (f *Factory) getAgentWorkspace(def *Definition) string {
	if def.Config != nil && def.Config.Workspace.Root != "" {
		// Use configured root (relative to workspace root)
		return filepath.Join(f.workspaceRoot, def.Config.Workspace.Root)
	}
	// Default: agents/{id}
	return filepath.Join(f.workspaceRoot, "agents", def.ID)
}

// CreateDefaultAgent creates a default agent.
func (f *Factory) CreateDefaultAgent() (*ReActAgent, error) {
	def := &Definition{
		ID:          "default",
		Name:        "默认助手",
		Description: "通用的 AI 助手",
		Avatar:      "🤖",
		Config:      DefaultAgentConfig(),
		IsActive:    true,
		IsDefault:   true,
	}

	return f.CreateAgent(def)
}

// GetAgentWorkspace returns the workspace path for an agent ID.
func (f *Factory) GetAgentWorkspace(agentID string) string {
	return filepath.Join(f.workspaceRoot, "agents", agentID)
}
