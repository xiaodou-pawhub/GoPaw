package cron

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// JobHandler is the function that executes the agent task.
// It returns the output content and an error if execution failed.
type JobHandler func(job *CronJob) (string, error)

type CronService struct {
	mu        sync.RWMutex
	storePath string
	store     *CronStore
	history   map[string][]CronRun // jobID -> runs (ring buffer)
	cron      *cron.Cron
	onJob     JobHandler
	logger    *zap.Logger
}

func NewCronService(workspaceDir string, logger *zap.Logger) *CronService {
	storePath := filepath.Join(workspaceDir, "cron", "jobs.json")
	
	c := cron.New(cron.WithSeconds())

	return &CronService{
		storePath: storePath,
		store:     &CronStore{Version: 1, Jobs: []CronJob{}},
		history:   make(map[string][]CronRun),
		cron:      c,
		logger:    logger.Named("cron"),
	}
}

func (s *CronService) SetHandler(handler JobHandler) {
	s.onJob = handler
}

func (s *CronService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.loadStore(); err != nil {
		s.logger.Warn("failed to load cron store, starting fresh", zap.Error(err))
	}

	for i := range s.store.Jobs {
		job := &s.store.Jobs[i]
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				s.logger.Error("failed to schedule loaded job", zap.String("id", job.ID), zap.Error(err))
			}
		}
	}

	s.cron.Start()
	s.logger.Info("cron service started", zap.Int("jobs", len(s.store.Jobs)))
	return nil
}

func (s *CronService) Stop() {
	s.cron.Stop()
}

func (s *CronService) AddJob(name, schedule, task, channel, targetID string) (*CronJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(schedule); err != nil {
		return nil, fmt.Errorf("invalid cron schedule: %w", err)
	}

	job := CronJob{
		ID:        uuid.New().String(),
		Name:      name,
		Schedule:  schedule,
		CronExpr:  schedule,
		Task:      task,
		Prompt:    task,
		Channel:   channel,
		TargetID:  targetID,
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	if err := s.scheduleJob(&job); err != nil {
		return nil, err
	}

	s.store.Jobs = append(s.store.Jobs, job)
	if err := s.saveStore(); err != nil {
		s.cron.Remove(cron.EntryID(job.EntryID))
		return nil, err
	}

	return &job, nil
}

func (s *CronService) RemoveJob(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, job := range s.store.Jobs {
		if job.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("job not found: %s", id)
	}

	job := &s.store.Jobs[idx]
	if job.EntryID != 0 {
		s.cron.Remove(cron.EntryID(job.EntryID))
	}

	s.store.Jobs = append(s.store.Jobs[:idx], s.store.Jobs[idx+1:]...)
	delete(s.history, id)
	
	return s.saveStore()
}

func (s *CronService) ListJobs() []CronJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	result := make([]CronJob, len(s.store.Jobs))
	copy(result, s.store.Jobs)
	return result
}

func (s *CronService) GetRuns(jobID string) []CronRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs, ok := s.history[jobID]
	if !ok {
		return []CronRun{}
	}

	result := make([]CronRun, len(runs))
	copy(result, runs)
	return result
}

func (s *CronService) UpdateJob(id, name, schedule, task, targetID string, enabled *bool) (*CronJob, error) {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(schedule); err != nil {
		return nil, fmt.Errorf("invalid cron schedule: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	job := &s.store.Jobs[idx]

	// Remove old cron entry before rescheduling
	if job.EntryID != 0 {
		s.cron.Remove(cron.EntryID(job.EntryID))
		job.EntryID = 0
	}

	job.Name = name
	job.Schedule = schedule
	job.CronExpr = schedule
	job.Task = task
	job.Prompt = task
	job.TargetID = targetID
	if enabled != nil {
		job.Enabled = *enabled
	}

	if job.Enabled {
		if err := s.scheduleJob(job); err != nil {
			return nil, err
		}
	}

	if err := s.saveStore(); err != nil {
		return nil, err
	}

	result := *job
	return &result, nil
}

func (s *CronService) TriggerJob(id string) (*CronRun, error) {
	s.mu.RLock()
	var found bool
	for _, job := range s.store.Jobs {
		if job.ID == id {
			found = true
			break
		}
	}
	s.mu.RUnlock()

	if !found {
		return nil, fmt.Errorf("job not found: %s", id)
	}

	run := &CronRun{
		ID:          uuid.New().String(),
		JobID:       id,
		TriggeredAt: time.Now(),
		Status:      "running",
	}

	go s.runJobWrapper(id)

	return run, nil
}

// Internal helpers

func (s *CronService) scheduleJob(job *CronJob) error {
	entryID, err := s.cron.AddFunc(job.Schedule, func() {
		if s.onJob != nil {
			s.runJobWrapper(job.ID)
		}
	})
	if err != nil {
		return err
	}
	job.EntryID = int(entryID)
	return nil
}

func (s *CronService) runJobWrapper(jobID string) {
	s.mu.RLock()
	var job *CronJob
	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == jobID {
			job = &s.store.Jobs[i]
			break
		}
	}
	s.mu.RUnlock()

	if job == nil {
		return
	}

	start := time.Now()
	output, err := s.onJob(job)
	duration := time.Since(start)

	run := CronRun{
		ID:          uuid.New().String(),
		JobID:       jobID,
		TriggeredAt: start,
		FinishedAt:  start.Add(duration),
		Status:      "success",
		Output:      output,
	}
	
	if err != nil {
		run.Status = "error"
		run.Error = err.Error()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Update job state
	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == jobID {
			s.store.Jobs[i].LastRunAt = &start
			if err != nil {
				s.store.Jobs[i].LastResult = "error"
			} else {
				s.store.Jobs[i].LastResult = "success"
			}
			_ = s.saveStore()
			break
		}
	}

	// Append history (Ring buffer logic: max 10 items)
	hist := s.history[jobID]
	if len(hist) >= 10 {
		copy(hist, hist[1:])
		hist[len(hist)-1] = run
	} else {
		hist = append(hist, run)
	}
	s.history[jobID] = hist
}

func (s *CronService) loadStore() error {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, s.store)
}

func (s *CronService) saveStore() error {
	data, err := json.MarshalIndent(s.store, "", "  ")
	if err != nil {
		return err
	}
	
	dir := filepath.Dir(s.storePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpPath := s.storePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, s.storePath)
}
