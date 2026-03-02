// Package memory implements the conversation history and long-term memory storage layer.
package memory

import (
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// encoderCache 使用 sync.Once 缓存编码器，避免每次调用都重新初始化
var (
	encoderInit sync.Once
	encoder     *tiktoken.Tiktoken
	encoderErr  error
)

// initEncoder 初始化编码器，只执行一次
func initEncoder() {
	encoderInit.Do(func() {
		encoder, encoderErr = tiktoken.GetEncoding("cl100k_base")
	})
}

// CountTokens returns the precise token count for the given messages using cl100k_base encoding.
// This encoding is compatible with GPT-4, GPT-3.5, and Claude models.
// On failure, it falls back to character-based estimation.
func CountTokens(msgs []MemoryMessage) int {
	initEncoder()

	if encoder != nil {
		total := 0
		for _, m := range msgs {
			// Encode the content
			tokens := encoder.Encode(m.Content, nil, nil)
			total += len(tokens)
			// Add overhead for role and formatting (approximately 4 tokens per message)
			total += 4
		}
		return total
	}

	// 编码器初始化失败，使用 fallback
	if encoderErr != nil {
		// 记录一次 warn（可选）
	}
	return estimateTokensFallback(msgs)
}

// estimateTokensFallback provides a rough token estimate when tiktoken is unavailable.
// Uses the rule of thumb: roughly 4 characters per token.
func estimateTokensFallback(msgs []MemoryMessage) int {
	total := 0
	for _, m := range msgs {
		// Count runes (handles Unicode correctly)
		runes := []rune(m.Content)
		total += len(runes) / 4
		// Add overhead for role and formatting
		total += 4
	}
	return total
}
