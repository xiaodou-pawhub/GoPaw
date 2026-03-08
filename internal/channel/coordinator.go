// Package channel provides channel management and routing utilities.
package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
	"github.com/gopaw/gopaw/pkg/types"
	"go.uber.org/zap"
)

// CapabilityCoordinator wraps Manager and transparently applies optional
// plugin capabilities (typing indicator, reactions, placeholders) around agent
// processing, degrading gracefully when the active plugin lacks the capability.
type CapabilityCoordinator struct {
	mgr    *Manager
	store  *MediaStore
	logger *zap.Logger

	// placeholders maps "channel:chatID:msgID" → placeholder message ID
	placeholders sync.Map
	// typingStops maps "channel:chatID" → stop func
	typingStops sync.Map
}

// Ensure CapabilityCoordinator implements tool.ApprovalUI
var _ tool.ApprovalUI = (*CapabilityCoordinator)(nil)

// NewCapabilityCoordinator creates a coordinator backed by mgr and store.
func NewCapabilityCoordinator(mgr *Manager, store *MediaStore) *CapabilityCoordinator {
	c := &CapabilityCoordinator{
		mgr:    mgr,
		store:  store,
		logger: zap.L().Named("channel.coordinator"),
	}
	return c
}

// RequestApproval sends an interactive card to the channel requesting permission to execute a tool.
func (c *CapabilityCoordinator) RequestApproval(ctx context.Context, req *tool.ApprovalRequest) error {
	p, err := c.mgr.GetActivePlugin(req.ChannelID)
	if err != nil {
		return err
	}

	// For now, we only support Feishu for interactive approvals.
	// We'll send a specialized card with collapsible detail panel.
	if req.ChannelID == "feishu" {
		c.logger.Info("building feishu approval card", zap.String("tool", req.ToolName))
		
		// Build card with summary and optional detail
		card := buildApprovalCard(req)

		cardJSON, _ := json.Marshal(card)
		
		// Log card JSON for debugging
		c.logger.Error("approval card built", 
			zap.String("tool", req.ToolName),
			zap.String("card_json", string(cardJSON)))

		msg := &types.Message{
			Channel:  req.ChannelID,
			ChatID:   req.ChatID,
			Content:  string(cardJSON),
			MsgType:  types.MsgTypeMarkdown,
		}

		// Send the card and get message ID
		if feishuPlugin, ok := p.(interface{ SendWithMessageID(*types.Message) (string, error) }); ok {
			messageID, err := feishuPlugin.SendWithMessageID(msg)
			if err != nil {
				return err
			}
			req.MessageID = messageID
			return nil
		}

		// Fallback to normal Send if MessageID is not supported
		return p.Send(msg)
	}

	return fmt.Errorf("approval not supported on channel %s", req.ChannelID)
}

// buildApprovalCard builds a Feishu approval card with summary and collapsible detail.
func buildApprovalCard(req *tool.ApprovalRequest) map[string]interface{} {
	// Get the tool from registry to check if it supports approval summary
	toolRegistry := tool.Global()
	toolInstance, ok := toolRegistry.Get(req.ToolName)

	var summaryText, detailText string

	// Check if tool supports ApprovalSummaryCapable interface
	if ok {
		if summaryTool, ok := toolInstance.(plugin.ApprovalSummaryCapable); ok {
			summaryText = summaryTool.ApprovalSummary(req.Args)
			detailText = summaryTool.ApprovalDetail(req.Args)
		}
	}
	
	// Fallback: show JSON if tool not found or doesn't support interface
	if summaryText == "" {
		argsJSON, _ := json.MarshalIndent(req.Args, "", "  ")
		summaryText = fmt.Sprintf("**操作请求**\n%s\n\n**详情参数**:\n```json\n%s\n```", req.Summary, string(argsJSON))
		detailText = "" // No collapsible panel for fallback
	}

	// Build card elements - simplified version without collapsible panel for now
	elements := []interface{}{
		map[string]interface{}{
			"tag":     "markdown",
			"content": summaryText,
		},
	}

	// Add detail as separate markdown if not empty (simplified approach)
	if detailText != "" {
		elements = append(elements, map[string]interface{}{
			"tag":     "markdown",
			"content": detailText,
		})
	}

	// Add action buttons
	elements = append(elements, map[string]interface{}{
		"tag": "column_set",
		"flex_mode": "stretch",
		"columns": []interface{}{
			map[string]interface{}{
				"tag": "column",
				"width": "weighted",
				"weight": 1,
				"elements": []interface{}{
					map[string]interface{}{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "允许",
						},
						"type": "primary",
						"value": map[string]interface{}{
							"action":     "tool_approve",
							"request_id": req.ID,
							"verdict":    string(tool.VerdictAllowed),
						},
					},
				},
			},
			map[string]interface{}{
				"tag": "column",
				"width": "weighted",
				"weight": 1,
				"elements": []interface{}{
					map[string]interface{}{
						"tag": "button",
						"text": map[string]interface{}{
							"tag":     "plain_text",
							"content": "拒绝",
						},
						"type": "danger",
						"value": map[string]interface{}{
							"action":     "tool_approve",
							"request_id": req.ID,
							"verdict":    string(tool.VerdictDenied),
						},
					},
				},
			},
		},
	})

	// Save approval context for later display in status card
	toolDisplay := extractToolDisplay(summaryText)
	tool.GlobalApprovalStore.SetApprovalContext(req.ID, &tool.ApprovalContext{
		ToolName:    req.ToolName,
		ToolDisplay: toolDisplay,
		Summary:     summaryText,
		Detail:      detailText,
		Timestamp:   time.Now().UnixMilli(),
	})

	return map[string]interface{}{
		"schema": "2.0",
		"header": map[string]interface{}{
			"title": map[string]interface{}{
				"tag":     "plain_text",
				"content": "⚠️ 安全审批",
			},
			"template": "orange",
		},
		"body": map[string]interface{}{
			"elements": elements,
		},
	}
}

