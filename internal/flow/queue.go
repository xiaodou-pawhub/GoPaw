// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"container/heap"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ========== 任务队列数据模型 ==========

// TaskPriority 任务优先级
type TaskPriority int

const (
	PriorityLow    TaskPriority = 1
	PriorityNormal TaskPriority = 5
	PriorityHigh   TaskPriority = 10
	PriorityUrgent TaskPriority = 20
)

// TaskStatus 任务状态
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"    // 等待执行
	TaskStatusQueued    TaskStatus = "queued"     // 已入队
	TaskStatusRunning   TaskStatus = "running"    // 执行中
	TaskStatusCompleted TaskStatus = "completed"  // 已完成
	TaskStatusFailed    TaskStatus = "failed"     // 失败
	TaskStatusCancelled TaskStatus = "cancelled"  // 已取消
	TaskStatusRetry     TaskStatus = "retry"      // 等待重试
)

// Task 队列任务
type Task struct {
	ID           string                 `json:"id" db:"id"`
	Type         string                 `json:"type" db:"type"`               // 任务类型: flow_execute, webhook_callback, etc.
	Priority     TaskPriority           `json:"priority" db:"priority"`       // 优先级
	Status       TaskStatus             `json:"status" db:"status"`           // 状态
	Payload      map[string]interface{} `json:"payload" db:"payload"`         // 任务负载
	Result       map[string]interface{} `json:"result" db:"result"`           // 执行结果
	Error        string                 `json:"error,omitempty" db:"error"`   // 错误信息
	RetryCount   int                    `json:"retry_count" db:"retry_count"` // 重试次数
	MaxRetries   int                    `json:"max_retries" db:"max_retries"` // 最大重试次数
	ScheduledAt  *time.Time             `json:"scheduled_at" db:"scheduled_at"` // 计划执行时间（延迟任务）
	StartedAt    *time.Time             `json:"started_at" db:"started_at"`   // 开始时间
	CompletedAt  *time.Time             `json:"completed_at" db:"completed_at"` // 完成时间
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`   // 创建时间
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`   // 更新时间
	Timeout      int                    `json:"timeout" db:"timeout"`         // 超时时间（秒）
	WorkerID     string                 `json:"worker_id" db:"worker_id"`     // 执行的 Worker ID
	Metadata     map[string]string      `json:"metadata" db:"metadata"`       // 元数据
	index        int                    // 堆索引（内部使用）
}

// TaskType 任务类型
const (
	TaskTypeFlowExecute    = "flow_execute"
	TaskTypeWebhookCallback = "webhook_callback"
	TaskTypeSubflowExecute  = "subflow_execute"
	TaskTypeRetryExecution  = "retry_execution"
)

// ========== 优先级队列实现 ==========

// PriorityQueue 优先级队列（基于堆）
type PriorityQueue struct {
	items []*Task
	mu    sync.RWMutex
}

// Len 实现 heap.Interface
func (pq *PriorityQueue) Len() int {
	return len(pq.items)
}

// Less 实现 heap.Interface - 优先级高的排前面，相同优先级按创建时间排序
func (pq *PriorityQueue) Less(i, j int) bool {
	if pq.items[i].Priority != pq.items[j].Priority {
		return pq.items[i].Priority > pq.items[j].Priority
	}
	// 相同优先级，先创建的先执行
	return pq.items[i].CreatedAt.Before(pq.items[j].CreatedAt)
}

// Swap 实现 heap.Interface
func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
	pq.items[i].index = i
	pq.items[j].index = j
}

// Push 实现 heap.Interface
func (pq *PriorityQueue) Push(x interface{}) {
	n := len(pq.items)
	item := x.(*Task)
	item.index = n
	pq.items = append(pq.items, item)
}

// Pop 实现 heap.Interface
func (pq *PriorityQueue) Pop() interface{} {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	pq.items = old[0 : n-1]
	return item
}

// PushTask 线程安全地添加任务
func (pq *PriorityQueue) PushTask(task *Task) {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	heap.Push(pq, task)
}

// PopTask 线程安全地取出任务
func (pq *PriorityQueue) PopTask() *Task {
	pq.mu.Lock()
	defer pq.mu.Unlock()
	if pq.Len() == 0 {
		return nil
	}
	return heap.Pop(pq).(*Task)
}

