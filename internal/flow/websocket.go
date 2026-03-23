// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// ExecutionEvent 执行事件
type ExecutionEvent struct {
	Type        string      `json:"type"`         // 事件类型: started, node_started, node_completed, node_failed, completed, failed
	ExecutionID string      `json:"execution_id"` // 执行 ID
	FlowID      string      `json:"flow_id"`      // 流程 ID
	NodeID      string      `json:"node_id"`      // 节点 ID（可选）
	NodeName    string      `json:"node_name"`    // 节点名称
	Status      string      `json:"status"`       // 状态
	Output      interface{} `json:"output"`       // 输出（可选）
	Error       string      `json:"error"`        // 错误信息（可选）
	Timestamp   int64       `json:"timestamp"`    // 时间戳
}

// WebSocketClient WebSocket 客户端
type WebSocketClient struct {
	conn     *websocket.Conn
	flowID   string
	send     chan []byte
	hub      *WebSocketHub
	logger   *zap.Logger
}

// WebSocketHub WebSocket 连接管理中心
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	flowSubs   map[string]map[*WebSocketClient]bool // flowID -> clients
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mu         sync.RWMutex
	logger     *zap.Logger
}

// NewWebSocketHub 创建 WebSocket Hub
func NewWebSocketHub(logger *zap.Logger) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		flowSubs:   make(map[string]map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		logger:     logger.Named("ws_hub"),
	}
}

// Run 运行 Hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			if client.flowID != "" {
				if h.flowSubs[client.flowID] == nil {
					h.flowSubs[client.flowID] = make(map[*WebSocketClient]bool)
				}
				h.flowSubs[client.flowID][client] = true
			}
			h.mu.Unlock()
			h.logger.Debug("client registered", zap.String("flow_id", client.flowID))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				if client.flowID != "" && h.flowSubs[client.flowID] != nil {
					delete(h.flowSubs[client.flowID], client)
				}
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Debug("client unregistered", zap.String("flow_id", client.flowID))

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToFlow 向订阅特定流程的客户端广播消息
func (h *WebSocketHub) BroadcastToFlow(flowID string, event ExecutionEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	subscribers := h.flowSubs[flowID]
	if len(subscribers) == 0 {
		return
	}

	event.Timestamp = event.Timestamp / 1e6 // 转换为毫秒
	message, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to marshal event", zap.Error(err))
		return
	}

	for client := range subscribers {
		select {
		case client.send <- message:
		default:
			// 客户端阻塞，跳过
		}
	}
}

// BroadcastEvent 广播执行事件
func (h *WebSocketHub) BroadcastEvent(event ExecutionEvent) {
	if event.FlowID != "" {
		h.BroadcastToFlow(event.FlowID, event)
	}
}

// RegisterClient 注册客户端
func (h *WebSocketHub) RegisterClient(client *WebSocketClient) {
	h.register <- client
}

// UnregisterClient 注销客户端
func (h *WebSocketHub) UnregisterClient(client *WebSocketClient) {
	h.unregister <- client
}

// NewWebSocketClient 创建 WebSocket 客户端
func NewWebSocketClient(conn *websocket.Conn, flowID string, hub *WebSocketHub, logger *zap.Logger) *WebSocketClient {
	return &WebSocketClient{
		conn:   conn,
		flowID: flowID,
		send:   make(chan []byte, 256),
		hub:    hub,
		logger: logger,
	}
}

// ReadPump 读取消息
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// WritePump 写入消息
func (c *WebSocketClient) WritePump() {
	defer c.conn.Close()

	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			break
		}
	}
}