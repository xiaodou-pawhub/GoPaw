// Package scheduler manages cron-based scheduled tasks.
package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/gopaw/gopaw/pkg/types"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

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