// Peek 查看队首任务（不移除）
func (pq *PriorityQueue) Peek() *Task {
	pq.mu.RLock()
	defer pq.mu.RUnlock()
	if pq.Len() == 0 {
		return nil
	}
	return pq.items[0]
}

// ========== 任务队列服务 ==========

// TaskQueue 任务队列服务
type TaskQueue struct {
	db          *sql.DB
	logger      *zap.Logger
	pq          *PriorityQueue
	delayed     []*Task          // 延迟任务列表
	workers     map[string]*Worker
	workerPool  chan *Worker
	taskChan    chan *Task
	resultChan  chan *TaskResult
	maxWorkers  int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
	mu          sync.RWMutex
	handlers    map[string]TaskHandler
}

// TaskHandler 任务处理函数
type TaskHandler func(ctx context.Context, task *Task) (map[string]interface{}, error)

// TaskResult 任务执行结果
type TaskResult struct {
	TaskID string
	Result map[string]interface{}
	Error  error
}

// Worker 工作协程
type Worker struct {
	ID       string
	taskChan chan *Task
	quit     chan struct{}
	busy     bool
}

// NewTaskQueue 创建任务队列
func NewTaskQueue(db *sql.DB, maxWorkers int, logger *zap.Logger) *TaskQueue {
	ctx, cancel := context.WithCancel(context.Background())

	return &TaskQueue{
		db:         db,
		logger:     logger.Named("task_queue"),
		pq:         &PriorityQueue{items: make([]*Task, 0)},
		delayed:    make([]*Task, 0),
		workers:    make(map[string]*Worker),
		workerPool: make(chan *Worker, maxWorkers),
		taskChan:   make(chan *Task, 1000),
		resultChan: make(chan *TaskResult, 1000),
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
		handlers:   make(map[string]TaskHandler),
	}
}

// InitQueueSchema 初始化队列表
func InitQueueSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS task_queue (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			priority INTEGER DEFAULT 5,
			status TEXT DEFAULT 'pending',
			payload TEXT,
			result TEXT,
			error TEXT,
			retry_count INTEGER DEFAULT 0,
			max_retries INTEGER DEFAULT 3,
			scheduled_at DATETIME,
			started_at DATETIME,
			completed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			timeout INTEGER DEFAULT 300,
			worker_id TEXT,
			metadata TEXT
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_task_status ON task_queue(status)`,
		`CREATE INDEX IF NOT EXISTS idx_task_priority ON task_queue(priority DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_task_scheduled ON task_queue(scheduled_at)`,
		`CREATE INDEX IF NOT EXISTS idx_task_type ON task_queue(type)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

// RegisterHandler 注册任务处理器
func (q *TaskQueue) RegisterHandler(taskType string, handler TaskHandler) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.handlers[taskType] = handler
}

// Start 启动任务队列
func (q *TaskQueue) Start() error {
	// 初始化数据库表
	if err := InitQueueSchema(q.db); err != nil {
		return fmt.Errorf("failed to init queue schema: %w", err)
	}

	// 恢复未完成的任务
	if err := q.restoreTasks(); err != nil {
		q.logger.Warn("failed to restore tasks", zap.Error(err))
	}

	// 启动 Worker
	for i := 0; i < q.maxWorkers; i++ {
		worker := q.createWorker(i)
		q.workers[worker.ID] = worker
		go worker.run(q)
	}

	// 启动调度器
	go q.scheduler()
	go q.resultProcessor()

	q.logger.Info("task queue started",
		zap.Int("max_workers", q.maxWorkers))

	return nil
}

// Stop 停止任务队列
func (q *TaskQueue) Stop() {
	q.cancel()
	q.wg.Wait()

	close(q.taskChan)
	close(q.resultChan)

	q.logger.Info("task queue stopped")
}

// createWorker 创建 Worker
func (q *TaskQueue) createWorker(index int) *Worker {
	return &Worker{
		ID:       fmt.Sprintf("worker_%d", index),
		taskChan: make(chan *Task, 1),
		quit:     make(chan struct{}),
	}
}

// Worker 运行循环
func (w *Worker) run(q *TaskQueue) {
	for {
		select {
		case task := <-w.taskChan:
			w.busy = true
			q.processTask(w, task)
			w.busy = false
			// 将自己放回池中
			select {
			case q.workerPool <- w:
			default:
			}
		case <-w.quit:
			return
		case <-q.ctx.Done():
			return
		}
	}
}

// scheduler 调度器
func (q *TaskQueue) scheduler() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 检查延迟任务
			q.checkDelayedTasks()

			// 分配任务给空闲 Worker
			q.dispatchTasks()

		case <-q.ctx.Done():
			return
		}
	}
}

