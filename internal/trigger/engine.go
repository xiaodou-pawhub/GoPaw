// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trigger

import (
	"fmt"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"go.uber.org/zap"
)

const (
	// TickInterval is the trigger evaluation interval.
	TickInterval = 30 * time.Second
	// DedupWindow is the deduplication window for the same agent.
	DedupWindow = 30 * time.Second
)

// Engine is the trigger execution engine.
type Engine struct {
	manager     *Manager
	agentRouter *agent.Router
	evaluator   *Evaluator
	logger      *zap.Logger
	tickInterval time.Duration
	stopCh      chan struct{}
	mu          sync.RWMutex
	lastFired   map[string]time.Time // agent_id -> last fired time
	running     bool
}

// NewEngine creates a new trigger engine.
func NewEngine(manager *Manager, agentRouter *agent.Router, logger *zap.Logger) *Engine {
	return &Engine{
		manager:      manager,
		agentRouter:  agentRouter,
		evaluator:    NewEvaluator(),
		logger:       logger.Named("trigger_engine"),
		tickInterval: TickInterval,
		stopCh:       make(chan struct{}),
		lastFired:    make(map[string]time.Time),
	}
}

// Start starts the trigger engine.
func (e *Engine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	e.logger.Info("trigger engine started", zap.Duration("tick_interval", e.tickInterval))

	// Start tick loop
	go e.tickLoop()
}

// Stop stops the trigger engine.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	e.mu.Unlock()

	close(e.stopCh)
	e.logger.Info("trigger engine stopped")
}

// tickLoop runs the tick loop.
func (e *Engine) tickLoop() {
	ticker := time.NewTicker(e.tickInterval)
	defer ticker.Stop()

	// Run immediately on start
	e.tick()

	for {
		select {
		case <-ticker.C:
			e.tick()
		case <-e.stopCh:
			return
		}
	}
}

// tick performs a single evaluation tick.
func (e *Engine) tick() {
	now := time.Now()

	// Load enabled triggers
	triggers := e.manager.ListEnabled()
	if len(triggers) == 0 {
		return
	}

	e.logger.Debug("tick evaluation", zap.Int("trigger_count", len(triggers)))

	// Group triggers by agent
	agentTriggers := e.groupByAgent(triggers)

	// Evaluate each agent's triggers
	for agentID, triggers := range agentTriggers {
		// Check deduplication
		if e.isRecentlyFired(agentID, now) {
			e.logger.Debug("agent recently fired, skipping",
				zap.String("agent_id", agentID),
				zap.Time("last_fired", e.lastFired[agentID]))
			continue
		}

		// Find the first trigger that should fire
		for _, t := range triggers {
			if e.shouldFire(t, now) {
				if err := e.fire(t, now); err != nil {
					e.logger.Error("failed to fire trigger",
						zap.String("trigger_id", t.ID),
						zap.Error(err))
				}
				break // Only fire one trigger per agent per tick
			}
		}
	}
}

// groupByAgent groups triggers by agent ID.
func (e *Engine) groupByAgent(triggers []*Trigger) map[string][]*Trigger {
	result := make(map[string][]*Trigger)
	for _, t := range triggers {
		result[t.AgentID] = append(result[t.AgentID], t)
	}
	return result
}

// isRecentlyFired checks if an agent was recently fired.
func (e *Engine) isRecentlyFired(agentID string, now time.Time) bool {
	e.mu.RLock()
	lastFired, ok := e.lastFired[agentID]
	e.mu.RUnlock()

	if !ok {
		return false
	}

	return now.Sub(lastFired) < DedupWindow
}

// shouldFire checks if a trigger should fire.
func (e *Engine) shouldFire(trigger *Trigger, now time.Time) bool {
	// Check max fires
	if trigger.MaxFires != nil && trigger.FireCount >= *trigger.MaxFires {
		return false
	}

	// Check cooldown
	if trigger.CooldownSec > 0 && trigger.LastFiredAt != nil {
		cooldown := time.Duration(trigger.CooldownSec) * time.Second
		if now.Sub(*trigger.LastFiredAt) < cooldown {
			return false
		}
	}

	// Evaluate based on type
	return e.evaluator.ShouldFire(trigger, now)
}

