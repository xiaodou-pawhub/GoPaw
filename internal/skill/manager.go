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

// FragmentsForInput returns skill prompt fragments matched against the current user input.
// Skills with always:true are always included; others are included only when the input
// contains at least one of their declared keywords.
func (m *Manager) FragmentsForInput(input string) string {
	return m.registry.ActivePromptFragmentsForInput(input)
}

// Reload clears all registered skills and re-scans the skills directory.
// Useful when the user adds or removes skill files at runtime.
func (m *Manager) Reload() error {
	m.registry.Clear()
	m.logger.Info("skill registry cleared, reloading from disk")
	return m.Load(nil)
}

// Registry returns the underlying skill registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}
