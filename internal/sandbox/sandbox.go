// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package sandbox provides session-based filesystem isolation for Agent tools.
// Each session gets its own sandbox directory, and all file operations are
// restricted to that directory.
package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages sandboxes for sessions.
type Manager struct {
	root      string
	sandboxes map[string]*Sandbox
	mu        sync.RWMutex
	logger    *zap.Logger
}

// Sandbox represents a single session's isolated workspace.
type Sandbox struct {
	SessionID string
	Root      string
	FilesDir  string
	TempDir   string
	CreatedAt time.Time
}

// NewManager creates a new sandbox manager.
func NewManager(root string, logger *zap.Logger) (*Manager, error) {
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sandbox root: %w", err)
	}

	m := &Manager{
		root:      root,
		sandboxes: make(map[string]*Sandbox),
		logger:    logger.Named("sandbox"),
	}

	// Clean up old sandboxes on startup
	go m.cleanupOldSandboxes()

	return m, nil
}

// GetOrCreate returns an existing sandbox or creates a new one.
func (m *Manager) GetOrCreate(sessionID string) (*Sandbox, error) {
	m.mu.RLock()
	if sb, ok := m.sandboxes[sessionID]; ok {
		m.mu.RUnlock()
		return sb, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if sb, ok := m.sandboxes[sessionID]; ok {
		return sb, nil
	}

	sb, err := m.createSandbox(sessionID)
	if err != nil {
		return nil, err
	}

	m.sandboxes[sessionID] = sb
	return sb, nil
}

// Get returns an existing sandbox or nil if not found.
func (m *Manager) Get(sessionID string) *Sandbox {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.sandboxes[sessionID]
}

// Remove removes a sandbox and deletes its directory.
func (m *Manager) Remove(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sb, ok := m.sandboxes[sessionID]
	if !ok {
		return nil
	}

	delete(m.sandboxes, sessionID)

	// Delete sandbox directory asynchronously
	go func() {
		if err := os.RemoveAll(sb.Root); err != nil {
			m.logger.Warn("failed to remove sandbox directory",
				zap.String("session_id", sessionID),
				zap.String("path", sb.Root),
				zap.Error(err),
			)
		} else {
			m.logger.Info("sandbox removed",
				zap.String("session_id", sessionID),
				zap.String("path", sb.Root),
			)
		}
	}()

	return nil
}

// createSandbox creates a new sandbox directory structure.
func (m *Manager) createSandbox(sessionID string) (*Sandbox, error) {
	// Sanitize session ID for filesystem
	safeID := sanitizeSessionID(sessionID)
	root := filepath.Join(m.root, safeID)

	sb := &Sandbox{
		SessionID: sessionID,
		Root:      root,
		FilesDir:  filepath.Join(root, "files"),
		TempDir:   filepath.Join(root, "temp"),
		CreatedAt: time.Now(),
	}

	// Create directories
	for _, dir := range []string{sb.FilesDir, sb.TempDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create sandbox directory %s: %w", dir, err)
		}
	}

	m.logger.Info("sandbox created",
		zap.String("session_id", sessionID),
		zap.String("path", root),
	)

	return sb, nil
}

// ResolvePath converts a user-provided path to a safe path within the sandbox.
// Returns error if the path attempts to escape the sandbox.
func (sb *Sandbox) ResolvePath(userPath string) (string, error) {
	// Clean the path
	userPath = filepath.Clean(userPath)

	// Check for path traversal attempts
	if strings.Contains(userPath, "..") {
		return "", fmt.Errorf("path traversal not allowed: %s", userPath)
	}

	// Remove leading slash if present
	userPath = strings.TrimPrefix(userPath, "/")

	// Join with files directory
	fullPath := filepath.Join(sb.FilesDir, userPath)

	// Ensure the resolved path is within the sandbox
	// This is a safety check in case filepath.Clean doesn't catch everything
	if !strings.HasPrefix(fullPath, sb.FilesDir) {
		return "", fmt.Errorf("path escapes sandbox: %s", userPath)
	}

	return fullPath, nil
}

// ResolvePathForSession is a convenience method on Manager.
func (m *Manager) ResolvePathForSession(sessionID, userPath string) (string, error) {
	sb, err := m.GetOrCreate(sessionID)
	if err != nil {
		return "", err
	}
	return sb.ResolvePath(userPath)
}

// GetSandboxRoot returns the root directory for a session's sandbox.
func (m *Manager) GetSandboxRoot(sessionID string) (string, error) {
	sb, err := m.GetOrCreate(sessionID)
	if err != nil {
		return "", err
	}
	return sb.Root, nil
}

// GetFilesDir returns the files directory for a session's sandbox.
func (m *Manager) GetFilesDir(sessionID string) (string, error) {
	sb, err := m.GetOrCreate(sessionID)
	if err != nil {
		return "", err
	}
	return sb.FilesDir, nil
}

// ListSandboxes returns a list of all active sandboxes.
func (m *Manager) ListSandboxes() []*Sandbox {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Sandbox, 0, len(m.sandboxes))
	for _, sb := range m.sandboxes {
		result = append(result, sb)
	}
	return result
}

// cleanupOldSandboxes removes sandboxes older than retention period.
func (m *Manager) cleanupOldSandboxes() {
	// Check for old sandboxes on disk (not in memory)
	entries, err := os.ReadDir(m.root)
	if err != nil {
		m.logger.Warn("failed to read sandbox root", zap.Error(err))
		return
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour) // 7 days retention

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(m.root, entry.Name())
			if err := os.RemoveAll(path); err != nil {
				m.logger.Warn("failed to cleanup old sandbox",
					zap.String("path", path),
					zap.Error(err),
				)
			} else {
				m.logger.Info("cleaned up old sandbox",
					zap.String("path", path),
					zap.Time("modified", info.ModTime()),
				)
			}
		}
	}
}

// sanitizeSessionID sanitizes a session ID for use as a directory name.
func sanitizeSessionID(id string) string {
	// Replace potentially problematic characters
	id = strings.ReplaceAll(id, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, ":", "_")
	id = strings.ReplaceAll(id, "..", "_")
	return id
}
