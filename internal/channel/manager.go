// Package channel manages the lifecycle of channel plugins and routes messages to the agent.
package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// MessageHandler is a function that processes an inbound message and returns a response.
type MessageHandler func(ctx context.Context, msg *types.Message) (*types.Message, error)

// Manager starts and monitors all enabled channel plugins.
type Manager struct {
	registry *Registry
	logger   *zap.Logger
	store    *MediaStore

	// active holds the plugins that were successfully initialised and started.
	active []plugin.ChannelPlugin
	mu     sync.RWMutex

	// aggregated is the merged inbound message channel (1000 buffer).
	aggregated chan *types.Message

	// ctx is the process-level lifecycle context for plugin background tasks.
	ctx context.Context
}

// NewManager creates a Manager backed by the given registry and store.
func NewManager(registry *Registry, store *MediaStore, logger *zap.Logger) *Manager {
	return &Manager{
		registry:   registry,
		logger:     logger,
		store:      store,
		aggregated: make(chan *types.Message, 1000),
	}
}

// Start initialises and starts all registered channel plugins.
func (m *Manager) Start(ctx context.Context, pluginCfgs map[string]json.RawMessage) error {
	m.ctx = ctx

	for _, p := range m.registry.All() {
		name := p.Name()
		cfg := pluginCfgs[name]
		if cfg == nil {
			cfg = json.RawMessage("{}")
		}

		// Inject MediaStore if the plugin supports it.
		if msr, ok := p.(plugin.MediaStoreReceiver); ok && m.store != nil {
			msr.SetMediaStore(m.store)
		}

		if err := p.Init(cfg); err != nil {
			if errors.Is(err, plugin.ErrMissingCredentials) {
				m.logger.Info("channel not configured, skipping", zap.String("plugin", name))
			} else {
				m.logger.Warn("channel plugin init failed, skipping", zap.String("plugin", name), zap.Error(err))
			}
			continue
		}
		
		if err := p.Start(ctx); err != nil {
			m.logger.Warn("channel plugin start failed, skipping", zap.String("plugin", name), zap.Error(err))
			continue
		}

		m.mu.Lock()
		m.active = append(m.active, p)
		m.mu.Unlock()

		go m.supervisedFanIn(ctx, p)

		m.logger.Info("channel started", zap.String("plugin", name))
	}
	return nil
}

// supervisedFanIn copies messages from a plugin into m.aggregated with backoff.
func (m *Manager) supervisedFanIn(ctx context.Context, p plugin.ChannelPlugin) {
	const maxAttempts = 10
	bo := defaultBackoff()

	for attempt := 0; attempt < maxAttempts; attempt++ {
		healthy := m.drainMessages(ctx, p)
		if ctx.Err() != nil {
			return 
		}
		if healthy {
			bo.Reset()
		}
		if attempt == maxAttempts-1 {
			m.logger.Error("channel plugin max restart attempts reached, giving up",
				zap.String("plugin", p.Name()), zap.Int("attempts", maxAttempts))
			return
		}

		delay := bo.Next()
		m.logger.Warn("channel plugin disconnected, restarting",
			zap.String("plugin", p.Name()), zap.Int("attempt", attempt+1), zap.Duration("delay", delay))

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		_ = p.Stop()
		if err := p.Start(ctx); err != nil {
			m.logger.Error("channel plugin restart failed", zap.String("plugin", p.Name()), zap.Error(err))
			continue
		}
		m.logger.Info("channel plugin restarted", zap.String("plugin", p.Name()))
	}
}

func (m *Manager) drainMessages(ctx context.Context, p plugin.ChannelPlugin) (healthy bool) {
	recv := p.Receive()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-recv:
			if !ok {
				return
			}
			healthy = true
			select {
			case m.aggregated <- msg:
			default:
				m.logger.Warn("aggregated channel full, dropping message", zap.String("channel", p.Name()))
			}
		}
	}
}

// Messages returns a read-only channel of all inbound messages.
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
			m.logger.Warn("channel stop error", zap.String("plugin", p.Name()), zap.Error(err))
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
func (m *Manager) GetPlugin(name string) (plugin.ChannelPlugin, error) {
	return m.registry.Get(name)
}

// GetActivePlugin returns an active plugin by name.
func (m *Manager) GetActivePlugin(name string) (plugin.ChannelPlugin, error) {
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
func (m *Manager) Reinit(name string, cfg json.RawMessage) error {
	if m.ctx == nil {
		return fmt.Errorf("channel: manager not started, cannot reinit")
	}

	p, err := m.registry.Get(name)
	if err != nil {
		return fmt.Errorf("channel: plugin %q not registered", name)
	}

	var oldPlugin plugin.ChannelPlugin
	m.mu.Lock()
	for i, ap := range m.active {
		if ap.Name() == name {
			oldPlugin = ap
			m.active = append(m.active[:i], m.active[i+1:]...)
			break
		}
	}
	m.mu.Unlock()

	if oldPlugin != nil {
		_ = oldPlugin.Stop()
	}

	if msr, ok := p.(plugin.MediaStoreReceiver); ok && m.store != nil {
		msr.SetMediaStore(m.store)
	}

	if err := p.Init(cfg); err != nil {
		return fmt.Errorf("channel: reinit %q: %w", name, err)
	}

	if err := p.Start(m.ctx); err != nil {
		return fmt.Errorf("channel: start %q after reinit: %w", name, err)
	}

	m.mu.Lock()
	m.active = append(m.active, p)
	m.mu.Unlock()

	go m.supervisedFanIn(m.ctx, p)

	return nil
}
