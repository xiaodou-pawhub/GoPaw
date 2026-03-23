// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"fmt"
	"sync"

	"github.com/gopaw/gopaw/internal/cron"
	"go.uber.org/zap"
)

// CronServiceInterface 定义 CronService 接口，避免循环依赖
type CronServiceInterface interface {
	AddJob(name, schedule, task, channel, targetID string) (*cron.CronJob, error)
	RemoveJob(id string) error
}

// TriggerManager 触发器管理器
type TriggerManager struct {
	cronService CronServiceInterface
	logger      *zap.Logger

	// 记录流程的 cron job ID
	// flowID -> cronJobID
	cronJobs map[string]string
	mu       sync.RWMutex
}

// NewTriggerManager 创建触发器管理器
func NewTriggerManager(cronService CronServiceInterface, logger *zap.Logger) *TriggerManager {
	return &TriggerManager{
		cronService: cronService,
		logger:      logger.Named("trigger_manager"),
		cronJobs:    make(map[string]string),
	}
}

// RegisterTrigger 注册流程触发器
func (tm *TriggerManager) RegisterTrigger(flow *Flow, executeFn func() error) error {
	if flow.Trigger == nil {
		return nil
	}

	switch flow.Trigger.Type {
	case "cron":
		return tm.registerCronTrigger(flow, executeFn)
	case "webhook":
		return tm.registerWebhookTrigger(flow)
	case "manual":
		// 手动触发，无需注册
		return nil
	case "event":
		// 事件触发，TODO: 实现事件订阅
		tm.logger.Warn("event trigger not implemented", zap.String("flow_id", flow.ID))
		return nil
	default:
		tm.logger.Warn("unknown trigger type",
			zap.String("flow_id", flow.ID),
			zap.String("type", flow.Trigger.Type),
		)
		return nil
	}
}

// UnregisterTrigger 注销流程触发器
func (tm *TriggerManager) UnregisterTrigger(flow *Flow) error {
	if flow.Trigger == nil {
		return nil
	}

	switch flow.Trigger.Type {
	case "cron":
		return tm.unregisterCronTrigger(flow)
	case "webhook":
		// Webhook 触发器不需要注销，只是流程状态变更
		return nil
	case "event":
		// TODO: 取消事件订阅
		return nil
	default:
		return nil
	}
}

// registerCronTrigger 注册 cron 触发器
func (tm *TriggerManager) registerCronTrigger(flow *Flow, executeFn func() error) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 获取 cron 表达式
	schedule, ok := flow.Trigger.Config["schedule"].(string)
	if !ok || schedule == "" {
		return fmt.Errorf("cron schedule is required for cron trigger")
	}

	// 获取执行任务描述
	task, _ := flow.Trigger.Config["task"].(string)
	if task == "" {
		task = fmt.Sprintf("执行流程: %s", flow.Name)
	}

	// 检查是否已注册
	if existingJobID, exists := tm.cronJobs[flow.ID]; exists {
		// 先移除旧的
		if tm.cronService != nil {
			tm.cronService.RemoveJob(existingJobID)
		}
	}

	// 注册新的 cron job
	if tm.cronService == nil {
		tm.logger.Warn("cron service not available", zap.String("flow_id", flow.ID))
		return nil
	}

	job, err := tm.cronService.AddJob(
		fmt.Sprintf("flow_%s", flow.ID), // job name
		schedule,                         // cron schedule
		task,                             // task description
		"",                               // channel (not used for flow)
		flow.ID,                          // targetID = flow ID
	)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	tm.cronJobs[flow.ID] = job.ID
	tm.logger.Info("registered cron trigger",
		zap.String("flow_id", flow.ID),
		zap.String("flow_name", flow.Name),
		zap.String("schedule", schedule),
		zap.String("job_id", job.ID),
	)

	return nil
}

// unregisterCronTrigger 注销 cron 触发器
func (tm *TriggerManager) unregisterCronTrigger(flow *Flow) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	jobID, exists := tm.cronJobs[flow.ID]
	if !exists {
		return nil
	}

	if tm.cronService != nil {
		if err := tm.cronService.RemoveJob(jobID); err != nil {
			tm.logger.Warn("failed to remove cron job",
				zap.String("flow_id", flow.ID),
				zap.String("job_id", jobID),
				zap.Error(err),
			)
		}
	}

	delete(tm.cronJobs, flow.ID)
	tm.logger.Info("unregistered cron trigger",
		zap.String("flow_id", flow.ID),
		zap.String("job_id", jobID),
	)

	return nil
}

// registerWebhookTrigger 注册 webhook 触发器
func (tm *TriggerManager) registerWebhookTrigger(flow *Flow) error {
	// Webhook 触发器不需要预先注册
	// 当外部 POST /api/webhooks/{flowID} 时，检查流程是否 active 即可
	// 这里只是记录日志
	tm.logger.Info("webhook trigger ready",
		zap.String("flow_id", flow.ID),
		zap.String("flow_name", flow.Name),
		zap.String("webhook_url", fmt.Sprintf("/api/webhooks/flow/%s", flow.ID)),
	)
	return nil
}

// GetCronJobID 获取流程的 cron job ID
func (tm *TriggerManager) GetCronJobID(flowID string) string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.cronJobs[flowID]
}