// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ========== 运行轨迹数据模型 ==========

// Trace 一次流程执行的完整追踪
type Trace struct {
	ID           string        `json:"id" db:"id"`                       // 追踪 ID（与执行 ID 相同）
	FlowID       string        `json:"flow_id" db:"flow_id"`             // 流程 ID
	FlowName     string        `json:"flow_name" db:"flow_name"`         // 流程名称
	ExecutionID  string        `json:"execution_id" db:"execution_id"`   // 执行 ID
	Trigger      string        `json:"trigger" db:"trigger"`             // 触发来源
	Status       string        `json:"status" db:"status"`               // 状态: running, completed, failed
	StartedAt    time.Time     `json:"started_at" db:"started_at"`       // 开始时间
	CompletedAt  *time.Time    `json:"completed_at" db:"completed_at"`   // 结束时间
	Duration     int64         `json:"duration" db:"duration"`           // 总耗时（毫秒）
	TotalTokens  int           `json:"total_tokens" db:"total_tokens"`   // 总 Token 数
	TotalCost    float64       `json:"total_cost" db:"total_cost"`       // 总成本
	Spans        []*Span       `json:"spans,omitempty"`                  // Span 列表
	RootSpanID   string        `json:"root_span_id" db:"root_span_id"`   // 根 Span ID
	Error        string        `json:"error,omitempty" db:"error"`       // 错误信息
	Metadata     TraceMetadata `json:"metadata" db:"metadata"`           // 元数据
}

// TraceMetadata 追踪元数据
type TraceMetadata struct {
	Input          string                 `json:"input,omitempty"`
	Output         string                 `json:"output,omitempty"`
	Variables      map[string]interface{} `json:"variables,omitempty"`
	UserID         string                 `json:"user_id,omitempty"`
	SessionID      string                 `json:"session_id,omitempty"`
	ClientIP       string                 `json:"client_ip,omitempty"`
	UserAgent      string                 `json:"user_agent,omitempty"`
	CustomLabels   map[string]string      `json:"custom_labels,omitempty"`
}

// Span 一个节点执行的追踪单元
type Span struct {
	ID           string      `json:"id" db:"id"`                       // Span ID
	TraceID      string      `json:"trace_id" db:"trace_id"`           // 所属 Trace ID
	ParentSpanID string      `json:"parent_span_id" db:"parent_span_id"` // 父 Span ID
	NodeID       string      `json:"node_id" db:"node_id"`             // 节点 ID
	NodeName     string      `json:"node_name" db:"node_name"`         // 节点名称
	NodeType     string      `json:"node_type" db:"node_type"`         // 节点类型
	Status       string      `json:"status" db:"status"`               // 状态: running, completed, failed, skipped
	StartedAt    time.Time   `json:"started_at" db:"started_at"`       // 开始时间
	CompletedAt  *time.Time  `json:"completed_at" db:"completed_at"`   // 结束时间
	Duration     int64       `json:"duration" db:"duration"`           // 耗时（毫秒）
	TokensIn     int         `json:"tokens_in" db:"tokens_in"`         // 输入 Token
	TokensOut    int         `json:"tokens_out" db:"tokens_out"`       // 输出 Token
	Cost         float64     `json:"cost" db:"cost"`                   // 成本
	AgentID      string      `json:"agent_id" db:"agent_id"`           // Agent ID（如果是 Agent 节点）
	AgentName    string      `json:"agent_name" db:"agent_name"`       // Agent 名称
	Model        string      `json:"model" db:"model"`                 // 使用的模型
	Input        string      `json:"input,omitempty" db:"input"`       // 输入内容
	Output       string      `json:"output,omitempty" db:"output"`     // 输出内容
	Error        string      `json:"error,omitempty" db:"error"`       // 错误信息
	Events       []*Event    `json:"events,omitempty"`                 // 事件列表
	Tags         SpanTags    `json:"tags" db:"tags"`                   // 标签
}

// SpanTags Span 标签
type SpanTags struct {
	LoopIteration  int    `json:"loop_iteration,omitempty"`  // 循环迭代次数
	BranchName     string `json:"branch_name,omitempty"`     // 分支名称
	ParallelIndex  int    `json:"parallel_index,omitempty"`  // 并行索引
	RetryAttempt   int    `json:"retry_attempt,omitempty"`   // 重试次数
	IsFallback     bool   `json:"is_fallback,omitempty"`     // 是否 fallback
	CacheHit       bool   `json:"cache_hit,omitempty"`       // 是否缓存命中
	CustomTags     map[string]string `json:"custom_tags,omitempty"` // 自定义标签
}

// Event 具体事件
type Event struct {
	ID          string                 `json:"id" db:"id"`
	SpanID      string                 `json:"span_id" db:"span_id"`
	Name        string                 `json:"name" db:"name"`           // 事件名称
	Type        string                 `json:"type" db:"type"`           // 事件类型
	Timestamp   time.Time              `json:"timestamp" db:"timestamp"` // 时间戳
	Attributes  map[string]interface{} `json:"attributes" db:"attributes"` // 事件属性
}

