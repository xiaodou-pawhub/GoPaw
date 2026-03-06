// Package tools registers all built-in GoPaw tool implementations.
package builtin

import (
	"sync"
)

var (
	memoryDirMu sync.RWMutex
	memoryDir   string

	notesDirMu sync.RWMutex
	notesDir   string

	workspaceRootMu sync.RWMutex
	workspaceRoot   string
)

// SetWorkspaceRoot sets the base directory for path sandboxing.
func SetWorkspaceRoot(root string) {
	workspaceRootMu.Lock()
	defer workspaceRootMu.Unlock()
	workspaceRoot = root
}

func getWorkspaceRoot() string {
	workspaceRootMu.RLock()
	defer workspaceRootMu.RUnlock()
	return workspaceRoot
}
// SetMemoryDir sets the base directory for file-based memory.
func SetMemoryDir(dir string) {
	memoryDirMu.Lock()
	defer memoryDirMu.Unlock()
	memoryDir = dir
}

func getMemoryDir() string {
	memoryDirMu.RLock()
	defer memoryDirMu.RUnlock()
	return memoryDir
}

// SetMemoryNotesDir sets the directory containing daily notes.
func SetMemoryNotesDir(dir string) {
	notesDirMu.Lock()
	defer notesDirMu.Unlock()
	notesDir = dir
}

func getMemoryNotesDir() string {
	notesDirMu.RLock()
	defer notesDirMu.RUnlock()
	return notesDir
}
