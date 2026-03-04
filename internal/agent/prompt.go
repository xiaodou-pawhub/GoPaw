// Package agent implements the native Function Calling agent engine.
package agent

import (
	"strings"

	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
)

// buildSystemPrompt assembles the full system prompt for an agent invocation.
// Tools are passed via the API Tools field and are not embedded in the system prompt.
func buildSystemPrompt(basePrompt, memoryContent, skillFragments string) string {
	var sb strings.Builder
	sb.WriteString(basePrompt)
	if memoryContent != "" {
		sb.WriteString("\n\n---\n## Long-term Memory\n\n")
		sb.WriteString(memoryContent)
	}
	if skillFragments != "" {
		sb.WriteString("\n\n---\n")
		sb.WriteString(skillFragments)
	}
	return sb.String()
}

// buildMessages constructs the LLM messages array from system prompt, history and the current query.
func buildMessages(systemPrompt string, history []memory.MemoryMessage, userContent string) []llm.ChatMessage {
	msgs := []llm.ChatMessage{
		{Role: llm.RoleSystem, Content: systemPrompt},
	}
	for _, h := range history {
		msgs = append(msgs, llm.ChatMessage{
			Role:    llm.Role(h.Role),
			Content: h.Content,
		})
	}
	msgs = append(msgs, llm.ChatMessage{
		Role:    llm.RoleUser,
		Content: userContent,
	})
	return msgs
}