// EventType 事件类型
const (
	EventTypeNodeStart     = "node_start"
	EventTypeNodeComplete  = "node_complete"
	EventTypeNodeFail      = "node_fail"
	EventTypeNodeRetry     = "node_retry"
	EventTypeLLMCallStart  = "llm_call_start"
	EventTypeLLMCallEnd    = "llm_call_end"
	EventTypeToolCallStart = "tool_call_start"
	EventTypeToolCallEnd   = "tool_call_end"
	EventTypeHumanInput    = "human_input"
	EventTypeHumanOutput   = "human_output"
	EventTypeConditionEval = "condition_eval"
	EventTypeLoopIterate   = "loop_iterate"
	EventTypeParallelStart = "parallel_start"
	EventTypeParallelEnd   = "parallel_end"
	EventTypeCacheHit      = "cache_hit"
	EventTypeCacheMiss     = "cache_miss"
	EventTypeError         = "error"
	EventTypeWarning       = "warning"
	EventTypeInfo          = "info"
)

// ========== 追踪服务 ==========

// TraceService 追踪服务
type TraceService struct {
	db     *sql.DB
	logger *zap.Logger
	// 内存缓存（用于实时追踪）
	traces map[string]*Trace
	spans  map[string]*Span
	mu     sync.RWMutex
}

// NewTraceService 创建追踪服务
func NewTraceService(db *sql.DB, logger *zap.Logger) *TraceService {
	return &TraceService{
		db:     db,
		logger: logger.Named("trace"),
		traces: make(map[string]*Trace),
		spans:  make(map[string]*Span),
	}
}