// fire triggers an agent.
func (e *Engine) fire(trigger *Trigger, now time.Time) error {
	e.logger.Info("firing trigger",
		zap.String("trigger_id", trigger.ID),
		zap.String("agent_id", trigger.AgentID),
		zap.String("type", trigger.Type))

	// Update last fired time
	e.mu.Lock()
	e.lastFired[trigger.AgentID] = now
	e.mu.Unlock()

	// Update trigger stats
	if err := e.manager.UpdateLastFired(trigger.ID); err != nil {
		e.logger.Warn("failed to update trigger stats", zap.Error(err))
	}

	// Get or create agent instance
	_, err := e.agentRouter.GetOrCreateAgent(trigger.AgentID)
	if err != nil {
		// Record failure
		e.manager.RecordFire(trigger.ID, trigger.AgentID, nil, false, err.Error())
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Create trigger context
	ctx := &TriggerContext{
		TriggerID:   trigger.ID,
		TriggerType: trigger.Type,
		Reason:      trigger.Reason,
		Payload:     make(map[string]interface{}),
		FiredAt:     now,
	}

	// Add type-specific payload
	switch trigger.Type {
	case "cron":
		ctx.Payload["expression"] = trigger.Config.(*CronConfig).Expression
	case "webhook":
		ctx.Payload["secret"] = trigger.Config.(*WebhookConfig).Secret
	case "message":
		ctx.Payload["from_agent"] = trigger.Config.(*MessageConfig).FromAgent
	}

	// Record success
	if err := e.manager.RecordFire(trigger.ID, trigger.AgentID, ctx.Payload, true, ""); err != nil {
		e.logger.Warn("failed to record trigger fire", zap.Error(err))
	}

	// Invoke agent (async)
	go func() {
		// TODO: Pass trigger context to agent
		// For now, just log
		e.logger.Info("agent invoked by trigger",
			zap.String("agent_id", trigger.AgentID),
			zap.String("trigger_id", trigger.ID))
	}()

	return nil
}

// FireTrigger manually fires a trigger by ID.
func (e *Engine) FireTrigger(triggerID string, payload map[string]interface{}) error {
	trigger, err := e.manager.Get(triggerID)
	if err != nil {
		return err
	}

	if !trigger.IsEnabled {
		return fmt.Errorf("trigger is disabled: %s", triggerID)
	}

	now := time.Now()
	return e.fire(trigger, now)
}

// FireWebhook fires a webhook trigger.
func (e *Engine) FireWebhook(triggerID string, secret string, payload map[string]interface{}) error {
	trigger, err := e.manager.Get(triggerID)
	if err != nil {
		return err
	}

	if trigger.Type != "webhook" {
		return fmt.Errorf("trigger is not a webhook: %s", triggerID)
	}

	// Validate secret
	config := trigger.Config.(*WebhookConfig)
	if config.Secret != "" && config.Secret != secret {
		return fmt.Errorf("invalid webhook secret")
	}

	// Merge payload
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["webhook"] = true

	now := time.Now()
	return e.fire(trigger, now)
}

// FireMessage fires a message trigger.
func (e *Engine) FireMessage(fromAgent, toAgent string, payload map[string]interface{}) error {
	// Find message trigger for the target agent
	triggers := e.manager.ListByAgent(toAgent)
	var targetTrigger *Trigger
	for _, t := range triggers {
		if t.Type == "message" && t.IsEnabled {
			config := t.Config.(*MessageConfig)
			if config.FromAgent == "" || config.FromAgent == fromAgent {
				targetTrigger = t
				break
			}
		}
	}

	if targetTrigger == nil {
		return fmt.Errorf("no message trigger found for agent: %s", toAgent)
	}

	// Merge payload
	if payload == nil {
		payload = make(map[string]interface{})
	}
	payload["from_agent"] = fromAgent
	payload["message"] = true

	now := time.Now()
	return e.fire(targetTrigger, now)
}

// IsRunning returns whether the engine is running.
func (e *Engine) IsRunning() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.running
}
