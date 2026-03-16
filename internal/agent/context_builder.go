// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package agent implements the native Function Calling agent engine.
package agent

import (
	"context"
	"fmt"
	"os"
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
	persona        string           // Base persona from AGENT.md
	memoryMgr      *memory.Manager  // Memory manager for retrieval
	ltmStore       *memory.LTMStore // Long-term memory store for Core memories
	skillMgr       *skill.Manager   // Skill manager for matching
	memoryNotesDir string           // Directory for daily notes
	tokenBudget    int              // Maximum tokens for dynamic content
	logger         *zap.Logger
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
	ltmStore *memory.LTMStore,
	skillMgr *skill.Manager,
	memoryNotesDir string,
	tokenBudget int,
	logger *zap.Logger,
) *ContextBuilder {
	if tokenBudget <= 0 {
		tokenBudget = 2000 // Default token budget for dynamic content
	}

	return &ContextBuilder{
		persona:        persona,
		memoryMgr:      memoryMgr,
		ltmStore:       ltmStore,
		skillMgr:       skillMgr,
		memoryNotesDir: memoryNotesDir,
		tokenBudget:    tokenBudget,
		logger:         logger.Named("context_builder"),
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

	// 2. Memory Section (Core + Daily Notes + Relevant)
	memoriesSection, memoriesUsed := b.buildMemorySection(ctx, sessionID, userInput)
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

// buildMemorySection builds the complete memory section including Core, Daily Notes, and Relevant memories.
func (b *ContextBuilder) buildMemorySection(ctx context.Context, sessionID, query string) (string, int) {
	var sections []string
	totalUsed := 0
	totalTokens := 0
	maxTokens := b.tokenBudget

	// 1. Core Memories (from LTMStore)
	if b.ltmStore != nil {
		cores, err := b.ltmStore.List(memory.CategoryCore, 5)
		if err == nil && len(cores) > 0 {
			var sb strings.Builder
			sb.WriteString("### 核心记忆\n\n")
			for _, e := range cores {
				content := fmt.Sprintf("- **%s**: %s\n", e.Key, e.Content)
				estimatedTokens := len(content) / 4
				if totalTokens+estimatedTokens > maxTokens {
					break
				}
				sb.WriteString(content)
				totalTokens += estimatedTokens
				totalUsed++
			}
			if totalUsed > 0 {
				sections = append(sections, sb.String())
			}
		}
	}

	// 2. Daily Notes (last 3 days)
	if b.memoryNotesDir != "" {
		notes := b.buildDailyNotes()
		if notes != "" {
			estimatedTokens := len(notes) / 4
			if totalTokens+estimatedTokens <= maxTokens {
				sections = append(sections, notes)
				totalTokens += estimatedTokens
			}
		}
	}

	// 3. Relevant Memories (based on user input)
	if b.memoryMgr != nil && totalTokens < maxTokens {
		snippets, err := b.memoryMgr.Search(ctx, sessionID, query, 5, 0.5)
		if err == nil && len(snippets) > 0 {
			var sb strings.Builder
			sb.WriteString("### 相关记忆\n\n")
			used := 0
			for _, snippet := range snippets {
				content := fmt.Sprintf("- %s\n", snippet.Content)
				estimatedTokens := len(content) / 4
				if totalTokens+estimatedTokens > maxTokens {
					break
				}
				sb.WriteString(content)
				totalTokens += estimatedTokens
				used++
				totalUsed++
			}
			if used > 0 {
				sections = append(sections, sb.String())
			}
		}
	}

	if len(sections) == 0 {
		return "", 0
	}

	var result strings.Builder
	result.WriteString("## 记忆\n\n")
	result.WriteString(strings.Join(sections, "\n"))

	return result.String(), totalUsed
}

// buildDailyNotes reads the last 3 days of daily notes.
func (b *ContextBuilder) buildDailyNotes() string {
	var sb strings.Builder
	now := time.Now()

	for i := 0; i < 3; i++ {
		date := now.AddDate(0, 0, -i)
		monthDir := b.memoryNotesDir + "/" + date.Format("200601")
		dayFile := monthDir + "/" + date.Format("20060102") + ".md"

		data, err := os.ReadFile(dayFile)
		if err != nil {
			continue
		}

		if sb.Len() == 0 {
			sb.WriteString("### 最近日记\n\n")
		}
		sb.WriteString(fmt.Sprintf("**%s**:\n", date.Format("2006-01-02")))
		sb.Write(data)
		sb.WriteString("\n\n")
	}

	return sb.String()
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
