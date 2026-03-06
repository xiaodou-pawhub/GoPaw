package tool

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ApprovalVerdict represents the user's decision on a tool execution.
type ApprovalVerdict string

const (
	VerdictAllowed ApprovalVerdict = "allowed"
	VerdictDenied  ApprovalVerdict = "denied"
	VerdictTimeout ApprovalVerdict = "timeout"
)

// ApprovalRequest holds the state of a pending tool execution.
type ApprovalRequest struct {
	ID        string
	ToolName  string
	Args      map[string]interface{}
	Summary   string // Human-readable summary of the operation
	ChannelID string
	ChatID    string
	SessionID string
	CreatedAt time.Time
	Result    chan ApprovalVerdict
}

// ApprovalStore manages pending approvals.
type ApprovalStore struct {
	mu      sync.RWMutex
	pending map[string]*ApprovalRequest
}

func NewApprovalStore() *ApprovalStore {
	return &ApprovalStore{
		pending: make(map[string]*ApprovalRequest),
	}
}

// GlobalApprovalStore is the singleton for the whole application.
var GlobalApprovalStore = NewApprovalStore()

// CreateRequest registers a new pending approval and returns the request ID.
func (s *ApprovalStore) CreateRequest(toolName string, args map[string]interface{}, channel, chatID, session string) *ApprovalRequest {
	summary := fmt.Sprintf("执行工具 %s", toolName)
	req := &ApprovalRequest{
		ID:        uuid.New().String(),
		ToolName:  toolName,
		Args:      args,
		Summary:   summary,
		ChannelID: channel,
		ChatID:    chatID,
		SessionID: session,
		CreatedAt: time.Now(),
		Result:    make(chan ApprovalVerdict, 1),
	}

	s.mu.Lock()
	s.pending[req.ID] = req
	s.mu.Unlock()

	return req
}

// Resolve marks a pending request as approved or denied.
func (s *ApprovalStore) Resolve(id string, verdict ApprovalVerdict) error {
	s.mu.Lock()
	req, ok := s.pending[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("approval request %s not found", id)
	}
	delete(s.pending, id)
	s.mu.Unlock()

	select {
	case req.Result <- verdict:
	default:
	}
	return nil
}

// WaitForVerdict blocks until the user decides or timeout occurs.
func (s *ApprovalStore) WaitForVerdict(ctx context.Context, req *ApprovalRequest, timeout time.Duration) ApprovalVerdict {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case v := <-req.Result:
		return v
	case <-timer.C:
		s.Resolve(req.ID, VerdictTimeout)
		return VerdictTimeout
	case <-ctx.Done():
		s.Resolve(req.ID, VerdictDenied)
		return VerdictDenied
	}
}
