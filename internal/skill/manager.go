// Package skill manages the loading, registration and lifecycle of GoPaw Skills.
package skill

import (
	"fmt"

	"github.com/gopaw/gopaw/internal/tool"
	"go.uber.org/zap"
)

// Manager coordinates Skill loading and exposes helpers used by other modules.
type Manager struct {
	registry  *Registry
	loader    *Loader
	toolReg   *tool.Registry
	logger    *zap.Logger
}

// NewManager creates a Manager.
func NewManager(skillsDir string, toolReg *tool.Registry, logger *zap.Logger) *Manager {
	registry := NewRegistry()
	loader := NewLoader(skillsDir, registry, logger)
	return &Manager{
		registry: registry,
		loader:   loader,
		toolReg:  toolReg,
		logger:   logger,
	}
}

// Load discovers skills from the filesystem and initialises Level-3 code skills.
func (m *Manager) Load(enabledList []string) error {
	if err := m.loader.LoadAll(enabledList); err != nil {
		return fmt.Errorf("skill manager: load: %w", err)
	}

	// Register tools from Level-3 code skills that are enabled.
	for _, e := range m.registry.All() {
		if !e.Enabled || e.CodeSkill == nil {
			continue
		}
		for _, t := range e.CodeSkill.Tools() {
			m.toolReg.Register(t)
			m.logger.Info("skill tool registered",
				zap.String("skill", e.Manifest.Name),
				zap.String("tool", t.Name()),
			)
		}
	}
	return nil
}

// SystemPromptFragment returns the combined prompt text to inject into the agent system prompt.
func (m *Manager) SystemPromptFragment() string {
	return m.registry.ActivePromptFragments()
}

// Registry returns the underlying skill registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}
