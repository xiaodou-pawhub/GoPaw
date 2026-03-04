// Package agent implements the ReAct agent engine.
package agent

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionStrategy 定义了如何将物理消息映射到逻辑会话。
type SessionStrategy string

const (
	// StrategyPerSender 为每个用户建立独立会话 (私聊常用)。
	StrategyPerSender SessionStrategy = "per-sender"
	// StrategyPerChannel 为每个频道/群组建立独立会话 (群聊常用)。
	StrategyPerChannel SessionStrategy = "per-channel"
	// StrategyMain 所有人共享同一个全局会话。
	StrategyMain SessionStrategy = "main"
)

// SessionStatus represents the current execution state of an agent session.
type SessionStatus string

const (
	StatusIdle    SessionStatus = "idle"
	StatusRunning SessionStatus = "running"
)

// Session represents an active conversation context with concurrency protection.
type Session struct {
	ID        string
	UserID    string
	Channel   string
	Status    SessionStatus
	CreatedAt time.Time
	UpdatedAt time.Time
	
	mu sync.Mutex
}

// Acquire attempts to lock the session for execution.
func (s *Session) Acquire() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.Status == StatusRunning {
		return fmt.Errorf("session %s is busy: another request is being processed", s.ID)
	}

	s.Status = StatusRunning
	s.UpdatedAt = time.Now()
	return nil
}

// Release marks the session as idle again.
func (s *Session) Release() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = StatusIdle
	s.UpdatedAt = time.Now()
}

// SessionManager tracks active sessions in memory for fast lookup.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewSessionManager creates an empty SessionManager.
func NewSessionManager() *SessionManager {
	return &SessionManager{sessions: make(map[string]*Session)}
}

// GetOrCreate returns an existing session by ID or creates a new one.
func (sm *SessionManager) GetOrCreate(sessionID, userID, channel string) (*Session, error) {
	if sessionID == "" {
		sessionID = uuid.New().String()
	}

	sessionID = SanitizeSessionID(sessionID)

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if s, ok := sm.sessions[sessionID]; ok {
		return s, nil
	}

	s := &Session{
		ID:        sessionID,
		UserID:    userID,
		Channel:   channel,
		Status:    StatusIdle,
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

// SanitizeSessionID cleans up session IDs for cross-platform safety.
func SanitizeSessionID(id string) string {
	r := strings.NewReplacer(":", "_", "/", "_", "\\", "_")
	return r.Replace(id)
}

// DeriveSessionID 根据策略计算最终的会话 ID。 (借鉴 ZeroClaw / OpenClaw)
// channel: 渠道名 (如 "feishu", "console")
// userID: 发送者唯一标识
// chatID: 聊天场景标识 (如群组 ID，私聊时通常等于 userID)
func DeriveSessionID(strategy SessionStrategy, channel, userID, chatID string) string {
	switch strategy {
	case StrategyMain:
		return "main"
	case StrategyPerChannel:
		// 整个群组或整个渠道共享一个会话
		if chatID != "" {
			return fmt.Sprintf("%s:%s", channel, chatID)
		}
		return channel
	case StrategyPerSender:
		// 每个用户独立会话，增加 channel 前缀防止不同渠道 ID 碰撞
		return fmt.Sprintf("%s:%s", channel, userID)
	default:
		// 默认行为：使用原始传入 ID 或回退到 PerSender
		if userID != "" {
			return fmt.Sprintf("%s:%s", channel, userID)
		}
		return "default"
	}
}
