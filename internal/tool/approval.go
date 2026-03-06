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
	MessageID string // Feishu message ID of the approval card (for editing)
	CreatedAt time.Time
	Result    chan ApprovalVerdict
	done      chan struct{} // closed by Resolve to signal auto-timeout goroutine
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
// It automatically starts a 5-minute timeout goroutine that will auto-deny if not resolved.
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
		done:      make(chan struct{}),
	}

	s.mu.Lock()
	s.pending[req.ID] = req
	s.mu.Unlock()

	// Start auto-timeout goroutine (5 minutes).
	// Uses req.done (closed by Resolve) instead of reading req.Result,
	// so it never races with WaitForVerdict for the verdict value.
	go func() {
		timer := time.NewTimer(5 * time.Minute)
		defer timer.Stop()

		select {
		case <-timer.C:
			// Timeout - auto deny
			s.mu.Lock()
			if _, exists := s.pending[req.ID]; exists {
				delete(s.pending, req.ID)
				select {
				case req.Result <- VerdictTimeout:
				default:
				}
			}
			s.mu.Unlock()
		case <-req.done:
			// Already resolved by Resolve(), exit cleanly.
		}
	}()

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
	close(req.done) // signal auto-timeout goroutine to exit
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
