// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package alert

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

// Service 告警服务
type Service struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewService 创建告警服务
func NewService(db *sql.DB, logger *zap.Logger) (*Service, error) {
	if err := InitSchema(db); err != nil {
		return nil, err
	}
	return &Service{
		db:     db,
		logger: logger.Named("alert"),
	}, nil
}

// ========== 告警规则管理 ==========

// CreateRule 创建告警规则
func (s *Service) CreateRule(ctx context.Context, rule *AlertRule) error {
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	channelsJSON, _ := json.Marshal(rule.Channels)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO alert_rules (id, name, description, type, condition, channels, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, rule.ID, rule.Name, rule.Description, rule.Type, rule.Condition, string(channelsJSON), rule.Enabled, rule.CreatedAt, rule.UpdatedAt)

	return err
}

// GetRule 获取告警规则
func (s *Service) GetRule(ctx context.Context, id string) (*AlertRule, error) {
	rule := &AlertRule{}
	var channelsJSON sql.NullString
	var lastTriggered sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, type, condition, channels, enabled, last_triggered, created_at, updated_at
		FROM alert_rules WHERE id = ?
	`, id).Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.Type, &rule.Condition,
		&channelsJSON, &rule.Enabled, &lastTriggered, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if channelsJSON.Valid {
		json.Unmarshal([]byte(channelsJSON.String), &rule.Channels)
	}
	if lastTriggered.Valid {
		rule.LastTriggered = &lastTriggered.Time
	}

	return rule, nil
}

// ListRules 列出告警规则
func (s *Service) ListRules(ctx context.Context) ([]AlertRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, type, condition, channels, enabled, last_triggered, created_at, updated_at
		FROM alert_rules ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []AlertRule
	for rows.Next() {
		rule := AlertRule{}
		var channelsJSON sql.NullString
		var lastTriggered sql.NullTime

		err := rows.Scan(
			&rule.ID, &rule.Name, &rule.Description, &rule.Type, &rule.Condition,
			&channelsJSON, &rule.Enabled, &lastTriggered, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if channelsJSON.Valid {
			json.Unmarshal([]byte(channelsJSON.String), &rule.Channels)
		}
		if lastTriggered.Valid {
			rule.LastTriggered = &lastTriggered.Time
		}

		rules = append(rules, rule)
	}

	return rules, nil
}

// UpdateRule 更新告警规则
func (s *Service) UpdateRule(ctx context.Context, id string, rule *AlertRule) error {
	rule.UpdatedAt = time.Now()
	channelsJSON, _ := json.Marshal(rule.Channels)

	_, err := s.db.ExecContext(ctx, `
		UPDATE alert_rules SET name = ?, description = ?, type = ?, condition = ?, channels = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`, rule.Name, rule.Description, rule.Type, rule.Condition, string(channelsJSON), rule.Enabled, rule.UpdatedAt, id)

	return err
}

// DeleteRule 删除告警规则
func (s *Service) DeleteRule(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM alert_rules WHERE id = ?`, id)
	return err
}

// ========== 通知渠道管理 ==========

// CreateChannel 创建通知渠道
func (s *Service) CreateChannel(ctx context.Context, channel *NotificationChannel) error {
	if channel.ID == "" {
		channel.ID = fmt.Sprintf("ch_%d", time.Now().UnixNano())
	}
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO notification_channels (id, name, type, config, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, channel.ID, channel.Name, channel.Type, channel.Config, channel.Enabled, channel.CreatedAt, channel.UpdatedAt)

	return err
}

