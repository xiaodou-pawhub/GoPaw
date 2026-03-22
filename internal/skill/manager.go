// Package skill manages the loading, registration and lifecycle of GoPaw Skills.
package skill

import (
	"fmt"
	"strings"

	"github.com/gopaw/gopaw/internal/tool"
	"go.uber.org/zap"
)

// Manager coordinates Skill loading and exposes helpers used by other modules.
type Manager struct {
	registry    *Registry
	loader      *Loader
	selector    *SmartSelector
	toolToSkill map[string]string // tool name -> skill name (reserved for future code skills)
	logger      *zap.Logger
}

// NewManager creates a Manager.
// toolReg is accepted for interface compatibility but currently unused.
// UsageStore is optional - if nil, usage data will not be persisted.
func NewManager(skillsDir string, _ *tool.Registry, store *UsageStore, logger *zap.Logger) *Manager {
	registry := NewRegistry()
	loader := NewLoader(skillsDir, registry, logger)
	selector := NewSmartSelector(registry, store, logger)
	return &Manager{
		registry:    registry,
		loader:      loader,
		selector:    selector,
		toolToSkill: make(map[string]string),
		logger:      logger,
	}
}

// Load discovers skills from the filesystem.
func (m *Manager) Load(enabledList []string) error {
	if err := m.loader.LoadAll(enabledList); err != nil {
		return fmt.Errorf("skill manager: load: %w", err)
	}
	return nil
}

// SmartSelectForInput returns all enabled Level-1 skill prompt fragments,
// ordered by usage frequency, within the given token budget.
func (m *Manager) SmartSelectForInput(input string, contextBudget int) string {
	opts := DefaultSelectionOptions()
	opts.ContextBudget = contextBudget

	scores := m.selector.SelectSkills(input, opts)
	if len(scores) == 0 {
		return ""
	}

	var sb strings.Builder
	totalTokens := 0

	for _, score := range scores {
		if score.Entry.Prompt == "" {
			continue
		}
		estimated := len(score.Entry.Prompt) / 4
		if totalTokens+estimated > contextBudget {
			m.logger.Debug("skill token budget reached",
				zap.String("skill", score.Entry.Manifest.Name),
				zap.Int("budget", contextBudget),
			)
			break
		}
		sb.WriteString(score.Entry.Prompt)
		sb.WriteString("\n\n")
		totalTokens += estimated
	}

	return sb.String()
}

// RecordSkillUsage records that a skill was used (for learning).
func (m *Manager) RecordSkillUsage(skillName string) {
	m.selector.RecordUsage(skillName)
}

// GetSkillUsageStats returns usage statistics for all skills.
func (m *Manager) GetSkillUsageStats() map[string]int {
	return m.selector.GetUsageStats()
}

// GetSkillByTool returns the skill name that owns the given tool.
// Returns empty string if the tool doesn't belong to any skill.
func (m *Manager) GetSkillByTool(toolName string) string {
	return m.toolToSkill[toolName]
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
