// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trace

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Manager manages execution traces with async writing.
type Manager struct {
	storage   *Storage
	logger    *zap.Logger
	writeCh   chan *writeRequest
	stopCh    chan struct{}
	wg        sync.WaitGroup
	retention time.Duration
}

// writeRequest represents a request to write data to storage.
type writeRequest struct {
	trace *Trace
	step  *Step
}

// NewManager creates a new trace manager.
func NewManager(dbPath string, retentionDays int, logger *zap.Logger) (*Manager, error) {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create trace directory: %w", err)
	}

	storage, err := NewStorage(dbPath)
	if err != nil {
		return nil, err
	}

	m := &Manager{
		storage:   storage,
		logger:    logger.Named("trace"),
		writeCh:   make(chan *writeRequest, 100), // Buffer up to 100 writes
		stopCh:    make(chan struct{}),
		retention: time.Duration(retentionDays) * 24 * time.Hour,
	}

	// Start async writer
	m.wg.Add(1)
	go m.asyncWriter()

	// Start cleanup routine
	if retentionDays > 0 {
		m.wg.Add(1)
		go m.cleanupRoutine()
	}

	m.logger.Info("trace manager initialized",
		zap.String("db_path", dbPath),
		zap.Int("retention_days", retentionDays),
	)

	return m, nil
}

// Close stops the manager and cleans up resources.
func (m *Manager) Close() error {
	close(m.stopCh)
	m.wg.Wait()
	return m.storage.Close()
}

// StartTrace starts a new trace.
func (m *Manager) StartTrace(sessionID string) *Trace {
	trace := NewTrace(sessionID)

	// Send to async writer (non-blocking)
	select {
	case m.writeCh <- &writeRequest{trace: trace}:
	default:
		m.logger.Warn("trace write channel full, dropping trace start",
			zap.String("trace_id", trace.ID),
		)
	}

	m.logger.Debug("trace started",
		zap.String("trace_id", trace.ID),
		zap.String("session_id", sessionID),
	)

	return trace
}

// AddStep adds a step to a trace.
func (m *Manager) AddStep(traceID string, step *Step) {
	step.TraceID = traceID

	// Send to async writer (non-blocking)
	select {
	case m.writeCh <- &writeRequest{step: step}:
	default:
		m.logger.Warn("trace write channel full, dropping step",
			zap.String("trace_id", traceID),
			zap.String("step_id", step.ID),
		)
	}

	m.logger.Debug("step added",
		zap.String("trace_id", traceID),
		zap.String("step_id", step.ID),
		zap.String("step_type", string(step.StepType)),
	)
}

// EndTrace ends a trace.
func (m *Manager) EndTrace(trace *Trace) {
	trace.End()

	// Send to async writer (non-blocking)
	select {
	case m.writeCh <- &writeRequest{trace: trace}:
	default:
		m.logger.Warn("trace write channel full, dropping trace end",
			zap.String("trace_id", trace.ID),
		)
	}

	m.logger.Debug("trace ended",
		zap.String("trace_id", trace.ID),
		zap.String("status", string(trace.Status)),
		zap.Duration("duration", trace.Duration()),
	)
}

// EndTraceWithError ends a trace with an error.
func (m *Manager) EndTraceWithError(trace *Trace, err error) {
	trace.EndWithError(err)

	// Send to async writer (non-blocking)
	select {
	case m.writeCh <- &writeRequest{trace: trace}:
	default:
		m.logger.Warn("trace write channel full, dropping trace end with error",
			zap.String("trace_id", trace.ID),
		)
	}

	m.logger.Debug("trace ended with error",
		zap.String("trace_id", trace.ID),
		zap.String("error", err.Error()),
	)
}

// asyncWriter processes write requests asynchronously.
func (m *Manager) asyncWriter() {
	defer m.wg.Done()

	for {
		select {
		case req := <-m.writeCh:
			if req.trace != nil {
				if err := m.storage.SaveTrace(req.trace); err != nil {
					m.logger.Error("failed to save trace",
						zap.String("trace_id", req.trace.ID),
						zap.Error(err),
					)
				}
			}
			if req.step != nil {
				if err := m.storage.SaveStep(req.step); err != nil {
					m.logger.Error("failed to save step",
						zap.String("step_id", req.step.ID),
						zap.Error(err),
					)
				}
			}
		case <-m.stopCh:
			// Process remaining items in channel
			for {
				select {
				case req := <-m.writeCh:
					if req.trace != nil {
						if err := m.storage.SaveTrace(req.trace); err != nil {
							m.logger.Error("failed to save trace on shutdown",
								zap.String("trace_id", req.trace.ID),
								zap.Error(err),
							)
						}
					}
					if req.step != nil {
						if err := m.storage.SaveStep(req.step); err != nil {
							m.logger.Error("failed to save step on shutdown",
								zap.String("step_id", req.step.ID),
								zap.Error(err),
							)
						}
					}
				default:
					return
				}
			}
		}
	}
}

// cleanupRoutine periodically cleans up old traces.
func (m *Manager) cleanupRoutine() {
	defer m.wg.Done()

	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	// Run cleanup immediately on start
	m.cleanup()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopCh:
			return
		}
	}
}

// cleanup deletes traces older than retention period.
func (m *Manager) cleanup() {
	if m.retention <= 0 {
		return
	}

	before := time.Now().UTC().Add(-m.retention)
	deleted, err := m.storage.Cleanup(before)
	if err != nil {
		m.logger.Error("failed to cleanup old traces",
			zap.Time("before", before),
			zap.Error(err),
		)
		return
	}

	if deleted > 0 {
		m.logger.Info("cleaned up old traces",
			zap.Int64("deleted", deleted),
			zap.Time("before", before),
		)
	}
}

// GetTrace retrieves a trace by ID.
func (m *Manager) GetTrace(id string) (*Trace, error) {
	return m.storage.GetTrace(id)
}

// QueryTraces queries traces with filters.
func (m *Manager) QueryTraces(opts QueryOptions) ([]*Trace, error) {
	return m.storage.QueryTraces(opts)
}

// GetDBPath returns the database path.
func (m *Manager) GetDBPath() string {
	return m.storage.DBPath()
}
