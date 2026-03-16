// Package websocket provides WebSocket-based notification and approval handling.
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketNotifier implements the Notifier and ApprovalHandler interfaces.
type WebSocketNotifier struct {
	upgrader websocket.Upgrader
	clients  map[string]*Client
	mu       sync.RWMutex
	logger   *zap.Logger
}

// Client represents a WebSocket client connection.
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	Notifier *WebSocketNotifier
}

// Message represents a WebSocket message.
type Message struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// NotificationPayload represents a notification message.
type NotificationPayload struct {
	Message   string    `json:"message"`
	ToolName  string    `json:"tool_name,omitempty"`
	Args      string    `json:"args,omitempty"`
	Result    string    `json:"result,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ApprovalRequestPayload represents an approval request.
type ApprovalRequestPayload struct {
	ID          string    `json:"id"`
	ToolName    string    `json:"tool_name"`
	Args        string    `json:"args"`
	Level       string    `json:"level"`
	RequestedAt time.Time `json:"requested_at"`
	SessionID   string    `json:"session_id"`
	AgentID     string    `json:"agent_id"`
}

// ApprovalResponsePayload represents an approval response.
type ApprovalResponsePayload struct {
	RequestID   string    `json:"request_id"`
	Approved    bool      `json:"approved"`
	Reason      string    `json:"reason,omitempty"`
	RespondedAt time.Time `json:"responded_at"`
}

// NewWebSocketNotifier creates a new WebSocket notifier.
func NewWebSocketNotifier(logger *zap.Logger) *WebSocketNotifier {
	return &WebSocketNotifier{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *gin.Context) bool {
				return true // Allow all origins in development
			},
		},
		clients: make(map[string]*Client),
		logger:  logger.Named("websocket_notifier"),
	}
}

// HandleWebSocket handles WebSocket connections.
func (n *WebSocketNotifier) HandleWebSocket(c *gin.Context) {
	conn, err := n.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		n.logger.Error("failed to upgrade connection", zap.Error(err))
		return
	}

	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = fmt.Sprintf("client_%d", time.Now().UnixNano())
	}

	client := &Client{
		ID:       clientID,
		Conn:     conn,
		Send:     make(chan []byte, 256),
		Notifier: n,
	}

	n.mu.Lock()
	n.clients[clientID] = client
	n.mu.Unlock()

	n.logger.Info("client connected", zap.String("client_id", clientID))

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection.
func (c *Client) readPump() {
	defer func() {
		c.Notifier.unregister(c.ID)
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Notifier.logger.Error("websocket error", zap.Error(err))
			}
			break
		}

		// Handle incoming messages (e.g., approval responses)
		c.Notifier.handleMessage(c.ID, message)
	}
}

// writePump writes messages to the WebSocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// unregister removes a client from the registry.
func (n *WebSocketNotifier) unregister(clientID string) {
	n.mu.Lock()
	if client, ok := n.clients[clientID]; ok {
		close(client.Send)
		delete(n.clients, clientID)
	}
	n.mu.Unlock()
	n.logger.Info("client disconnected", zap.String("client_id", clientID))
}

// handleMessage handles incoming WebSocket messages.
func (n *WebSocketNotifier) handleMessage(clientID string, data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		n.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	switch msg.Type {
	case "approval_response":
		n.handleApprovalResponse(clientID, msg.Payload)
	default:
		n.logger.Debug("unknown message type", zap.String("type", msg.Type))
	}
}

// handleApprovalResponse handles approval response from client.
func (n *WebSocketNotifier) handleApprovalResponse(clientID string, payload json.RawMessage) {
	var resp ApprovalResponsePayload
	if err := json.Unmarshal(payload, &resp); err != nil {
		n.logger.Error("failed to unmarshal approval response", zap.Error(err))
		return
	}

	// Forward to pending approvals
	n.mu.RLock()
	pendingCh, ok := n.pendingApprovals[resp.RequestID]
	n.mu.RUnlock()

	if ok {
		approvalResp := &agent.ApprovalResponse{
			RequestID:   resp.RequestID,
			Approved:    resp.Approved,
			Reason:      resp.Reason,
			RespondedAt: resp.RespondedAt,
		}
		pendingCh <- approvalResp
	}
}

// pendingApprovals stores channels for pending approval requests.
var pendingApprovals = make(map[string]chan *agent.ApprovalResponse)
var pendingMu sync.RWMutex

// Notify implements the Notifier interface.
func (n *WebSocketNotifier) Notify(message string) {
	payload := NotificationPayload{
		Message:   message,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(Message{
		Type:    "notification",
		Payload: mustMarshal(payload),
	})

	n.broadcast(data)
}

// NotifyWithDetails implements the Notifier interface.
func (n *WebSocketNotifier) NotifyWithDetails(toolName string, args string, result string) {
	payload := NotificationPayload{
		Message:   fmt.Sprintf("Tool executed: %s", toolName),
		ToolName:  toolName,
		Args:      args,
		Result:    result,
		Timestamp: time.Now(),
	}

	data, _ := json.Marshal(Message{
		Type:    "notification",
		Payload: mustMarshal(payload),
	})

	n.broadcast(data)
}

// RequestApproval implements the ApprovalHandler interface.
func (n *WebSocketNotifier) RequestApproval(ctx context.Context, req *agent.ApprovalRequest) (*agent.ApprovalResponse, error) {
	// Create response channel
	respCh := make(chan *agent.ApprovalResponse, 1)

	pendingMu.Lock()
	pendingApprovals[req.ID] = respCh
	pendingMu.Unlock()

	defer func() {
		pendingMu.Lock()
		delete(pendingApprovals, req.ID)
		pendingMu.Unlock()
	}()

	// Send approval request to all clients
	payload := ApprovalRequestPayload{
		ID:          req.ID,
		ToolName:    req.ToolName,
		Args:        req.Args,
		Level:       req.Level.String(),
		RequestedAt: req.RequestedAt,
		SessionID:   req.SessionID,
		AgentID:     req.AgentID,
	}

	data, _ := json.Marshal(Message{
		Type:    "approval_request",
		Payload: mustMarshal(payload),
	})

	n.broadcast(data)

	// Wait for response or timeout
	select {
	case resp := <-respCh:
		return resp, nil
	case <-ctx.Done():
		return &agent.ApprovalResponse{
			RequestID:   req.ID,
			Approved:    false,
			Reason:      "timeout",
			RespondedAt: time.Now(),
		}, ctx.Err()
	case <-time.After(5 * time.Minute): // 5 minute timeout
		return &agent.ApprovalResponse{
			RequestID:   req.ID,
			Approved:    false,
			Reason:      "timeout",
			RespondedAt: time.Now(),
		}, fmt.Errorf("approval timeout")
	}
}

// broadcast sends a message to all connected clients.
func (n *WebSocketNotifier) broadcast(data []byte) {
	n.mu.RLock()
	clients := make([]*Client, 0, len(n.clients))
	for _, client := range n.clients {
		clients = append(clients, client)
	}
	n.mu.RUnlock()

	for _, client := range clients {
		select {
		case client.Send <- data:
		default:
			// Channel full, close connection
			n.unregister(client.ID)
		}
	}
}

// mustMarshal marshals data to JSON, panics on error (should never happen).
func mustMarshal(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}

// GetClientCount returns the number of connected clients.
func (n *WebSocketNotifier) GetClientCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.clients)
}