// GetChannel 获取通知渠道
func (s *Service) GetChannel(ctx context.Context, id string) (*NotificationChannel, error) {
	channel := &NotificationChannel{}
	var lastUsed sql.NullTime

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, type, config, enabled, last_used, created_at, updated_at
		FROM notification_channels WHERE id = ?
	`, id).Scan(
		&channel.ID, &channel.Name, &channel.Type, &channel.Config,
		&channel.Enabled, &lastUsed, &channel.CreatedAt, &channel.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if lastUsed.Valid {
		channel.LastUsed = &lastUsed.Time
	}

	return channel, nil
}

// ListChannels 列出通知渠道
func (s *Service) ListChannels(ctx context.Context) ([]NotificationChannel, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, type, config, enabled, last_used, created_at, updated_at
		FROM notification_channels ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []NotificationChannel
	for rows.Next() {
		channel := NotificationChannel{}
		var lastUsed sql.NullTime

		err := rows.Scan(
			&channel.ID, &channel.Name, &channel.Type, &channel.Config,
			&channel.Enabled, &lastUsed, &channel.CreatedAt, &channel.UpdatedAt,
		)
		if err != nil {
			continue
		}

		if lastUsed.Valid {
			channel.LastUsed = &lastUsed.Time
		}

		channels = append(channels, channel)
	}

	return channels, nil
}

// UpdateChannel 更新通知渠道
func (s *Service) UpdateChannel(ctx context.Context, id string, channel *NotificationChannel) error {
	channel.UpdatedAt = time.Now()

	_, err := s.db.ExecContext(ctx, `
		UPDATE notification_channels SET name = ?, type = ?, config = ?, enabled = ?, updated_at = ?
		WHERE id = ?
	`, channel.Name, channel.Type, channel.Config, channel.Enabled, channel.UpdatedAt, id)

	return err
}

// DeleteChannel 删除通知渠道
func (s *Service) DeleteChannel(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notification_channels WHERE id = ?`, id)
	return err
}

// ========== 告警历史 ==========

// CreateHistory 创建告警历史
func (s *Service) CreateHistory(ctx context.Context, history *AlertHistory) error {
	if history.ID == "" {
		history.ID = fmt.Sprintf("ah_%d", time.Now().UnixNano())
	}
	history.CreatedAt = time.Now()

	channelsJSON, _ := json.Marshal(history.Channels)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO alert_history (id, rule_id, rule_name, type, message, value, threshold, status, channels, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, history.ID, history.RuleID, history.RuleName, history.Type, history.Message,
		history.Value, history.Threshold, history.Status, string(channelsJSON), history.CreatedAt)

	return err
}

// ListHistory 列出告警历史
func (s *Service) ListHistory(ctx context.Context, limit int) ([]AlertHistory, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rule_id, rule_name, type, message, value, threshold, status, channels, created_at, resolved_at
		FROM alert_history ORDER BY created_at DESC LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var histories []AlertHistory
	for rows.Next() {
		h := AlertHistory{}
		var channelsJSON sql.NullString
		var resolvedAt sql.NullTime

		err := rows.Scan(
			&h.ID, &h.RuleID, &h.RuleName, &h.Type, &h.Message,
			&h.Value, &h.Threshold, &h.Status, &channelsJSON, &h.CreatedAt, &resolvedAt,
		)
		if err != nil {
			continue
		}

		if channelsJSON.Valid {
			json.Unmarshal([]byte(channelsJSON.String), &h.Channels)
		}
		if resolvedAt.Valid {
			h.ResolvedAt = &resolvedAt.Time
		}

		histories = append(histories, h)
	}

	return histories, nil
}

// ========== 发送通知 ==========

// SendNotification 发送通知
func (s *Service) SendNotification(ctx context.Context, channelID string, title, message string) error {
	channel, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}

	if !channel.Enabled {
		return fmt.Errorf("channel %s is disabled", channelID)
	}

	var sendErr error
	switch channel.Type {
	case ChannelTypeEmail:
		sendErr = s.sendEmail(channel, title, message)
	case ChannelTypeDingTalk:
		sendErr = s.sendDingTalk(channel, title, message)
	case ChannelTypeWeCom:
		sendErr = s.sendWeCom(channel, title, message)
	case ChannelTypeWebhook:
		sendErr = s.sendWebhook(channel, title, message)
	default:
		sendErr = fmt.Errorf("unsupported channel type: %s", channel.Type)
	}

	if sendErr != nil {
		s.logger.Error("failed to send notification",
			zap.String("channel", channelID),
			zap.Error(sendErr))
		return sendErr
	}

	// 更新最后使用时间
	now := time.Now()
	s.db.ExecContext(ctx, `UPDATE notification_channels SET last_used = ? WHERE id = ?`, now, channelID)

	return nil
}

func (s *Service) sendEmail(channel *NotificationChannel, title, message string) error {
	cfg := channel.Config
	if cfg.SMTPHost == "" || len(cfg.To) == 0 {
		return fmt.Errorf("email config incomplete")
	}

	auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: [GoPaw Alert] %s\r\n\r\n%s",
		cfg.From, strings.Join(cfg.To, ","), title, message)

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)
	return smtp.SendMail(addr, auth, cfg.From, cfg.To, []byte(msg))
}

