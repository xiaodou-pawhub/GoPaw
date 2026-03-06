// Package agent implements the native Function Calling agent engine.
package agent

import (
	"fmt"
	"strings"

	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
)

// buildSystemPrompt assembles the full system prompt for an agent invocation.
func buildSystemPrompt(basePrompt, memoryContent, skillFragments, capabilityFragment string) string {
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
	if capabilityFragment != "" {
		sb.WriteString("\n\n---\n")
		sb.WriteString(capabilityFragment)
	}
	return sb.String()
}

// buildCapabilityFragment generates a dynamic capability guide based on which tools are currently registered.
func buildCapabilityFragment(tools []plugin.Tool) string {
	var sb strings.Builder
	sb.WriteString("\n## Active Tool Use Strategy\n\n")
	sb.WriteString("You are connected to the real world through tools. Follow these protocols:\n")
	sb.WriteString("1. **Information Gap**: If a user asks for data after your knowledge cutoff (e.g., current prices, news, weather), DO NOT apologize. Immediately use `web_search` or `http_client`.\n")
	sb.WriteString("2. **API Interaction**: If a user mentions a service with a known API, use `http_client` to interact with it directly.\n")
	sb.WriteString("3. **Multi-step Reasoning**: You can chain tools. For example, use `web_search` to find a public API endpoint, then use `http_client` to fetch the data.\n")
	sb.WriteString("4. **Proactive Updates**: If you suspect information might be stale, verify it using tools without being asked.\n")
	sb.WriteString("5. **Browser/HTTP Synergy**: If a service requires login, use `browser_control` to authenticate, then use `get_cookies` action to extract the session. You can then pass these cookies to `http_client` headers for faster, structured data access.\n")
	sb.WriteString("6. **Visual Debugging**: If browser actions fail (e.g., selector not found), inspect the returned screenshot to understand the page state or identify overlays.\n")

	// Multi-modal check
	hasImage := false
	for _, t := range tools {
		name := t.Name()
		if name == "image_info" || name == "image_process" || name == "send_to_user" {
			hasImage = true
			break
		}
	}
	
	if hasImage {
		sb.WriteString("\n## Multimodal Capabilities\n\n")
		sb.WriteString("You can process images and files. When media resources are present, a **[System: Media Resources]** block appears.\n")
		sb.WriteString("**Workflow:**\n")
		sb.WriteString("1. **image_info** — inspect specs first\n")
		sb.WriteString("2. **image_process** — edit as needed\n")
		sb.WriteString("3. **send_to_user** — deliver result immediately\n")
	}

	return sb.String()
}

// buildMediaManifest produces a structured resource-manifest suffix appended to the user message.
func buildMediaManifest(files []types.FileAttachment) string {
	if len(files) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("\n\n---\n")
	if len(files) == 1 {
		sb.WriteString("**[System: 1 active media resource attached]**\n")
	} else {
		fmt.Fprintf(&sb, "**[System: %d active media resources attached]**\n", len(files))
	}
	for _, f := range files {
		ref := f.URL // media://uuid
		label := f.Name
		if label == "" {
			label = "unnamed"
		}
		mime := f.MIMEType
		if mime == "" {
			mime = "application/octet-stream"
		}
		if f.Size > 0 {
			fmt.Fprintf(&sb, "• %s — %s (%s, %d bytes)\n", ref, label, mime, f.Size)
		} else {
			fmt.Fprintf(&sb, "• %s — %s (%s)\n", ref, label, mime)
		}
	}
	sb.WriteString("\nIMPORTANT: Only use the `media://` references listed above for the current task. References from older messages may be expired.\n")
	sb.WriteString("Use `image_info` to inspect, `image_process` to transform, `send_to_user` to deliver.")
	return sb.String()
}

// buildMessages constructs the LLM messages array.
func buildMessages(systemPrompt string, history []memory.MemoryMessage, userContent string, files []types.FileAttachment) []llm.ChatMessage {
	msgs := []llm.ChatMessage{
		{Role: llm.RoleSystem, Content: systemPrompt},
	}
	for _, h := range history {
		msgs = append(msgs, llm.ChatMessage{
			Role:    llm.Role(h.Role),
			Content: h.Content,
		})
	}

	content := userContent
	if manifest := buildMediaManifest(files); manifest != "" {
		content += manifest
	}

	msgs = append(msgs, llm.ChatMessage{
		Role:    llm.RoleUser,
		Content: content,
	})
	return msgs
}
