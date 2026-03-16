package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// setupTestSkillDir creates a temporary skill directory for testing.
func setupTestSkillDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "skill-test-*")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// createSkillFile creates a skill file in the test directory.
func createSkillFile(t *testing.T, dir, path, content string) {
	fullPath := filepath.Join(dir, path)
	err := os.MkdirAll(filepath.Dir(fullPath), 0755)
	require.NoError(t, err)
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(t, err)
}

// setupTestManager creates a test manager with all dependencies.
func setupTestManager(t *testing.T, skillDir string) *Manager {
	logger := zap.NewNop()
	toolReg := tool.NewRegistry(logger)
	manager := NewManager(skillDir, toolReg, logger)
	return manager
}

// TestNewManager tests creating a new manager.
func TestNewManager(t *testing.T) {
	dir := setupTestSkillDir(t)
	manager := setupTestManager(t, dir)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.registry)
	assert.NotNil(t, manager.loader)
}

// TestManager_Load tests loading skills from directory.
func TestManager_Load(t *testing.T) {
	dir := setupTestSkillDir(t)

	// Create a valid skill
	createSkillFile(t, dir, "test_skill/manifest.yaml", `
name: test_skill
version: 1.0.0
display_name: Test Skill
description: A test skill
level: 1
author: test
`)
	createSkillFile(t, dir, "test_skill/prompt.md", `
# Test Skill

You are a test skill.
`)

	manager := setupTestManager(t, dir)
	err := manager.Load(nil)

	require.NoError(t, err)

	// Verify skill was loaded
	registry := manager.Registry()
	skill, err := registry.Get("test_skill")
	require.NoError(t, err)
	assert.Equal(t, "test_skill", skill.Manifest.Name)
	assert.Equal(t, "Test Skill", skill.Manifest.DisplayName)
	assert.Equal(t, 1, int(skill.Manifest.Level))
}

