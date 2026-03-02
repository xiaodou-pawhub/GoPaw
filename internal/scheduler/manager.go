// Package scheduler manages cron-based scheduled tasks.
package scheduler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gopaw/gopaw/pkg/types"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// ErrInvalidCronExpr is returned when a cron expression is invalid.
// 中文：Cron 表达式无效时返回的错误
// English: Error returned when a cron expression is invalid
var ErrInvalidCronExpr = errors.New("scheduler: invalid cron expression")

// AgentProcessor is the function signature used to trigger the agent from a scheduled job.
type AgentProcessor func(ctx context.Context, req *types.Request) (*types.Response, error)

// ChannelSender is the function signature used to deliver a response back to a channel.
type ChannelSender func(msg *types.Message) error

// Manager wraps robfig/cron and persists job metadata in SQLite.
type Manager struct {
	cron    *cron.Cron
	store   *JobStore
	process AgentProcessor
	send    ChannelSender
	logger  *zap.Logger

	// entryMap maps job ID → cron.EntryID for removal support.
	entryMap map[string]cron.EntryID
}

// NewManager creates a Manager.
func NewManager(store *JobStore, process AgentProcessor, send ChannelSender, logger *zap.Logger) *Manager {
	c := cron.New(cron.WithSeconds())
	return &Manager{
		cron:     c,
		store:    store,
		process:  process,
		send:     send,
		logger:   logger,
		entryMap: make(map[string]cron.EntryID),
	}
}

// Start loads all enabled jobs from the store and starts the cron engine.
func (m *Manager) Start(ctx context.Context) error {
	jobs, err := m.store.List()
	if err != nil {
		return fmt.Errorf("scheduler: load jobs: %w", err)
	}

	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		if err := m.scheduleJob(ctx, &job); err != nil {
			m.logger.Warn("scheduler: failed to schedule job",
				zap.String("id", job.ID), zap.Error(err))
		}
	}

	m.cron.Start()
	m.logger.Info("scheduler started", zap.Int("jobs", len(m.entryMap)))
	return nil
}

// Stop gracefully shuts down the cron engine.
func (m *Manager) Stop() {
	m.cron.Stop()
}

// AddJob persists a new job and schedules it immediately.
func (m *Manager) AddJob(ctx context.Context, job *CronJob) (string, error) {
	id, err := m.store.Create(job)
	if err != nil {
		return "", fmt.Errorf("scheduler: add job: %w", err)
	}
	job.ID = id
	if job.Enabled {
		if err := m.scheduleJob(ctx, job); err != nil {
			return id, fmt.Errorf("scheduler: schedule new job: %w", err)
		}
	}
	return id, nil
}

// RemoveJob cancels a scheduled job and deletes it from the store.
func (m *Manager) RemoveJob(id string) error {
	if entryID, ok := m.entryMap[id]; ok {
		m.cron.Remove(entryID)
		delete(m.entryMap, id)
	}
	return m.store.Delete(id)
}

// TriggerJob executes a job immediately regardless of its schedule.
func (m *Manager) TriggerJob(ctx context.Context, id string) error {
	job, err := m.store.Get(id)
	if err != nil {
		return err
	}
	go m.runJob(ctx, job)
	return nil
}

// ListJobs returns all jobs from the store.
func (m *Manager) ListJobs() ([]CronJob, error) {
	return m.store.List()
}

// UpdateJobRequest contains the updatable fields for a CronJob.
// 中文：包含 CronJob 可更新字段的请求结构
// English: Request structure containing updatable fields for CronJob
type UpdateJobRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	CronExpr    *string `json:"cron_expr"`
	Channel     *string `json:"channel"`
	SessionID   *string `json:"session_id"`
	Prompt      *string `json:"prompt"`
	Enabled     *bool   `json:"enabled"`
	ActiveFrom  *string `json:"active_from"`
	ActiveUntil *string `json:"active_until"`
}

