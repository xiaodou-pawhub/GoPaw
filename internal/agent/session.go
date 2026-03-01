// Package agent implements the ReAct agent engine.
package agent

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Session represents an active conversation context.
type Session struct {
	ID        string
	UserID    string
	Channel   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// SessionManager tracks active sessions in memory for fast lookup.
// Persistence is delegated to the memory store.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates an empty SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session)}
}

// GetOrCreate returns an existing session by ID or creates a new one.
// If sessionID is empty a UUID is generated automatically.
func (sm *SessionManager) GetOrCreate(sessionID, userID, channel string) (*Session, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[sessionID]; ok {
		s.UpdatedAt = time.Now()
		return s, nil
	}

	s := &Session{
		ID:        sessionID,
		UserID:    userID,
		Channel:   channel,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	sm.sessions[sessionID] = s
	return s, nil
}

// Get retrieves a session by ID.
func (sm *SessionManager) Get(sessionID string) (*Session, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session %q not found", sessionID)
	}
	return s, nil
}

// Delete removes a session from the in-memory cache.
func (sm *SessionManager) Delete(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions, sessionID)
}

// All returns all active sessions.
func (sm *SessionManager) All() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		out = append(out, s)
	}
	return out
}