// InitTraceSchema 初始化追踪表
func InitTraceSchema(db *sql.DB) error {
	// 追踪表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_traces (
			id TEXT PRIMARY KEY,
			flow_id TEXT NOT NULL,
			flow_name TEXT,
			execution_id TEXT NOT NULL,
			trigger TEXT DEFAULT 'manual',
			status TEXT DEFAULT 'running',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			duration INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
			root_span_id TEXT,
			error TEXT,
			metadata TEXT,
			FOREIGN KEY (flow_id) REFERENCES flows(id) ON DELETE CASCADE,
			FOREIGN KEY (execution_id) REFERENCES flow_executions(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// Span 表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_spans (
			id TEXT PRIMARY KEY,
			trace_id TEXT NOT NULL,
			parent_span_id TEXT,
			node_id TEXT NOT NULL,
			node_name TEXT,
			node_type TEXT,
			status TEXT DEFAULT 'running',
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			completed_at DATETIME,
			duration INTEGER DEFAULT 0,
			tokens_in INTEGER DEFAULT 0,
			tokens_out INTEGER DEFAULT 0,
			cost REAL DEFAULT 0,
			agent_id TEXT,
			agent_name TEXT,
			model TEXT,
			input TEXT,
			output TEXT,
			error TEXT,
			tags TEXT,
			FOREIGN KEY (trace_id) REFERENCES flow_traces(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 事件表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_events (
			id TEXT PRIMARY KEY,
			span_id TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			attributes TEXT,
			FOREIGN KEY (span_id) REFERENCES flow_spans(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_traces_flow ON flow_traces(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_status ON flow_traces(status)`,
		`CREATE INDEX IF NOT EXISTS idx_traces_started ON flow_traces(started_at)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_trace ON flow_spans(trace_id)`,
		`CREATE INDEX IF NOT EXISTS idx_spans_node ON flow_spans(node_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_span ON flow_events(span_id)`,
		`CREATE INDEX IF NOT EXISTS idx_events_type ON flow_events(type)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

// StartTrace 开始追踪
func (s *TraceService) StartTrace(flowID, flowName, executionID, trigger string, metadata TraceMetadata) (*Trace, error) {
	trace := &Trace{
		ID:          executionID,
		FlowID:      flowID,
		FlowName:    flowName,
		ExecutionID: executionID,
		Trigger:     trigger,
		Status:      "running",
		StartedAt:   time.Now(),
		Metadata:    metadata,
	}

	// 保存到内存
	s.mu.Lock()
	s.traces[trace.ID] = trace
	s.mu.Unlock()

	// 保存到数据库
	if err := s.saveTrace(trace); err != nil {
		return nil, err
	}

	s.logger.Debug("trace started",
		zap.String("trace_id", trace.ID),
		zap.String("flow_id", flowID))

	return trace, nil
}

// StartSpan 开始 Span
func (s *TraceService) StartSpan(traceID, parentSpanID, nodeID, nodeName, nodeType string) *Span {
	spanID := fmt.Sprintf("span_%s_%s", traceID[:8], nodeID)

	span := &Span{
		ID:           spanID,
		TraceID:      traceID,
		ParentSpanID: parentSpanID,
		NodeID:       nodeID,
		NodeName:     nodeName,
		NodeType:     nodeType,
		Status:       "running",
		StartedAt:    time.Now(),
		Events:       make([]*Event, 0),
	}

	// 保存到内存
	s.mu.Lock()
	s.spans[span.ID] = span
	// 更新 Trace 的 RootSpanID
	if trace, ok := s.traces[traceID]; ok && trace.RootSpanID == "" {
		trace.RootSpanID = spanID
	}
	s.mu.Unlock()

	// 保存到数据库
	s.saveSpan(span)

	s.logger.Debug("span started",
		zap.String("span_id", span.ID),
		zap.String("node_id", nodeID),
		zap.String("node_type", nodeType))

	return span
}

// EndSpan 结束 Span
func (s *TraceService) EndSpan(spanID string, output string, err error) {
	s.mu.Lock()
	span, ok := s.spans[spanID]
	if !ok {
		s.mu.Unlock()
		return
	}

	now := time.Now()
	span.CompletedAt = &now
	span.Duration = now.Sub(span.StartedAt).Milliseconds()
	span.Output = output

	if err != nil {
		span.Status = "failed"
		span.Error = err.Error()
	} else {
		span.Status = "completed"
	}
	s.mu.Unlock()

	// 保存到数据库
	s.updateSpan(span)

	s.logger.Debug("span ended",
		zap.String("span_id", spanID),
		zap.String("status", span.Status),
		zap.Int64("duration_ms", span.Duration))
}

// AddEvent 添加事件
func (s *TraceService) AddEvent(spanID, name, eventType string, attributes map[string]interface{}) {
	s.mu.RLock()
	span, ok := s.spans[spanID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	event := &Event{
		ID:         fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		SpanID:     spanID,
		Name:       name,
		Type:       eventType,
		Timestamp:  time.Now(),
		Attributes: attributes,
	}

	s.mu.Lock()
	span.Events = append(span.Events, event)
	s.mu.Unlock()

	// 保存到数据库
	s.saveEvent(event)

	s.logger.Debug("event added",
		zap.String("span_id", spanID),
		zap.String("event_type", eventType),
		zap.String("event_name", name))
}

// SetSpanTokens 设置 Span Token 信息
func (s *TraceService) SetSpanTokens(spanID string, tokensIn, tokensOut int, cost float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	span, ok := s.spans[spanID]
	if !ok {
		return
	}

	span.TokensIn = tokensIn
	span.TokensOut = tokensOut
	span.Cost = cost

	// 更新 Trace 总计
	if trace, ok := s.traces[span.TraceID]; ok {
		trace.TotalTokens += tokensIn + tokensOut
		trace.TotalCost += cost
	}
}

// SetSpanAgent 设置 Span Agent 信息
func (s *TraceService) SetSpanAgent(spanID, agentID, agentName, model string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	span, ok := s.spans[spanID]
	if !ok {
		return
	}

	span.AgentID = agentID
	span.AgentName = agentName
	span.Model = model
}

// SetSpanInput 设置 Span 输入
func (s *TraceService) SetSpanInput(spanID, input string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	span, ok := s.spans[spanID]
	if !ok {
		return
	}

	span.Input = input
}

// EndTrace 结束追踪
func (s *TraceService) EndTrace(traceID string, output string, err error) {
	s.mu.Lock()
	trace, ok := s.traces[traceID]
	if !ok {
		s.mu.Unlock()
		return
	}

	now := time.Now()
	trace.CompletedAt = &now
	trace.Duration = now.Sub(trace.StartedAt).Milliseconds()
	trace.Metadata.Output = output

	if err != nil {
		trace.Status = "failed"
		trace.Error = err.Error()
	} else {
		trace.Status = "completed"
	}
	s.mu.Unlock()

	// 更新数据库
	s.updateTrace(trace)

	s.logger.Info("trace ended",
		zap.String("trace_id", traceID),
		zap.String("status", trace.Status),
		zap.Int64("duration_ms", trace.Duration),
		zap.Int("total_tokens", trace.TotalTokens))
}

// GetTrace 获取追踪
func (s *TraceService) GetTrace(traceID string) (*Trace, error) {
	// 先从内存获取
	s.mu.RLock()
	if trace, ok := s.traces[traceID]; ok {
		s.mu.RUnlock()
		return trace, nil
	}
	s.mu.RUnlock()

	// 从数据库获取
	return s.loadTrace(traceID)
}

// GetSpan 获取 Span
func (s *TraceService) GetSpan(spanID string) (*Span, error) {
	s.mu.RLock()
	if span, ok := s.spans[spanID]; ok {
		s.mu.RUnlock()
		return span, nil
	}
	s.mu.RUnlock()

	return s.loadSpan(spanID)
}

// ListTraces 列出追踪
func (s *TraceService) ListTraces(flowID string, limit int) ([]*Trace, error) {
	query := `SELECT id, flow_id, flow_name, execution_id, trigger, status, started_at, completed_at, duration, total_tokens, total_cost, root_span_id, error, metadata
		FROM flow_traces`
	args := []interface{}{}

	if flowID != "" {
		query += " WHERE flow_id = ?"
		args = append(args, flowID)
	}

	query += " ORDER BY started_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list traces: %w", err)
	}
	defer rows.Close()

	var traces []*Trace
	for rows.Next() {
		trace := &Trace{}
		var flowName, rootSpanID, errMsg, metadataJSON sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(&trace.ID, &trace.FlowID, &flowName, &trace.ExecutionID, &trace.Trigger, &trace.Status,
			&trace.StartedAt, &completedAt, &trace.Duration, &trace.TotalTokens, &trace.TotalCost, &rootSpanID, &errMsg, &metadataJSON)
		if err != nil {
			continue
		}

		if flowName.Valid {
			trace.FlowName = flowName.String
		}
		if rootSpanID.Valid {
			trace.RootSpanID = rootSpanID.String
		}
		if errMsg.Valid {
			trace.Error = errMsg.String
		}
		if completedAt.Valid {
			trace.CompletedAt = &completedAt.Time
		}
		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &trace.Metadata)
		}

		traces = append(traces, trace)
	}

	return traces, nil
}

// GetTraceSpans 获取 Trace 的所有 Span
func (s *TraceService) GetTraceSpans(traceID string) ([]*Span, error) {
	rows, err := s.db.Query(`
		SELECT id, trace_id, parent_span_id, node_id, node_name, node_type, status, started_at, completed_at, duration,
			tokens_in, tokens_out, cost, agent_id, agent_name, model, input, output, error, tags
		FROM flow_spans WHERE trace_id = ? ORDER BY started_at
	`, traceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trace spans: %w", err)
	}
	defer rows.Close()

	var spans []*Span
	for rows.Next() {
		span := &Span{}
		var parentSpanID, agentID, agentName, model, input, output, errMsg, tagsJSON sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(&span.ID, &span.TraceID, &parentSpanID, &span.NodeID, &span.NodeName, &span.NodeType, &span.Status,
			&span.StartedAt, &completedAt, &span.Duration, &span.TokensIn, &span.TokensOut, &span.Cost,
			&agentID, &agentName, &model, &input, &output, &errMsg, &tagsJSON)
		if err != nil {
			continue
		}

		if parentSpanID.Valid {
			span.ParentSpanID = parentSpanID.String
		}
		if agentID.Valid {
			span.AgentID = agentID.String
		}
		if agentName.Valid {
			span.AgentName = agentName.String
		}
		if model.Valid {
			span.Model = model.String
		}
		if input.Valid {
			span.Input = input.String
		}
		if output.Valid {
			span.Output = output.String
		}
		if errMsg.Valid {
			span.Error = errMsg.String
		}
		if completedAt.Valid {
			span.CompletedAt = &completedAt.Time
		}
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &span.Tags)
		}

		// 加载事件
		span.Events, _ = s.loadSpanEvents(span.ID)

		spans = append(spans, span)
	}

	return spans, nil
}

// ========== 数据库操作 ==========

func (s *TraceService) saveTrace(trace *Trace) error {
	metadataJSON, _ := json.Marshal(trace.Metadata)

	_, err := s.db.Exec(`
		INSERT INTO flow_traces (id, flow_id, flow_name, execution_id, trigger, status, started_at, metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, trace.ID, trace.FlowID, trace.FlowName, trace.ExecutionID, trace.Trigger, trace.Status, trace.StartedAt, string(metadataJSON))

	return err
}

func (s *TraceService) updateTrace(trace *Trace) error {
	metadataJSON, _ := json.Marshal(trace.Metadata)

	_, err := s.db.Exec(`
		UPDATE flow_traces SET status = ?, completed_at = ?, duration = ?, total_tokens = ?, total_cost = ?, error = ?, metadata = ?
		WHERE id = ?
	`, trace.Status, trace.CompletedAt, trace.Duration, trace.TotalTokens, trace.TotalCost, trace.Error, string(metadataJSON), trace.ID)

	return err
}

func (s *TraceService) saveSpan(span *Span) error {
	tagsJSON, _ := json.Marshal(span.Tags)

	_, err := s.db.Exec(`
		INSERT INTO flow_spans (id, trace_id, parent_span_id, node_id, node_name, node_type, status, started_at, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, span.ID, span.TraceID, span.ParentSpanID, span.NodeID, span.NodeName, span.NodeType, span.Status, span.StartedAt, string(tagsJSON))

	return err
}

func (s *TraceService) updateSpan(span *Span) error {
	tagsJSON, _ := json.Marshal(span.Tags)

	_, err := s.db.Exec(`
		UPDATE flow_spans SET status = ?, completed_at = ?, duration = ?, tokens_in = ?, tokens_out = ?, cost = ?,
			agent_id = ?, agent_name = ?, model = ?, input = ?, output = ?, error = ?, tags = ?
		WHERE id = ?
	`, span.Status, span.CompletedAt, span.Duration, span.TokensIn, span.TokensOut, span.Cost,
		span.AgentID, span.AgentName, span.Model, span.Input, span.Output, span.Error, string(tagsJSON), span.ID)

	return err
}

func (s *TraceService) saveEvent(event *Event) error {
	attrsJSON, _ := json.Marshal(event.Attributes)

	_, err := s.db.Exec(`
		INSERT INTO flow_events (id, span_id, name, type, timestamp, attributes)
		VALUES (?, ?, ?, ?, ?, ?)
	`, event.ID, event.SpanID, event.Name, event.Type, event.Timestamp, string(attrsJSON))

	return err
}

func (s *TraceService) loadTrace(traceID string) (*Trace, error) {
	trace := &Trace{}
	var flowName, rootSpanID, errMsg, metadataJSON sql.NullString
	var completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, flow_id, flow_name, execution_id, trigger, status, started_at, completed_at, duration, total_tokens, total_cost, root_span_id, error, metadata
		FROM flow_traces WHERE id = ?
	`, traceID).Scan(&trace.ID, &trace.FlowID, &flowName, &trace.ExecutionID, &trace.Trigger, &trace.Status,
		&trace.StartedAt, &completedAt, &trace.Duration, &trace.TotalTokens, &trace.TotalCost, &rootSpanID, &errMsg, &metadataJSON)
	if err != nil {
		return nil, fmt.Errorf("trace not found: %s", traceID)
	}

	if flowName.Valid {
		trace.FlowName = flowName.String
	}
	if rootSpanID.Valid {
		trace.RootSpanID = rootSpanID.String
	}
	if errMsg.Valid {
		trace.Error = errMsg.String
	}
	if completedAt.Valid {
		trace.CompletedAt = &completedAt.Time
	}
	if metadataJSON.Valid {
		json.Unmarshal([]byte(metadataJSON.String), &trace.Metadata)
	}

	// 加载 Spans
	trace.Spans, _ = s.GetTraceSpans(traceID)

	return trace, nil
}

func (s *TraceService) loadSpan(spanID string) (*Span, error) {
	span := &Span{}
	var parentSpanID, agentID, agentName, model, input, output, errMsg, tagsJSON sql.NullString
	var completedAt sql.NullTime

	err := s.db.QueryRow(`
		SELECT id, trace_id, parent_span_id, node_id, node_name, node_type, status, started_at, completed_at, duration,
			tokens_in, tokens_out, cost, agent_id, agent_name, model, input, output, error, tags
		FROM flow_spans WHERE id = ?
	`, spanID).Scan(&span.ID, &span.TraceID, &parentSpanID, &span.NodeID, &span.NodeName, &span.NodeType, &span.Status,
		&span.StartedAt, &completedAt, &span.Duration, &span.TokensIn, &span.TokensOut, &span.Cost,
		&agentID, &agentName, &model, &input, &output, &errMsg, &tagsJSON)
	if err != nil {
		return nil, fmt.Errorf("span not found: %s", spanID)
	}

	if parentSpanID.Valid {
		span.ParentSpanID = parentSpanID.String
	}
	if agentID.Valid {
		span.AgentID = agentID.String
	}
	if agentName.Valid {
		span.AgentName = agentName.String
	}
	if model.Valid {
		span.Model = model.String
	}
	if input.Valid {
		span.Input = input.String
	}
	if output.Valid {
		span.Output = output.String
	}
	if errMsg.Valid {
		span.Error = errMsg.String
	}
	if completedAt.Valid {
		span.CompletedAt = &completedAt.Time
	}
	if tagsJSON.Valid {
		json.Unmarshal([]byte(tagsJSON.String), &span.Tags)
	}

	// 加载事件
	span.Events, _ = s.loadSpanEvents(span.ID)

	return span, nil
}

func (s *TraceService) loadSpanEvents(spanID string) ([]*Event, error) {
	rows, err := s.db.Query(`
		SELECT id, span_id, name, type, timestamp, attributes
		FROM flow_events WHERE span_id = ? ORDER BY timestamp
	`, spanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		event := &Event{}
		var attrsJSON sql.NullString

		err := rows.Scan(&event.ID, &event.SpanID, &event.Name, &event.Type, &event.Timestamp, &attrsJSON)
		if err != nil {
			continue
		}

		if attrsJSON.Valid {
			json.Unmarshal([]byte(attrsJSON.String), &event.Attributes)
		}

		events = append(events, event)
	}

	return events, nil
}

// GetTraceStats 获取追踪统计
func (s *TraceService) GetTraceStats(flowID string, days int) (*TraceStats, error) {
	stats := &TraceStats{}

	query := `
		SELECT 
			COUNT(*) as total_executions,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as completed,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			COALESCE(AVG(duration), 0) as avg_duration,
			COALESCE(MAX(duration), 0) as max_duration,
			COALESCE(MIN(duration), 0) as min_duration,
			COALESCE(SUM(total_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0) as total_cost
		FROM flow_traces
		WHERE started_at >= datetime('now', '-' || ? || ' days')
	`
	args := []interface{}{days}

	if flowID != "" {
		query += " AND flow_id = ?"
		args = append(args, flowID)
	}

	err := s.db.QueryRow(query, args...).Scan(
		&stats.TotalExecutions, &stats.Completed, &stats.Failed,
		&stats.AvgDuration, &stats.MaxDuration, &stats.MinDuration,
		&stats.TotalTokens, &stats.TotalCost,
	)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

// TraceStats 追踪统计
type TraceStats struct {
	TotalExecutions int     `json:"total_executions"`
	Completed       int     `json:"completed"`
	Failed          int     `json:"failed"`
	AvgDuration     float64 `json:"avg_duration"`
	MaxDuration     int64   `json:"max_duration"`
	MinDuration     int64   `json:"min_duration"`
	TotalTokens     int     `json:"total_tokens"`
	TotalCost       float64 `json:"total_cost"`
}

// NodePerformanceStats 节点性能统计
type NodePerformanceStats struct {
	NodeID          string  `json:"node_id"`
	NodeName        string  `json:"node_name"`
	NodeType        string  `json:"node_type"`
	ExecutionCount  int     `json:"execution_count"`
	SuccessCount    int     `json:"success_count"`
	FailedCount     int     `json:"failed_count"`
	AvgDuration     float64 `json:"avg_duration"`
	MaxDuration     int64   `json:"max_duration"`
	MinDuration     int64   `json:"min_duration"`
	TotalDuration   int64   `json:"total_duration"`
	TotalTokensIn   int     `json:"total_tokens_in"`
	TotalTokensOut  int     `json:"total_tokens_out"`
	TotalCost       float64 `json:"total_cost"`
	SuccessRate     float64 `json:"success_rate"`
	IsBottleneck    bool    `json:"is_bottleneck"`
}

// PerformanceAnalysis 性能分析结果
type PerformanceAnalysis struct {
	FlowID          string                  `json:"flow_id"`
	FlowName        string                  `json:"flow_name"`
	TotalTraces     int                     `json:"total_traces"`
	AvgDuration     float64                 `json:"avg_duration"`
	TotalDuration   int64                   `json:"total_duration"`
	NodeStats       []*NodePerformanceStats `json:"node_stats"`
	Bottlenecks     []*NodePerformanceStats `json:"bottlenecks"`
	SlowestNodes    []*NodePerformanceStats `json:"slowest_nodes"`
	ErrorProneNodes []*NodePerformanceStats `json:"error_prone_nodes"`
	Recommendations []string                `json:"recommendations"`
}

// GetPerformanceAnalysis 获取性能分析
func (s *TraceService) GetPerformanceAnalysis(flowID string, days int) (*PerformanceAnalysis, error) {
	analysis := &PerformanceAnalysis{
		FlowID:          flowID,
		NodeStats:       make([]*NodePerformanceStats, 0),
		Bottlenecks:     make([]*NodePerformanceStats, 0),
		SlowestNodes:    make([]*NodePerformanceStats, 0),
		ErrorProneNodes: make([]*NodePerformanceStats, 0),
		Recommendations: make([]string, 0),
	}

	// 获取流程名称
	if flowID != "" {
		s.db.QueryRow("SELECT name FROM flows WHERE id = ?", flowID).Scan(&analysis.FlowName)
	}

	// 获取总体统计
	query := `
		SELECT COUNT(*), COALESCE(AVG(duration), 0), COALESCE(SUM(duration), 0)
		FROM flow_traces
		WHERE started_at >= datetime('now', '-' || ? || ' days')
	`
	args := []interface{}{days}
	if flowID != "" {
		query += " AND flow_id = ?"
		args = append(args, flowID)
	}

	s.db.QueryRow(query, args...).Scan(&analysis.TotalTraces, &analysis.AvgDuration, &analysis.TotalDuration)

	// 获取节点性能统计
	nodeQuery := `
		SELECT 
			node_id,
			node_name,
			node_type,
			COUNT(*) as exec_count,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as success_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(AVG(duration), 0) as avg_duration,
			COALESCE(MAX(duration), 0) as max_duration,
			COALESCE(MIN(duration), 0) as min_duration,
			COALESCE(SUM(duration), 0) as total_duration,
			COALESCE(SUM(tokens_in), 0) as total_tokens_in,
			COALESCE(SUM(tokens_out), 0) as total_tokens_out,
			COALESCE(SUM(cost), 0) as total_cost
		FROM flow_spans
		WHERE trace_id IN (
			SELECT id FROM flow_traces 
			WHERE started_at >= datetime('now', '-' || ? || ' days')
	`
	nodeArgs := []interface{}{days}
	if flowID != "" {
		nodeQuery += " AND flow_id = ?"
		nodeArgs = append(nodeArgs, flowID)
	}
	nodeQuery += `)
		GROUP BY node_id, node_name, node_type
		ORDER BY total_duration DESC
	`

	rows, err := s.db.Query(nodeQuery, nodeArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get node stats: %w", err)
	}
	defer rows.Close()

	var totalNodeDuration int64 = 0
	for rows.Next() {
		stat := &NodePerformanceStats{}
		var nodeName, nodeType sql.NullString

		err := rows.Scan(
			&stat.NodeID, &nodeName, &nodeType,
			&stat.ExecutionCount, &stat.SuccessCount, &stat.FailedCount,
			&stat.AvgDuration, &stat.MaxDuration, &stat.MinDuration, &stat.TotalDuration,
			&stat.TotalTokensIn, &stat.TotalTokensOut, &stat.TotalCost,
		)
		if err != nil {
			continue
		}

		if nodeName.Valid {
			stat.NodeName = nodeName.String
		}
		if nodeType.Valid {
			stat.NodeType = nodeType.String
		}

		// 计算成功率
		if stat.ExecutionCount > 0 {
			stat.SuccessRate = float64(stat.SuccessCount) / float64(stat.ExecutionCount) * 100
		}

		analysis.NodeStats = append(analysis.NodeStats, stat)
		totalNodeDuration += stat.TotalDuration
	}

	// 识别瓶颈（耗时占比超过 20% 的节点）
	for _, stat := range analysis.NodeStats {
		if totalNodeDuration > 0 && float64(stat.TotalDuration)/float64(totalNodeDuration) > 0.2 {
			stat.IsBottleneck = true
			analysis.Bottlenecks = append(analysis.Bottlenecks, stat)
		}
	}

	// 找出最慢的节点（按平均耗时）
	if len(analysis.NodeStats) > 0 {
		sortedByAvg := make([]*NodePerformanceStats, len(analysis.NodeStats))
		copy(sortedByAvg, analysis.NodeStats)
		for i := 0; i < len(sortedByAvg)-1; i++ {
			for j := i + 1; j < len(sortedByAvg); j++ {
				if sortedByAvg[i].AvgDuration < sortedByAvg[j].AvgDuration {
					sortedByAvg[i], sortedByAvg[j] = sortedByAvg[j], sortedByAvg[i]
				}
			}
		}
		// 取前 5 个
		for i := 0; i < 5 && i < len(sortedByAvg); i++ {
			analysis.SlowestNodes = append(analysis.SlowestNodes, sortedByAvg[i])
		}
	}

	// 找出错误率最高的节点
	for _, stat := range analysis.NodeStats {
		if stat.FailedCount > 0 && stat.SuccessRate < 80 {
			analysis.ErrorProneNodes = append(analysis.ErrorProneNodes, stat)
		}
	}

	// 生成建议
	analysis.Recommendations = s.generateRecommendations(analysis)

	return analysis, nil
}

// generateRecommendations 生成优化建议
func (s *TraceService) generateRecommendations(analysis *PerformanceAnalysis) []string {
	recommendations := make([]string, 0)

	// 瓶颈建议
	for _, b := range analysis.Bottlenecks {
		if b.NodeType == "agent" {
			recommendations = append(recommendations, 
				fmt.Sprintf("节点 [%s] 是性能瓶颈，考虑优化 Agent 提示词或使用更快的模型", b.NodeName))
		} else if b.NodeType == "condition" {
			recommendations = append(recommendations,
				fmt.Sprintf("节点 [%s] 条件判断耗时较长，考虑简化条件逻辑", b.NodeName))
		} else if b.NodeType == "loop" {
			recommendations = append(recommendations,
				fmt.Sprintf("节点 [%s] 循环执行耗时较长，考虑减少循环次数或优化循环体", b.NodeName))
		} else {
			recommendations = append(recommendations,
				fmt.Sprintf("节点 [%s] 是性能瓶颈，占总耗时 %.1f%%", b.NodeName, 
					float64(b.TotalDuration)/float64(analysis.TotalDuration)*100))
		}
	}

	// 错误率建议
	for _, e := range analysis.ErrorProneNodes {
		recommendations = append(recommendations,
			fmt.Sprintf("节点 [%s] 错误率较高 (%.1f%%)，建议检查配置或添加重试机制", 
				e.NodeName, 100-e.SuccessRate))
	}

	// 总体建议
	if analysis.AvgDuration > 10000 { // 超过 10 秒
		recommendations = append(recommendations, 
			"流程平均执行时间较长，建议考虑并行执行或拆分子流程")
	}

	if len(analysis.NodeStats) > 10 {
		recommendations = append(recommendations,
			"流程节点较多，建议评估是否可以简化流程结构")
	}

	return recommendations
}

// GetDailyStats 获取每日统计
func (s *TraceService) GetDailyStats(flowID string, days int) ([]*DailyStats, error) {
	query := `
		SELECT 
			date(started_at) as date,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			AVG(duration) as avg_duration,
			SUM(total_tokens) as total_tokens,
			SUM(total_cost) as total_cost
		FROM flow_traces
		WHERE started_at >= datetime('now', '-' || ? || ' days')
	`
	args := []interface{}{days}

	if flowID != "" {
		query += " AND flow_id = ?"
		args = append(args, flowID)
	}

	query += " GROUP BY date(started_at) ORDER BY date"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []*DailyStats
	for rows.Next() {
		ds := &DailyStats{}
		err := rows.Scan(&ds.Date, &ds.Total, &ds.Completed, &ds.Failed, 
			&ds.AvgDuration, &ds.TotalTokens, &ds.TotalCost)
		if err != nil {
			continue
		}
		stats = append(stats, ds)
	}

	return stats, nil
}

// DailyStats 每日统计
type DailyStats struct {
	Date        string  `json:"date"`
	Total       int     `json:"total"`
	Completed   int     `json:"completed"`
	Failed      int     `json:"failed"`
	AvgDuration float64 `json:"avg_duration"`
	TotalTokens int     `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
}

// CostReport 成本报表
type CostReport struct {
	Period       string         `json:"period"`
	TotalCost    float64        `json:"total_cost"`
	TotalTokens  int            `json:"total_tokens"`
	TokensIn     int            `json:"tokens_in"`
	TokensOut    int            `json:"tokens_out"`
	Executions   int            `json:"executions"`
	AvgCostPerRun float64       `json:"avg_cost_per_run"`
	ByFlow       []*FlowCost    `json:"by_flow"`
	ByModel      []*ModelCost   `json:"by_model"`
	ByNode       []*NodeCost    `json:"by_node"`
	DailyTrend   []*DailyStats  `json:"daily_trend"`
}

// FlowCost 流程成本
type FlowCost struct {
	FlowID       string  `json:"flow_id"`
	FlowName     string  `json:"flow_name"`
	Executions   int     `json:"executions"`
	TotalCost    float64 `json:"total_cost"`
	TotalTokens  int     `json:"total_tokens"`
	AvgCostPerRun float64 `json:"avg_cost_per_run"`
}

// ModelCost 模型成本
type ModelCost struct {
	Model       string  `json:"model"`
	Calls       int     `json:"calls"`
	TotalCost   float64 `json:"total_cost"`
	TokensIn    int     `json:"tokens_in"`
	TokensOut   int     `json:"tokens_out"`
	AvgCostPerCall float64 `json:"avg_cost_per_call"`
}

// NodeCost 节点成本
type NodeCost struct {
	NodeType    string  `json:"node_type"`
	Executions  int     `json:"executions"`
	TotalCost   float64 `json:"total_cost"`
	TotalTokens int     `json:"total_tokens"`
}

// GetCostReport 获取成本报表
func (s *TraceService) GetCostReport(days int) (*CostReport, error) {
	report := &CostReport{
		Period:     fmt.Sprintf("最近 %d 天", days),
		ByFlow:     make([]*FlowCost, 0),
		ByModel:    make([]*ModelCost, 0),
		ByNode:     make([]*NodeCost, 0),
		DailyTrend: make([]*DailyStats, 0),
	}

	// 总体统计
	err := s.db.QueryRow(`
		SELECT 
			COALESCE(SUM(total_cost), 0),
			COALESCE(SUM(total_tokens), 0),
			COUNT(*)
		FROM flow_traces
		WHERE started_at >= datetime('now', '-' || ? || ' days')
	`, days).Scan(&report.TotalCost, &report.TotalTokens, &report.Executions)
	if err != nil {
		return nil, err
	}

	if report.Executions > 0 {
		report.AvgCostPerRun = report.TotalCost / float64(report.Executions)
	}

	// Token 统计
	s.db.QueryRow(`
		SELECT 
			COALESCE(SUM(tokens_in), 0),
			COALESCE(SUM(tokens_out), 0)
		FROM flow_spans
		WHERE trace_id IN (
			SELECT id FROM flow_traces 
			WHERE started_at >= datetime('now', '-' || ? || ' days')
		)
	`, days).Scan(&report.TokensIn, &report.TokensOut)

	// 按流程统计
	flowRows, err := s.db.Query(`
		SELECT 
			t.flow_id,
			f.name as flow_name,
			COUNT(*) as executions,
			COALESCE(SUM(t.total_cost), 0) as total_cost,
			COALESCE(SUM(t.total_tokens), 0) as total_tokens
		FROM flow_traces t
		LEFT JOIN flows f ON t.flow_id = f.id
		WHERE t.started_at >= datetime('now', '-' || ? || ' days')
		GROUP BY t.flow_id
		ORDER BY total_cost DESC
		LIMIT 20
	`, days)
	if err == nil {
		defer flowRows.Close()
		for flowRows.Next() {
			fc := &FlowCost{}
			var flowName sql.NullString
			err := flowRows.Scan(&fc.FlowID, &flowName, &fc.Executions, &fc.TotalCost, &fc.TotalTokens)
			if err != nil {
				continue
			}
			if flowName.Valid {
				fc.FlowName = flowName.String
			}
			if fc.Executions > 0 {
				fc.AvgCostPerRun = fc.TotalCost / float64(fc.Executions)
			}
			report.ByFlow = append(report.ByFlow, fc)
		}
	}

	// 按模型统计
	modelRows, err := s.db.Query(`
		SELECT 
			model,
			COUNT(*) as calls,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(SUM(tokens_in), 0) as tokens_in,
			COALESCE(SUM(tokens_out), 0) as tokens_out
		FROM flow_spans
		WHERE trace_id IN (
			SELECT id FROM flow_traces 
			WHERE started_at >= datetime('now', '-' || ? || ' days')
		) AND model != '' AND model IS NOT NULL
		GROUP BY model
		ORDER BY total_cost DESC
		LIMIT 20
	`, days)
	if err == nil {
		defer modelRows.Close()
		for modelRows.Next() {
			mc := &ModelCost{}
			err := modelRows.Scan(&mc.Model, &mc.Calls, &mc.TotalCost, &mc.TokensIn, &mc.TokensOut)
			if err != nil {
				continue
			}
			if mc.Calls > 0 {
				mc.AvgCostPerCall = mc.TotalCost / float64(mc.Calls)
			}
			report.ByModel = append(report.ByModel, mc)
		}
	}

	// 按节点类型统计
	nodeRows, err := s.db.Query(`
		SELECT 
			node_type,
			COUNT(*) as executions,
			COALESCE(SUM(cost), 0) as total_cost,
			COALESCE(SUM(tokens_in) + SUM(tokens_out), 0) as total_tokens
		FROM flow_spans
		WHERE trace_id IN (
			SELECT id FROM flow_traces 
			WHERE started_at >= datetime('now', '-' || ? || ' days')
		)
		GROUP BY node_type
		ORDER BY total_cost DESC
	`, days)
	if err == nil {
		defer nodeRows.Close()
		for nodeRows.Next() {
			nc := &NodeCost{}
			err := nodeRows.Scan(&nc.NodeType, &nc.Executions, &nc.TotalCost, &nc.TotalTokens)
			if err != nil {
				continue
			}
			report.ByNode = append(report.ByNode, nc)
		}
	}

	// 每日趋势
	dailyStats, err := s.GetDailyStats("", days)
	if err == nil {
		report.DailyTrend = dailyStats
	}

	return report, nil
}