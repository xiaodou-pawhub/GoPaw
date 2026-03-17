// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package trace

import (
	"encoding/json"
	"fmt"
	"time"
)

// Query provides methods to query and analyze traces.
type Query struct {
	storage *Storage
}

// NewQuery creates a new query instance.
func NewQuery(storage *Storage) *Query {
	return &Query{storage: storage}
}

// ListRecent returns recent traces.
func (q *Query) ListRecent(limit int) ([]*Trace, error) {
	return q.storage.QueryTraces(QueryOptions{
		Limit: limit,
	})
}

// ListBySession returns traces for a specific session.
func (q *Query) ListBySession(sessionID string, limit int) ([]*Trace, error) {
	return q.storage.QueryTraces(QueryOptions{
		SessionID: sessionID,
		Limit:     limit,
	})
}

// ListByTimeRange returns traces within a time range.
func (q *Query) ListByTimeRange(start, end time.Time, limit int) ([]*Trace, error) {
	return q.storage.QueryTraces(QueryOptions{
		StartedAfter:  start,
		StartedBefore: end,
		Limit:         limit,
	})
}

// GetTraceWithSteps returns a trace with all its steps.
func (q *Query) GetTraceWithSteps(traceID string) (*Trace, error) {
	return q.storage.GetTrace(traceID)
}

// Stats returns statistics about traces.
func (q *Query) Stats() (*TraceStats, error) {
	// This is a placeholder - in real implementation,
	// we would add a method to Storage to get stats
	return &TraceStats{}, nil
}

// TraceStats holds statistics about traces.
type TraceStats struct {
	TotalTraces   int64         `json:"total_traces"`
	TotalSteps    int64         `json:"total_steps"`
	AvgDuration   time.Duration `json:"avg_duration"`
	AvgSteps      float64       `json:"avg_steps"`
	ErrorRate     float64       `json:"error_rate"`
}

// FormatTrace formats a trace for display.
func FormatTrace(t *Trace) string {
	var result string
	result += fmt.Sprintf("Trace ID: %s\n", t.ID)
	result += fmt.Sprintf("Session: %s\n", t.SessionID)
	result += fmt.Sprintf("Status: %s\n", t.Status)
	result += fmt.Sprintf("Started: %s\n", t.StartedAt.Format("2006-01-02 15:04:05"))
	if !t.EndedAt.IsZero() {
		result += fmt.Sprintf("Ended: %s\n", t.EndedAt.Format("2006-01-02 15:04:05"))
		result += fmt.Sprintf("Duration: %s\n", t.Duration())
	}
	if t.ErrorMessage != "" {
		result += fmt.Sprintf("Error: %s\n", t.ErrorMessage)
	}
	result += fmt.Sprintf("Steps: %d\n", len(t.Steps))
	return result
}

// FormatStep formats a step for display.
func FormatStep(s *Step) string {
	var result string
	result += fmt.Sprintf("  Step %d: %s\n", s.StepNumber, s.StepType)
	result += fmt.Sprintf("    Duration: %s\n", s.Duration())
	if len(s.Input) > 0 {
		result += fmt.Sprintf("    Input: %s\n", truncate(string(s.Input), 100))
	}
	if len(s.Output) > 0 {
		result += fmt.Sprintf("    Output: %s\n", truncate(string(s.Output), 100))
	}
	return result
}

// ToJSON converts a trace to JSON string.
func ToJSON(t *Trace) (string, error) {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// truncate shortens a string to max length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
