// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package agent implements the native Function Calling agent engine.
package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/internal/skill"
	"go.uber.org/zap"
)

// ContextBuilder dynamically builds system prompts based on user input and context.
// It retrieves relevant memories, matches active skills, and includes time context
// to provide personalized and contextual responses.
type ContextBuilder struct {
	persona      string           // Base persona from AGENT.md
	memoryMgr    *memory.Manager  // Memory manager for retrieval
	skillMgr     *skill.Manager   // Skill manager for matching
	tokenBudget  int              // Maximum tokens for dynamic content
	logger       *zap.Logger
}

// ContextBuildResult holds the result of context building.
type ContextBuildResult struct {
	SystemPrompt    string   // Final system prompt
	MemoriesUsed    int      // Number of memories included
	SkillsMatched   int      // Number of skills matched
	BuildTime       time.Duration // Time taken to build
}

// NewContextBuilder creates a new context builder.
func NewContextBuilder(
	persona string,
	memoryMgr *memory.Manager,
	skillMgr *skill.Manager,
	tokenBudget int,
	logger *zap.Logger,
) *ContextBuilder {
	if tokenBudget <= 0 {
		tokenBudget = 2000 // Default token budget for dynamic content
	}

	return &ContextBuilder{
		persona:     persona,
		memoryMgr:   memoryMgr,
		skillMgr:    skillMgr,
		tokenBudget: tokenBudget,
		logger:      logger.Named("context_builder"),
	}
}

// Build constructs a dynamic system prompt based on user input.
func (b *ContextBuilder) Build(ctx context.Context, sessionID, userInput string) (*ContextBuildResult, error) {
	start := time.Now()
	result := &ContextBuildResult{}

	b.logger.Debug("building context",
		zap.String("session", sessionID),
		zap.String("input", userInput[:min(len(userInput), 50)]),
	)

	var parts []string

	// 1. Base Persona (always included)
	parts = append(parts, b.persona)

	// 2. Relevant Memories (dynamic, token budget controlled)
	memoriesSection, memoriesUsed := b.buildRelevantMemories(ctx, sessionID, userInput)
	if memoriesSection != "" {
		parts = append(parts, memoriesSection)
		result.MemoriesUsed = memoriesUsed
	}

	// 3. Active Skills (dynamic, based on input matching)
	skillsSection, skillsMatched := b.buildActiveSkills(userInput)
	if skillsSection != "" {
		parts = append(parts, skillsSection)
		result.SkillsMatched = skillsMatched
	}

	// 4. Time Context (dynamic)
	parts = append(parts, b.buildTimeContext())

	// Combine all parts
	result.SystemPrompt = strings.Join(parts, "\n\n")
	result.BuildTime = time.Since(start)

	b.logger.Info("context built",
		zap.String("session", sessionID),
		zap.Int("memories_used", result.MemoriesUsed),
		zap.Int("skills_matched", result.SkillsMatched),
		zap.Duration("build_time", result.BuildTime),
		zap.Int("prompt_length", len(result.SystemPrompt)),
	)

	return result, nil
}

// buildRelevantMemories retrieves and formats relevant memories.
func (b *ContextBuilder) buildRelevantMemories(ctx context.Context, sessionID, query string) (string, int) {
	if b.memoryMgr == nil {
		return "", 0
	}

	// Search for relevant memories
	snippets, err := b.memoryMgr.Search(ctx, sessionID, query, 5, 0.5)
	if err != nil {
		b.logger.Warn("failed to search memories", zap.Error(err))
		return "", 0
	}

	if len(snippets) == 0 {
		return "", 0
	}

	var sb strings.Builder
	sb.WriteString("## 相关记忆\n\n")

	used := 0
	totalTokens := 0
	maxTokens := b.tokenBudget

	for _, snippet := range snippets {
		// Estimate tokens (rough approximation: 4 chars ≈ 1 token)
		content := fmt.Sprintf("- %s\n", snippet.Content)
		estimatedTokens := len(content) / 4

		if totalTokens+estimatedTokens > maxTokens {
			b.logger.Debug("memory token budget exceeded",
				zap.Int("used", used),
				zap.Int("total_tokens", totalTokens),
			)
			break
		}

		sb.WriteString(content)
		totalTokens += estimatedTokens
		used++
	}

	if used == 0 {
		return "", 0
	}

	return sb.String(), used
}

// buildActiveSkills matches and formats active skills.
func (b *ContextBuilder) buildActiveSkills(input string) (string, int) {
	if b.skillMgr == nil {
		return "", 0
	}

	// Get skill fragments based on input
	fragments := b.skillMgr.FragmentsForInput(input)
	if fragments == "" {
		return "", 0
	}

	var sb strings.Builder
	sb.WriteString("## 相关技能\n")
	sb.WriteString(fragments)
	sb.WriteString("\n\n⚠️ 使用技能前请先读取完整文件")

	// Count matched skills (rough estimate based on lines)
	lines := strings.Split(fragments, "\n")
	skillsMatched := 0
	for _, line := range lines {
		if strings.Contains(line, "|") && !strings.Contains(line, "---") {
			skillsMatched++
		}
	}

	return sb.String(), skillsMatched
}

// buildTimeContext generates time-related context.
func (b *ContextBuilder) buildTimeContext() string {
	now := time.Now()
	weekday := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}[now.Weekday()]

	return fmt.Sprintf("## 当前时间\n%s（%s）",
		now.Format("2006-01-02 15:04"),
		weekday,
	)
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
