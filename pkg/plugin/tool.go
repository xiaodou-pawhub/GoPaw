// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

import "context"

// ToolProperty describes a single parameter property in JSON Schema style.
type ToolProperty struct {
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Enum        []string `json:"enum,omitempty"`
}

// ToolParameters is the JSON Schema object describing a tool's accepted parameters.
// It follows the OpenAI function-calling parameter format.
type ToolParameters struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required"`
}

// Tool is the interface that every callable tool must implement.
// Tools are the atomic operations that the ReAct agent can invoke.
type Tool interface {
	// Name returns the unique snake_case identifier used in LLM prompts.
	Name() string

	// Description explains what the tool does and when the LLM should call it.
	Description() string

	// Parameters returns the JSON Schema for the tool's input arguments.
	Parameters() ToolParameters

	// Execute runs the tool with the given arguments and returns a string result.
	// args values are typed according to the JSON Schema (string, float64, bool …).
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}
