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
	hasImage := false
	for _, t := range tools {
		name := t.Name()
		if name == "image_info" || name == "image_process" || name == "send_to_user" {
			hasImage = true
			break
		}
	}
	if !hasImage {
		return ""
	}
	return `## Multimodal Capabilities

You can process images and files sent by users. When a message contains media resources,
a **[System: Media Resources]** block will appear at the end of the user's message listing
the available media:// references.

**Recommended image-processing workflow:**
1. **image_info** — inspect the image's dimensions, format, and size first
2. **image_process** — resize / crop / rotate / grayscale as needed (returns a new media:// ref)
3. **send_to_user** — deliver each result immediately; do not wait until all work is done

If the user simply asks you to "look at" or "describe" an image, pass the media:// reference
directly as context; you do not need to call image_info unless specs are specifically needed.`
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
