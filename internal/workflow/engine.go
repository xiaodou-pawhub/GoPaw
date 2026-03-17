// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package workflow

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gopaw/gopaw/internal/agent"
	"github.com/gopaw/gopaw/internal/agent/message"
	"go.uber.org/zap"
)

// Engine manages workflow definitions and executions.
type Engine struct {
	db          *sql.DB
	msgMgr      *message.Manager
	agentRouter *agent.Router
	logger      *zap.Logger
	runners     map[string]*Runner
	mu          sync.RWMutex
}

// NewEngine creates a new workflow engine.
func NewEngine(db *sql.DB, msgMgr *message.Manager, agentRouter *agent.Router, logger *zap.Logger) (*Engine, error) {
	e := &Engine{
		db:          db,
		msgMgr:      msgMgr,
		agentRouter: agentRouter,
		logger:      logger.Named("workflow_engine"),
		runners:     make(map[string]*Runner),
	}

	if err := e.initSchema(); err != nil {
		return nil, err
	}

	return e, nil
}

// initSchema creates the database tables.
func (e *Engine) initSchema() error {
	schema := `
CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    definition TEXT NOT NULL,
    version TEXT,
    status TEXT DEFAULT 'draft',
    trigger_type TEXT,
    trigger_config TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_executions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    status TEXT DEFAULT 'pending',
    input TEXT,
    output TEXT,
    variables TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT,
    triggered_by TEXT
);

CREATE TABLE IF NOT EXISTS workflow_steps (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    agent_id TEXT,
    status TEXT DEFAULT 'pending',
    input TEXT,
    output TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error TEXT,
    retry_count INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_workflows_status ON workflows(status);
CREATE INDEX IF NOT EXISTS idx_executions_workflow ON workflow_executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON workflow_executions(status);
CREATE INDEX IF NOT EXISTS idx_steps_execution ON workflow_steps(execution_id);
CREATE INDEX IF NOT EXISTS idx_steps_status ON workflow_steps(status);
`
	_, err := e.db.Exec(schema)
	return err
}

