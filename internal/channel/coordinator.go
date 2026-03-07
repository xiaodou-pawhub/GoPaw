// Package channel provides channel management and routing utilities.
package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	// We'll send a specialized card.
	if req.ChannelID == "feishu" {
		argsJSON, _ := json.MarshalIndent(req.Args, "", "  ")

		// Build an interactive card with buttons
		card := map[string]interface{}{
			"schema": "2.0",
			"header": map[string]interface{}{
				"title":    map[string]string{"tag": "plain_text", "content": "⚠️ 安全审批"},
				"template": "orange",
			},
			"body": map[string]interface{}{
				"elements": []interface{}{
					map[string]interface{}{
						"tag":     "markdown",
						"content": fmt.Sprintf("**操作请求**\n%s\n\n**详情参数**:\n```json\n%s\n```", req.Summary, string(argsJSON)),
					},
					map[string]interface{}{
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
										"text": map[string]string{"tag": "plain_text", "content": "允许"},
										"type": "primary",
										"value": map[string]string{
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
										"text": map[string]string{"tag": "plain_text", "content": "拒绝"},
										"type": "danger",
										"value": map[string]string{
											"action":     "tool_approve",
											"request_id": req.ID,
											"verdict":    string(tool.VerdictDenied),
										},
									},
								},
							},
						},
					},
				},
			},
		}

		cardJSON, _ := json.Marshal(card)
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
// It stops the typing indicator, updates reactions, edits the placeholder, and cleans up media resources.
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

	// 3. Handle Placeholder or normal Send
	if err == nil {
		if v, ok := c.placeholders.LoadAndDelete(placeholderKey(inbound)); ok {
			placeholderID := v.(string)
			if me, ok := p.(plugin.MessageEditor); ok {
				if editErr := me.EditMessage(ctx, inbound.ChatID, placeholderID, reply.Content); editErr != nil {
					c.logger.Warn("placeholder edit failed, falling back to send",
						zap.String("channel", inbound.Channel), zap.Error(editErr))
					return c.mgr.Send(reply)
				}
				return nil
			}
		}
	}

	/* 
	   REMOVED: Aggressive cleanup of all session media after every turn.
	   This prevents multi-turn operations on the same image (e.g. info -> process -> send).
	   The MediaStore's internal TTL janitor (1h) will handle background cleanup.
	   
	   if c.store != nil {
	       c.store.ReleaseAll(inbound.SessionID)
	   }
	*/

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