// extractToolDisplay extracts the tool display name from the summary text.
// e.g., "📧 **发送邮件**\n收件人：..." → "📧 发送邮件"
func extractToolDisplay(summary string) string {
	lines := strings.Split(summary, "\n")
	if len(lines) > 0 {
		// Remove markdown bold markers
		display := strings.ReplaceAll(lines[0], "**", "")
		return strings.TrimSpace(display)
	}
	return "工具操作"
}

// PreProcess is called before the agent starts processing msg.
// It starts a typing indicator, sends a placeholder, and/or adds a "waiting" reaction.
func (c *CapabilityCoordinator) PreProcess(ctx context.Context, msg *types.Message) {
	p, err := c.mgr.GetActivePlugin(msg.Channel)
	if err != nil {
		return
	}

	// 1. Emoji Reaction (ACK)
	if rc, ok := p.(plugin.ReactionCapable); ok {
		if err := rc.AddReaction(ctx, msg.ChatID, msg.ID, plugin.ReactionWait); err != nil {
			c.logger.Debug("failed to add waiting reaction", zap.Error(err))
		}
	}

	// 2. Typing indicator
	if tc, ok := p.(plugin.TypingCapable); ok {
		stop, err := tc.StartTyping(ctx, msg.ChatID)
		if err != nil {
			c.logger.Warn("typing indicator failed", zap.String("channel", msg.Channel), zap.Error(err))
		} else {
			c.typingStops.Store(typingKey(msg), stop)
		}
	}

}
// Note: SendPlaceholder (thinking card) is intentionally not called here.
// The wait reaction emoji is sufficient to indicate the message was received.

// PostProcess is called after the agent produces reply.
// It stops the typing indicator, updates reactions, and sends the reply message.
func (c *CapabilityCoordinator) PostProcess(ctx context.Context, inbound, reply *types.Message) error {
	p, err := c.mgr.GetActivePlugin(inbound.Channel)

	// 1. Cleanup Typing indicator
	if v, ok := c.typingStops.LoadAndDelete(typingKey(inbound)); ok {
		v.(func())()
	}

	// 2. Update Reactions
	if err == nil {
		if rc, ok := p.(plugin.ReactionCapable); ok {
			_ = rc.RemoveReaction(ctx, inbound.ChatID, inbound.ID, plugin.ReactionWait)
			_ = rc.AddReaction(ctx, inbound.ChatID, inbound.ID, plugin.ReactionSuccess)
		}
	}

	// 3. Send reply message (always)
	// Note: SendPlaceholder is no longer called in PreProcess, so we always send
	// a new message here instead of editing a placeholder.
	if p == nil {
		return fmt.Errorf("plugin gone")
	}
	return c.mgr.Send(reply)
}

func typingKey(msg *types.Message) string {
	return fmt.Sprintf("%s:%s", msg.Channel, msg.ChatID)
}

func placeholderKey(msg *types.Message) string {
	return fmt.Sprintf("%s:%s:%s", msg.Channel, msg.ChatID, msg.ID)
}