// TestManager_Load_InvalidSkill tests loading with invalid skill.
func TestManager_Load_InvalidSkill(t *testing.T) {
	dir := setupTestSkillDir(t)

	// Create an invalid skill (no manifest)
	_ = os.MkdirAll(filepath.Join(dir, "invalid_skill"), 0755)

	manager := setupTestManager(t, dir)
	err := manager.Load(nil)

	// Should not error, just skip invalid skills
	require.NoError(t, err)

	// Verify no skills were loaded
	registry := manager.Registry()
	_, err = registry.Get("invalid_skill")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestManager_Load_WithEnabledList tests loading with enabled list filter.
func TestManager_Load_WithEnabledList(t *testing.T) {
	dir := setupTestSkillDir(t)

	// Create multiple skills
	for i := 1; i <= 3; i++ {
		name := fmt.Sprintf("skill_%d", i)
		createSkillFile(t, dir, name+"/manifest.yaml", fmt.Sprintf(`
name: %s
version: 1.0.0
display_name: Skill %d
level: 1
`, name, i))
		createSkillFile(t, dir, name+"/prompt.md", fmt.Sprintf("# Skill %d", i))
	}

	manager := setupTestManager(t, dir)
	// Only enable skill_1 and skill_3
	err := manager.Load([]string{"skill_1", "skill_3"})
	require.NoError(t, err)

	registry := manager.Registry()

	skill1, _ := registry.Get("skill_1")
	assert.True(t, skill1.Enabled)

	skill2, _ := registry.Get("skill_2")
	assert.False(t, skill2.Enabled)

	skill3, _ := registry.Get("skill_3")
	assert.True(t, skill3.Enabled)
}

// TestManager_FragmentsForInput tests fragment matching.
func TestManager_FragmentsForInput(t *testing.T) {
	dir := setupTestSkillDir(t)

	// Create skill with keywords
	createSkillFile(t, dir, "translator/manifest.yaml", `
name: translator
version: 1.0.0
display_name: Translator
level: 1
activation:
  keywords:
    - translate
    - translation
    - language
`)
	createSkillFile(t, dir, "translator/prompt.md", "# Translator")

	manager := setupTestManager(t, dir)
	_ = manager.Load(nil)

	// Should match keywords
	fragments := manager.FragmentsForInput("Please translate this text")
	assert.NotEmpty(t, fragments)
	assert.Contains(t, fragments, "Translator")

	// Should not match
	fragments = manager.FragmentsForInput("Hello world")
	assert.Empty(t, fragments)
}

// TestManager_Reload tests reloading skills.
func TestManager_Reload(t *testing.T) {
	dir := setupTestSkillDir(t)

	createSkillFile(t, dir, "test_skill/manifest.yaml", `
name: test_skill
version: 1.0.0
level: 1
`)
	createSkillFile(t, dir, "test_skill/prompt.md", "# Original")

	manager := setupTestManager(t, dir)
	_ = manager.Load(nil)

	// Modify skill
	createSkillFile(t, dir, "test_skill/prompt.md", "# Updated")

	// Reload
	err := manager.Reload()
	require.NoError(t, err)

	// Verify skill was reloaded
	registry := manager.Registry()
	skill, err := registry.Get("test_skill")
	require.NoError(t, err)
	assert.NotNil(t, skill)
}

// TestManager_Level2Skill tests loading Level 2 skill.
func TestManager_Level2Skill(t *testing.T) {
	dir := setupTestSkillDir(t)

	createSkillFile(t, dir, "workflow_skill/manifest.yaml", `
name: workflow_skill
version: 1.0.0
display_name: Workflow Skill
level: 2
`)
	createSkillFile(t, dir, "workflow_skill/workflow.yaml", `
steps:
  - name: step1
    tool: tool1
    input:
      param: "value"
`)

	manager := setupTestManager(t, dir)
	err := manager.Load(nil)

	require.NoError(t, err)

	registry := manager.Registry()
	skill, err := registry.Get("workflow_skill")
	require.NoError(t, err)
	assert.Equal(t, 2, int(skill.Manifest.Level))
}

// TestManager_Level3Skill tests loading Level 3 skill.
func TestManager_Level3Skill(t *testing.T) {
	dir := setupTestSkillDir(t)

	createSkillFile(t, dir, "code_skill/manifest.yaml", `
name: code_skill
version: 1.0.0
display_name: Code Skill
level: 3
`)
	createSkillFile(t, dir, "code_skill/skill.go", `
package main

import "context"

type Skill struct{}

func (s *Skill) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return "Code skill executed", nil
}
`)

	manager := setupTestManager(t, dir)
	err := manager.Load(nil)

	require.NoError(t, err)

	registry := manager.Registry()
	skill, err := registry.Get("code_skill")
	require.NoError(t, err)
	assert.Equal(t, 3, int(skill.Manifest.Level))
}

// TestManager_MultipleSkills tests loading multiple skills.
func TestManager_MultipleSkills(t *testing.T) {
	dir := setupTestSkillDir(t)

	// Create multiple skills
	for i := 1; i <= 5; i++ {
		name := fmt.Sprintf("skill_%d", i)
		createSkillFile(t, dir, name+"/manifest.yaml", fmt.Sprintf(`
name: %s
version: 1.0.0
display_name: Skill %d
level: 1
`, name, i))
		createSkillFile(t, dir, name+"/prompt.md", fmt.Sprintf("# Skill %d", i))
	}

	manager := setupTestManager(t, dir)
	err := manager.Load(nil)
	require.NoError(t, err)

	registry := manager.Registry()
	allSkills := registry.All()
	assert.Len(t, allSkills, 5)
}

// TestManager_Registry tests accessing the registry.
func TestManager_Registry(t *testing.T) {
	dir := setupTestSkillDir(t)

	createSkillFile(t, dir, "test_skill/manifest.yaml", `
name: test_skill
version: 1.0.0
level: 1
`)
	createSkillFile(t, dir, "test_skill/prompt.md", "# Test")

	manager := setupTestManager(t, dir)
	_ = manager.Load(nil)

	registry := manager.Registry()
	assert.NotNil(t, registry)

	// Test registry operations
	skill, err := registry.Get("test_skill")
	require.NoError(t, err)
	assert.Equal(t, "test_skill", skill.Manifest.Name)
}

// BenchmarkManager_Load benchmarks skill loading.
func BenchmarkManager_Load(b *testing.B) {
	dir, _ := os.MkdirTemp("", "skill-bench-*")
	defer os.RemoveAll(dir)
	logger := zap.NewNop()

	// Create test skills
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("skill_%d", i)
		_ = os.MkdirAll(filepath.Join(dir, name), 0755)
		_ = os.WriteFile(filepath.Join(dir, name, "manifest.yaml"), []byte(fmt.Sprintf(`
name: %s
version: 1.0.0
level: 1
`, name)), 0644)
		_ = os.WriteFile(filepath.Join(dir, name, "prompt.md"), []byte("# "+name), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		toolReg := tool.NewRegistry(logger)
		manager := NewManager(dir, toolReg, logger)
		_ = manager.Load(nil)
	}
}

// BenchmarkManager_FragmentsForInput benchmarks fragment matching.
func BenchmarkManager_FragmentsForInput(b *testing.B) {
	dir, _ := os.MkdirTemp("", "skill-bench-*")
	defer os.RemoveAll(dir)
	logger := zap.NewNop()

	// Create skills with keywords
	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("skill_%d", i)
		_ = os.MkdirAll(filepath.Join(dir, name), 0755)
		_ = os.WriteFile(filepath.Join(dir, name, "manifest.yaml"), []byte(fmt.Sprintf(`
name: %s
version: 1.0.0
level: 1
keywords:
  - keyword_%d
`, name, i)), 0644)
		_ = os.WriteFile(filepath.Join(dir, name, "prompt.md"), []byte("# "+name), 0644)
	}

	toolReg := tool.NewRegistry(logger)
	manager := NewManager(dir, toolReg, logger)
	_ = manager.Load(nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.FragmentsForInput("test keyword_5 input")
	}
}

