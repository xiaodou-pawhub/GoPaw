// Package plugin defines the public interfaces that all GoPaw plugins must implement.
package plugin

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gopaw/gopaw/pkg/types"
)

// MediaMeta holds metadata for a stored media item.
type MediaMeta struct {
	Filename    string
	ContentType string
	Source      string
}

// MediaStore is an interface for storing and resolving media files.
type MediaStore interface {
	Store(localPath string, meta MediaMeta, scope string) (string, error)
	Resolve(refID string) (string, error)
	ResolveWithMeta(refID string) (string, MediaMeta, error)
	Delete(refID string) error
	TempPath(ext string) string
}

// MediaStoreReceiver is an optional interface for plugins that need access
// to the global MediaStore for handling images, files, etc.
type MediaStoreReceiver interface {
	SetMediaStore(s MediaStore)
}

// HealthStatus describes the current operational status of a channel plugin.
type HealthStatus struct {
	Running bool
	Message string
	Since   time.Time
}

// TestResult describes the result of a channel connection test.
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ChannelPlugin is the interface that every channel plugin must satisfy.
type ChannelPlugin interface {
	Name() string
	DisplayName() string
	Init(cfg json.RawMessage) error
	Start(ctx context.Context) error
	Stop() error
	Receive() <-chan *types.Message
	Send(msg *types.Message) error
	Health() HealthStatus
	Test(ctx context.Context) TestResult
}
