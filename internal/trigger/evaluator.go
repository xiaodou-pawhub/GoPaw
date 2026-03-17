// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trigger

import (
	"time"

	"github.com/robfig/cron/v3"
)

// Evaluator evaluates triggers.
type Evaluator struct {
	cronParser cron.Parser
}

// NewEvaluator creates a new evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{
		cronParser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// ShouldFire evaluates if a trigger should fire at the given time.
func (e *Evaluator) ShouldFire(trigger *Trigger, now time.Time) bool {
	switch trigger.Type {
	case "cron":
		return e.evaluateCron(trigger, now)
	case "webhook":
		return e.evaluateWebhook(trigger, now)
	case "message":
		return e.evaluateMessage(trigger, now)
	default:
		return false
	}
}

// evaluateCron evaluates a cron trigger.
func (e *Evaluator) evaluateCron(trigger *Trigger, now time.Time) bool {
	config := trigger.Config.(*CronConfig)
	if config.Expression == "" {
		return false
	}

	schedule, err := e.cronParser.Parse(config.Expression)
	if err != nil {
		return false
	}

	// Get the next scheduled time
	next := schedule.Next(now.Add(-time.Minute))

	// Check if it should have fired in the last minute
	return next.Before(now) || next.Equal(now)
}

// evaluateWebhook evaluates a webhook trigger.
// Webhook triggers never fire automatically - they must be triggered via HTTP.
func (e *Evaluator) evaluateWebhook(trigger *Trigger, now time.Time) bool {
	return false
}

// evaluateMessage evaluates a message trigger.
// Message triggers never fire automatically - they must be triggered via message API.
func (e *Evaluator) evaluateMessage(trigger *Trigger, now time.Time) bool {
	return false
}

// ValidateCron validates a cron expression.
func (e *Evaluator) ValidateCron(expression string) error {
	_, err := e.cronParser.Parse(expression)
	return err
}

// NextFireTime returns the next time a cron trigger should fire.
func (e *Evaluator) NextFireTime(trigger *Trigger) (*time.Time, error) {
	if trigger.Type != "cron" {
		return nil, nil
	}

	config := trigger.Config.(*CronConfig)
	if config.Expression == "" {
		return nil, nil
	}

	schedule, err := e.cronParser.Parse(config.Expression)
	if err != nil {
		return nil, err
	}

	next := schedule.Next(time.Now())
	return &next, nil
}

// DescribeCron returns a human-readable description of a cron expression.
func (e *Evaluator) DescribeCron(expression string) string {
	// Simple descriptions for common patterns
	descriptions := map[string]string{
		"@yearly":   "每年执行一次",
		"@annually": "每年执行一次",
		"@monthly":  "每月执行一次",
		"@weekly":   "每周执行一次",
		"@daily":    "每天执行一次",
		"@midnight": "每天午夜执行",
		"@hourly":   "每小时执行一次",
		"* * * * *": "每分钟执行",
	}

	if desc, ok := descriptions[expression]; ok {
		return desc
	}

	// Try to parse and describe
	schedule, err := e.cronParser.Parse(expression)
	if err != nil {
		return "自定义定时"
	}

	// Get next run time
	next := schedule.Next(time.Now())
	return "下次执行: " + next.Format("yyyy-MM-dd HH:mm")
}