// resultProcessor 结果处理器
func (q *TaskQueue) resultProcessor() {
	for {
		select {
		case result := <-q.resultChan:
			q.handleResult(result)
		case <-q.ctx.Done():
			return
		}
	}
}

// checkDelayedTasks 检查延迟任务
func (q *TaskQueue) checkDelayedTasks() {
	q.mu.Lock()
	defer q.mu.Unlock()

	now := time.Now()
	var ready []*Task
	var stillDelayed []*Task

	for _, task := range q.delayed {
		if task.ScheduledAt == nil || task.ScheduledAt.Before(now) {
			ready = append(ready, task)
		} else {
			stillDelayed = append(stillDelayed, task)
		}
	}

	q.delayed = stillDelayed

	// 将就绪的任务加入优先级队列
	for _, task := range ready {
		task.Status = TaskStatusQueued
		q.pq.PushTask(task)
		q.updateTask(task)
	}
}

// dispatchTasks 分配任务
func (q *TaskQueue) dispatchTasks() {
	for {
		// 获取空闲 Worker
		select {
		case worker := <-q.workerPool:
			// 获取任务
			task := q.pq.PopTask()
			if task == nil {
				// 没有任务，把 Worker 放回
				select {
				case q.workerPool <- worker:
				default:
				}
				return
			}

			// 分配任务给 Worker
			task.Status = TaskStatusRunning
			task.WorkerID = worker.ID
			now := time.Now()
			task.StartedAt = &now
			q.updateTask(task)

			worker.taskChan <- task

		default:
			// 没有空闲 Worker
			return
		}
	}
}

// processTask 处理任务
func (q *TaskQueue) processTask(worker *Worker, task *Task) {
	ctx, cancel := context.WithTimeout(q.ctx, time.Duration(task.Timeout)*time.Second)
	defer cancel()

	result := &TaskResult{TaskID: task.ID}

	// 获取处理器
	q.mu.RLock()
	handler, ok := q.handlers[task.Type]
	q.mu.RUnlock()

	if !ok {
		result.Error = fmt.Errorf("no handler for task type: %s", task.Type)
		q.resultChan <- result
		return
	}

	// 执行任务
	output, err := handler(ctx, task)
	if err != nil {
		result.Error = err
	} else {
		result.Result = output
	}

	q.resultChan <- result
}

// handleResult 处理结果
func (q *TaskQueue) handleResult(result *TaskResult) {
	// 获取任务
	task, err := q.getTask(result.TaskID)
	if err != nil {
		q.logger.Warn("task not found for result", zap.String("task_id", result.TaskID))
		return
	}

	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	if result.Error != nil {
		task.Error = result.Error.Error()

		// 检查是否需要重试
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Status = TaskStatusRetry
			task.Error = "" // 清除错误，等待重试

			// 重新入队
			q.mu.Lock()
			q.pq.PushTask(task)
			q.mu.Unlock()

			q.logger.Info("task scheduled for retry",
				zap.String("task_id", task.ID),
				zap.Int("retry_count", task.RetryCount),
				zap.Int("max_retries", task.MaxRetries))
		} else {
			task.Status = TaskStatusFailed
			q.logger.Warn("task failed after max retries",
				zap.String("task_id", task.ID),
				zap.Int("retries", task.RetryCount))
		}
	} else {
		task.Status = TaskStatusCompleted
		task.Result = result.Result
	}

	q.updateTask(task)
}

// ========== 公共 API ==========

