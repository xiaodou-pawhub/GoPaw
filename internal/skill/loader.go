// Package skill manages the loading, registration and lifecycle of GoPaw Skills.
package skill

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gopaw/gopaw/pkg/plugin"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// Loader discovers and loads skills from the filesystem.
type Loader struct {
	skillsDir string
	registry  *Registry
	logger    *zap.Logger
}

// NewLoader creates a Loader that reads from the given directory.
func NewLoader(skillsDir string, registry *Registry, logger *zap.Logger) *Loader {
	return &Loader{skillsDir: skillsDir, registry: registry, logger: logger}
}

// LoadAll scans the skills directory and loads every valid skill it finds.
// enabledList restricts loading to named skills; pass nil to load all.
func (l *Loader) LoadAll(enabledList []string) error {
	enabledSet := toSet(enabledList)

	entries, err := os.ReadDir(l.skillsDir)
	if err != nil {
		if os.IsNotExist(err) {
			l.logger.Info("skills directory does not exist, skipping", zap.String("dir", l.skillsDir))
			return nil
		}
		return fmt.Errorf("skill loader: read dir %q: %w", l.skillsDir, err)
	}

	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		skillDir := filepath.Join(l.skillsDir, de.Name())
		entry, err := l.loadSkill(skillDir)
		if err != nil {
			l.logger.Warn("skill load failed", zap.String("dir", skillDir), zap.Error(err))
			continue
		}
		// Determine if skill should be enabled.
		if len(enabledSet) > 0 {
			entry.Enabled = enabledSet[entry.Manifest.Name]
		} else {
			entry.Enabled = true
		}
		if err := l.registry.Register(entry); err != nil {
			return fmt.Errorf("skill loader: register %q: %w", entry.Manifest.Name, err)
		}
		l.logger.Info("skill loaded",
			zap.String("name", entry.Manifest.Name),
			zap.Int("level", int(entry.Manifest.Level)),
			zap.Bool("enabled", entry.Enabled),
		)
	}
	return nil
}

// loadSkill parses a single skill directory and returns the populated Entry.
func (l *Loader) loadSkill(dir string) (*Entry, error) {
	// --- manifest.yaml (required) ---
	manifestPath := filepath.Join(dir, "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	var manifest plugin.SkillManifest
	if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	if manifest.Name == "" {
		return nil, fmt.Errorf("manifest.name is required")
	}

	// --- prompt.md (required for Level 1 prompt skills, optional for Level 2 code skills) ---
	promptPath := filepath.Join(dir, "prompt.md")
	promptData, _ := os.ReadFile(promptPath)

	entry := &Entry{
		Manifest: &manifest,
		Prompt:   string(promptData),
	}

	l.logger.Debug("skill parsed",
		zap.String("name", manifest.Name),
		zap.Int("level", int(manifest.Level)),
		zap.Bool("has_prompt", len(promptData) > 0),
	)

	return entry, nil
}

// toSet converts a string slice to a membership set for O(1) lookup.
func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]bool, len(items))
	for _, item := range items {
		s[item] = true
	}
	return s
}
