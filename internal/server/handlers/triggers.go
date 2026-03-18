// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/trigger"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// TriggerHandler handles trigger-related HTTP requests.
type TriggerHandler struct {
	manager *trigger.Manager
	engine  *trigger.Engine
	logger  *zap.Logger
}

// NewTriggerHandler creates a new trigger handler.
func NewTriggerHandler(manager *trigger.Manager, engine *trigger.Engine, logger *zap.Logger) *TriggerHandler {
	return &TriggerHandler{
		manager: manager,
		engine:  engine,
		logger:  logger.Named("trigger_handler"),
	}
}

// CreateTriggerRequest represents a request to create a trigger.
type CreateTriggerRequest struct {
	ID          string                 `json:"id" binding:"required"`
	AgentID     string                 `json:"agent_id" binding:"required"`
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Type        string                 `json:"type" binding:"required"` // cron/webhook/message
	Config      map[string]interface{} `json:"config" binding:"required"`
	Reason      string                 `json:"reason"`
	IsEnabled   bool                   `json:"is_enabled"`
	MaxFires    *int                   `json:"max_fires"`
	CooldownSec int                    `json:"cooldown_seconds"`
}

// UpdateTriggerRequest represents a request to update a trigger.
type UpdateTriggerRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Reason      string                 `json:"reason"`
	IsEnabled   *bool                  `json:"is_enabled"`
	MaxFires    *int                   `json:"max_fires"`
	CooldownSec *int                   `json:"cooldown_seconds"`
}

// ListTriggers returns all triggers.
func (h *TriggerHandler) ListTriggers(c *gin.Context) {
	triggers := h.manager.List()
	api.Success(c, triggers)
}

// GetTrigger returns a specific trigger.
func (h *TriggerHandler) GetTrigger(c *gin.Context) {
	id := c.Param("id")
	trigger, err := h.manager.Get(id)
	if err != nil {
		api.NotFound(c, "trigger")
		return
	}
	api.Success(c, trigger)
}

