// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package handlers

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/trace"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// TraceHandler handles /api/traces routes for execution traces.
type TraceHandler struct {
	manager *trace.Manager
	logger  *zap.Logger
}

// NewTraceHandler creates a TraceHandler.
func NewTraceHandler(manager *trace.Manager, logger *zap.Logger) *TraceHandler {
	return &TraceHandler{manager: manager, logger: logger}
}

// traceJSON is the JSON representation of a trace.
type traceJSON struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	StartedAt    int64     `json:"started_at"`
	EndedAt      int64     `json:"ended_at,omitempty"`
	Status       string    `json:"status"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Duration     int64     `json:"duration_ms"`
	StepCount    int       `json:"step_count"`
}

// stepJSON is the JSON representation of a trace step.
type stepJSON struct {
	ID         int64           `json:"id"`
	StepNumber int             `json:"step_number"`
	StepType   string          `json:"step_type"`
	StartedAt  int64           `json:"started_at"`
	EndedAt    int64           `json:"ended_at,omitempty"`
	Duration   int64           `json:"duration_ms"`
	Input      json.RawMessage `json:"input,omitempty"`
	Output     json.RawMessage `json:"output,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

// traceDetailJSON is the JSON representation of a trace with steps.
type traceDetailJSON struct {
	traceJSON
	Steps []stepJSON `json:"steps"`
}

// List handles GET /api/traces
// Query params: session_id, status, limit (default 50), offset
func (h *TraceHandler) List(c *gin.Context) {
	if h.manager == nil {
		api.InternalError(c, "trace manager not initialized")
		return
	}

	sessionID := c.Query("session_id")
	status := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	opts := trace.QueryOptions{
		SessionID: sessionID,
		Limit:     limit,
	}
	if status != "" {
		opts.Status = status
	}

	traces, err := h.manager.QueryTraces(opts)
	if err != nil {
		h.logger.Error("failed to query traces", zap.Error(err))
		api.InternalError(c, "failed to query traces")
		return
	}

	result := make([]traceJSON, len(traces))
	for i, t := range traces {
		result[i] = traceToJSON(t)
	}

	api.List(c, result, len(result))
}

// Get handles GET /api/traces/:id
func (h *TraceHandler) Get(c *gin.Context) {
	if h.manager == nil {
		api.InternalError(c, "trace manager not initialized")
		return
	}

	id := c.Param("id")
	if id == "" {
		api.BadRequest(c, "trace id is required")
		return
	}

	t, err := h.manager.GetTrace(id)
	if err != nil {
		h.logger.Error("failed to get trace", zap.String("id", id), zap.Error(err))
		api.InternalError(c, "failed to get trace")
		return
	}
	if t == nil {
		api.NotFound(c, "trace")
		return
	}

	result := traceDetailToJSON(t)
	api.Success(c, result)
}

// Stats handles GET /api/traces/stats
func (h *TraceHandler) Stats(c *gin.Context) {
	if h.manager == nil {
		api.InternalError(c, "trace manager not initialized")
		return
	}

	// Get recent traces for stats
	traces, err := h.manager.QueryTraces(trace.QueryOptions{Limit: 1000})
	if err != nil {
		h.logger.Error("failed to query traces for stats", zap.Error(err))
		api.InternalError(c, "failed to query traces")
		return
	}

	var totalDuration time.Duration
	var completedCount, errorCount int
	stepTypeCount := make(map[string]int)

	for _, t := range traces {
		totalDuration += t.Duration()
		if t.Status == trace.StatusCompleted {
			completedCount++
		} else if t.Status == trace.StatusError {
			errorCount++
		}
		for _, s := range t.Steps {
			stepTypeCount[string(s.StepType)]++
		}
	}

	avgDuration := int64(0)
	if len(traces) > 0 {
		avgDuration = int64(totalDuration / time.Duration(len(traces)))
	}

	api.Success(c, gin.H{
		"total_traces":   len(traces),
		"completed":      completedCount,
		"errors":         errorCount,
		"avg_duration_ms": avgDuration / int64(time.Millisecond),
		"step_types":     stepTypeCount,
	})
}

// traceToJSON converts a Trace to traceJSON.
func traceToJSON(t *trace.Trace) traceJSON {
	result := traceJSON{
		ID:        t.ID,
		SessionID: t.SessionID,
		StartedAt: t.StartedAt.UnixMilli(),
		Status:    string(t.Status),
		Duration:  int64(t.Duration() / time.Millisecond),
		StepCount: len(t.Steps),
	}
	if !t.EndedAt.IsZero() {
		result.EndedAt = t.EndedAt.UnixMilli()
	}
	if t.ErrorMessage != "" {
		result.ErrorMessage = t.ErrorMessage
	}
	return result
}

// traceDetailToJSON converts a Trace to traceDetailJSON.
func traceDetailToJSON(t *trace.Trace) traceDetailJSON {
	result := traceDetailJSON{
		traceJSON: traceToJSON(t),
		Steps:     make([]stepJSON, len(t.Steps)),
	}
	for i, s := range t.Steps {
		result.Steps[i] = stepToJSON(s)
	}
	return result
}

// stepToJSON converts a Step to stepJSON.
func stepToJSON(s *trace.Step) stepJSON {
	result := stepJSON{
		ID:         0, // SQLite rowid not exposed
		StepNumber: s.StepNumber,
		StepType:   string(s.StepType),
		StartedAt:  s.StartedAt.UnixMilli(),
		Duration:   int64(s.Duration() / time.Millisecond),
		Input:      s.Input,
		Output:     s.Output,
		Metadata:   s.Metadata,
	}
	if !s.EndedAt.IsZero() {
		result.EndedAt = s.EndedAt.UnixMilli()
	}
	return result
}