// Enqueue 入队任务
func (q *TaskQueue) Enqueue(taskType string, payload map[string]interface{}, opts ...TaskOption) (*Task, error) {
	task := &Task{
		ID:          generateID(),
		Type:        taskType,
		Priority:    PriorityNormal,
		Status:      TaskStatusPending,
		Payload:     payload,
		MaxRetries:  3,
		Timeout:     300,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Metadata:    make(map[string]string),
	}

	// 应用选项
	for _, opt := range opts {
		opt(task)
	}

	// 保存到数据库
	if err := q.saveTask(task); err != nil {
		return nil, fmt.Errorf("failed to save task: %w", err)
	}

	// 加入队列
	q.mu.Lock()
	if task.ScheduledAt != nil && task.ScheduledAt.After(time.Now()) {
		// 延迟任务
		q.delayed = append(q.delayed, task)
	} else {
		// 立即执行
		task.Status = TaskStatusQueued
		q.pq.PushTask(task)
	}
	q.mu.Unlock()

	q.logger.Debug("task enqueued",
		zap.String("task_id", task.ID),
		zap.String("type", taskType),
		zap.Int("priority", int(task.Priority)))

	return task, nil
}

// EnqueueWithPriority 带优先级入队
func (q *TaskQueue) EnqueueWithPriority(taskType string, payload map[string]interface{}, priority TaskPriority) (*Task, error) {
	return q.Enqueue(taskType, payload, WithPriority(priority))
}

// EnqueueDelayed 延迟入队
func (q *TaskQueue) EnqueueDelayed(taskType string, payload map[string]interface{}, delay time.Duration) (*Task, error) {
	scheduledAt := time.Now().Add(delay)
	return q.Enqueue(taskType, payload, WithScheduledAt(scheduledAt))
}

// GetTask 获取任务
func (q *TaskQueue) GetTask(taskID string) (*Task, error) {
	return q.getTask(taskID)
}

// CancelTask 取消任务
func (q *TaskQueue) CancelTask(taskID string) error {
	task, err := q.getTask(taskID)
	if err != nil {
		return err
	}

	if task.Status == TaskStatusRunning {
		return fmt.Errorf("cannot cancel running task")
	}

	task.Status = TaskStatusCancelled
	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	return q.updateTask(task)
}

// ListTasks 列出任务
func (q *TaskQueue) ListTasks(status TaskStatus, limit int) ([]*Task, error) {
	query := "SELECT id, type, priority, status, payload, result, error, retry_count, max_retries, scheduled_at, started_at, completed_at, created_at, updated_at, timeout, worker_id, metadata FROM task_queue"
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = ?"
		args = append(args, string(status))
	}
	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := q.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	return q.scanTasks(rows)
}

// GetQueueStats 获取队列统计
func (q *TaskQueue) GetQueueStats() *QueueStats {
	q.mu.RLock()
	defer q.mu.RUnlock()

	stats := &QueueStats{
		Pending:   q.pq.Len(),
		Delayed:   len(q.delayed),
		Workers:   len(q.workers),
	}

	// 统计忙碌的 Worker
	for _, w := range q.workers {
		if w.busy {
			stats.BusyWorkers++
		}
	}

	return stats
}

// QueueStats 队列统计
type QueueStats struct {
	Pending      int `json:"pending"`
	Delayed      int `json:"delayed"`
	Workers      int `json:"workers"`
	BusyWorkers  int `json:"busy_workers"`
}

// ========== 任务选项 ==========

// TaskOption 任务选项
type TaskOption func(*Task)

// WithPriority 设置优先级
func WithPriority(priority TaskPriority) TaskOption {
	return func(t *Task) {
		t.Priority = priority
	}
}

// WithScheduledAt 设置计划执行时间
func WithScheduledAt(scheduledAt time.Time) TaskOption {
	return func(t *Task) {
		t.ScheduledAt = &scheduledAt
	}
}

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(maxRetries int) TaskOption {
	return func(t *Task) {
		t.MaxRetries = maxRetries
	}
}

// WithTimeout 设置超时时间
func WithTimeout(timeout int) TaskOption {
	return func(t *Task) {
		t.Timeout = timeout
	}
}

// WithMetadata 设置元数据
func WithMetadata(metadata map[string]string) TaskOption {
	return func(t *Task) {
		t.Metadata = metadata
	}
}

// ========== 数据库操作 ==========

