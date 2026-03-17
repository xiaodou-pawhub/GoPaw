// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trace

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Status represents the status of a trace.
type Status string

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

// StepType represents the type of a trace step.
type StepType string

const (
	StepTypeContextBuild   StepType = "context_build"
	StepTypeLLMCall        StepType = "llm_call"
	StepTypeToolExecution  StepType = "tool_execution"
	StepTypeHookExecution  StepType = "hook_execution"
	StepTypeFinalAnswer    StepType = "final_answer"
)

// Trace represents a complete execution trace.
type Trace struct {
	ID            string    `json:"id"`
	SessionID     string    `json:"session_id"`
	StartedAt     time.Time `json:"started_at"`
	EndedAt       time.Time `json:"ended_at,omitempty"`
	Status        Status    `json:"status"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	Steps         []*Step   `json:"steps,omitempty"`
}

// NewTrace creates a new trace.
func NewTrace(sessionID string) *Trace {
	return &Trace{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		StartedAt: time.Now().UTC(),
		Status:    StatusRunning,
		Steps:     []*Step{},
	}
}

// End marks the trace as completed.
func (t *Trace) End() {
	t.EndedAt = time.Now().UTC()
	if t.Status == StatusRunning {
		t.Status = StatusCompleted
	}
}

// EndWithError marks the trace as error.
func (t *Trace) EndWithError(err error) {
	t.EndedAt = time.Now().UTC()
	t.Status = StatusError
	if err != nil {
		t.ErrorMessage = err.Error()
	}
}

// Duration returns the duration of the trace.
func (t *Trace) Duration() time.Duration {
	if t.EndedAt.IsZero() {
		return time.Since(t.StartedAt)
	}
	return t.EndedAt.Sub(t.StartedAt)
}

// Step represents a single step in the execution trace.
type Step struct {
	ID         string          `json:"id"`
	TraceID    string          `json:"trace_id"`
	StepNumber int             `json:"step_number"`
	StepType   StepType        `json:"step_type"`
	StartedAt  time.Time       `json:"started_at"`
	EndedAt    time.Time       `json:"ended_at,omitempty"`
	Input      json.RawMessage `json:"input,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// NewStep creates a new step.
func NewStep(traceID string, stepNumber int, stepType StepType) *Step {
	return &Step{
		ID:         uuid.New().String(),
		TraceID:    traceID,
		StepNumber: stepNumber,
		StepType:   stepType,
		StartedAt:  time.Now().UTC(),
	}
}

// End marks the step as completed.
func (s *Step) End() {
	s.EndedAt = time.Now().UTC()
}

// Duration returns the duration of the step.
func (s *Step) Duration() time.Duration {
	if s.EndedAt.IsZero() {
		return time.Since(s.StartedAt)
	}
	return s.EndedAt.Sub(s.StartedAt)
}

// SetInput sets the input data.
func (s *Step) SetInput(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Input = data
	return nil
}

// SetOutput sets the output data.
func (s *Step) SetOutput(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Output = data
	return nil
}

// SetMetadata sets the metadata.
func (s *Step) SetMetadata(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	s.Metadata = data
	return nil
}

// ContextBuildInput represents input for context build step.
type ContextBuildInput struct {
	UserInput  string `json:"user_input"`
	SessionID  string `json:"session_id"`
}

// ContextBuildOutput represents output for context build step.
type ContextBuildOutput struct {
	SystemPrompt   string `json:"system_prompt"`
	MemoriesUsed   int    `json:"memories_used"`
	SkillsMatched  int    `json:"skills_matched"`
	FocusTasks     string `json:"focus_tasks,omitempty"`
}

// ContextBuildMetadata represents metadata for context build step.
type ContextBuildMetadata struct {
	TokenBudget int `json:"token_budget"`
	BuildTimeMs int `json:"build_time_ms"`
}

// LLMCallInput represents input for LLM call step.
type LLMCallInput struct {
	Messages []map[string]interface{} `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

// LLMCallOutput represents output for LLM call step.
type LLMCallOutput struct {
	Message      map[string]interface{} `json:"message"`
	FinishReason string                 `json:"finish_reason"`
	Usage        struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// LLMCallMetadata represents metadata for LLM call step.
type LLMCallMetadata struct {
	Model       string  `json:"model"`
	Temperature float32 `json:"temperature"`
	DurationMs  int     `json:"duration_ms"`
}

// ToolExecutionInput represents input for tool execution step.
type ToolExecutionInput struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolExecutionOutput represents output for tool execution step.
type ToolExecutionOutput struct {
	Result   string `json:"result"`
	Error    string `json:"error,omitempty"`
	Duration int    `json:"duration_ms"`
}

// ToolExecutionMetadata represents metadata for tool execution step.
type ToolExecutionMetadata struct {
	Parallel   bool   `json:"parallel"`
	ToolCallID string `json:"tool_call_id"`
}

// HookExecutionInput represents input for hook execution step.
type HookExecutionInput struct {
	HookType string                   `json:"hook_type"`
	Messages []map[string]interface{} `json:"messages"`
}

// HookExecutionOutput represents output for hook execution step.
type HookExecutionOutput struct {
	Modified bool                     `json:"modified"`
	Messages []map[string]interface{} `json:"messages"`
}

// FinalAnswerInput represents input for final answer step.
type FinalAnswerInput struct {
	MaxStepsReached bool `json:"max_steps_reached"`
}

// FinalAnswerOutput represents output for final answer step.
type FinalAnswerOutput struct {
	Answer string `json:"answer"`
}
