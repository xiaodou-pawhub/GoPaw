// Package server provides composite approval UI handling for multiple channels.
package server

import (
	"context"

	"github.com/gopaw/gopaw/internal/tool"
	"go.uber.org/zap"
)

// CompositeApprovalUI combines multiple approval UI handlers.
// It routes approval requests to the appropriate handler based on channel.
type CompositeApprovalUI struct {
	feishu    tool.ApprovalUI // Feishu channel coordinator
	webConsole tool.ApprovalUI // Web Console WebSocket handler
	logger    *zap.Logger
}

// NewCompositeApprovalUI creates a composite approval UI.
func NewCompositeApprovalUI(feishuUI tool.ApprovalUI, webConsoleUI tool.ApprovalUI, logger *zap.Logger) *CompositeApprovalUI {
	return &CompositeApprovalUI{
		feishu:     feishuUI,
		webConsole: webConsoleUI,
		logger:     logger.Named("composite_approval_ui"),
	}
}

// RequestApproval routes the approval request to the appropriate handler.
func (c *CompositeApprovalUI) RequestApproval(ctx context.Context, req *tool.ApprovalRequest) error {
	c.logger.Info("routing approval request",
		zap.String("tool", req.ToolName),
		zap.String("channel", req.ChannelID))

	// Route based on channel
	switch req.ChannelID {
	case "feishu":
		if c.feishu != nil {
			return c.feishu.RequestApproval(ctx, req)
		}
		c.logger.Warn("feishu approval UI not configured")
		return nil

	case "web", "":
		// Web Console or empty channel (default to web)
		if c.webConsole != nil {
			return c.webConsole.RequestApproval(ctx, req)
		}
		c.logger.Warn("web console approval UI not configured")
		return nil

	default:
		// For other channels, try feishu first, then web console
		if c.feishu != nil {
			if err := c.feishu.RequestApproval(ctx, req); err == nil {
				return nil
			}
		}
		if c.webConsole != nil {
			return c.webConsole.RequestApproval(ctx, req)
		}
		c.logger.Warn("no approval UI configured for channel", zap.String("channel", req.ChannelID))
		return nil
	}
}

// Ensure CompositeApprovalUI implements tool.ApprovalUI
var _ tool.ApprovalUI = (*CompositeApprovalUI)(nil)

// WebConsoleApprovalUI implements tool.ApprovalUI for Web Console.
// It stores the request in the approval store for WebSocket clients to pick up.
type WebConsoleApprovalUI struct {
	store  *tool.ApprovalStore
	logger *zap.Logger
}

// NewWebConsoleApprovalUI creates a new Web Console approval UI.
func NewWebConsoleApprovalUI(store *tool.ApprovalStore, logger *zap.Logger) *WebConsoleApprovalUI {
	return &WebConsoleApprovalUI{
		store:  store,
		logger: logger.Named("web_console_approval"),
	}
}

// RequestApproval implements tool.ApprovalUI.
// The request is already stored in the ApprovalStore by the executor.
// This method just logs the request for WebSocket clients.
func (w *WebConsoleApprovalUI) RequestApproval(ctx context.Context, req *tool.ApprovalRequest) error {
	w.logger.Info("web console approval: request available",
		zap.String("id", req.ID),
		zap.String("tool", req.ToolName),
		zap.String("session", req.SessionID))

	// The request is already in the store, WebSocket clients will poll or receive push
	return nil
}

// Ensure WebConsoleApprovalUI implements tool.ApprovalUI
var _ tool.ApprovalUI = (*WebConsoleApprovalUI)(nil)
