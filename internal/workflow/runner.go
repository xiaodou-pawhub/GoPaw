// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/agent/message"
	"go.uber.org/zap"
)

// Runner executes a workflow instance.
type Runner struct {
	execution *Execution
	workflow  *Workflow
	engine    *Engine
	ctx       context.Context
	cancel    context.CancelFunc
	resolver  *Resolver
	evaluator *Evaluator
	stepOutputs map[string]*StepOutput
	mu        sync.RWMutex
}

// NewRunner creates a new workflow runner.
func NewRunner(execution *Execution, workflow *Workflow, engine *Engine, ctx context.Context) *Runner {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize resolver and evaluator
	resolver := NewResolver(execution.Variables, nil, make(map[string]*StepOutput))
	evaluator := NewEvaluator(resolver)

	return &Runner{
		execution:   execution,
		workflow:    workflow,
		engine:      engine,
		ctx:         ctx,
		cancel:      cancel,
		resolver:    resolver,
		evaluator:   evaluator,
		stepOutputs: make(map[string]*StepOutput),
	}
}

// Run executes the workflow.
func (r *Runner) Run() error {
	// Update execution status to running
	now := time.Now().UTC()
	r.execution.Status = ExecutionStatusRunning
	r.execution.StartedAt = &now

	if err := r.updateExecutionStatus(); err != nil {
		return err
	}

	r.engine.logger.Info("workflow execution started",
		zap.String("execution_id", r.execution.ID),
		zap.String("workflow_id", r.workflow.ID))

	// Execute steps
	if err := r.executeSteps(); err != nil {
		r.handleError(err)
		return err
	}

	// Mark as completed
	completedAt := time.Now().UTC()
	r.execution.Status = ExecutionStatusCompleted
	r.execution.CompletedAt = &completedAt

	if err := r.updateExecutionStatus(); err != nil {
		return err
	}

	r.engine.logger.Info("workflow execution completed",
		zap.String("execution_id", r.execution.ID))

	return nil
}

// Cancel cancels the workflow execution.
func (r *Runner) Cancel() {
	r.cancel()
}

