// Package tool manages the registration and lookup of Tool plugins.
package tool

import (
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/pkg/plugin"
)

// Registry is a thread-safe store of named Tool implementations.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]plugin.Tool
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]plugin.Tool)}
}

// globalRegistry is the process-wide tool registry used by init() registrations.
var globalRegistry = NewRegistry()

// Register adds a tool to the global registry.
// It is typically called from tool package init() functions.
func Register(t plugin.Tool) {
	globalRegistry.Register(t)
}

// Global returns the process-wide tool registry.
func Global() *Registry {
	return globalRegistry
}

// Register adds a tool to this registry.
func (r *Registry) Register(t plugin.Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Unregister removes a tool by name.
func (r *Registry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// Get returns the tool with the given name, or an error if not found.
func (r *Registry) Get(name string) (plugin.Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool: %q not registered", name)
	}
	return t, nil
}

// All returns a snapshot of all registered tools.
func (r *Registry) All() []plugin.Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]plugin.Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Names returns the names of all registered tools.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}