func (q *TaskQueue) saveTask(task *Task) error {
	payloadJSON, _ := json.Marshal(task.Payload)
	metadataJSON, _ := json.Marshal(task.Metadata)

	_, err := q.db.Exec(`
		INSERT INTO task_queue (id, type, priority, status, payload, retry_count, max_retries, scheduled_at, timeout, metadata, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.Type, task.Priority, task.Status, string(payloadJSON), task.RetryCount, task.MaxRetries, task.ScheduledAt, task.Timeout, string(metadataJSON), task.CreatedAt, task.UpdatedAt)

	return err
}

func (q *TaskQueue) updateTask(task *Task) error {
	resultJSON, _ := json.Marshal(task.Result)
	metadataJSON, _ := json.Marshal(task.Metadata)

	_, err := q.db.Exec(`
		UPDATE task_queue SET status = ?, result = ?, error = ?, retry_count = ?, started_at = ?, completed_at = ?, updated_at = ?, worker_id = ?, metadata = ?
		WHERE id = ?
	`, task.Status, string(resultJSON), task.Error, task.RetryCount, task.StartedAt, task.CompletedAt, task.UpdatedAt, task.WorkerID, string(metadataJSON), task.ID)

	return err
}

func (q *TaskQueue) getTask(taskID string) (*Task, error) {
	task := &Task{}
	var payloadJSON, resultJSON, metadataJSON sql.NullString
	var scheduledAt, startedAt, completedAt sql.NullTime
	var workerID sql.NullString

	err := q.db.QueryRow(`
		SELECT id, type, priority, status, payload, result, error, retry_count, max_retries, scheduled_at, started_at, completed_at, created_at, updated_at, timeout, worker_id, metadata
		FROM task_queue WHERE id = ?
	`, taskID).Scan(&task.ID, &task.Type, &task.Priority, &task.Status, &payloadJSON, &resultJSON, &task.Error, &task.RetryCount, &task.MaxRetries, &scheduledAt, &startedAt, &completedAt, &task.CreatedAt, &task.UpdatedAt, &task.Timeout, &workerID, &metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	if payloadJSON.Valid {
		json.Unmarshal([]byte(payloadJSON.String), &task.Payload)
	}
	if resultJSON.Valid {
		json.Unmarshal([]byte(resultJSON.String), &task.Result)
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &task.Metadata)
	}
	if scheduledAt.Valid {
		task.ScheduledAt = &scheduledAt.Time
	}
	if startedAt.Valid {
		task.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		task.CompletedAt = &completedAt.Time
	}
	if workerID.Valid {
		task.WorkerID = workerID.String
	}

	return task, nil
}

func (q *TaskQueue) restoreTasks() error {
	// 恢复 queued 和 running 状态的任务
	rows, err := q.db.Query(`
		SELECT id, type, priority, status, payload, result, error, retry_count, max_retries, scheduled_at, started_at, completed_at, created_at, updated_at, timeout, worker_id, metadata
		FROM task_queue WHERE status IN ('queued', 'running', 'pending', 'retry')
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tasks, err := q.scanTasks(rows)
	if err != nil {
		return err
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	for _, task := range tasks {
		if task.ScheduledAt != nil && task.ScheduledAt.After(time.Now()) {
			q.delayed = append(q.delayed, task)
		} else {
			task.Status = TaskStatusQueued
			q.pq.PushTask(task)
		}
	}

	q.logger.Info("restored tasks", zap.Int("count", len(tasks)))
	return nil
}

func (q *TaskQueue) scanTasks(rows *sql.Rows) ([]*Task, error) {
	var tasks []*Task

	for rows.Next() {
		task := &Task{}
		var payloadJSON, resultJSON, metadataJSON sql.NullString
		var scheduledAt, startedAt, completedAt sql.NullTime
		var workerID sql.NullString

		err := rows.Scan(&task.ID, &task.Type, &task.Priority, &task.Status, &payloadJSON, &resultJSON, &task.Error, &task.RetryCount, &task.MaxRetries, &scheduledAt, &startedAt, &completedAt, &task.CreatedAt, &task.UpdatedAt, &task.Timeout, &workerID, &metadataJSON)
		if err != nil {
			continue
		}

		if payloadJSON.Valid {
			json.Unmarshal([]byte(payloadJSON.String), &task.Payload)
		}
		if resultJSON.Valid {
			json.Unmarshal([]byte(resultJSON.String), &task.Result)
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &task.Metadata)
		}
		if scheduledAt.Valid {
			task.ScheduledAt = &scheduledAt.Time
		}
		if startedAt.Valid {
			task.StartedAt = &startedAt.Time
		}
		if completedAt.Valid {
			task.CompletedAt = &completedAt.Time
		}
		if workerID.Valid {
			task.WorkerID = workerID.String
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}