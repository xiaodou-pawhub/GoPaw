// Package config handles configuration loading, validation and access throughout GoPaw.
package config

import (
	"go.uber.org/zap"
)

// Watcher wraps the config manager's hot-reload functionality behind a simple
// start/stop lifecycle that matches the rest of GoPaw's component patterns.
type Watcher struct {
	manager *Manager
	logger  *zap.Logger
}

// NewWatcher creates a Watcher backed by the given Manager.
func NewWatcher(m *Manager, logger *zap.Logger) *Watcher {
	return &Watcher{manager: m, logger: logger}
}

// Start activates the Viper filesystem watcher.
// It is non-blocking; change notifications are delivered via the callbacks
// registered with Manager.OnChange.
func (w *Watcher) Start() {
	w.logger.Info("config watcher started")
	w.manager.WatchConfig()
}
