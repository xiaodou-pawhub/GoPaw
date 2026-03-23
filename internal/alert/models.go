// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package alert

import (
	"database/sql"
	"encoding/json"
	"time"
)

// AlertRule 告警规则
type AlertRule struct {
	ID          string          `json:"id" db:"id"`
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Type        AlertType       `json:"type" db:"type"` // metric, error, custom
	Condition   AlertCondition  `json:"condition" db:"condition"`
	Channels    []string        `json:"channels" db:"channels"` // 通知渠道 ID 列表
	Enabled     bool            `json:"enabled" db:"enabled"`
	LastTriggered *time.Time    `json:"last_triggered,omitempty" db:"last_triggered"`
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
}

// AlertType 告警类型
type AlertType string

const (
	AlertTypeMetric AlertType = "metric" // 指标告警
	AlertTypeError  AlertType = "error"  // 错误告警
	AlertTypeCustom AlertType = "custom" // 自定义告警
)

// AlertCondition 告警条件
type AlertCondition struct {
	Metric      string  `json:"metric"`      // 指标名称: latency, error_rate, token_usage
	Operator    string  `json:"operator"`    // 操作符: >, <, >=, <=, ==, !=
	Threshold   float64 `json:"threshold"`   // 阈值
	Duration    int     `json:"duration"`    // 持续时间（秒）
	Aggregation string  `json:"aggregation"` // 聚合方式: avg, max, min, sum
}

// Value 实现 driver.Valuer
func (c AlertCondition) Value() (interface{}, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner
func (c *AlertCondition) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	}
	return nil
}

// NotificationChannel 通知渠道
type NotificationChannel struct {
	ID          string              `json:"id" db:"id"`
	Name        string              `json:"name" db:"name"`
	Type        ChannelType         `json:"type" db:"type"` // email, dingtalk, wecom, webhook
	Config      ChannelConfig       `json:"config" db:"config"`
	Enabled     bool                `json:"enabled" db:"enabled"`
	LastUsed    *time.Time          `json:"last_used,omitempty" db:"last_used"`
	CreatedAt   time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at" db:"updated_at"`
}

// ChannelType 渠道类型
type ChannelType string

const (
	ChannelTypeEmail    ChannelType = "email"
	ChannelTypeDingTalk ChannelType = "dingtalk"
	ChannelTypeWeCom    ChannelType = "wecom"
	ChannelTypeWebhook  ChannelType = "webhook"
)

// ChannelConfig 渠道配置
type ChannelConfig struct {
	// Email
	SMTPHost     string `json:"smtp_host,omitempty"`
	SMTPPort     int    `json:"smtp_port,omitempty"`
	SMTPUser     string `json:"smtp_user,omitempty"`
	SMTPPassword string `json:"smtp_password,omitempty"`
	From         string `json:"from,omitempty"`
	To           []string `json:"to,omitempty"`

	// DingTalk
	DingTalkWebhook string `json:"dingtalk_webhook,omitempty"`
	DingTalkSecret  string `json:"dingtalk_secret,omitempty"`

	// WeCom
	WeComWebhook string `json:"wecom_webhook,omitempty"`

	// Webhook
	WebhookURL    string            `json:"webhook_url,omitempty"`
	WebhookMethod string            `json:"webhook_method,omitempty"`
	WebhookHeaders map[string]string `json:"webhook_headers,omitempty"`
}

// Value 实现 driver.Valuer
func (c ChannelConfig) Value() (interface{}, error) {
	return json.Marshal(c)
}

// Scan 实现 sql.Scanner
func (c *ChannelConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	}
	return nil
}

// AlertHistory 告警历史
type AlertHistory struct {
	ID         string    `json:"id" db:"id"`
	RuleID     string    `json:"rule_id" db:"rule_id"`
	RuleName   string    `json:"rule_name" db:"rule_name"`
	Type       AlertType `json:"type" db:"type"`
	Message    string    `json:"message" db:"message"`
	Value      float64   `json:"value" db:"value"`
	Threshold  float64   `json:"threshold" db:"threshold"`
	Status     string    `json:"status" db:"status"` // triggered, resolved
	Channels   []string  `json:"channels" db:"channels"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" db:"resolved_at"`
}

// InitSchema 初始化数据库表
func InitSchema(db *sql.DB) error {
	// 告警规则表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS alert_rules (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			type TEXT NOT NULL,
			condition TEXT NOT NULL,
			channels TEXT,
			enabled BOOLEAN DEFAULT 1,
			last_triggered TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// 通知渠道表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS notification_channels (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			config TEXT NOT NULL,
			enabled BOOLEAN DEFAULT 1,
			last_used TIMESTAMP,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		return err
	}

	// 告警历史表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS alert_history (
			id TEXT PRIMARY KEY,
			rule_id TEXT NOT NULL,
			rule_name TEXT NOT NULL,
			type TEXT NOT NULL,
			message TEXT NOT NULL,
			value REAL,
			threshold REAL,
			status TEXT DEFAULT 'triggered',
			channels TEXT,
			created_at TIMESTAMP NOT NULL,
			resolved_at TIMESTAMP,
			FOREIGN KEY (rule_id) REFERENCES alert_rules(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_alert_rules_enabled ON alert_rules(enabled)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_alert_history_rule ON alert_history(rule_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_alert_history_created ON alert_history(created_at)`)
	if err != nil {
		return err
	}

	return nil
}