// Package tool manages the tool lifecycle and execution for the agent.
package tool

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/gopaw/gopaw/pkg/plugin"
	"go.uber.org/zap"
)

// Registry maintains a thread-safe set of available tools.
type Registry struct {
	mu     sync.RWMutex
	tools  map[string]plugin.Tool
	logger *zap.Logger
	store  plugin.MediaStore

	// Filter settings
	allowedTools map[string]struct{}
	deniedTools  map[string]struct{}
}

// NewRegistry creates a new tool registry.
func NewRegistry(logger *zap.Logger) *Registry {
	return &Registry{
		tools:        make(map[string]plugin.Tool),
		logger:       logger.Named("tool_registry"),
		allowedTools: make(map[string]struct{}),
		deniedTools:  make(map[string]struct{}),
	}
}

// globalRegistry is the process-wide tool registry.
var globalRegistry = NewRegistry(zap.L())

// Register adds a tool to the global registry.
func Register(t plugin.Tool) {
	globalRegistry.Register(t)
}

// Global returns the process-wide tool registry.
func Global() *Registry {
	return globalRegistry
}

// SetMediaStore configures the media store and injects it into all registered tools.
func (r *Registry) SetMediaStore(s plugin.MediaStore) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.store = s
	for _, t := range r.tools {
		if msr, ok := t.(plugin.MediaStoreReceiver); ok {
			msr.SetMediaStore(s)
		}
	}
}

// SetFilter configures which tools are allowed or denied.
func (r *Registry) SetFilter(allowed, denied []string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.allowedTools = make(map[string]struct{})
	for _, name := range allowed {
		r.allowedTools[name] = struct{}{}
	}

	r.deniedTools = make(map[string]struct{})
	for _, name := range denied {
		r.deniedTools[name] = struct{}{}
	}
}

// isAllowed checks if a tool is permitted to run.
func (r *Registry) isAllowed(name string) bool {
	// Denylist has highest priority
	if _, denied := r.deniedTools[name]; denied {
		return false
	}
	// If allowlist is not empty, only allowed tools can run
	if len(r.allowedTools) > 0 {
		_, allowed := r.allowedTools[name]
		return allowed
	}
	return true
}

// Register adds a tool to the registry.
func (r *Registry) Register(t plugin.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := t.Name()
	if _, exists := r.tools[name]; exists {
		r.logger.Warn("tool already registered, overwriting", zap.String("tool", name))
	}

	// Inject MediaStore if already available
	if r.store != nil {
		if msr, ok := t.(plugin.MediaStoreReceiver); ok {
			msr.SetMediaStore(r.store)
		}
	}

	r.tools[name] = t
}

// Get returns the tool with the given name.
func (r *Registry) Get(name string) (plugin.Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// Names returns a sorted list of all registered tool names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		if r.isAllowed(name) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// All returns all registered tools that pass the filter.
func (r *Registry) All() []plugin.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]plugin.Tool, 0, len(r.tools))
	for name, t := range r.tools {
		if r.isAllowed(name) {
			tools = append(tools, t)
		}
	}
	return tools
}

// MCPServerConfig defines how to connect to an MCP server.
type MCPServerConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// LoadMCPServers connects to multiple MCP servers and registers their tools.
func (r *Registry) LoadMCPServers(ctx context.Context, configs []MCPServerConfig) error {
	for _, cfg := range configs {
		client := NewMCPClient(cfg.Name, cfg.Command, cfg.Args)
		if err := client.Start(ctx); err != nil {
			r.logger.Error("failed to start MCP server", zap.String("server", cfg.Name), zap.Error(err))
			continue
		}

		mcpTools := client.GetTools()
		for _, info := range mcpTools {
			wrapper := NewMCPToolWrapper(client, info, cfg.Name)
			r.Register(wrapper)
		}
		r.logger.Info("registered tools from MCP server", zap.String("server", cfg.Name), zap.Int("count", len(mcpTools)))
	}
	return nil
}

// Execute calls a tool by name with the provided context and arguments.
func (r *Registry) Execute(ctx context.Context, name string, args map[string]interface{}, channel, session, user string) *plugin.ToolResult {
	r.mu.RLock()
	allowed := r.isAllowed(name)
	r.mu.RUnlock()

	if !allowed {
		return plugin.ErrorResult(fmt.Sprintf("tool %q is restricted by security policy", name))
	}

	t, ok := r.Get(name)
	if !ok {
		return plugin.ErrorResult(fmt.Sprintf("tool %q not found", name))
	}

	// Inject context if the tool requires it.
	if ct, ok := t.(plugin.ContextualTool); ok {
		ct.SetContext(channel, session, user)
	}

	return t.Execute(ctx, args)
}
