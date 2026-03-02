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
	active   []plugin.ChannelPlugin
	mu       sync.RWMutex

	// aggregated is the merged inbound message channel (1000 buffer).
	aggregated chan *types.Message
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
func (m *Manager) Start(ctx context.Context, enabledNames []string, pluginCfgs map[string]json.RawMessage) error {
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

// GetPlugin returns the plugin with the given name.
// This is a wrapper around the registry's Get method for use by HTTP handlers.
func (m *Manager) GetPlugin(name string) (plugin.ChannelPlugin, error) {
	return m.registry.Get(name)
}
