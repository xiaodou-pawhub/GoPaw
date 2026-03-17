// Package server provides the HTTP server, WebSocket handler and middleware for GoPaw.
package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Allow all origins in development; restrict in production via config.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// wsMessage is the JSON structure used for WebSocket communication.
type wsMessage struct {
	Type      string `json:"type"`      // "message" | "delta" | "done" | "error" | "switch_agent" | "agent_switched"
	Content   string `json:"content,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	AgentID   string `json:"agent_id,omitempty"`   // For switch_agent
	AgentName string `json:"agent_name,omitempty"` // For agent_switched response
}

// WSHandler holds the dependencies for the WebSocket endpoint.
type WSHandler struct {
	agent  *agent.ReActAgent
	router *agent.Router
	logger *zap.Logger
}

// NewWSHandler creates a WSHandler.
func NewWSHandler(a *agent.ReActAgent, router *agent.Router, logger *zap.Logger) *WSHandler {
	return &WSHandler{agent: a, router: router, logger: logger}
}

// Handle is the Gin handler for GET /ws.
func (h *WSHandler) Handle(c *gin.Context) {
	sessionID := c.Query("session_id")

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Warn("ws: upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(24 * time.Hour)) //nolint:errcheck

	h.logger.Info("ws: client connected", zap.String("session_id", sessionID))

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				h.logger.Warn("ws: connection closed unexpectedly", zap.Error(err))
			}
			return
		}

		var inMsg wsMessage
		if err := json.Unmarshal(raw, &inMsg); err != nil {
			h.sendWS(conn, wsMessage{Type: "error", Content: "invalid message format"})
			continue
		}

		// Update session ID from message if provided
		if inMsg.SessionID != "" {
			sessionID = inMsg.SessionID
		}

		switch inMsg.Type {
		case "message":
			h.handleMessage(conn, sessionID, inMsg)
		case "switch_agent":
			h.handleSwitchAgent(conn, sessionID, inMsg)
		default:
			h.sendWS(conn, wsMessage{Type: "error", Content: "unknown message type: " + inMsg.Type})
		}
	}
}

// handleMessage processes a chat message.
func (h *WSHandler) handleMessage(conn *websocket.Conn, sessionID string, inMsg wsMessage) {
	// Get agent for session (or default)
	agentInstance, agentID, err := h.getAgentForSession(sessionID)
	if err != nil {
		h.sendWS(conn, wsMessage{Type: "error", Content: "failed to get agent: " + err.Error()})
		return
	}

	h.logger.Debug("ws: processing message",
		zap.String("session_id", sessionID),
		zap.String("agent_id", agentID),
	)

	req := &types.Request{
		SessionID: sessionID,
		Content:   inMsg.Content,
		MsgType:   types.MsgTypeText,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	resp, err := agentInstance.Process(ctx, req)
	cancel()

	if err != nil {
		h.sendWS(conn, wsMessage{Type: "error", Content: err.Error()})
		return
	}

	h.sendWS(conn, wsMessage{Type: "delta", Content: resp.Content, AgentID: agentID})
	h.sendWS(conn, wsMessage{Type: "done"})
}

// handleSwitchAgent handles agent switching.
func (h *WSHandler) handleSwitchAgent(conn *websocket.Conn, sessionID string, inMsg wsMessage) {
	if inMsg.AgentID == "" {
		h.sendWS(conn, wsMessage{Type: "error", Content: "agent_id is required"})
		return
	}

	if h.router == nil {
		h.sendWS(conn, wsMessage{Type: "error", Content: "agent router not available"})
		return
	}

	if err := h.router.SwitchAgent(sessionID, inMsg.AgentID); err != nil {
		h.sendWS(conn, wsMessage{Type: "error", Content: "failed to switch agent: " + err.Error()})
		return
	}

	// Get agent name from manager for response
	agentName := inMsg.AgentID
	if def, err := h.router.GetAgentDefinition(inMsg.AgentID); err == nil {
		agentName = def.Name
	}

	h.sendWS(conn, wsMessage{
		Type:      "agent_switched",
		AgentID:   inMsg.AgentID,
		AgentName: agentName,
	})

	h.logger.Info("ws: agent switched",
		zap.String("session_id", sessionID),
		zap.String("agent_id", inMsg.AgentID),
	)
}

// getAgentForSession returns the agent instance for a session.
func (h *WSHandler) getAgentForSession(sessionID string) (*agent.ReActAgent, string, error) {
	// If router is available, use it for multi-agent support
	if h.router != nil {
		return h.router.GetAgentForSession(sessionID)
	}

	// Fallback to single agent mode
	return h.agent, "default", nil
}

func (h *WSHandler) sendWS(conn *websocket.Conn, msg wsMessage) {
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		h.logger.Warn("ws: write failed", zap.Error(err))
	}
}
