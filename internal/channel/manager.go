// Package channel manages the lifecycle of channel plugins and routes messages to the agent.
package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// MessageHandler is a function that processes an inbound message and returns a response.
// Returning a non-nil response causes the manager to send it back via the originating channel.
type MessageHandler func(ctx context.Context, msg *types.Message) (*types.Message, error)

// Manager starts and monitors all enabled channel plugins.
// It aggregates messages from all channels into a single stream and hands them to the agent.
type Manager struct {
	registry *Registry
	logger   *zap.Logger

	// active holds the plugins that were successfully initialised and started.
	active []plugin.ChannelPlugin
	mu     sync.RWMutex

	// aggregated is the merged inbound message channel (1000 buffer).
	aggregated chan *types.Message

	// ctx is the process-level lifecycle context for plugin background tasks.
	// It should not be tied to HTTP request contexts.
	ctx context.Context
}

// NewManager creates a Manager backed by the given registry.
func NewManager(registry *Registry, logger *zap.Logger) *Manager {
	return &Manager{
		registry:   registry,
		logger:     logger,
		aggregated: make(chan *types.Message, 1000),
	}
}

// Start initialises and starts all enabled channel plugins.
// enabledNames is the list from config.yaml plugins.enabled.
// pluginCfgs is the raw plugin configuration sub-tree keyed by plugin name.
// The context is saved for hot-reload operations to ensure plugin background
// tasks are not tied to HTTP request lifecycles.
func (m *Manager) Start(ctx context.Context, enabledNames []string, pluginCfgs map[string]json.RawMessage) error {
	// Save the process-level context for Reinit
	m.ctx = ctx

	for _, name := range enabledNames {
		p, err := m.registry.Get(name)
		if err != nil {
			m.logger.Warn("channel plugin not registered, skipping",
				zap.String("plugin", name), zap.Error(err))
			continue
		}

		cfg := pluginCfgs[name]
		if err := p.Init(cfg); err != nil {
			return fmt.Errorf("channel manager: init %q: %w", name, err)
		}
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("channel manager: start %q: %w", name, err)
		}

		m.mu.Lock()
		m.active = append(m.active, p)
		m.mu.Unlock()

		// Fan all inbound messages into the aggregated channel.
		go m.fanIn(ctx, p)

		m.logger.Info("channel started", zap.String("plugin", name))
	}
	return nil
}

// fanIn copies messages from a single plugin's Receive() channel into m.aggregated.
func (m *Manager) fanIn(ctx context.Context, p plugin.ChannelPlugin) {
	recv := p.Receive()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-recv:
			if !ok {
				return
			}
			select {
			case m.aggregated <- msg:
			default:
				m.logger.Warn("aggregated channel full, dropping message",
					zap.String("channel", p.Name()))
			}
		}
	}
}

// Messages returns a read-only channel of all inbound messages across all channels.
func (m *Manager) Messages() <-chan *types.Message {
	return m.aggregated
}

// Send delivers a message via the plugin identified by msg.Channel.
func (m *Manager) Send(msg *types.Message) error {
	p, err := m.registry.Get(msg.Channel)
	if err != nil {
		return fmt.Errorf("channel manager: send: %w", err)
	}
	return p.Send(msg)
}

// Stop gracefully shuts down all active channel plugins.
func (m *Manager) Stop() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.active {
		if err := p.Stop(); err != nil {
			m.logger.Warn("channel stop error",
				zap.String("plugin", p.Name()), zap.Error(err))
		}
	}
}

// Health returns a snapshot of every active plugin's health status.
func (m *Manager) Health() map[string]plugin.HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make(map[string]plugin.HealthStatus, len(m.active))
	for _, p := range m.active {
		out[p.Name()] = p.Health()
	}
	return out
}

// Test triggers a connection test for the specified channel plugin.
func (m *Manager) Test(ctx context.Context, name string) (plugin.TestResult, error) {
	p, err := m.registry.Get(name)
	if err != nil {
		return plugin.TestResult{}, fmt.Errorf("channel manager: test: %w", err)
	}
	return p.Test(ctx), nil
}

// GetPlugin returns the plugin with the given name from the registry.
// Note: This returns registered plugins regardless of whether they are started.
// Use GetActivePlugin for HTTP handlers that require a running plugin.
func (m *Manager) GetPlugin(name string) (plugin.ChannelPlugin, error) {
	return m.registry.Get(name)
}

// GetActivePlugin returns an active (initialized and started) plugin by name.
// Returns error if the plugin is not registered or not in the active list.
func (m *Manager) GetActivePlugin(name string) (plugin.ChannelPlugin, error) {
	// First check if plugin is registered
	_, err := m.registry.Get(name)
	if err != nil {
		return nil, err
	}

	// Then check if it's in the active list
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ap := range m.active {
		if ap.Name() == name {
			return ap, nil
		}
	}
	return nil, fmt.Errorf("channel: plugin %q not active", name)
}

// Reinit reinitializes a channel plugin with new configuration.
// It stops the old instance, initializes with new config, and restarts.
// This enables hot-reload of channel configuration without process restart.
// Note: It uses the process-level context saved during Manager.Start(),
// not the HTTP request context, to ensure plugin background tasks persist.
func (m *Manager) Reinit(name string, cfg json.RawMessage) error {
	// Use process-level context, not request context
	if m.ctx == nil {
		return fmt.Errorf("channel: manager not started, cannot reinit")
	}

	// 1. Get the plugin from registry (no lock needed for registry)
	p, err := m.registry.Get(name)
	if err != nil {
		return fmt.Errorf("channel: plugin %q not registered", name)
	}

	// 2. Find the old plugin instance and remove from active list (short lock)
	var oldPlugin plugin.ChannelPlugin
	wasActive := false
	m.mu.Lock()
	for i, ap := range m.active {
		if ap.Name() == name {
			oldPlugin = ap
			wasActive = true
			// Remove from active list
			m.active = append(m.active[:i], m.active[i+1:]...)
			break
		}
	}
	m.mu.Unlock()

	// 3. Stop old instance (outside lock - may be slow)
	if oldPlugin != nil {
		if err := oldPlugin.Stop(); err != nil {
			m.logger.Warn("channel: stop old plugin failed",
				zap.String("name", name),
				zap.Error(err),
			)
		}
	}

	// 4. Initialize with new config (outside lock - may be slow)
	if err := p.Init(cfg); err != nil {
		return fmt.Errorf("channel: reinit %q: %w", name, err)
	}

	// 5. Start the plugin with process-level context (outside lock - may be slow)
	if err := p.Start(m.ctx); err != nil {
		return fmt.Errorf("channel: start %q after reinit: %w", name, err)
	}

	// 6. Add to active list (short lock)
	m.mu.Lock()
	m.active = append(m.active, p)
	m.mu.Unlock()

	// 7. Only start fanIn if this plugin was not previously active
	// (already active plugins have fanIn running from initial Start)
	if !wasActive {
		go m.fanIn(m.ctx, p)
	}

	m.logger.Info("channel: reinit completed", zap.String("name", name))
	return nil
}
