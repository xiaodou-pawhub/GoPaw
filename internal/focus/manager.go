// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package focus

import (
	"fmt"
	"os"
	"sync"

	"go.uber.org/zap"
)

// Manager manages the focus tasks.
type Manager struct {
	focusPath string
	tasks     []Task
	notes     string
	mu        sync.RWMutex
	logger    *zap.Logger
}

// NewManager creates a new focus manager.
func NewManager(focusPath string, logger *zap.Logger) *Manager {
	return &Manager{
		focusPath: focusPath,
		tasks:     []Task{},
		logger:    logger.Named("focus"),
	}
}

// Load loads tasks from the focus file.
func (m *Manager) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tasks, notes, err := ParseFile(m.focusPath)
	if err != nil {
		return fmt.Errorf("failed to parse focus file: %w", err)
	}

	m.tasks = tasks
	m.notes = notes

	m.logger.Info("focus loaded",
		zap.Int("tasks", len(tasks)),
		zap.String("path", m.focusPath),
	)

	return nil
}

// Save saves tasks to the focus file.
func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	content := FormatTasks(m.tasks, m.notes)

	if err := os.WriteFile(m.focusPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write focus file: %w", err)
	}

	m.logger.Info("focus saved",
		zap.Int("tasks", len(m.tasks)),
		zap.String("path", m.focusPath),
	)

	return nil
}

// GetFocusText returns the formatted focus text for display.
func (m *Manager) GetFocusText() string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.tasks) == 0 {
		return "No active focus tasks."
	}

	var result string
	for _, task := range m.tasks {
		result += fmt.Sprintf("%s %s\n", task.Status.Symbol(), task.Title)
	}

	if m.notes != "" {
		result += "\n" + m.notes + "\n"
	}

	return result
}

// GetActiveTask returns the current active task (in-progress or first pending).
func (m *Manager) GetActiveTask() *Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return GetActiveTask(m.tasks)
}

// UpdateTask updates a task's status by title.
// Thread-safe: locks only for memory update, saves without lock.
func (m *Manager) UpdateTask(title string, status Status) error {
	// Step 1: Update memory with lock
	m.mu.Lock()
	found := false
	for i := range m.tasks {
		if m.tasks[i].Title == title {
			m.tasks[i].Status = status
			found = true
			break
		}
	}
	m.mu.Unlock()

	if !found {
		return fmt.Errorf("task not found: %s", title)
	}

	// Step 2: Save to file without lock
	if err := m.Save(); err != nil {
		// Log error but don't rollback memory state for simplicity
		m.logger.Error("failed to save focus after update",
			zap.String("title", title),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save after update: %w", err)
	}

	m.logger.Info("task updated",
		zap.String("title", title),
		zap.String("status", status.String()),
	)

	return nil
}

// AddTask adds a new task.
// Thread-safe: locks only for memory update, saves without lock.
func (m *Manager) AddTask(title string, status Status) error {
	// Step 1: Update memory with lock
	m.mu.Lock()
	// Check if task already exists
	for _, task := range m.tasks {
		if task.Title == title {
			m.mu.Unlock()
			return fmt.Errorf("task already exists: %s", title)
		}
	}
	m.tasks = append(m.tasks, Task{
		Title:  title,
		Status: status,
	})
	m.mu.Unlock()

	// Step 2: Save to file without lock
	if err := m.Save(); err != nil {
		m.logger.Error("failed to save focus after add",
			zap.String("title", title),
			zap.Error(err),
		)
		return fmt.Errorf("failed to save after add: %w", err)
	}

	m.logger.Info("task added",
		zap.String("title", title),
		zap.String("status", status.String()),
	)

	return nil
}

// GetTasks returns a copy of all tasks.
func (m *Manager) GetTasks() []Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Task, len(m.tasks))
	copy(result, m.tasks)
	return result
}

// GetPath returns the focus file path.
func (m *Manager) GetPath() string {
	return m.focusPath
}
