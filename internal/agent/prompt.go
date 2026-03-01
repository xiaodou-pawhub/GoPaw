// Package agent implements the ReAct agent engine.
package agent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/pkg/plugin"
)

// buildSystemPrompt assembles the full system prompt for an agent invocation.
// It concatenates the base prompt, active skill fragments, and tool descriptions.
func buildSystemPrompt(basePrompt, skillFragments string, tools []plugin.Tool) string {
	var sb strings.Builder
	sb.WriteString(basePrompt)
	if skillFragments != "" {
		sb.WriteString("\n\n---\n")
		sb.WriteString(skillFragments)
	}
	if len(tools) > 0 {
		sb.WriteString("\n\n---\n## Available Tools\n\n")
		sb.WriteString(formatToolsDescription(tools))
	}
	sb.WriteString("\n\n---\n")
	sb.WriteString(reactInstructions)
	return sb.String()
}

// reactInstructions is appended to every system prompt to teach the LLM the ReAct format.
const reactInstructions = `## Response Format

Use the following format for your responses:

Thought: think about what to do
Action: tool_name
Action Input: {"arg1": "value1", "arg2": "value2"}
Observation: (tool result will be inserted here by the system)
... (repeat Thought/Action/Observation as needed)
Thought: I now know the final answer
Final Answer: your final answer to the user`

// formatToolsDescription converts tool definitions to a human-readable prompt section.
func formatToolsDescription(tools []plugin.Tool) string {
	var sb strings.Builder
	for i, t := range tools {
		if i > 0 {
			sb.WriteString("\n")
		}
		paramsJSON, _ := json.MarshalIndent(t.Parameters(), "", "  ")
		sb.WriteString(fmt.Sprintf("### %s\n%s\n\nParameters:\n```json\n%s\n```\n",
			t.Name(), t.Description(), string(paramsJSON)))
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

// appendReActStep appends an assistant reply and the tool observation to the message list.
func appendReActStep(msgs []llm.ChatMessage, assistantReply, observation string) []llm.ChatMessage {
	msgs = append(msgs,
		llm.ChatMessage{Role: llm.RoleAssistant, Content: assistantReply},
		llm.ChatMessage{Role: llm.RoleUser, Content: "Observation: " + observation},
	)
	return msgs
}