func (s *Service) sendDingTalk(channel *NotificationChannel, title, message string) error {
	cfg := channel.Config
	if cfg.DingTalkWebhook == "" {
		return fmt.Errorf("dingtalk webhook not configured")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("[GoPaw Alert] %s\n\n%s", title, message),
		},
	}

	body, _ := json.Marshal(payload)

	webhookURL := cfg.DingTalkWebhook
	if cfg.DingTalkSecret != "" {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		sign := s.generateDingTalkSign(timestamp, cfg.DingTalkSecret)
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", webhookURL, timestamp, url.QueryEscape(sign))
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *Service) generateDingTalkSign(timestamp, secret string) string {
	stringToSign := timestamp + "\n" + secret
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(stringToSign))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *Service) sendWeCom(channel *NotificationChannel, title, message string) error {
	cfg := channel.Config
	if cfg.WeComWebhook == "" {
		return fmt.Errorf("wecom webhook not configured")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": fmt.Sprintf("[GoPaw Alert] %s\n\n%s", title, message),
		},
	}

	body, _ := json.Marshal(payload)

	resp, err := http.Post(cfg.WeComWebhook, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (s *Service) sendWebhook(channel *NotificationChannel, title, message string) error {
	cfg := channel.Config
	if cfg.WebhookURL == "" {
		return fmt.Errorf("webhook url not configured")
	}

	method := cfg.WebhookMethod
	if method == "" {
		method = "POST"
	}

	payload := map[string]interface{}{
		"title":   title,
		"message": message,
		"source":  "GoPaw",
		"time":    time.Now().Format(time.RFC3339),
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequest(method, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range cfg.WebhookHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// ========== 告警检查 ==========

// CheckRules 检查所有启用的告警规则
func (s *Service) CheckRules(ctx context.Context) error {
	rules, err := s.ListRules(ctx)
	if err != nil {
		return err
	}

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}

		// 检查告警条件
		triggered, value, err := s.checkCondition(ctx, rule)
		if err != nil {
			s.logger.Error("failed to check rule", zap.String("rule", rule.ID), zap.Error(err))
			continue
		}

		if triggered {
			// 创建告警历史
			history := &AlertHistory{
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Type:      rule.Type,
				Message:   fmt.Sprintf("Alert triggered: %s (value: %.2f, threshold: %.2f)", rule.Name, value, rule.Condition.Threshold),
				Value:     value,
				Threshold: rule.Condition.Threshold,
				Status:    "triggered",
				Channels:  rule.Channels,
			}
			s.CreateHistory(ctx, history)

			// 发送通知
			for _, channelID := range rule.Channels {
				go s.SendNotification(context.Background(), channelID, rule.Name, history.Message)
			}

			// 更新最后触发时间
			now := time.Now()
			s.db.ExecContext(ctx, `UPDATE alert_rules SET last_triggered = ? WHERE id = ?`, now, rule.ID)
		}
	}

	return nil
}

func (s *Service) checkCondition(ctx context.Context, rule AlertRule) (bool, float64, error) {
	// 简化实现：检查 traces 表中的指标
	cond := rule.Condition
	var value float64

	switch cond.Metric {
	case "latency":
		// 检查平均延迟
		err := s.db.QueryRowContext(ctx, `
			SELECT COALESCE(AVG(duration_ms), 0)
			FROM traces
			WHERE created_at >= datetime('now', '-5 minutes')
		`).Scan(&value)
		if err != nil {
			return false, 0, err
		}

	case "error_rate":
		// 检查错误率
		var total, errors int
		err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(*), SUM(CASE WHEN status = 'error' THEN 1 ELSE 0 END)
			FROM traces
			WHERE created_at >= datetime('now', '-5 minutes')
		`).Scan(&total, &errors)
		if err != nil {
			return false, 0, err
		}
		if total > 0 {
			value = float64(errors) / float64(total) * 100
		}

	case "token_usage":
		// 检查 Token 使用量
		err := s.db.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(total_tokens), 0)
			FROM traces
			WHERE created_at >= datetime('now', '-1 hour')
		`).Scan(&value)
		if err != nil {
			return false, 0, err
		}

	default:
		return false, 0, fmt.Errorf("unknown metric: %s", cond.Metric)
	}

	// 检查条件
	triggered := false
	switch cond.Operator {
	case ">":
		triggered = value > cond.Threshold
	case "<":
		triggered = value < cond.Threshold
	case ">=":
		triggered = value >= cond.Threshold
	case "<=":
		triggered = value <= cond.Threshold
	case "==":
		triggered = value == cond.Threshold
	case "!=":
		triggered = value != cond.Threshold
	}

	return triggered, value, nil
}