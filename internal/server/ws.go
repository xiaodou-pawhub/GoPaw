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
	Type      string `json:"type"`    // "message" | "delta" | "done" | "error"
	Content   string `json:"content,omitempty"`
	SessionID string `json:"session_id,omitempty"`
}

// WSHandler holds the dependencies for the WebSocket endpoint.
type WSHandler struct {
	agent  *agent.ReActAgent
	logger *zap.Logger
}

// NewWSHandler creates a WSHandler.
func NewWSHandler(a *agent.ReActAgent, logger *zap.Logger) *WSHandler {
	return &WSHandler{agent: a, logger: logger}
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

		if inMsg.Type != "message" {
			continue
		}
		if sessionID == "" && inMsg.SessionID != "" {
			sessionID = inMsg.SessionID
		}

		req := &types.Request{
			SessionID: sessionID,
			Content:   inMsg.Content,
			MsgType:   types.MsgTypeText,
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 120*time.Second)
		resp, err := h.agent.Process(ctx, req)
		cancel()

		if err != nil {
			h.sendWS(conn, wsMessage{Type: "error", Content: err.Error()})
			continue
		}

		h.sendWS(conn, wsMessage{Type: "delta", Content: resp.Content})
		h.sendWS(conn, wsMessage{Type: "done"})
	}
}

func (h *WSHandler) sendWS(conn *websocket.Conn, msg wsMessage) {
	data, _ := json.Marshal(msg)
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		h.logger.Warn("ws: write failed", zap.Error(err))
	}
}
