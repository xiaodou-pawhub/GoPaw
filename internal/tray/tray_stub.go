//go:build !tray

// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package tray provides a no-op stub when compiled without -tags tray.
// Server and Docker builds use this file — no CGo dependency, no systray.
package tray

import "go.uber.org/zap"

// Config mirrors the desktop version so callers don't need build tags.
type Config struct {
	AppURL string
	OnQuit func()
	Logger *zap.Logger
}

// Run is a no-op in server builds. The --tray flag is silently ignored.
func Run(cfg Config) {
	if cfg.Logger != nil {
		cfg.Logger.Warn("tray: build tag 'tray' not set; systray is unavailable in this binary")
	}
	// Block forever so callers still behave as if systray is running.
	select {}
}
