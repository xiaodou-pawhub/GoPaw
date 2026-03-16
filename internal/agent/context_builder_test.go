// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestNewContextBuilder(t *testing.T) {
	logger := zap.NewNop()
	builder := NewContextBuilder("test persona", nil, nil, nil, "", 1000, logger)

	assert.NotNil(t, builder)
	assert.Equal(t, "test persona", builder.persona)
	assert.Equal(t, 1000, builder.tokenBudget)
}

func TestNewContextBuilder_DefaultTokenBudget(t *testing.T) {
	logger := zap.NewNop()
	builder := NewContextBuilder("test persona", nil, nil, nil, "", 0, logger)

	assert.NotNil(t, builder)
	assert.Equal(t, 2000, builder.tokenBudget) // Default value
}

func TestContextBuilder_Build_NoManagers(t *testing.T) {
	logger := zap.NewNop()
	builder := NewContextBuilder("Test Persona", nil, nil, nil, "", 2000, logger)

	ctx := testingContext()
	result, err := builder.Build(ctx, "session1", "hello")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result.SystemPrompt, "Test Persona")
	assert.NotContains(t, result.SystemPrompt, "记忆")
	assert.NotContains(t, result.SystemPrompt, "相关技能")
	assert.Contains(t, result.SystemPrompt, "当前时间")
	assert.Equal(t, 0, result.MemoriesUsed)
	assert.Equal(t, 0, result.SkillsMatched)
}

func TestContextBuilder_buildTimeContext(t *testing.T) {
	logger := zap.NewNop()
	builder := NewContextBuilder("test", nil, nil, nil, "", 2000, logger)

	timeContext := builder.buildTimeContext()

	assert.NotEmpty(t, timeContext)
	assert.Contains(t, timeContext, "当前时间")
	assert.Contains(t, timeContext, "星期")
}

func TestMin(t *testing.T) {
	assert.Equal(t, 1, min(1, 2))
	assert.Equal(t, 1, min(2, 1))
	assert.Equal(t, 0, min(0, 0))
	assert.Equal(t, -1, min(-1, 1))
}

// Helper function to create a test context
func testingContext() context.Context {
	return context.Background()
}
