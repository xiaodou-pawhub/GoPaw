// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package config provides configuration management and hot-reload capabilities.
package config

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

// FileWatcher manages hot-reload for a single file.
type FileWatcher struct {
	watcher   *fsnotify.Watcher
	path      string
	callback  func()
	logger    *zap.Logger
	stopCh    chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex
	timer     *time.Timer
	lastHash  string // content hash to detect actual changes
}

// WatchFile watches a single file and triggers callback on change.
// Returns a cancel function to stop watching.
func WatchFile(path string, callback func(), logger *zap.Logger) (context.CancelFunc, error) {
	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return nil, err
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := fsWatcher.Add(path); err != nil {
		fsWatcher.Close()
		return nil, err
	}

	fw := &FileWatcher{
		watcher:  fsWatcher,
		path:     path,
		callback: callback,
		logger:   logger.Named("file_watcher"),
		stopCh:   make(chan struct{}),
	}

	// Calculate initial hash
	fw.lastHash = fw.calculateHash()

	logger.Info("watching file for hot-reload",
		zap.String("path", path),
	)

	fw.wg.Add(1)
	go fw.run()

	return func() { fw.Stop() }, nil
}

// run is the main event loop.
func (fw *FileWatcher) run() {
	defer fw.wg.Done()
	defer fw.watcher.Close()

	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				fw.handleChange()
			}

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fw.logger.Error("watcher error",
				zap.String("path", fw.path),
				zap.Error(err),
			)

		case <-fw.stopCh:
			return
		}
	}
}

// handleChange processes file change with debouncing and hash check.
func (fw *FileWatcher) handleChange() {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Cancel existing timer
	if fw.timer != nil {
		fw.timer.Stop()
	}

	// Debounce: wait 500ms after last change
	fw.timer = time.AfterFunc(500*time.Millisecond, func() {
		// Check if content actually changed
		newHash := fw.calculateHash()
		if newHash == "" {
			fw.logger.Warn("failed to read file, skipping reload",
				zap.String("path", fw.path),
			)
			return
		}

		fw.mu.Lock()
		if newHash == fw.lastHash {
			fw.mu.Unlock()
			fw.logger.Debug("file content unchanged, skipping reload",
				zap.String("path", fw.path),
			)
			return
		}
		fw.lastHash = newHash
		fw.mu.Unlock()

		fw.logger.Info("file changed, triggering reload",
			zap.String("path", fw.path),
		)
		fw.callback()
	})
}

// calculateHash calculates a simple hash of file content.
func (fw *FileWatcher) calculateHash() string {
	data, err := os.ReadFile(fw.path)
	if err != nil {
		return ""
	}
	// Simple hash: first 100 chars + length + last 100 chars
	if len(data) == 0 {
		return "empty"
	}
	prefix := data[:min(100, len(data))]
	suffix := data[max(0, len(data)-100):]
	return string(prefix) + "|" + string(suffix) + "|" + string(rune(len(data)))
}

// Stop stops the watcher.
func (fw *FileWatcher) Stop() {
	close(fw.stopCh)
	fw.wg.Wait()
	fw.mu.Lock()
	if fw.timer != nil {
		fw.timer.Stop()
	}
	fw.mu.Unlock()
}

// DirWatcher manages hot-reload for a directory.
type DirWatcher struct {
	watcher      *fsnotify.Watcher
	dir          string
	callback     func(path string)
	logger       *zap.Logger
	stopCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	globalTimer  *time.Timer // global debounce timer
	pendingFiles map[string]bool // files pending reload
}

// WatchDir watches a directory and triggers callback when any file changes.
// Uses global debouncing: all changes within 500ms trigger single reload.
func WatchDir(dir string, callback func(path string), logger *zap.Logger) (context.CancelFunc, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); err != nil {
		return nil, err
	}

	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	if err := fsWatcher.Add(dir); err != nil {
		fsWatcher.Close()
		return nil, err
	}

	dw := &DirWatcher{
		watcher:      fsWatcher,
		dir:          dir,
		callback:     callback,
		logger:       logger.Named("dir_watcher"),
		stopCh:       make(chan struct{}),
		pendingFiles: make(map[string]bool),
	}

	logger.Info("watching directory for hot-reload",
		zap.String("dir", dir),
	)

	dw.wg.Add(1)
	go dw.run()

	return func() { dw.Stop() }, nil
}

// run is the main event loop.
func (dw *DirWatcher) run() {
	defer dw.wg.Done()
	defer dw.watcher.Close()

	for {
		select {
		case event, ok := <-dw.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				dw.handleChange(event.Name)
			}

		case err, ok := <-dw.watcher.Errors:
			if !ok {
				return
			}
			dw.logger.Error("watcher error",
				zap.String("dir", dw.dir),
				zap.Error(err),
			)

		case <-dw.stopCh:
			return
		}
	}
}

// handleChange processes directory change with global debouncing.
func (dw *DirWatcher) handleChange(path string) {
	dw.mu.Lock()
	defer dw.mu.Unlock()

	// Track changed file
	dw.pendingFiles[path] = true

	// Cancel existing global timer
	if dw.globalTimer != nil {
		dw.globalTimer.Stop()
	}

	// Global debounce: wait 500ms after last change, then reload all
	dw.globalTimer = time.AfterFunc(500*time.Millisecond, func() {
		dw.mu.Lock()
		files := make([]string, 0, len(dw.pendingFiles))
		for f := range dw.pendingFiles {
			files = append(files, f)
		}
		dw.pendingFiles = make(map[string]bool) // clear pending
		dw.mu.Unlock()

		if len(files) > 0 {
			dw.logger.Info("directory changed, triggering reload",
				zap.String("dir", dw.dir),
				zap.Int("files_changed", len(files)),
			)
			// Trigger reload for the first changed file
			// (skill manager will reload all skills)
			dw.callback(files[0])
		}
	})
}

// Stop stops the watcher.
func (dw *DirWatcher) Stop() {
	close(dw.stopCh)
	dw.wg.Wait()
	dw.mu.Lock()
	if dw.globalTimer != nil {
		dw.globalTimer.Stop()
	}
	dw.mu.Unlock()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
