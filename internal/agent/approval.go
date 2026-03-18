package agent

import (
	"time"
)

// AutonomyLevel represents the autonomy level for tool execution.
type AutonomyLevel string

func (l AutonomyLevel) String() string {
	return string(l)
}

// ApprovalRequest represents a request for human approval.
type ApprovalRequest struct {
	ID          string
	ToolName    string
	Args        string
	Level       AutonomyLevel
	RequestedAt time.Time
	SessionID   string
	AgentID     string
}

// ApprovalResponse represents a response to an approval request.
type ApprovalResponse struct {
	RequestID   string
	Approved    bool
	Reason      string
	RespondedAt time.Time
}
