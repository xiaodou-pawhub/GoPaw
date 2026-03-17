// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package workflow

import (
	"encoding/json"
	"time"
)

// WorkflowStatus represents the status of a workflow.
type WorkflowStatus string

const (
	WorkflowStatusDraft     WorkflowStatus = "draft"
	WorkflowStatusActive    WorkflowStatus = "active"
	WorkflowStatusDisabled  WorkflowStatus = "disabled"
)

// ExecutionStatus represents the status of a workflow execution.
type ExecutionStatus string

const (
	ExecutionStatusPending    ExecutionStatus = "pending"
	ExecutionStatusRunning    ExecutionStatus = "running"
	ExecutionStatusCompleted  ExecutionStatus = "completed"
	ExecutionStatusFailed     ExecutionStatus = "failed"
	ExecutionStatusCancelled  ExecutionStatus = "cancelled"
)

// StepStatus represents the status of a workflow step.
type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusRunning    StepStatus = "running"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusFailed     StepStatus = "failed"
	StepStatusSkipped    StepStatus = "skipped"
	StepStatusCancelled  StepStatus = "cancelled"
)

// Workflow represents a workflow definition.
type Workflow struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Definition  *WorkflowDef   `json:"definition"`
	Version     string         `json:"version"`
	Status      WorkflowStatus `json:"status"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// WorkflowDef represents the workflow definition structure.
type WorkflowDef struct {
	Variables      []VariableDef      `json:"variables,omitempty"`
	Steps          []StepDef          `json:"steps"`
	Trigger        *TriggerConfig     `json:"trigger,omitempty"`
	ErrorHandlers  []ErrorHandlerDef  `json:"error_handlers,omitempty"`
	ParallelConfig *ParallelConfig    `json:"parallel,omitempty"`
}

// VariableDef represents a workflow variable definition.
type VariableDef struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"` // string, number, boolean, object, array
	Required bool        `json:"required,omitempty"`
	Default  interface{} `json:"default,omitempty"`
}

// StepDef represents a workflow step definition.
type StepDef struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Agent       string                 `json:"agent"`
	Action      string                 `json:"action"` // task, notify, query
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      []string               `json:"output,omitempty"`
	DependsOn   []string               `json:"depends_on,omitempty"`
	Condition   string                 `json:"condition,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"` // seconds
	Retry       int                    `json:"retry,omitempty"`
	RetryDelay  int                    `json:"retry_delay,omitempty"` // seconds
}

// TriggerConfig represents the trigger configuration for a workflow.
type TriggerConfig struct {
	Type   string                 `json:"type"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// ErrorHandlerDef represents an error handler definition.
type ErrorHandlerDef struct {
	Condition string                 `json:"condition"` // any, step_id, or expression
	Action    string                 `json:"action"`
	Agent     string                 `json:"agent,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
}

// ParallelConfig represents parallel execution configuration.
type ParallelConfig struct {
	MaxConcurrent int `json:"max_concurrent,omitempty"`
}

// Execution represents a workflow execution instance.
type Execution struct {
	ID          string            `json:"id"`
	WorkflowID  string            `json:"workflow_id"`
	Status      ExecutionStatus   `json:"status"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	Variables   map[string]interface{} `json:"variables,omitempty"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
	Error       string            `json:"error,omitempty"`
	TriggeredBy string            `json:"triggered_by,omitempty"`
	Steps       []*StepExecution  `json:"steps,omitempty"`
}

// StepExecution represents the execution of a workflow step.
type StepExecution struct {
	ID           string                 `json:"id"`
	ExecutionID  string                 `json:"execution_id"`
	StepID       string                 `json:"step_id"`
	AgentID      string                 `json:"agent_id"`
	Status       StepStatus             `json:"status"`
	Input        map[string]interface{} `json:"input,omitempty"`
	Output       map[string]interface{} `json:"output,omitempty"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Error        string                 `json:"error,omitempty"`
	RetryCount   int                    `json:"retry_count,omitempty"`
}

// StepOutput represents the output of a completed step.
type StepOutput struct {
	StepID string
	Status StepStatus
	Output map[string]interface{}
	Error  string
}

// ExecutionStats represents execution statistics.
type ExecutionStats struct {
	TotalExecutions   int `json:"total_executions"`
	CompletedCount    int `json:"completed_count"`
	FailedCount       int `json:"failed_count"`
	RunningCount      int `json:"running_count"`
	PendingCount      int `json:"pending_count"`
	CancelledCount    int `json:"cancelled_count"`
	AvgExecutionTime  int `json:"avg_execution_time"` // seconds
}

// MarshalDefinition marshals workflow definition to JSON.
func MarshalDefinition(def *WorkflowDef) ([]byte, error) {
	return json.Marshal(def)
}

// UnmarshalDefinition unmarshals JSON to workflow definition.
func UnmarshalDefinition(data []byte) (*WorkflowDef, error) {
	var def WorkflowDef
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, err
	}
	return &def, nil
}

// GetStep returns a step definition by ID.
func (def *WorkflowDef) GetStep(stepID string) *StepDef {
	for _, step := range def.Steps {
		if step.ID == stepID {
			return &step
		}
	}
	return nil
}

// GetDependencies returns the dependency graph for steps.
func (def *WorkflowDef) GetDependencies() map[string][]string {
	deps := make(map[string][]string)
	for _, step := range def.Steps {
		deps[step.ID] = step.DependsOn
	}
	return deps
}

// GetReadySteps returns steps that are ready to execute.
func (def *WorkflowDef) GetReadySteps(completedSteps map[string]bool) []StepDef {
	var ready []StepDef
	for _, step := range def.Steps {
		if completedSteps[step.ID] {
			continue
		}
		// Check if all dependencies are completed
		allDepsCompleted := true
		for _, dep := range step.DependsOn {
			if !completedSteps[dep] {
				allDepsCompleted = false
				break
			}
		}
		if allDepsCompleted {
			ready = append(ready, step)
		}
	}
	return ready
}
