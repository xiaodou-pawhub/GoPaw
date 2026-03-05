package tool

import (
	"context"

	"github.com/gopaw/gopaw/pkg/plugin"
)

// MCPToolWrapper wraps a single tool from an MCP server into the plugin.Tool interface.
type MCPToolWrapper struct {
	client   *MCPClient
	toolInfo MCPToolInfo
	prefix   string
}

// NewMCPToolWrapper creates a new wrapper. The prefix helps avoid name collisions.
func NewMCPToolWrapper(client *MCPClient, info MCPToolInfo, prefix string) *MCPToolWrapper {
	return &MCPToolWrapper{
		client:   client,
		toolInfo: info,
		prefix:   prefix,
	}
}

func (w *MCPToolWrapper) Name() string {
	if w.prefix != "" {
		return w.prefix + "__" + w.toolInfo.Name
	}
	return w.toolInfo.Name
}

func (w *MCPToolWrapper) Description() string {
	return w.toolInfo.Description
}

func (w *MCPToolWrapper) Parameters() plugin.ToolParameters {
	// MCP inputSchema is already a JSON Schema object.
	// We convert it to plugin.ToolParameters.
	params := plugin.ToolParameters{
		Type:       "object",
		Properties: make(map[string]plugin.ToolProperty),
	}

	if schema, ok := w.toolInfo.InputSchema["properties"].(map[string]interface{}); ok {
		for k, v := range schema {
			if prop, ok := v.(map[string]interface{}); ok {
				p := plugin.ToolProperty{
					Type:        prop["type"].(string),
					Description: prop["description"].(string),
				}
				if enum, ok := prop["enum"].([]interface{}); ok {
					for _, e := range enum {
						p.Enum = append(p.Enum, e.(string))
					}
				}
				params.Properties[k] = p
			}
		}
	}

	if req, ok := w.toolInfo.InputSchema["required"].([]interface{}); ok {
		for _, r := range req {
			params.Required = append(params.Required, r.(string))
		}
	}

	return params
}

func (w *MCPToolWrapper) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	// Strip prefix before sending to MCP server
	return w.client.ExecuteTool(ctx, w.toolInfo.Name, args)
}
