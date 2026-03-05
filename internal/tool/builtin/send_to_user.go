package builtin

import (
	"context"
	"fmt"
	"strings"

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
	return "Send a file, image, or text message immediately to the user. " +
		"Use this to share generated assets or important notifications during a task."
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

func (t *SendToUserTool) SetContext(channel, session, user string) {
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
