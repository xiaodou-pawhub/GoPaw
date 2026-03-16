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
	toolReg     *tool.Registry
	selector    *SmartSelector
	toolToSkill map[string]string // tool name -> skill name
	logger      *zap.Logger
}

// NewManager creates a Manager.
// UsageStore is optional - if nil, usage data will not be persisted.
func NewManager(skillsDir string, toolReg *tool.Registry, store *UsageStore, logger *zap.Logger) *Manager {
	registry := NewRegistry()
	loader := NewLoader(skillsDir, registry, logger)
	selector := NewSmartSelector(registry, store, logger)
	return &Manager{
		registry:    registry,
		loader:      loader,
		toolReg:     toolReg,
		selector:    selector,
		toolToSkill: make(map[string]string),
		logger:      logger,
	}
}

// Load discovers skills from the filesystem and initialises Level-3 code skills.
func (m *Manager) Load(enabledList []string) error {
	if err := m.loader.LoadAll(enabledList); err != nil {
		return fmt.Errorf("skill manager: load: %w", err)
	}

	// Register tools from Level-3 code skills that are enabled.
	// Build tool -> skill mapping for usage tracking.
	for _, e := range m.registry.All() {
		if !e.Enabled || e.CodeSkill == nil {
			continue
		}
		for _, t := range e.CodeSkill.Tools() {
			m.toolReg.Register(t)
			m.toolToSkill[t.Name()] = e.Manifest.Name // Build mapping
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
// Deprecated: Use SmartSelectForInput for intelligent skill selection.
func (m *Manager) FragmentsForInput(input string) string {
	return m.registry.ActivePromptFragmentsForInput(input)
}

// SmartSelectForInput returns skill fragments using intelligent selection.
// It considers keyword matching, usage frequency, and context budget.
func (m *Manager) SmartSelectForInput(input string, contextBudget int) string {
	opts := DefaultSelectionOptions()
	opts.ContextBudget = contextBudget

	scores := m.selector.SelectSkills(input, opts)
	if len(scores) == 0 {
		return ""
	}

	var fragments strings.Builder
	fragments.WriteString("## 相关技能\n\n")
	fragments.WriteString("| 技能 | 描述 | 文件 |\n")
	fragments.WriteString("|------|------|------|\n")

	totalTokens := 0
	maxTokens := contextBudget

	for _, score := range scores {
		// Estimate tokens
		line := fmt.Sprintf("| %s | %s | skills/%s.md |\n",
			score.Entry.Manifest.Name,
			score.Entry.Manifest.Description,
			score.Entry.Manifest.Name)
		estimatedTokens := len(line) / 4

		if totalTokens+estimatedTokens > maxTokens {
			m.logger.Debug("skill token budget exceeded",
				zap.Int("selected", len(scores)),
				zap.Int("total_tokens", totalTokens),
			)
			break
		}

		fragments.WriteString(line)
		totalTokens += estimatedTokens
	}

	fragments.WriteString("\n⚠️ 使用技能前请先读取完整文件")

	return fragments.String()
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
	m.toolToSkill = make(map[string]string) // Clear tool mapping
	m.logger.Info("skill registry cleared, reloading from disk")
	return m.Load(nil)
}

// Registry returns the underlying skill registry.
func (m *Manager) Registry() *Registry {
	return m.registry
}
