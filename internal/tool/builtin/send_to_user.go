package builtin

import (
	"context"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&SendToUserTool{})
}

type SendToUserTool struct {
	channel string
	session string
	user    string
}

func (t *SendToUserTool) Name() string { return "send_to_user" }

func (t *SendToUserTool) Description() string {
	return "Immediately deliver a file (image, document, etc.) or text message to the user. " +
		"WHEN TO USE: call this as soon as you have a result ready — do NOT accumulate multiple " +
		"outputs and send them all at the end. Each completed image or file should trigger its own " +
		"send_to_user call so the user sees progress in real time. " +
		"Pass a media:// reference in 'path' to send an image or file; use 'text' for a plain message."
}

func (t *SendToUserTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"text": {
				Type:        "string",
				Description: "Optional text message to send.",
			},
			"path": {
				Type:        "string",
				Description: "Optional media reference (media://uuid) or local path to a file/image.",
			},
		},
		Required: []string{},
	}
}

func (t *SendToUserTool) SetContext(channel, chatID, session, user string) {
	t.channel = channel
	t.session = session
	t.user = user
}

func (t *SendToUserTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	text, _ := args["text"].(string)
	path, _ := args["path"].(string)

	if text == "" && path == "" {
		return plugin.ErrorResult("either 'text' or 'path' must be provided")
	}

	result := &plugin.ToolResult{
		LLMOutput:  "Message sent to user successfully.",
		UserOutput: text,
	}

	if path != "" {
		result.Media = []string{path}
	}

	// The Executor is responsible for seeing this result and 
	// performing the actual push to the channel.
	return result
}
