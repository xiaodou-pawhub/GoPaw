//go:build tray

// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package tray provides system tray integration for GoPaw desktop use.
// Built only when compiled with -tags tray.
// Server / Docker builds omit this file and link the stub instead (no CGo).
package tray

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/getlantern/systray"
	"go.uber.org/zap"
)

// Config holds the parameters needed to set up the system tray.
type Config struct {
	AppURL string
	OnQuit func()
	Logger *zap.Logger
}

// Run starts the system tray and blocks until the user clicks Quit.
// Must be called from the main goroutine on macOS.
func Run(cfg Config) {
	systray.Run(func() { onReady(cfg) }, func() { onExit(cfg) })
}

func onReady(cfg Config) {
	systray.SetTitle("GoPaw")
	systray.SetTooltip("GoPaw AI Assistant")
	systray.SetIcon(iconData())

	mOpen := systray.AddMenuItem("Open GoPaw", "Open the web interface")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit GoPaw", "Stop the GoPaw server")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				openBrowser(cfg.AppURL, cfg.Logger)
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func onExit(cfg Config) {
	if cfg.OnQuit != nil {
		cfg.OnQuit()
	}
	os.Exit(0)
}

func openBrowser(url string, logger *zap.Logger) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd, args = "open", []string{url}
	case "windows":
		cmd, args = "rundll32", []string{"url.dll,FileProtocolHandler", url}
	default:
		cmd, args = "xdg-open", []string{url}
	}
	if err := exec.Command(cmd, args...).Start(); err != nil && logger != nil {
		logger.Warn("tray: failed to open browser", zap.String("url", url), zap.Error(err))
	}
}

// iconData returns a minimal 1×1 transparent PNG as the tray icon placeholder.
// Replace with a proper icon file embed for production builds.
func iconData() []byte {
	return []byte{
		0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
		0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4, 0x89, 0x00, 0x00, 0x00,
		0x0a, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9c, 0x62, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00, 0x00, 0x00, 0x49,
		0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
	}
}
