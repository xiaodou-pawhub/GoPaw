// Package server provides WebSocket-based approval handling for Web Console.
package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/gopaw/gopaw/internal/tool"
	"go.uber.org/zap"
)

// ApprovalWSHandler handles WebSocket connections for tool approval.
type ApprovalWSHandler struct {
	store  *tool.ApprovalStore
	logger *zap.Logger
}

// NewApprovalWSHandler creates a new approval WebSocket handler.
func NewApprovalWSHandler(store *tool.ApprovalStore, logger *zap.Logger) *ApprovalWSHandler {
	return &ApprovalWSHandler{
		store:  store,
		logger: logger.Named("approval_ws"),
	}
}

// ApprovalRequestMessage is sent from server to client when approval is needed.
type ApprovalRequestMessage struct {
	Type      string                 `json:"type"` // "approval_request"
	ID        string                 `json:"id"`
	ToolName  string                 `json:"tool_name"`
	Args      map[string]interface{} `json:"args"`
	Summary   string                 `json:"summary"`
	SessionID string                 `json:"session_id"`
	Timestamp time.Time              `json:"timestamp"`
}

// ApprovalResponseMessage is sent from client to server with user's decision.
type ApprovalResponseMessage struct {
	Type     string `json:"type"` // "approval_response"
	ID       string `json:"id"`
	Approved bool   `json:"approved"`
	Reason   string `json:"reason,omitempty"`
}

// WebSocket upgrader for approval connections.
var approvalUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// Handle upgrades HTTP to WebSocket and handles approval communication.
func (h *ApprovalWSHandler) Handle(c *gin.Context) {
	conn, err := approvalUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("approval ws: upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	h.logger.Info("approval ws: client connected")

	// Handle incoming messages (approval responses)
	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				h.logger.Warn("approval ws: connection closed", zap.Error(err))
			}
			return
		}

		var msg ApprovalResponseMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			h.logger.Warn("approval ws: invalid message", zap.Error(err))
			continue
		}

		if msg.Type != "approval_response" {
			continue
		}

		// Resolve the approval request
		h.logger.Info("approval ws: received response",
			zap.String("id", msg.ID),
			zap.Bool("approved", msg.Approved))

		verdict := tool.VerdictDenied
		if msg.Approved {
			verdict = tool.VerdictAllowed
		}

		if err := h.store.Resolve(msg.ID, verdict); err != nil {
			h.logger.Warn("approval ws: failed to resolve", zap.Error(err))
		}
	}
}

// BroadcastApproval sends an approval request to all connected WebSocket clients.
// In production, this should target specific sessions. For now, we use a simple broadcast.
func (h *ApprovalWSHandler) BroadcastApproval(req *tool.ApprovalRequest) error {
	// Store the request for WebSocket clients to pick up
	// In a real implementation, this would push to specific connections
	h.logger.Info("approval ws: broadcasting request",
		zap.String("id", req.ID),
		zap.String("tool", req.ToolName))

	// The actual broadcast would be implemented with a connection manager
	// For now, the WebSocket client polls or the request is stored
	return nil
}
