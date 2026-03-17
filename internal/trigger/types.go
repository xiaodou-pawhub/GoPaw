// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trigger

import (
	"encoding/json"
	"time"
)

// Trigger represents a trigger configuration.
type Trigger struct {
	ID          string     `json:"id"`
	AgentID     string     `json:"agent_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        string     `json:"type"` // cron/webhook/message
	Config      Config     `json:"config"`
	Reason      string     `json:"reason"`
	IsEnabled   bool       `json:"is_enabled"`
	LastFiredAt *time.Time `json:"last_fired_at"`
	FireCount   int        `json:"fire_count"`
	MaxFires    *int       `json:"max_fires"`
	CooldownSec int        `json:"cooldown_seconds"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Config is the interface for trigger configurations.
type Config interface {
	Type() string
}

// CronConfig for cron triggers.
type CronConfig struct {
	Expression string `json:"expression"`
}

// Type returns the trigger type.
func (c *CronConfig) Type() string { return "cron" }

// WebhookConfig for webhook triggers.
type WebhookConfig struct {
	Secret string `json:"secret"`
}

// Type returns the trigger type.
func (c *WebhookConfig) Type() string { return "webhook" }

// MessageConfig for message triggers.
type MessageConfig struct {
	FromAgent string `json:"from_agent"`
}

// Type returns the trigger type.
func (c *MessageConfig) Type() string { return "message" }

// TriggerContext passed to Agent when triggered.
type TriggerContext struct {
	TriggerID   string                 `json:"trigger_id"`
	TriggerType string                 `json:"trigger_type"`
	Reason      string                 `json:"reason"`
	Payload     map[string]interface{} `json:"payload"`
	FiredAt     time.Time              `json:"fired_at"`
}

// History represents a trigger fire history record.
type History struct {
	ID           int64     `json:"id"`
	TriggerID    string    `json:"trigger_id"`
	AgentID      string    `json:"agent_id"`
	FiredAt      time.Time `json:"fired_at"`
	Payload      string    `json:"payload"` // JSON string
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message"`
}

// ParseConfig parses config JSON based on trigger type.
func ParseConfig(triggerType string, configJSON []byte) (Config, error) {
	switch triggerType {
	case "cron":
		var cfg CronConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	case "webhook":
		var cfg WebhookConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	case "message":
		var cfg MessageConfig
		if err := json.Unmarshal(configJSON, &cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	default:
		return nil, nil
	}
}

// MarshalConfig marshals config to JSON.
func MarshalConfig(cfg Config) ([]byte, error) {
	return json.Marshal(cfg)
}
