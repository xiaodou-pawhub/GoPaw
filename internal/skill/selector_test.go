// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package skill

import (
	"testing"

	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewSmartSelector(t *testing.T) {
	registry := NewRegistry()
	logger := zap.NewNop()
	selector := NewSmartSelector(registry, nil, logger)

	assert.NotNil(t, selector)
	assert.NotNil(t, selector.registry)
	assert.NotNil(t, selector.usageCount)
	assert.NotNil(t, selector.lastUsed)
}

func TestSmartSelector_SelectSkills_Always(t *testing.T) {
	registry := NewRegistry()
	logger := zap.NewNop()
	selector := NewSmartSelector(registry, nil, logger)

	// Register a skill with always:true
	entry := &Entry{
		Manifest: &plugin.SkillManifest{
			Name:        "always_skill",
			Description: "Always included",
			Activation: plugin.SkillActivation{
				Always: true,
			},
		},
		Prompt:  "This skill is always included",
		Enabled: true,
	}
	registry.Register(entry)

	opts := DefaultSelectionOptions()
	scores := selector.SelectSkills("test input", opts)

	assert.Len(t, scores, 1)
	assert.Equal(t, "always_skill", scores[0].Entry.Manifest.Name)
	assert.Equal(t, 1.0, scores[0].Score)
	assert.Equal(t, "always", scores[0].MatchType)
}

func TestSmartSelector_SelectSkills_KeywordMatch(t *testing.T) {
	registry := NewRegistry()
	logger := zap.NewNop()
	selector := NewSmartSelector(registry, nil, logger)

	// Register a skill with keywords
	entry := &Entry{
		Manifest: &plugin.SkillManifest{
			Name:        "code_review",
			Description: "Review code",
			Activation: plugin.SkillActivation{
				Always:   false,
				Keywords: []string{"code", "review", "pr"},
			},
		},
		Prompt:  "Code review skill",
		Enabled: true,
	}
	registry.Register(entry)

	opts := DefaultSelectionOptions()

	// Test matching input
	scores := selector.SelectSkills("Please review my code", opts)
	assert.Len(t, scores, 1)
	assert.Equal(t, "code_review", scores[0].Entry.Manifest.Name)
	assert.Greater(t, scores[0].Score, 0.0)
	assert.Equal(t, "keyword", scores[0].MatchType)

	// Test non-matching input
	scores = selector.SelectSkills("What's the weather?", opts)
	assert.Len(t, scores, 0)
}

func TestSmartSelector_SelectSkills_Disabled(t *testing.T) {
	registry := NewRegistry()
	logger := zap.NewNop()
	selector := NewSmartSelector(registry, nil, logger)

	// Register a disabled skill
	entry := &Entry{
		Manifest: &plugin.SkillManifest{
			Name:        "disabled_skill",
			Description: "Disabled",
			Activation: plugin.SkillActivation{
				Always:   true,
				Keywords: []string{"test"},
			},
		},
		Prompt:  "Disabled skill",
		Enabled: false,
	}
	registry.Register(entry)

	opts := DefaultSelectionOptions()
	scores := selector.SelectSkills("test", opts)

	assert.Len(t, scores, 0)
}

func TestSmartSelector_RecordUsage(t *testing.T) {
	registry := NewRegistry()
	logger := zap.NewNop()
	selector := NewSmartSelector(registry, nil, logger)

	// Record usage
	selector.RecordUsage("skill1")
	selector.RecordUsage("skill1")
	selector.RecordUsage("skill2")

	// Check stats
	stats := selector.GetUsageStats()
	assert.Equal(t, 2, stats["skill1"])
	assert.Equal(t, 1, stats["skill2"])
}

func TestSmartSelector_GetFrequencyBoost(t *testing.T) {
	registry := NewRegistry()
	logger := zap.NewNop()
	selector := NewSmartSelector(registry, nil, logger)

	// Initially no boost
	boost := selector.getFrequencyBoost("skill1")
	assert.Equal(t, 0.0, boost)

	// Record usage
	for i := 0; i < 10; i++ {
		selector.RecordUsage("skill1")
	}

	// Should have some boost now
	boost = selector.getFrequencyBoost("skill1")
	assert.Greater(t, boost, 0.0)
	assert.LessOrEqual(t, boost, 0.3) // Max boost is 0.3
}

func TestSortSkillsByScore(t *testing.T) {
	scores := []SkillScore{
		{Entry: &Entry{Manifest: &plugin.SkillManifest{Name: "skill1"}}, Score: 0.5},
		{Entry: &Entry{Manifest: &plugin.SkillManifest{Name: "skill2"}}, Score: 0.9},
		{Entry: &Entry{Manifest: &plugin.SkillManifest{Name: "skill3"}}, Score: 0.3},
	}

	sortSkillsByScore(scores)

	assert.Equal(t, "skill2", scores[0].Entry.Manifest.Name)
	assert.Equal(t, "skill1", scores[1].Entry.Manifest.Name)
	assert.Equal(t, "skill3", scores[2].Entry.Manifest.Name)
}

func TestDefaultSelectionOptions(t *testing.T) {
	opts := DefaultSelectionOptions()

	assert.Equal(t, 3, opts.MaxSkills)
	assert.Equal(t, 0.1, opts.MinScore)
	assert.True(t, opts.UseFrequency)
	assert.True(t, opts.UseSemantic)
	assert.Equal(t, 500, opts.ContextBudget)
}

func TestMin(t *testing.T) {
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(2, 1))
	assert.Equal(t, 0, min(0, 0))
}