// UpdateJob updates the specified fields of an existing job.
// 中文：更新现有任务的指定字段
// English: Update specified fields of an existing job
func (m *Manager) UpdateJob(ctx context.Context, id string, req UpdateJobRequest) error {
	// 中文：从数据库读取当前值
	// English: Get current job from database
	job, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("scheduler: get job %q: %w", id, err)
	}

	// 中文：合并更新字段（只更新非 nil 字段）
	// English: Merge updated fields (only update non-nil fields)
	if req.Name != nil {
		job.Name = *req.Name
	}
	if req.Description != nil {
		job.Description = *req.Description
	}
	if req.CronExpr != nil {
		// 中文：校验 cron 表达式（使用临时实例，避免副作用）
		// English: Validate cron expression (use temporary instance to avoid side effects)
		tmpCron := cron.New()
		if _, err := tmpCron.AddFunc(*req.CronExpr, func() {}); err != nil {
			return fmt.Errorf("%w: %q: %v", ErrInvalidCronExpr, *req.CronExpr, err)
		}
		job.CronExpr = *req.CronExpr
	}
	if req.Channel != nil {
		job.Channel = *req.Channel
	}
	if req.SessionID != nil {
		job.SessionID = *req.SessionID
	}
	if req.Prompt != nil {
		job.Prompt = *req.Prompt
	}
	if req.Enabled != nil {
		job.Enabled = *req.Enabled
	}
	if req.ActiveFrom != nil {
		job.ActiveFrom = *req.ActiveFrom
	}
	if req.ActiveUntil != nil {
		job.ActiveUntil = *req.ActiveUntil
	}

	// 中文：写回数据库
	// English: Write back to database
	if err := m.store.Update(job); err != nil {
		return fmt.Errorf("scheduler: update job: %w", err)
	}

	// 中文：如果 cron 表达式变了或启用状态变了，需要重新调度
	// English: Reschedule if cron expr or enabled state changed
	if req.CronExpr != nil || req.Enabled != nil {
		// 中文：先从 cron 中移除
		// English: Remove from cron first
		if entryID, ok := m.entryMap[id]; ok {
			m.cron.Remove(entryID)
			delete(m.entryMap, id)
		}
		// 中文：如果启用则重新添加
		// English: Re-add if enabled
		if job.Enabled {
			if err := m.scheduleJob(ctx, job); err != nil {
				return fmt.Errorf("scheduler: reschedule job: %w", err)
			}
		}
	}

	return nil
}

// scheduleJob registers a job with the cron engine.
func (m *Manager) scheduleJob(ctx context.Context, job *CronJob) error {
	jobCopy := *job
	entryID, err := m.cron.AddFunc(job.CronExpr, func() {
		m.runJob(ctx, &jobCopy)
	})
	if err != nil {
		return fmt.Errorf("invalid cron expression %q: %w", job.CronExpr, err)
	}
	m.entryMap[job.ID] = entryID
	m.logger.Info("job scheduled",
		zap.String("id", job.ID),
		zap.String("name", job.Name),
		zap.String("expr", job.CronExpr),
	)
	return nil
}

// runJob executes a single job, respecting the active time window.
func (m *Manager) runJob(ctx context.Context, job *CronJob) {
	if !m.withinActiveWindow(job) {
		m.logger.Debug("job outside active window, skipping",
			zap.String("id", job.ID))
		return
	}

	m.logger.Info("scheduler: running job",
		zap.String("id", job.ID), zap.String("name", job.Name))

	req := &types.Request{
		SessionID: job.SessionID,
		Channel:   job.Channel,
		Content:   job.Prompt,
		MsgType:   types.MsgTypeText,
		Metadata:  map[string]string{"scheduler_job_id": job.ID},
	}

	resp, err := m.process(ctx, req)
	if err != nil {
		m.logger.Error("scheduler: agent processing failed",
			zap.String("job_id", job.ID), zap.Error(err))
		return
	}

	msg := &types.Message{
		Channel:  job.Channel,
		Content:  resp.Content,
		MsgType:  resp.MsgType,
		Metadata: map[string]string{"scheduler_job_id": job.ID},
	}
	if err := m.send(msg); err != nil {
		m.logger.Error("scheduler: send response failed",
			zap.String("job_id", job.ID), zap.Error(err))
	}

	nextEntry := m.cron.Entry(m.entryMap[job.ID])
	_ = m.store.UpdateLastRun(job.ID, time.Now(), nextEntry.Next)
}

// withinActiveWindow checks whether the current time falls within the job's active window.
// If either boundary is empty, the check is skipped.
func (m *Manager) withinActiveWindow(job *CronJob) bool {
	if job.ActiveFrom == "" && job.ActiveUntil == "" {
		return true
	}
	now := time.Now()
	current := now.Format("15:04")
	if job.ActiveFrom != "" && current < job.ActiveFrom {
		return false
	}
	if job.ActiveUntil != "" && current > job.ActiveUntil {
		return false
	}
	return true
}
