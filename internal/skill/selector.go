// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package skill

import (
	"math"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SkillScore represents a skill with its relevance score.
type SkillScore struct {
	Entry     *Entry
	Score     float64 // 0.0 - 1.0
	MatchType string  // "always", "keyword", "semantic", "frequency"
}

// SmartSelector provides intelligent skill selection based on user input.
type SmartSelector struct {
	registry  *Registry
	store     *UsageStore
	logger    *zap.Logger

	// Usage tracking for learning
	usageMu       sync.RWMutex
	usageCount    map[string]int       // skill name -> usage count
	lastUsed      map[string]time.Time // skill name -> last used time
	totalUsage    int
}

// NewSmartSelector creates a new smart skill selector.
// If store is provided, it will load persisted usage data.
func NewSmartSelector(registry *Registry, store *UsageStore, logger *zap.Logger) *SmartSelector {
	s := &SmartSelector{
		registry:   registry,
		store:      store,
		logger:     logger.Named("skill_selector"),
		usageCount: make(map[string]int),
		lastUsed:   make(map[string]time.Time),
	}

	// Load persisted data if store is available
	if store != nil {
		if err := s.loadFromStore(); err != nil {
			logger.Warn("failed to load skill usage from store", zap.Error(err))
		}
	}

	return s
}

// loadFromStore loads usage data from the persistent store.
func (s *SmartSelector) loadFromStore() error {
	records, err := s.store.LoadAll()
	if err != nil {
		return err
	}

	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	for _, r := range records {
		s.usageCount[r.SkillName] = r.Count
		s.lastUsed[r.SkillName] = r.LastUsed
		s.totalUsage += r.Count
	}

	s.logger.Info("loaded skill usage from store",
		zap.Int("records", len(records)),
		zap.Int("total_usage", s.totalUsage),
	)
	return nil
}

// SelectSkills returns skills sorted by relevance to the user input.
// Options control the selection behavior.
func (s *SmartSelector) SelectSkills(input string, opts SelectionOptions) []SkillScore {
	start := time.Now()
	lowerInput := strings.ToLower(input)

	var scores []SkillScore

	for _, entry := range s.registry.All() {
		if !entry.Enabled || entry.Prompt == "" {
			continue
		}

		score := s.calculateScore(entry, lowerInput, opts)
		if score.Score > 0 {
			scores = append(scores, score)
		}
	}

	// Sort by score descending
	sortSkillsByScore(scores)

	s.logger.Debug("skills selected",
		zap.String("input", input[:min(len(input), 50)]),
		zap.Int("total_skills", len(scores)),
		zap.Int("max_skills", opts.MaxSkills),
		zap.Duration("duration", time.Since(start)),
	)

	// Apply limit
	if opts.MaxSkills > 0 && len(scores) > opts.MaxSkills {
		scores = scores[:opts.MaxSkills]
	}

	return scores
}

// SelectionOptions controls skill selection behavior.
type SelectionOptions struct {
	MaxSkills       int     // Maximum number of skills to return (0 = unlimited)
	MinScore        float64 // Minimum relevance score (0.0 - 1.0)
	UseFrequency    bool    // Whether to consider usage frequency
	UseSemantic     bool    // Whether to use semantic matching
	ContextLength   int     // Current context length for adaptive selection
	ContextBudget   int     // Token budget for skills
}

// DefaultSelectionOptions returns default options.
func DefaultSelectionOptions() SelectionOptions {
	return SelectionOptions{
		MaxSkills:     3,
		MinScore:      0.1,
		UseFrequency:  true,
		UseSemantic:   true,
		ContextBudget: 500, // Default token budget for skills
	}
}

// calculateScore calculates the relevance score for a skill.
func (s *SmartSelector) calculateScore(entry *Entry, lowerInput string, opts SelectionOptions) SkillScore {
	score := SkillScore{Entry: entry}

	// Always included skills get maximum priority
	if entry.Manifest.Activation.Always {
		score.Score = 1.0
		score.MatchType = "always"
		return score
	}

	var keywordScore float64
	var matchedKeyword bool

	// Keyword matching with TF-IDF-like scoring
	for _, kw := range entry.Manifest.Activation.Keywords {
		kwLower := strings.ToLower(kw)
		if strings.Contains(lowerInput, kwLower) {
			// Exact match gets higher score
			keywordScore += 1.0
			matchedKeyword = true
		} else if opts.UseSemantic {
			// Partial match (e.g., "code" matches "coding")
			if strings.Contains(lowerInput, kwLower[:min(len(kwLower), 4)]) {
				keywordScore += 0.5
				matchedKeyword = true
			}
		}
	}

	if matchedKeyword {
		// Normalize by number of keywords
		score.Score = keywordScore / float64(len(entry.Manifest.Activation.Keywords))
		score.MatchType = "keyword"
	}

	// Boost by usage frequency
	if opts.UseFrequency {
		freqBoost := s.getFrequencyBoost(entry.Manifest.Name)
		score.Score = score.Score*(1-freqBoost) + freqBoost
	}

	// Apply minimum score threshold
	if score.Score < opts.MinScore {
		score.Score = 0
	}

	return score
}

// getFrequencyBoost returns a boost factor based on usage frequency (0.0 - 0.3).
func (s *SmartSelector) getFrequencyBoost(skillName string) float64 {
	s.usageMu.RLock()
	defer s.usageMu.RUnlock()

	if s.totalUsage == 0 {
		return 0
	}

	count := s.usageCount[skillName]
	if count == 0 {
		return 0
	}

	// Calculate frequency ratio and apply logarithmic scaling
	ratio := float64(count) / float64(s.totalUsage)
	boost := math.Log1p(ratio*10) / 10 // Max boost ~0.23

	// Decay boost for skills not used recently
	if lastUse, ok := s.lastUsed[skillName]; ok {
		daysSinceUse := time.Since(lastUse).Hours() / 24
		if daysSinceUse > 7 {
			boost *= 0.5 // Halve boost if not used for a week
		}
	}

	return math.Min(boost, 0.3) // Cap at 0.3
}

// RecordUsage records that a skill was used.
func (s *SmartSelector) RecordUsage(skillName string) {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	s.usageCount[skillName]++
	s.lastUsed[skillName] = time.Now()
	s.totalUsage++

	// Persist to store if available
	if s.store != nil {
		record := &UsageRecord{
			SkillName: skillName,
			Count:     s.usageCount[skillName],
			LastUsed:  s.lastUsed[skillName],
		}
		if err := s.store.Save(record); err != nil {
			s.logger.Warn("failed to persist skill usage",
				zap.String("skill", skillName),
				zap.Error(err),
			)
		}
	}

	s.logger.Debug("skill usage recorded",
		zap.String("skill", skillName),
		zap.Int("count", s.usageCount[skillName]),
	)
}

// GetUsageStats returns usage statistics for all skills.
func (s *SmartSelector) GetUsageStats() map[string]int {
	s.usageMu.RLock()
	defer s.usageMu.RUnlock()

	stats := make(map[string]int)
	for name, count := range s.usageCount {
		stats[name] = count
	}
	return stats
}

// Cleanup removes stale usage records for skills not used in the given duration.
// This prevents memory leaks from accumulated usage data.
func (s *SmartSelector) Cleanup(maxAge time.Duration) {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for name, lastUse := range s.lastUsed {
		if lastUse.Before(cutoff) {
			// Remove stale records
			s.totalUsage -= s.usageCount[name]
			delete(s.usageCount, name)
			delete(s.lastUsed, name)
			cleaned++
		}
	}

	if cleaned > 0 {
		s.logger.Info("cleaned up stale skill usage records",
			zap.Int("cleaned", cleaned),
			zap.Duration("max_age", maxAge),
		)
	}
}

// Reset clears all usage data.
// Useful for testing or when user explicitly wants to reset learning.
func (s *SmartSelector) Reset() {
	s.usageMu.Lock()
	defer s.usageMu.Unlock()

	s.usageCount = make(map[string]int)
	s.lastUsed = make(map[string]time.Time)
	s.totalUsage = 0

	s.logger.Info("skill usage data reset")
}

// sortSkillsByScore sorts skills by score descending.
func sortSkillsByScore(scores []SkillScore) {
	// Simple bubble sort (sufficient for small lists)
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].Score > scores[i].Score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