// executeSteps executes all workflow steps.
func (r *Runner) executeSteps() error {
	completedSteps := make(map[string]bool)
	failedSteps := make(map[string]bool)
	var stepsMu sync.RWMutex
	
	// Prevent infinite loops - max iterations
	maxIterations := len(r.workflow.Definition.Steps) * 10
	iterations := 0

	for {
		iterations++
		if iterations > maxIterations {
			return fmt.Errorf("workflow execution exceeded maximum iterations, possible circular dependency")
		}
		
		// Check if cancelled
		if r.ctx.Err() != nil {
			return fmt.Errorf("execution cancelled")
		}

		// Get ready steps
		stepsMu.RLock()
		readySteps := r.workflow.Definition.GetReadySteps(completedSteps)
		stepsMu.RUnlock()
		
		if len(readySteps) == 0 {
			// Check if all steps are completed or failed
			stepsMu.RLock()
			allDone := true
			for _, step := range r.workflow.Definition.Steps {
				if !completedSteps[step.ID] && !failedSteps[step.ID] {
					// Check if step has unmet dependencies
					depFailed := false
					for _, dep := range step.DependsOn {
						if failedSteps[dep] {
							depFailed = true
							break
						}
					}
					if depFailed {
						stepsMu.RUnlock()
						stepsMu.Lock()
						failedSteps[step.ID] = true
						stepsMu.Unlock()
						stepsMu.RLock()
					} else {
						allDone = false
					}
				}
			}
			stepsMu.RUnlock()
			
			if allDone {
				break
			}
			// Wait a bit and retry
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Execute ready steps
		var wg sync.WaitGroup
		var stepErrors []error
		var stepMu sync.Mutex

		// Limit concurrent execution
		maxConcurrent := 3
		if r.workflow.Definition.ParallelConfig != nil && r.workflow.Definition.ParallelConfig.MaxConcurrent > 0 {
			maxConcurrent = r.workflow.Definition.ParallelConfig.MaxConcurrent
		}

		for i, step := range readySteps {
			if i >= maxConcurrent {
				break
			}

			wg.Add(1)
			go func(s StepDef) {
				defer wg.Done()

				if err := r.executeStep(&s); err != nil {
					stepMu.Lock()
					stepErrors = append(stepErrors, fmt.Errorf("step %s failed: %w", s.ID, err))
					stepsMu.Lock()
					failedSteps[s.ID] = true
					stepsMu.Unlock()
					stepMu.Unlock()
				} else {
					stepsMu.Lock()
					completedSteps[s.ID] = true
					stepsMu.Unlock()
				}
			}(step)
		}

		wg.Wait()

		if len(stepErrors) > 0 {
			// Check if we should continue or fail
			if !r.shouldContinueOnError() {
				return stepErrors[0]
			}
		}
	}

	return nil
}

// executeStep executes a single step.
func (r *Runner) executeStep(step *StepDef) error {
	// Check condition
	if step.Condition != "" {
		shouldExecute, err := r.evaluator.Evaluate(step.Condition)
		if err != nil {
			r.engine.logger.Warn("failed to evaluate condition",
				zap.String("step_id", step.ID),
				zap.Error(err))
		} else if !shouldExecute {
			// Skip this step
			r.engine.logger.Info("skipping step due to condition",
				zap.String("step_id", step.ID))
			r.recordStepResult(step, StepStatusSkipped, nil, nil)
			return nil
		}
	}

	// Create step execution record
	stepExec := &StepExecution{
		ID:          generateStepID(),
		ExecutionID: r.execution.ID,
		StepID:      step.ID,
		AgentID:     step.Agent,
		Status:      StepStatusRunning,
	}

	startedAt := time.Now().UTC()
	stepExec.StartedAt = &startedAt

	if err := r.saveStepExecution(stepExec); err != nil {
		return err
	}

	// Resolve input
	resolvedInput, err := r.resolver.Resolve(step.Input)
	if err != nil {
		return r.failStep(stepExec, fmt.Errorf("failed to resolve input: %w", err))
	}
	
	// Safe type conversion
	if resolvedInput == nil {
		stepExec.Input = make(map[string]interface{})
	} else if inputMap, ok := resolvedInput.(map[string]interface{}); ok {
		stepExec.Input = inputMap
	} else {
		return r.failStep(stepExec, fmt.Errorf("resolved input is not a map, got %T", resolvedInput))
	}

	// Execute the step action
	var output map[string]interface{}
	var execErr error

	for attempt := 0; attempt <= step.Retry; attempt++ {
		if attempt > 0 {
			r.engine.logger.Info("retrying step",
				zap.String("step_id", step.ID),
				zap.Int("attempt", attempt))
			time.Sleep(time.Duration(step.RetryDelay) * time.Second)
		}

		output, execErr = r.executeStepAction(step, stepExec.Input)
		if execErr == nil {
			break
		}
	}

	if execErr != nil {
		return r.failStep(stepExec, execErr)
	}

	// Complete step
	completedAt := time.Now().UTC()
	stepExec.Status = StepStatusCompleted
	stepExec.Output = output
	stepExec.CompletedAt = &completedAt

	if err := r.saveStepExecution(stepExec); err != nil {
		return err
	}

	// Update step outputs for resolver
	r.mu.Lock()
	r.stepOutputs[step.ID] = &StepOutput{
		StepID: step.ID,
		Status: StepStatusCompleted,
		Output: output,
	}
	r.resolver = NewResolver(r.execution.Variables, nil, r.stepOutputs)
	r.evaluator = NewEvaluator(r.resolver)
	r.mu.Unlock()

	r.engine.logger.Info("step completed",
		zap.String("step_id", step.ID),
		zap.String("agent", step.Agent))

	return nil
}

// executeStepAction executes the step action.
func (r *Runner) executeStepAction(step *StepDef, input map[string]interface{}) (map[string]interface{}, error) {
	// Set timeout
	timeout := 300 // default 5 minutes
	if step.Timeout > 0 {
		timeout = step.Timeout
	}

	ctx, cancel := context.WithTimeout(r.ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	switch step.Action {
	case "task":
		return r.executeTask(ctx, step, input)
	case "notify":
		return r.executeNotify(ctx, step, input)
	case "query":
		return r.executeQuery(ctx, step, input)
	default:
		return nil, fmt.Errorf("unknown action: %s", step.Action)
	}
}

// executeTask executes a task action.
func (r *Runner) executeTask(ctx context.Context, step *StepDef, input map[string]interface{}) (map[string]interface{}, error) {
	// Send task message to agent
	content := ""
	if c, ok := input["description"].(string); ok {
		content = c
	} else {
		content = fmt.Sprintf("Execute step: %s", step.Name)
	}

	payload := &message.TaskPayload{
		TaskID:      fmt.Sprintf("%s_%s", r.execution.ID, step.ID),
		Description: content,
		Data:        input,
	}

	if priority, ok := input["priority"].(string); ok {
		payload.Priority = priority
	}

	msg, err := r.engine.msgMgr.SendTask("workflow", step.Agent, content, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to send task: %w", err)
	}

	// Wait for response (simplified - in production, use proper async handling)
	// For now, return the message ID as output
	output := map[string]interface{}{
		"message_id": msg.ID,
		"task_id":    payload.TaskID,
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
func (r *Runner) executeNotify(ctx context.Context, step *StepDef, input map[string]interface{}) (map[string]interface{}, error) {
	content := ""
	if c, ok := input["message"].(string); ok {
		content = c
	} else {
		content = fmt.Sprintf("Notification from workflow: %s", r.workflow.Name)
	}

	event := "workflow_notification"
	if e, ok := input["event"].(string); ok {
		event = e
	}

	_, err := r.engine.msgMgr.SendNotify("workflow", step.Agent, event, input)
	if err != nil {
		return nil, fmt.Errorf("failed to send notify: %w", err)
	}

	return map[string]interface{}{
		"notified": true,
		"message":  content,
	}, nil
}

// executeQuery executes a query action.
func (r *Runner) executeQuery(ctx context.Context, step *StepDef, input map[string]interface{}) (map[string]interface{}, error) {
	question := ""
	if q, ok := input["question"].(string); ok {
		question = q
	} else {
		question = fmt.Sprintf("Query from workflow: %s", r.workflow.Name)
	}

	context := make(map[string]interface{})
	if c, ok := input["context"].(map[string]interface{}); ok {
		context = c
	}

	_, err := r.engine.msgMgr.SendQuery("workflow", step.Agent, question, context)
	if err != nil {
		return nil, fmt.Errorf("failed to send query: %w", err)
	}

	return map[string]interface{}{
		"queried":  true,
		"question": question,
	}, nil
}

// failStep marks a step as failed.
func (r *Runner) failStep(stepExec *StepExecution, err error) error {
	stepExec.Status = StepStatusFailed
	stepExec.Error = err.Error()
	completedAt := time.Now().UTC()
	stepExec.CompletedAt = &completedAt

	if err := r.saveStepExecution(stepExec); err != nil {
		return err
	}

	// Update step outputs
	r.mu.Lock()
	r.stepOutputs[stepExec.StepID] = &StepOutput{
		StepID: stepExec.StepID,
		Status: StepStatusFailed,
		Error:  err.Error(),
	}
	r.mu.Unlock()

	return err
}

// recordStepResult records a step result.
func (r *Runner) recordStepResult(step *StepDef, status StepStatus, output map[string]interface{}, err error) {
	stepExec := &StepExecution{
		ID:          generateStepID(),
		ExecutionID: r.execution.ID,
		StepID:      step.ID,
		AgentID:     step.Agent,
		Status:      status,
		Output:      output,
	}

	now := time.Now().UTC()
	stepExec.StartedAt = &now
	stepExec.CompletedAt = &now

	if err != nil {
		stepExec.Error = err.Error()
	}

	r.saveStepExecution(stepExec)

	// Update step outputs
	r.mu.Lock()
	r.stepOutputs[step.ID] = &StepOutput{
		StepID: step.ID,
		Status: status,
		Output: output,
		Error:  stepExec.Error,
	}
	r.mu.Unlock()
}

// saveStepExecution saves a step execution to the database.
func (r *Runner) saveStepExecution(stepExec *StepExecution) error {
	inputJSON, _ := json.Marshal(stepExec.Input)
	outputJSON, _ := json.Marshal(stepExec.Output)

	_, err := r.engine.db.Exec(`
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

// updateExecutionStatus updates the execution status in the database.
func (r *Runner) updateExecutionStatus() error {
	outputJSON, _ := json.Marshal(r.execution.Output)
	variablesJSON, _ := json.Marshal(r.execution.Variables)

	_, err := r.engine.db.Exec(`
		UPDATE workflow_executions
		SET status = ?, output = ?, variables = ?, started_at = ?, completed_at = ?, error = ?
		WHERE id = ?
	`, r.execution.Status, string(outputJSON), string(variablesJSON), r.execution.StartedAt, r.execution.CompletedAt, r.execution.Error, r.execution.ID)

	return err
}

// handleError handles workflow execution errors.
func (r *Runner) handleError(err error) {
	completedAt := time.Now().UTC()
	r.execution.Status = ExecutionStatusFailed
	r.execution.CompletedAt = &completedAt
	r.execution.Error = err.Error()

	r.updateExecutionStatus()

	// Execute error handlers if defined
	for _, handler := range r.workflow.Definition.ErrorHandlers {
		if handler.Condition == "any" || handler.Condition == "" {
			r.executeErrorHandler(&handler)
			break
		}
	}
}

// executeErrorHandler executes an error handler.
func (r *Runner) executeErrorHandler(handler *ErrorHandlerDef) {
	if handler.Agent == "" {
		return
	}

	content := handler.Message
	if content == "" {
		content = fmt.Sprintf("Workflow %s failed: %s", r.workflow.Name, r.execution.Error)
	}

	// Resolve variables in message
	resolved, _ := r.resolver.Resolve(content)
	content = fmt.Sprintf("%v", resolved)

	_, err := r.engine.msgMgr.SendNotify("workflow", handler.Agent, "workflow_failed", map[string]interface{}{
		"workflow_id":   r.workflow.ID,
		"execution_id":  r.execution.ID,
		"error":         r.execution.Error,
		"message":       content,
	})

	if err != nil {
		r.engine.logger.Error("failed to send error notification",
			zap.Error(err))
	}
}

// shouldContinueOnError checks if workflow should continue on step error.
func (r *Runner) shouldContinueOnError() bool {
	// For now, always fail on error
	// In the future, this could be configurable per workflow
	return false
}

// generateStepID generates a unique step execution ID.
func generateStepID() string {
	return "step_" + time.Now().Format("20060102150405") + "_" + randomString(6)
}