// Create creates a new workflow.
func (e *Engine) Create(workflow *Workflow) error {
	now := time.Now().UTC()
	workflow.CreatedAt = now
	workflow.UpdatedAt = now

	if workflow.Status == "" {
		workflow.Status = WorkflowStatusDraft
	}

	defJSON, err := json.Marshal(workflow.Definition)
	if err != nil {
		return fmt.Errorf("failed to marshal definition: %w", err)
	}

	var triggerType, triggerConfig interface{}
	if workflow.Definition.Trigger != nil {
		triggerType = workflow.Definition.Trigger.Type
		configJSON, _ := json.Marshal(workflow.Definition.Trigger.Config)
		triggerConfig = string(configJSON)
	}

	_, err = e.db.Exec(`
		INSERT INTO workflows (id, name, description, definition, version, status, trigger_type, trigger_config, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, workflow.ID, workflow.Name, workflow.Description, string(defJSON), workflow.Version, workflow.Status, triggerType, triggerConfig, workflow.CreatedAt, workflow.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to insert workflow: %w", err)
	}

	e.logger.Info("workflow created", zap.String("id", workflow.ID), zap.String("name", workflow.Name))
	return nil
}

// Update updates an existing workflow.
func (e *Engine) Update(id string, workflow *Workflow) error {
	workflow.UpdatedAt = time.Now().UTC()

	defJSON, err := json.Marshal(workflow.Definition)
	if err != nil {
		return fmt.Errorf("failed to marshal definition: %w", err)
	}

	var triggerType, triggerConfig interface{}
	if workflow.Definition.Trigger != nil {
		triggerType = workflow.Definition.Trigger.Type
		configJSON, _ := json.Marshal(workflow.Definition.Trigger.Config)
		triggerConfig = string(configJSON)
	}

	_, err = e.db.Exec(`
		UPDATE workflows
		SET name = ?, description = ?, definition = ?, version = ?, status = ?, trigger_type = ?, trigger_config = ?, updated_at = ?
		WHERE id = ?
	`, workflow.Name, workflow.Description, string(defJSON), workflow.Version, workflow.Status, triggerType, triggerConfig, workflow.UpdatedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	e.logger.Info("workflow updated", zap.String("id", id))
	return nil
}

// Delete deletes a workflow.
func (e *Engine) Delete(id string) error {
	_, err := e.db.Exec(`DELETE FROM workflows WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	e.logger.Info("workflow deleted", zap.String("id", id))
	return nil
}

// Get retrieves a workflow by ID.
func (e *Engine) Get(id string) (*Workflow, error) {
	workflow := &Workflow{}
	var defJSON string
	var triggerType, triggerConfig sql.NullString

	err := e.db.QueryRow(`
		SELECT id, name, description, definition, version, status, trigger_type, trigger_config, created_at, updated_at
		FROM workflows
		WHERE id = ?
	`, id).Scan(&workflow.ID, &workflow.Name, &workflow.Description, &defJSON, &workflow.Version, &workflow.Status, &triggerType, &triggerConfig, &workflow.CreatedAt, &workflow.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow not found: %s", id)
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(defJSON), &workflow.Definition); err != nil {
		return nil, fmt.Errorf("failed to unmarshal definition: %w", err)
	}

	return workflow, nil
}

// List returns all workflows.
func (e *Engine) List() ([]*Workflow, error) {
	rows, err := e.db.Query(`
		SELECT id, name, description, definition, version, status, trigger_type, trigger_config, created_at, updated_at
		FROM workflows
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*Workflow
	for rows.Next() {
		workflow := &Workflow{}
		var defJSON string
		var triggerType, triggerConfig sql.NullString

		err := rows.Scan(&workflow.ID, &workflow.Name, &workflow.Description, &defJSON, &workflow.Version, &workflow.Status, &triggerType, &triggerConfig, &workflow.CreatedAt, &workflow.UpdatedAt)
		if err != nil {
			e.logger.Warn("failed to scan workflow", zap.Error(err))
			continue
		}

		if err := json.Unmarshal([]byte(defJSON), &workflow.Definition); err != nil {
			e.logger.Warn("failed to unmarshal definition", zap.Error(err))
			continue
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workflows, nil
}

// ListByStatus returns workflows by status.
func (e *Engine) ListByStatus(status WorkflowStatus) ([]*Workflow, error) {
	rows, err := e.db.Query(`
		SELECT id, name, description, definition, version, status, trigger_type, trigger_config, created_at, updated_at
		FROM workflows
		WHERE status = ?
		ORDER BY updated_at DESC
	`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workflows []*Workflow
	for rows.Next() {
		workflow := &Workflow{}
		var defJSON string
		var triggerType, triggerConfig sql.NullString

		err := rows.Scan(&workflow.ID, &workflow.Name, &workflow.Description, &defJSON, &workflow.Version, &workflow.Status, &triggerType, &triggerConfig, &workflow.CreatedAt, &workflow.UpdatedAt)
		if err != nil {
			e.logger.Warn("failed to scan workflow", zap.Error(err))
			continue
		}

		if err := json.Unmarshal([]byte(defJSON), &workflow.Definition); err != nil {
			e.logger.Warn("failed to unmarshal definition", zap.Error(err))
			continue
		}

		workflows = append(workflows, workflow)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return workflows, nil
}

// Execute starts a workflow execution.
func (e *Engine) Execute(workflowID string, input map[string]interface{}, triggeredBy string) (*Execution, error) {
	workflow, err := e.Get(workflowID)
	if err != nil {
		return nil, err
	}

	if workflow.Status != WorkflowStatusActive {
		return nil, fmt.Errorf("workflow is not active: %s", workflow.Status)
	}

	// Create execution
	execution := &Execution{
		ID:          generateExecutionID(),
		WorkflowID:  workflowID,
		Status:      ExecutionStatusPending,
		Input:       input,
		Variables:   make(map[string]interface{}),
		TriggeredBy: triggeredBy,
	}

	// Initialize variables with defaults
	for _, v := range workflow.Definition.Variables {
		if v.Required && input[v.Name] == nil && v.Default == nil {
			return nil, fmt.Errorf("required variable not provided: %s", v.Name)
		}
		if input[v.Name] != nil {
			execution.Variables[v.Name] = input[v.Name]
		} else if v.Default != nil {
			execution.Variables[v.Name] = v.Default
		}
	}

	// Save execution
	variablesJSON, _ := json.Marshal(execution.Variables)
	inputJSON, _ := json.Marshal(execution.Input)

	_, err = e.db.Exec(`
		INSERT INTO workflow_executions (id, workflow_id, status, input, variables, triggered_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, execution.ID, execution.WorkflowID, execution.Status, string(inputJSON), string(variablesJSON), execution.TriggeredBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	// Start execution in background
	go e.runExecution(execution, workflow)

	e.logger.Info("workflow execution started",
		zap.String("execution_id", execution.ID),
		zap.String("workflow_id", workflowID))

	return execution, nil
}

// runExecution runs a workflow execution.
func (e *Engine) runExecution(execution *Execution, workflow *Workflow) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := NewRunner(execution, workflow, e, ctx)

	e.mu.Lock()
	e.runners[execution.ID] = runner
	e.mu.Unlock()

	defer func() {
		e.mu.Lock()
		delete(e.runners, execution.ID)
		e.mu.Unlock()
	}()

	if err := runner.Run(); err != nil {
		e.logger.Error("workflow execution failed",
			zap.String("execution_id", execution.ID),
			zap.Error(err))
	}
}

// Cancel cancels a running execution.
func (e *Engine) Cancel(executionID string) error {
	e.mu.RLock()
	runner, ok := e.runners[executionID]
	e.mu.RUnlock()

	if ok {
		runner.Cancel()
	}

	_, err := e.db.Exec(`
		UPDATE workflow_executions
		SET status = ?, completed_at = ?
		WHERE id = ? AND status IN ('pending', 'running')
	`, ExecutionStatusCancelled, time.Now().UTC(), executionID)
	if err != nil {
		return fmt.Errorf("failed to cancel execution: %w", err)
	}

	e.logger.Info("workflow execution cancelled", zap.String("execution_id", executionID))
	return nil
}

// GetExecution retrieves an execution by ID.
func (e *Engine) GetExecution(id string) (*Execution, error) {
	execution := &Execution{}
	var inputJSON, outputJSON, variablesJSON string

	err := e.db.QueryRow(`
		SELECT id, workflow_id, status, input, output, variables, started_at, completed_at, error, triggered_by
		FROM workflow_executions
		WHERE id = ?
	`, id).Scan(&execution.ID, &execution.WorkflowID, &execution.Status, &inputJSON, &outputJSON, &variablesJSON, &execution.StartedAt, &execution.CompletedAt, &execution.Error, &execution.TriggeredBy)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("execution not found: %s", id)
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(inputJSON), &execution.Input); err != nil {
		e.logger.Warn("failed to unmarshal input", zap.Error(err))
	}
	if err := json.Unmarshal([]byte(outputJSON), &execution.Output); err != nil {
		e.logger.Warn("failed to unmarshal output", zap.Error(err))
	}
	if err := json.Unmarshal([]byte(variablesJSON), &execution.Variables); err != nil {
		e.logger.Warn("failed to unmarshal variables", zap.Error(err))
	}

	// Load steps
	steps, err := e.GetExecutionSteps(id)
	if err != nil {
		e.logger.Warn("failed to load execution steps", zap.Error(err))
	} else {
		execution.Steps = steps
	}

	return execution, nil
}

// GetExecutionSteps retrieves all steps for an execution.
func (e *Engine) GetExecutionSteps(executionID string) ([]*StepExecution, error) {
	rows, err := e.db.Query(`
		SELECT id, execution_id, step_id, agent_id, status, input, output, started_at, completed_at, error, retry_count
		FROM workflow_steps
		WHERE execution_id = ?
		ORDER BY started_at ASC
	`, executionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var steps []*StepExecution
	for rows.Next() {
		step := &StepExecution{}
		var inputJSON, outputJSON string

		err := rows.Scan(&step.ID, &step.ExecutionID, &step.StepID, &step.AgentID, &step.Status, &inputJSON, &outputJSON, &step.StartedAt, &step.CompletedAt, &step.Error, &step.RetryCount)
		if err != nil {
			e.logger.Warn("failed to scan step", zap.Error(err))
			continue
		}

		if err := json.Unmarshal([]byte(inputJSON), &step.Input); err != nil {
			e.logger.Warn("failed to unmarshal step input", zap.Error(err))
		}
		if err := json.Unmarshal([]byte(outputJSON), &step.Output); err != nil {
			e.logger.Warn("failed to unmarshal step output", zap.Error(err))
		}

		steps = append(steps, step)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return steps, nil
}

// ListExecutions returns executions for a workflow.
func (e *Engine) ListExecutions(workflowID string, limit int) ([]*Execution, error) {
	rows, err := e.db.Query(`
		SELECT id, workflow_id, status, input, output, variables, started_at, completed_at, error, triggered_by
		FROM workflow_executions
		WHERE workflow_id = ?
		ORDER BY started_at DESC
		LIMIT ?
	`, workflowID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var executions []*Execution
	for rows.Next() {
		execution := &Execution{}
		var inputJSON, outputJSON, variablesJSON string

		err := rows.Scan(&execution.ID, &execution.WorkflowID, &execution.Status, &inputJSON, &outputJSON, &variablesJSON, &execution.StartedAt, &execution.CompletedAt, &execution.Error, &execution.TriggeredBy)
		if err != nil {
			e.logger.Warn("failed to scan execution", zap.Error(err))
			continue
		}

		if err := json.Unmarshal([]byte(inputJSON), &execution.Input); err != nil {
			e.logger.Warn("failed to unmarshal input", zap.Error(err))
		}
		if err := json.Unmarshal([]byte(outputJSON), &execution.Output); err != nil {
			e.logger.Warn("failed to unmarshal output", zap.Error(err))
		}
		if err := json.Unmarshal([]byte(variablesJSON), &execution.Variables); err != nil {
			e.logger.Warn("failed to unmarshal variables", zap.Error(err))
		}

		executions = append(executions, execution)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return executions, nil
}

// GetStats returns execution statistics for a workflow.
func (e *Engine) GetStats(workflowID string) (*ExecutionStats, error) {
	stats := &ExecutionStats{}

	// Total executions
	err := e.db.QueryRow(`SELECT COUNT(*) FROM workflow_executions WHERE workflow_id = ?`, workflowID).Scan(&stats.TotalExecutions)
	if err != nil {
		return nil, err
	}

	// Status counts
	statuses := []ExecutionStatus{ExecutionStatusCompleted, ExecutionStatusFailed, ExecutionStatusRunning, ExecutionStatusPending, ExecutionStatusCancelled}
	for _, status := range statuses {
		var count int
		err := e.db.QueryRow(`SELECT COUNT(*) FROM workflow_executions WHERE workflow_id = ? AND status = ?`, workflowID, status).Scan(&count)
		if err != nil {
			continue
		}
		switch status {
		case ExecutionStatusCompleted:
			stats.CompletedCount = count
		case ExecutionStatusFailed:
			stats.FailedCount = count
		case ExecutionStatusRunning:
			stats.RunningCount = count
		case ExecutionStatusPending:
			stats.PendingCount = count
		case ExecutionStatusCancelled:
			stats.CancelledCount = count
		}
	}

	// Average execution time
	var avgTime sql.NullInt64
	err = e.db.QueryRow(`
		SELECT AVG(strftime('%s', completed_at) - strftime('%s', started_at))
		FROM workflow_executions
		WHERE workflow_id = ? AND status = 'completed' AND started_at IS NOT NULL AND completed_at IS NOT NULL
	`, workflowID).Scan(&avgTime)
	if err == nil && avgTime.Valid {
		stats.AvgExecutionTime = int(avgTime.Int64)
	}

	return stats, nil
}

// generateExecutionID generates a unique execution ID.
func generateExecutionID() string {
	return "exec_" + time.Now().Format("20060102150405") + "_" + randomString(8)
}

// randomString generates a random string of given length.
func randomString(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}
