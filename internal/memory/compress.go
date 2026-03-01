// Package memory implements the conversation history and long-term memory storage layer.
package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/gopaw/gopaw/internal/llm"
)

// Compressor reduces old messages to a concise summary using the LLM.
type Compressor struct {
	llmClient llm.Client
}

// NewCompressor creates a Compressor backed by the given LLM client.
func NewCompressor(client llm.Client) *Compressor {
	return &Compressor{llmClient: client}
}

// Summarise sends the provided conversation turns to the LLM and returns a summary.
// messages should be the oldest half of the session context that needs to be compressed.
func (c *Compressor) Summarise(ctx context.Context, messages []StoredMessage) (string, error) {
	if len(messages) == 0 {
		return "", nil
	}

	// Build a human-readable transcript.
	var sb strings.Builder
	for _, m := range messages {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", m.Role, m.Content))
	}

	req := llm.ChatRequest{
		Messages: []llm.ChatMessage{
			{
				Role: llm.RoleSystem,
				Content: "你是一个对话摘要助手。请将以下对话历史压缩为不超过 150 字的摘要，" +
					"保留关键信息、用户偏好和重要决策，去除冗余细节。只输出摘要文本，不要解释。",
			},
			{
				Role:    llm.RoleUser,
				Content: sb.String(),
			},
		},
		MaxTokens: 300,
	}

	resp, err := c.llmClient.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("compressor: llm call failed: %w", err)
	}
	return strings.TrimSpace(resp.Message.Content), nil
}
