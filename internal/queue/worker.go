// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package queue

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Worker processes messages from a queue.
type Worker struct {
	id       string
	queue    string
	handler  Handler
	mgr      *Manager
	logger   *zap.Logger
	stop     chan struct{}
	stopped  bool
	mu       sync.Mutex
	interval time.Duration
}

// NewWorker creates a new worker.
func NewWorker(id, queue string, handler Handler, mgr *Manager, logger *zap.Logger) *Worker {
	return &Worker{
		id:       id,
		queue:    queue,
		handler:  handler,
		mgr:      mgr,
		logger:   logger.Named("worker").With(zap.String("worker_id", id), zap.String("queue", queue)),
		stop:     make(chan struct{}),
		interval: 1 * time.Second,
	}
}

// Start starts the worker.
func (w *Worker) Start() {
	w.logger.Info("worker started")
	go w.run()
}

// Stop stops the worker.
func (w *Worker) Stop() {
	w.mu.Lock()
	if w.stopped {
		w.mu.Unlock()
		return
	}
	w.stopped = true
	w.mu.Unlock()

	close(w.stop)
	w.logger.Info("worker stopped")
}

// IsStopped checks if the worker is stopped.
func (w *Worker) IsStopped() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stopped
}

// run is the main worker loop.
func (w *Worker) run() {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.processNext()
		case <-w.stop:
			return
		}
	}
}

// processNext processes the next available message.
func (w *Worker) processNext() {
	if w.IsStopped() {
		return
	}

	// Dequeue message
	msg, err := w.mgr.Dequeue(w.queue, w.id)
	if err != nil {
		w.logger.Error("failed to dequeue message", zap.Error(err))
		return
	}

	if msg == nil {
		// No message available
		return
	}

	w.logger.Info("processing message",
		zap.String("message_id", msg.ID),
		zap.String("type", msg.Type),
		zap.Int("attempt", msg.Attempts))

	// Process message
	err = w.handler(msg)

	if err != nil {
		w.logger.Error("message processing failed",
			zap.String("message_id", msg.ID),
			zap.Error(err))

		// Check if should retry
		if msg.Attempts < msg.MaxRetries {
			w.logger.Info("message will be retried",
				zap.String("message_id", msg.ID),
				zap.Int("attempts", msg.Attempts),
				zap.Int("max_retries", msg.MaxRetries))
			
			// Retry: reset status to pending
			if retryErr := w.mgr.Retry(msg.ID); retryErr != nil {
				w.logger.Error("failed to retry message", zap.Error(retryErr))
			}
		} else {
			// Max retries reached, mark as failed
			if failErr := w.mgr.Fail(msg.ID, err.Error()); failErr != nil {
				w.logger.Error("failed to mark message as failed", zap.Error(failErr))
			}
		}
	} else {
		w.logger.Info("message processed successfully",
			zap.String("message_id", msg.ID))

		// Mark as completed
		if err := w.mgr.Complete(msg.ID); err != nil {
			w.logger.Error("failed to mark message as completed", zap.Error(err))
		}
	}
}

// WorkerPool manages a pool of workers.
type WorkerPool struct {
	mgr     *Manager
	logger  *zap.Logger
	workers []*Worker
	mu      sync.RWMutex
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(mgr *Manager, logger *zap.Logger) *WorkerPool {
	return &WorkerPool{
		mgr:     mgr,
		logger:  logger.Named("worker_pool"),
		workers: make([]*Worker, 0),
	}
}

// AddWorker adds a worker to the pool.
func (p *WorkerPool) AddWorker(queue string, handler Handler) (*Worker, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	workerID := fmt.Sprintf("worker_%s_%d", queue, len(p.workers))
	worker := NewWorker(workerID, queue, handler, p.mgr, p.logger)
	p.workers = append(p.workers, worker)

	worker.Start()
	p.logger.Info("worker added to pool", zap.String("worker_id", workerID), zap.String("queue", queue))

	return worker, nil
}

// AddWorkers adds multiple workers to the pool.
func (p *WorkerPool) AddWorkers(queue string, handler Handler, count int) ([]*Worker, error) {
	workers := make([]*Worker, count)
	for i := 0; i < count; i++ {
		worker, err := p.AddWorker(queue, handler)
		if err != nil {
			return nil, err
		}
		workers[i] = worker
	}
	return workers, nil
}

// StopAll stops all workers in the pool.
func (p *WorkerPool) StopAll() {
	p.mu.RLock()
	workers := make([]*Worker, len(p.workers))
	copy(workers, p.workers)
	p.mu.RUnlock()

	for _, worker := range workers {
		worker.Stop()
	}

	p.logger.Info("all workers stopped")
}

// GetWorkerCount returns the number of workers.
func (p *WorkerPool) GetWorkerCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.workers)
}

// GetWorkersByQueue returns workers for a specific queue.
func (p *WorkerPool) GetWorkersByQueue(queue string) []*Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result []*Worker
	for _, worker := range p.workers {
		if worker.queue == queue {
			result = append(result, worker)
		}
	}
	return result
}
