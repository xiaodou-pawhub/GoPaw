// Package channel manages the lifecycle of channel plugins and routes messages to the agent.
package channel

import (
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/pkg/plugin"
)

// Registry is a thread-safe store of named ChannelPlugin implementations.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]plugin.ChannelPlugin
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{plugins: make(map[string]plugin.ChannelPlugin)}
}

// globalRegistry is the process-wide channel plugin registry.
var globalRegistry = NewRegistry()

// Register adds a plugin to the global channel registry.
// It is typically called from channel plugin package init() functions.
func Register(p plugin.ChannelPlugin) {
	globalRegistry.Register(p)
}

// Global returns the process-wide channel registry.
func Global() *Registry {
	return globalRegistry
}

// Register adds a plugin to this registry.
func (r *Registry) Register(p plugin.ChannelPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[p.Name()] = p
}

// Get returns the plugin with the given name.
func (r *Registry) Get(name string) (plugin.ChannelPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.plugins[name]
	if !ok {
		return nil, fmt.Errorf("channel registry: %q not registered", name)
	}
	return p, nil
}

// All returns all registered plugins.
func (r *Registry) All() []plugin.ChannelPlugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]plugin.ChannelPlugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	return out
}