// CreateTrigger creates a new trigger.
func (h *TriggerHandler) CreateTrigger(c *gin.Context) {
	var req CreateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	// Parse config based on type
	configJSON, _ := json.Marshal(req.Config)
	config, err := trigger.ParseConfig(req.Type, configJSON)
	if err != nil {
		api.BadRequestWithError(c, "invalid config", err)
		return
	}

	trigger := &trigger.Trigger{
		ID:          req.ID,
		AgentID:     req.AgentID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Config:      config,
		Reason:      req.Reason,
		IsEnabled:   req.IsEnabled,
		MaxFires:    req.MaxFires,
		CooldownSec: req.CooldownSec,
	}

	if err := h.manager.Create(trigger); err != nil {
		h.logger.Error("failed to create trigger", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to create trigger", err)
		return
	}

	api.Created(c, trigger)
}

// UpdateTrigger updates an existing trigger.
func (h *TriggerHandler) UpdateTrigger(c *gin.Context) {
	id := c.Param("id")

	existing, err := h.manager.Get(id)
	if err != nil {
		api.NotFound(c, "trigger")
		return
	}

	var req UpdateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	// Update fields
	if req.Name != "" {
		existing.Name = req.Name
	}
	if req.Description != "" {
		existing.Description = req.Description
	}
	if req.Type != "" {
		existing.Type = req.Type
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		config, err := trigger.ParseConfig(req.Type, configJSON)
		if err != nil {
			api.BadRequestWithError(c, "invalid config", err)
			return
		}
		existing.Config = config
	}
	if req.Reason != "" {
		existing.Reason = req.Reason
	}
	if req.IsEnabled != nil {
		existing.IsEnabled = *req.IsEnabled
	}
	if req.MaxFires != nil {
		existing.MaxFires = req.MaxFires
	}
	if req.CooldownSec != nil {
		existing.CooldownSec = *req.CooldownSec
	}

	if err := h.manager.Update(id, existing); err != nil {
		h.logger.Error("failed to update trigger", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to update trigger", err)
		return
	}

	api.Success(c, existing)
}

// DeleteTrigger deletes a trigger.
func (h *TriggerHandler) DeleteTrigger(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.Delete(id); err != nil {
		api.NotFound(c, "trigger")
		return
	}
	api.Success(c, gin.H{"message": "trigger deleted"})
}

// EnableTrigger enables a trigger.
func (h *TriggerHandler) EnableTrigger(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.SetEnabled(id, true); err != nil {
		api.NotFound(c, "trigger")
		return
	}
	api.Success(c, gin.H{"message": "trigger enabled"})
}

// DisableTrigger disables a trigger.
func (h *TriggerHandler) DisableTrigger(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.SetEnabled(id, false); err != nil {
		api.NotFound(c, "trigger")
		return
	}
	api.Success(c, gin.H{"message": "trigger disabled"})
}

// FireTrigger manually fires a trigger.
func (h *TriggerHandler) FireTrigger(c *gin.Context) {
	id := c.Param("id")

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = make(map[string]interface{})
	}

	if err := h.engine.FireTrigger(id, payload); err != nil {
		h.logger.Error("failed to fire trigger", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to fire trigger", err)
		return
	}

	api.Success(c, gin.H{"message": "trigger fired"})
}

// GetTriggerHistory returns trigger fire history.
func (h *TriggerHandler) GetTriggerHistory(c *gin.Context) {
	id := c.Param("id")
	limit := 50 // default limit

	history, err := h.manager.GetHistory(id, limit)
	if err != nil {
		h.logger.Error("failed to get trigger history", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to get trigger history", err)
		return
	}

	api.Success(c, history)
}

// ListTriggersByAgent returns triggers for a specific agent.
func (h *TriggerHandler) ListTriggersByAgent(c *gin.Context) {
	agentID := c.Param("agent_id")
	triggers := h.manager.ListByAgent(agentID)
	api.Success(c, triggers)
}

// ValidateCronRequest represents a request to validate a cron expression.
type ValidateCronRequest struct {
	Expression string `json:"expression" binding:"required"`
}

// ValidateCronResponse represents the response for cron validation.
type ValidateCronResponse struct {
	Valid       bool   `json:"valid"`
	Description string `json:"description,omitempty"`
	NextRun     string `json:"next_run,omitempty"`
	Error       string `json:"error,omitempty"`
}

// ValidateCron validates a cron expression.
func (h *TriggerHandler) ValidateCron(c *gin.Context) {
	var req ValidateCronRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	evaluator := trigger.NewEvaluator()
	err := evaluator.ValidateCron(req.Expression)

	resp := ValidateCronResponse{
		Valid:       err == nil,
		Description: evaluator.DescribeCron(req.Expression),
	}

	if err != nil {
		resp.Error = err.Error()
	} else {
		// Get next run time
		tmpTrigger := &trigger.Trigger{
			Type: "cron",
			Config: &trigger.CronConfig{Expression: req.Expression},
		}
		next, _ := evaluator.NextFireTime(tmpTrigger)
		if next != nil {
			resp.NextRun = next.Format(time.RFC3339)
		}
	}

	api.Success(c, resp)
}

// WebhookHandler handles webhook triggers.
type WebhookHandler struct {
	engine *trigger.Engine
	logger *zap.Logger
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(engine *trigger.Engine, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		engine: engine,
		logger: logger.Named("webhook_handler"),
	}
}

// HandleWebhook handles webhook requests.
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	triggerID := c.Param("id")
	secret := c.GetHeader("X-Webhook-Secret")

	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		payload = make(map[string]interface{})
	}

	if err := h.engine.FireWebhook(triggerID, secret, payload); err != nil {
		h.logger.Error("failed to fire webhook", zap.Error(err))
		api.BadRequestWithError(c, "invalid webhook request", err)
		return
	}

	api.Success(c, gin.H{"message": "webhook received"})
}

// MessageHandler handles message triggers between agents.
type MessageHandler struct {
	engine *trigger.Engine
	logger *zap.Logger
}

// NewMessageHandler creates a new message handler.
func NewMessageHandler(engine *trigger.Engine, logger *zap.Logger) *MessageHandler {
	return &MessageHandler{
		engine: engine,
		logger: logger.Named("message_handler"),
	}
}

// TriggerMessageRequest represents a request to send a message to an agent via trigger.
type TriggerMessageRequest struct {
	From    string                 `json:"from" binding:"required"`
	To      string                 `json:"to" binding:"required"`
	Payload map[string]interface{} `json:"payload"`
}

// SendMessage sends a message to trigger another agent.
func (h *MessageHandler) SendMessage(c *gin.Context) {
	var req TriggerMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.engine.FireMessage(req.From, req.To, req.Payload); err != nil {
		h.logger.Error("failed to send message", zap.Error(err))
		api.InternalErrorWithDetails(c, "failed to send message", err)
		return
	}

	api.Success(c, gin.H{"message": "message sent"})
}