// TestManager_GetSkillByTool tests the tool to skill mapping.
func TestManager_GetSkillByTool(t *testing.T) {
	dir := setupTestSkillDir(t)

	// Create a Level-3 skill with tools
	skillContent := `name: test_skill
version: "1.0"
display_name: Test Skill
description: A test skill with tools
level: 3
activation:
  always: false
  keywords:
    - test
`
	createSkillFile(t, dir, "test_skill/manifest.yaml", skillContent)

	// Create a simple Go skill file
	codeContent := `package main
// This is a test skill
`
	createSkillFile(t, dir, "test_skill/skill.go", codeContent)

	logger := zap.NewNop()
	toolReg := tool.NewRegistry(logger)
	manager := NewManager(dir, toolReg, logger)

	// Before loading, mapping should be empty
	skillName := manager.GetSkillByTool("nonexistent_tool")
	assert.Empty(t, skillName)

	// Load skills
	err := manager.Load(nil)
	require.NoError(t, err)

	// After loading, check that toolToSkill mapping is initialized
	assert.NotNil(t, manager.toolToSkill)
}

// TestManager_GetSkillUsageStats tests getting usage statistics.
func TestManager_GetSkillUsageStats(t *testing.T) {
	dir := setupTestSkillDir(t)
	manager := setupTestManager(t, dir)

	// Initially stats should be empty
	stats := manager.GetSkillUsageStats()
	assert.Empty(t, stats)

	// Record some usage
	manager.RecordSkillUsage("skill1")
	manager.RecordSkillUsage("skill1")
	manager.RecordSkillUsage("skill2")

	// Check stats
	stats = manager.GetSkillUsageStats()
	assert.Equal(t, 2, stats["skill1"])
	assert.Equal(t, 1, stats["skill2"])
}
