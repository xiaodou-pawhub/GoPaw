// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gopaw/gopaw/internal/queue"
	"go.uber.org/zap"
)

// Worker processes workflow step messages from the queue.
type Worker struct {
	mgr      *queue.Manager
	engine   *Engine
	logger   *zap.Logger
	stop     chan struct{}
	stopped  bool
}

// NewWorker creates a new workflow worker.
func NewWorker(mgr *queue.Manager, engine *Engine, logger *zap.Logger) *Worker {
	return &Worker{
		mgr:     mgr,
		engine:  engine,
		logger:  logger.Named("workflow_worker"),
		stop:    make(chan struct{}),
		stopped: false,
	}
}

// Start starts the worker.
func (w *Worker) Start() {
	w.logger.Info("workflow worker started")
	go w.run()
}

// Stop stops the worker.
func (w *Worker) Stop() {
	if w.stopped {
		return
	}
	w.stopped = true
	close(w.stop)
	w.logger.Info("workflow worker stopped")
}

// run is the main worker loop.
func (w *Worker) run() {
	ticker := time.NewTicker(1 * time.Second)
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
	if w.stopped {
		return
	}

	// Dequeue message from workflow queue
	msg, err := w.mgr.Dequeue("workflow", "workflow_worker")
	if err != nil {
		w.logger.Error("failed to dequeue message", zap.Error(err))
		return
	}

	if msg == nil {
		// No message available
		return
	}

	w.logger.Info("processing workflow message",
		zap.String("message_id", msg.ID),
		zap.String("type", msg.Type))

	// Process message based on type
	var processErr error
	switch msg.Type {
	case "execute_step":
		processErr = w.executeStep(msg)
	default:
		processErr = fmt.Errorf("unknown message type: %s", msg.Type)
	}

	if processErr != nil {
		w.logger.Error("message processing failed",
			zap.String("message_id", msg.ID),
			zap.Error(processErr))

		// Check if should retry
		if msg.Attempts < msg.MaxRetries {
			w.logger.Info("step will be retried",
				zap.String("message_id", msg.ID),
				zap.Int("attempts", msg.Attempts),
				zap.Int("max_retries", msg.MaxRetries))

			// Retry: reset status to pending
			if retryErr := w.mgr.Retry(msg.ID); retryErr != nil {
				w.logger.Error("failed to retry message", zap.Error(retryErr))
			}
		} else {
			// Max retries reached, mark as failed
			if failErr := w.mgr.Fail(msg.ID, processErr.Error()); failErr != nil {
				w.logger.Error("failed to mark message as failed", zap.Error(failErr))
			}
			// Note: Execution will detect failure via database polling
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

// executeStep executes a workflow step from a queue message.
func (w *Worker) executeStep(msg *queue.Message) error {
	// Parse payload
	payload := msg.Payload

	executionID, ok := payload["execution_id"].(string)
	if !ok {
		return fmt.Errorf("missing execution_id in payload")
	}

	stepID, ok := payload["step_id"].(string)
	if !ok {
		return fmt.Errorf("missing step_id in payload")
	}

	// Get execution
	execution, err := w.engine.GetExecution(executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	// Get workflow
	workflow, err := w.engine.Get(execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to get workflow: %w", err)
	}

	// Get step definition
	var stepDef *StepDef
	for i := range workflow.Definition.Steps {
		if workflow.Definition.Steps[i].ID == stepID {
			stepDef = &workflow.Definition.Steps[i]
			break
		}
	}
	if stepDef == nil {
		return fmt.Errorf("step %s not found in workflow", stepID)
	}

	// Create step execution record
	stepExec := &StepExecution{
		ID:          generateStepID(),
		ExecutionID: executionID,
		StepID:      stepID,
		AgentID:     stepDef.Agent,
		Status:      StepStatusRunning,
	}

	startedAt := time.Now().UTC()
	stepExec.StartedAt = &startedAt

	// Get input from payload
	if input, ok := payload["input"].(map[string]interface{}); ok {
		stepExec.Input = input
	} else {
		stepExec.Input = make(map[string]interface{})
	}

	if err := w.saveStepExecution(stepExec); err != nil {
		return fmt.Errorf("failed to save step execution: %w", err)
	}

	// Execute the step action
	output, execErr := w.executeStepAction(stepDef, stepExec.Input, payload)

	if execErr != nil {
		// Mark step as failed
		stepExec.Status = StepStatusFailed
		stepExec.Error = execErr.Error()
		completedAt := time.Now().UTC()
		stepExec.CompletedAt = &completedAt

		if err := w.saveStepExecution(stepExec); err != nil {
			w.logger.Error("failed to save failed step execution", zap.Error(err))
		}

		return execErr
	}

	// Complete step
	stepExec.Status = StepStatusCompleted
	stepExec.Output = output
	completedAt := time.Now().UTC()
	stepExec.CompletedAt = &completedAt

	if err := w.saveStepExecution(stepExec); err != nil {
		return fmt.Errorf("failed to save completed step execution: %w", err)
	}

	w.logger.Info("step completed",
		zap.String("execution_id", executionID),
		zap.String("step_id", stepID),
		zap.String("agent", stepDef.Agent))

	return nil
}

// executeStepAction executes the step action.
func (w *Worker) executeStepAction(step *StepDef, input map[string]interface{}, payload map[string]interface{}) (map[string]interface{}, error) {
	// Set timeout
	timeout := 300 // default 5 minutes
	if step.Timeout > 0 {
		timeout = step.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	switch step.Action {
	case "task":
		return w.executeTask(ctx, step, input, payload)
	case "notify":
		return w.executeNotify(ctx, step, input, payload)
	case "query":
		return w.executeQuery(ctx, step, input, payload)
	default:
		return nil, fmt.Errorf("unknown action: %s", step.Action)
	}
}

// executeTask executes a task action.
func (w *Worker) executeTask(ctx context.Context, step *StepDef, input map[string]interface{}, payload map[string]interface{}) (map[string]interface{}, error) {
	// Get execution ID from payload
	executionID, _ := payload["execution_id"].(string)

	content := ""
	if c, ok := input["description"].(string); ok {
		content = c
	} else {
		content = fmt.Sprintf("Execute step: %s", step.Name)
	}

	taskPayload := map[string]interface{}{
		"task_id":     fmt.Sprintf("%s_%s", executionID, step.ID),
		"description": content,
		"data":        input,
	}

	if priority, ok := input["priority"].(string); ok {
		taskPayload["priority"] = priority
	}

	// For now, return the task info as output
	// In a real implementation, this would send a message to an agent and wait for response
	output := map[string]interface{}{
		"task_id":     taskPayload["task_id"],
		"description": content,
		"executed":    true,
	}

	// Copy expected output fields
	for _, field := range step.Output {
		if val, ok := input[field]; ok {
			output[field] = val
		}
	}

	return output, nil
}

// executeNotify executes a notify action.
func (w *Worker) executeNotify(ctx context.Context, step *StepDef, input map[string]interface{}, payload map[string]interface{}) (map[string]interface{}, error) {
	content := ""
	if c, ok := input["message"].(string); ok {
		content = c
	} else {
		content = fmt.Sprintf("Notification from workflow step: %s", step.Name)
	}

	// For now, just return success
	// In a real implementation, this would send a notification
	return map[string]interface{}{
		"notified": true,
		"message":  content,
	}, nil
}

// executeQuery executes a query action.
func (w *Worker) executeQuery(ctx context.Context, step *StepDef, input map[string]interface{}, payload map[string]interface{}) (map[string]interface{}, error) {
	question := ""
	if q, ok := input["question"].(string); ok {
		question = q
	} else {
		question = fmt.Sprintf("Query from workflow step: %s", step.Name)
	}

	// For now, just return success
	// In a real implementation, this would send a query and wait for response
	return map[string]interface{}{
		"queried":  true,
		"question": question,
	}, nil
}

// saveStepExecution saves a step execution to the database.
func (w *Worker) saveStepExecution(stepExec *StepExecution) error {
	inputJSON, _ := json.Marshal(stepExec.Input)
	outputJSON, _ := json.Marshal(stepExec.Output)

	_, err := w.engine.db.Exec(`
		INSERT INTO workflow_steps (id, execution_id, step_id, agent_id, status, input, output, started_at, completed_at, error, retry_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			output = excluded.output,
			completed_at = excluded.completed_at,
			error = excluded.error,
			retry_count = excluded.retry_count
	`, stepExec.ID, stepExec.ExecutionID, stepExec.StepID, stepExec.AgentID, stepExec.Status, string(inputJSON), string(outputJSON), stepExec.StartedAt, stepExec.CompletedAt, stepExec.Error, stepExec.RetryCount)

	return err
}
