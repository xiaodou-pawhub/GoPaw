// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gopaw/gopaw/pkg/types"
)

// HealthStatus describes the current operational status of a channel plugin.
type HealthStatus struct {
	// Running indicates whether the channel is actively accepting and sending messages.
	Running bool
	// Message provides a human-readable status description.
	Message string
	// Since records when the current status was entered.
	Since time.Time
}

// TestResult describes the result of a channel connection test.
type TestResult struct {
	// Success indicates whether the test passed.
	Success bool `json:"success"`
	// Message provides a human-readable result description.
	Message string `json:"message"`
	// Details contains optional error details for debugging.
	Details string `json:"details,omitempty"`
}

// ChannelPlugin is the interface that every channel plugin must satisfy.
// A channel plugin adapts a specific messaging platform (Feishu, DingTalk, Webhook …)
// to the unified GoPaw message model.
type ChannelPlugin interface {
	// Name returns the unique snake_case identifier used in config.yaml.
	Name() string
	// DisplayName returns a human-readable label (e.g. "飞书").
	DisplayName() string

	// Init parses and applies the plugin-specific configuration blob.
	// cfg is the raw YAML/JSON sub-tree from the plugin section of config.yaml.
	Init(cfg json.RawMessage) error

	// Start begins accepting messages from the underlying platform.
	// Implementations should spawn their own goroutines and return immediately.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the plugin, draining any in-flight work.
	Stop() error

	// Receive returns a read-only channel that emits inbound messages.
	// The channel manager reads from all registered plugins via this channel.
	Receive() <-chan *types.Message

	// Send delivers a message to the underlying platform.
	Send(msg *types.Message) error

	// Health returns the current operational status.
	Health() HealthStatus

	// Test validates the channel connection and credentials.
	// It returns a TestResult indicating success/failure with a human-readable message.
	// This is used by the Web UI to verify configuration before/after saving.
	Test(ctx context.Context) TestResult
}
