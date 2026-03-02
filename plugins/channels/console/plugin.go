// Package console implements the built-in Web Console channel plugin.
// It provides a simple in-process message bus that the HTTP/WebSocket server uses
// to inject user messages and receive agent replies.
package console

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gopaw/gopaw/internal/channel"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

func init() {
	channel.Register(&Plugin{
		inbound: make(chan *types.Message, 256),
	})
}

// Plugin is the Web Console channel plugin.
type Plugin struct {
	inbound chan *types.Message
	started time.Time
	logger  *zap.Logger
}

func (p *Plugin) Name() string        { return "console" }
func (p *Plugin) DisplayName() string { return "Web Console" }

// Init parses the (empty) console configuration.
func (p *Plugin) Init(cfg json.RawMessage) error {
	p.logger = zap.L().Named("channel.console")
	return nil
}

// Start begins the console channel (no-op — messages arrive via Push).
func (p *Plugin) Start(_ context.Context) error {
	p.started = time.Now()
	p.logger.Info("console channel started")
	return nil
}

// Stop drains the inbound channel and signals shutdown.
func (p *Plugin) Stop() error {
	p.logger.Info("console channel stopped")
	return nil
}

// Receive returns the channel of inbound messages.
func (p *Plugin) Receive() <-chan *types.Message {
	return p.inbound
}

// Send delivers an outbound message. For the console channel this is a no-op because
// the HTTP handler reads responses directly from the agent; this method exists to satisfy
// the interface and for future in-process bridging.
func (p *Plugin) Send(msg *types.Message) error {
	p.logger.Debug("console: send", zap.String("content", msg.Content))
	return nil
}

// Health returns the current health status.
func (p *Plugin) Health() plugin.HealthStatus {
	return plugin.HealthStatus{
		Running: true,
		Message: "ok",
		Since:   p.started,
	}
}

// Test validates the console channel (always succeeds as it's internal).
func (p *Plugin) Test(ctx context.Context) plugin.TestResult {
	return plugin.TestResult{
		Success: true,
		Message: "Web Console 通道正常",
	}
}

// Push injects a message into the inbound queue from the HTTP layer.
func (p *Plugin) Push(msg *types.Message) {
	select {
	case p.inbound <- msg:
	default:
		p.logger.Warn("console: inbound queue full, dropping message")
	}
}
