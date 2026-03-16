// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package config provides configuration management and hot-reload capabilities.
package config

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// WatchFile watches a single file and triggers callback on change.
// Uses debouncing to avoid rapid successive reloads.
func WatchFile(path string, callback func(), logger *zap.Logger) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(path); err != nil {
		watcher.Close()
		return err
	}

	logger.Info("watching file for hot-reload",
		zap.String("path", path),
	)

	// Debounce timer
	var timer *time.Timer
	var mu sync.Mutex

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					mu.Lock()
					if timer != nil {
						timer.Stop()
					}
					timer = time.AfterFunc(500*time.Millisecond, func() {
						logger.Info("file changed, triggering reload",
							zap.String("path", path),
						)
						callback()
					})
					mu.Unlock()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("watcher error", zap.Error(err))
			}
		}
	}()

	return nil
}

// WatchDir watches a directory and triggers callback when any file changes.
// Uses debouncing to avoid rapid successive reloads.
func WatchDir(dir string, callback func(path string), logger *zap.Logger) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return err
	}

	logger.Info("watching directory for hot-reload",
		zap.String("dir", dir),
	)

	// Debounce timer per file
	timers := make(map[string]*time.Timer)
	var mu sync.Mutex

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					mu.Lock()
					if timer, exists := timers[event.Name]; exists {
						timer.Stop()
					}
					timers[event.Name] = time.AfterFunc(500*time.Millisecond, func() {
						mu.Lock()
						delete(timers, event.Name)
						mu.Unlock()
						logger.Debug("file changed in watched directory",
							zap.String("path", event.Name),
						)
						callback(event.Name)
					})
					mu.Unlock()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logger.Error("watcher error", zap.Error(err))
			}
		}
	}()

	return nil
}
