// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package flow

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// FlowVersion 流程版本
type FlowVersion struct {
	ID          string         `json:"id" db:"id"`
	FlowID      string         `json:"flow_id" db:"flow_id"`
	Version     int            `json:"version" db:"version"`           // 版本号，递增
	Name        string         `json:"name" db:"name"`                 // 版本名称（可选）
	Description string         `json:"description" db:"description"`   // 版本描述
	Definition  FlowDefinition `json:"definition" db:"definition"`     // 流程定义快照
	Trigger     *TriggerConfig `json:"trigger,omitempty" db:"trigger"` // 触发器配置快照
	CreatedAt   time.Time      `json:"created_at" db:"created_at"`
	CreatedBy   string         `json:"created_by" db:"created_by"` // 创建者
}

// CreateVersionRequest 创建版本请求
type CreateVersionRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

// VersionService 版本服务
type VersionService struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewVersionService 创建版本服务
func NewVersionService(db *sql.DB, logger *zap.Logger) *VersionService {
	return &VersionService{db: db, logger: logger.Named("version")}
}

// InitVersionSchema 初始化版本表
func InitVersionSchema(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS flow_versions (
			id TEXT PRIMARY KEY,
			flow_id TEXT NOT NULL,
			version INTEGER NOT NULL,
			name TEXT,
			description TEXT,
			definition TEXT NOT NULL,
			trigger TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			created_by TEXT,
			FOREIGN KEY (flow_id) REFERENCES flows(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_versions_flow ON flow_versions(flow_id)`,
		`CREATE INDEX IF NOT EXISTS idx_versions_version ON flow_versions(flow_id, version)`,
	}

	for _, idx := range indexes {
		if _, err := db.Exec(idx); err != nil {
			return err
		}
	}

	return nil
}

// CreateVersion 创建新版本
func (s *VersionService) CreateVersion(flowID string, req CreateVersionRequest, createdBy string) (*FlowVersion, error) {
	// 获取当前流程
	var flow Flow
	var defJSON, triggerJSON sql.NullString
	err := s.db.QueryRow(`
		SELECT id, name, definition, trigger FROM flows WHERE id = ?
	`, flowID).Scan(&flow.ID, &flow.Name, &defJSON, &triggerJSON)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("flow not found: %s", flowID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get flow: %w", err)
	}

	// 解析定义
	if defJSON.Valid {
		if err := json.Unmarshal([]byte(defJSON.String), &flow.Definition); err != nil {
			return nil, fmt.Errorf("failed to parse definition: %w", err)
		}
	}

	// 解析触发器
	if triggerJSON.Valid && triggerJSON.String != "" {
		var trigger TriggerConfig
		if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err != nil {
			s.logger.Warn("failed to parse trigger", zap.Error(err))
		} else {
			flow.Trigger = &trigger
		}
	}

	// 获取下一个版本号
	var maxVersion int
	err = s.db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM flow_versions WHERE flow_id = ?`, flowID).Scan(&maxVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get max version: %w", err)
	}

	newVersion := maxVersion + 1
	versionID := fmt.Sprintf("ver_%s_%d", flowID[:8], newVersion)

	// 序列化定义
	defBytes, err := json.Marshal(flow.Definition)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal definition: %w", err)
	}

	// 序列化触发器
	var triggerBytes []byte
	if flow.Trigger != nil {
		triggerBytes, _ = json.Marshal(flow.Trigger)
	}

	// 插入版本记录
	_, err = s.db.Exec(`
		INSERT INTO flow_versions (id, flow_id, version, name, description, definition, trigger, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, versionID, flowID, newVersion, req.Name, req.Description, string(defBytes), string(triggerBytes), time.Now(), createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return &FlowVersion{
		ID:          versionID,
		FlowID:      flowID,
		Version:     newVersion,
		Name:        req.Name,
		Description: req.Description,
		Definition:  flow.Definition,
		Trigger:     flow.Trigger,
		CreatedAt:   time.Now(),
		CreatedBy:   createdBy,
	}, nil
}

// ListVersions 列出流程的所有版本
func (s *VersionService) ListVersions(flowID string) ([]*FlowVersion, error) {
	rows, err := s.db.Query(`
		SELECT id, flow_id, version, name, description, definition, trigger, created_at, created_by
		FROM flow_versions
		WHERE flow_id = ?
		ORDER BY version DESC
	`, flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []*FlowVersion
	for rows.Next() {
		v := &FlowVersion{}
		var defJSON, triggerJSON sql.NullString
		var name, description, createdBy sql.NullString

		err := rows.Scan(&v.ID, &v.FlowID, &v.Version, &name, &description, &defJSON, &triggerJSON, &v.CreatedAt, &createdBy)
		if err != nil {
			continue
		}

		if name.Valid {
			v.Name = name.String
		}
		if description.Valid {
			v.Description = description.String
		}
		if createdBy.Valid {
			v.CreatedBy = createdBy.String
		}

		if defJSON.Valid {
			json.Unmarshal([]byte(defJSON.String), &v.Definition)
		}
		if triggerJSON.Valid && triggerJSON.String != "" {
			var trigger TriggerConfig
			if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err == nil {
				v.Trigger = &trigger
			}
		}

		versions = append(versions, v)
	}

	return versions, nil
}

// GetVersion 获取特定版本
func (s *VersionService) GetVersion(flowID string, version int) (*FlowVersion, error) {
	v := &FlowVersion{}
	var defJSON, triggerJSON sql.NullString
	var name, description, createdBy sql.NullString

	err := s.db.QueryRow(`
		SELECT id, flow_id, version, name, description, definition, trigger, created_at, created_by
		FROM flow_versions
		WHERE flow_id = ? AND version = ?
	`, flowID, version).Scan(&v.ID, &v.FlowID, &v.Version, &name, &description, &defJSON, &triggerJSON, &v.CreatedAt, &createdBy)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("version not found: flow=%s, version=%d", flowID, version)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	if name.Valid {
		v.Name = name.String
	}
	if description.Valid {
		v.Description = description.String
	}
	if createdBy.Valid {
		v.CreatedBy = createdBy.String
	}

	if defJSON.Valid {
		json.Unmarshal([]byte(defJSON.String), &v.Definition)
	}
	if triggerJSON.Valid && triggerJSON.String != "" {
		var trigger TriggerConfig
		if err := json.Unmarshal([]byte(triggerJSON.String), &trigger); err == nil {
			v.Trigger = &trigger
		}
	}

	return v, nil
}

// RollbackVersion 回滚到指定版本
func (s *VersionService) RollbackVersion(flowID string, version int, service *Service) (*Flow, error) {
	// 获取目标版本
	targetVersion, err := s.GetVersion(flowID, version)
	if err != nil {
		return nil, err
	}

	// 更新流程定义
	defBytes, _ := json.Marshal(targetVersion.Definition)
	var triggerBytes []byte
	if targetVersion.Trigger != nil {
		triggerBytes, _ = json.Marshal(targetVersion.Trigger)
	}

	_, err = s.db.Exec(`
		UPDATE flows SET definition = ?, trigger = ?, updated_at = ? WHERE id = ?
	`, string(defBytes), string(triggerBytes), time.Now(), flowID)
	if err != nil {
		return nil, fmt.Errorf("failed to rollback flow: %w", err)
	}

	// 返回更新后的流程
	return service.GetFlow(flowID)
}

// DeleteVersion 删除指定版本
func (s *VersionService) DeleteVersion(flowID string, version int) error {
	result, err := s.db.Exec(`DELETE FROM flow_versions WHERE flow_id = ? AND version = ?`, flowID, version)
	if err != nil {
		return fmt.Errorf("failed to delete version: %w", err)
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		return fmt.Errorf("version not found: flow=%s, version=%d", flowID, version)
	}

	return nil
}

// AutoSaveVersion 自动保存版本（在流程更新时调用）
func (s *VersionService) AutoSaveVersion(flowID string, createdBy string) (*FlowVersion, error) {
	return s.CreateVersion(flowID, CreateVersionRequest{
		Name:        "自动保存",
		Description: "流程更新时自动创建",
	}, createdBy)
}

// ========== 版本对比 ==========

// VersionDiff 版本差异
type VersionDiff struct {
	FlowID      string        `json:"flow_id"`
	VersionFrom int           `json:"version_from"`
	VersionTo   int           `json:"version_to"`
	FromName    string        `json:"from_name"`
	ToName      string        `json:"to_name"`
	NodeChanges []NodeChange  `json:"node_changes"`
	EdgeChanges []EdgeChange  `json:"edge_changes"`
	VarChanges  []VarChange   `json:"var_changes"`
	TriggerDiff *TriggerDiff  `json:"trigger_diff,omitempty"`
	Summary     DiffSummary   `json:"summary"`
}

// NodeChange 节点变更
type NodeChange struct {
	Type       string      `json:"type"` // added, removed, modified, moved
	NodeID     string      `json:"node_id"`
	NodeName   string      `json:"node_name"`
	NodeType   string      `json:"node_type"`
	OldValue   interface{} `json:"old_value,omitempty"`
	NewValue   interface{} `json:"new_value,omitempty"`
	Changes    []string    `json:"changes,omitempty"` // 具体变更的字段
}

// EdgeChange 连线变更
type EdgeChange struct {
	Type     string `json:"type"` // added, removed, modified
	EdgeID   string `json:"edge_id"`
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// VarChange 变量变更
type VarChange struct {
	Type     string `json:"type"` // added, removed, modified
	VarName  string `json:"var_name"`
	OldValue string `json:"old_value,omitempty"`
	NewValue string `json:"new_value,omitempty"`
}

// TriggerDiff 触发器变更
type TriggerDiff struct {
	TypeChanged bool   `json:"type_changed,omitempty"`
	OldType     string `json:"old_type,omitempty"`
	NewType     string `json:"new_type,omitempty"`
	ConfigDiff  string `json:"config_diff,omitempty"`
}

// DiffSummary 差异摘要
type DiffSummary struct {
	NodesAdded    int `json:"nodes_added"`
	NodesRemoved  int `json:"nodes_removed"`
	NodesModified int `json:"nodes_modified"`
	EdgesAdded    int `json:"edges_added"`
	EdgesRemoved  int `json:"edges_removed"`
	VarsChanged   int `json:"vars_changed"`
	TriggerChanged bool `json:"trigger_changed"`
}

// CompareVersions 对比两个版本
func (s *VersionService) CompareVersions(flowID string, versionFrom, versionTo int) (*VersionDiff, error) {
	// 获取两个版本
	from, err := s.GetVersion(flowID, versionFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", versionFrom, err)
	}

	to, err := s.GetVersion(flowID, versionTo)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", versionTo, err)
	}

	diff := &VersionDiff{
		FlowID:      flowID,
		VersionFrom: versionFrom,
		VersionTo:   versionTo,
		FromName:    from.Name,
		ToName:      to.Name,
		NodeChanges: make([]NodeChange, 0),
		EdgeChanges: make([]EdgeChange, 0),
		VarChanges:  make([]VarChange, 0),
	}

	// 对比节点
	fromNodes := make(map[string]FlowNode)
	toNodes := make(map[string]FlowNode)
	for _, n := range from.Definition.Nodes {
		fromNodes[n.ID] = n
	}
	for _, n := range to.Definition.Nodes {
		toNodes[n.ID] = n
	}

	// 检查新增和修改的节点
	for id, toNode := range toNodes {
		if fromNode, exists := fromNodes[id]; exists {
			// 检查是否修改
			changes := s.compareNodeValues(fromNode, toNode)
			if len(changes) > 0 {
				diff.NodeChanges = append(diff.NodeChanges, NodeChange{
					Type:     "modified",
					NodeID:   id,
					NodeName: toNode.Name,
					NodeType: string(toNode.Type),
					Changes:  changes,
				})
				diff.Summary.NodesModified++
			}
		} else {
			// 新增节点
			diff.NodeChanges = append(diff.NodeChanges, NodeChange{
				Type:     "added",
				NodeID:   id,
				NodeName: toNode.Name,
				NodeType: string(toNode.Type),
				NewValue: toNode,
			})
			diff.Summary.NodesAdded++
		}
	}

	// 检查删除的节点
	for id, fromNode := range fromNodes {
		if _, exists := toNodes[id]; !exists {
			diff.NodeChanges = append(diff.NodeChanges, NodeChange{
				Type:     "removed",
				NodeID:   id,
				NodeName: fromNode.Name,
				NodeType: string(fromNode.Type),
				OldValue: fromNode,
			})
			diff.Summary.NodesRemoved++
		}
	}

	// 对比连线
	fromEdges := make(map[string]FlowEdge)
	toEdges := make(map[string]FlowEdge)
	for _, e := range from.Definition.Edges {
		fromEdges[e.ID] = e
	}
	for _, e := range to.Definition.Edges {
		toEdges[e.ID] = e
	}

	for id, toEdge := range toEdges {
		if _, exists := fromEdges[id]; !exists {
			diff.EdgeChanges = append(diff.EdgeChanges, EdgeChange{
				Type:     "added",
				EdgeID:   id,
				NewValue: fmt.Sprintf("%s -> %s", toEdge.Source, toEdge.Target),
			})
			diff.Summary.EdgesAdded++
		}
	}

	for id, fromEdge := range fromEdges {
		if _, exists := toEdges[id]; !exists {
			diff.EdgeChanges = append(diff.EdgeChanges, EdgeChange{
				Type:     "removed",
				EdgeID:   id,
				OldValue: fmt.Sprintf("%s -> %s", fromEdge.Source, fromEdge.Target),
			})
			diff.Summary.EdgesRemoved++
		}
	}

	// 对比触发器
	if from.Trigger != nil || to.Trigger != nil {
		if from.Trigger == nil && to.Trigger != nil {
			diff.TriggerDiff = &TriggerDiff{
				TypeChanged: true,
				NewType:     string(to.Trigger.Type),
			}
			diff.Summary.TriggerChanged = true
		} else if from.Trigger != nil && to.Trigger == nil {
			diff.TriggerDiff = &TriggerDiff{
				TypeChanged: true,
				OldType:     string(from.Trigger.Type),
			}
			diff.Summary.TriggerChanged = true
		} else if from.Trigger != nil && to.Trigger != nil && from.Trigger.Type != to.Trigger.Type {
			diff.TriggerDiff = &TriggerDiff{
				TypeChanged: true,
				OldType:     string(from.Trigger.Type),
				NewType:     string(to.Trigger.Type),
			}
			diff.Summary.TriggerChanged = true
		}
	}

	return diff, nil
}

// compareNodeValues 对比节点值
func (s *VersionService) compareNodeValues(from, to FlowNode) []string {
	changes := make([]string, 0)

	if from.Name != to.Name {
		changes = append(changes, fmt.Sprintf("名称: %s → %s", from.Name, to.Name))
	}
	if from.Type != to.Type {
		changes = append(changes, fmt.Sprintf("类型: %s → %s", from.Type, to.Type))
	}
	if from.AgentID != to.AgentID {
		changes = append(changes, fmt.Sprintf("Agent: %s → %s", from.AgentID, to.AgentID))
	}
	if from.Prompt != to.Prompt {
		changes = append(changes, "Prompt 已修改")
	}
	if from.Role != to.Role {
		changes = append(changes, fmt.Sprintf("角色: %s → %s", from.Role, to.Role))
	}
	if from.Position.X != to.Position.X || from.Position.Y != to.Position.Y {
		changes = append(changes, "位置已移动")
	}
	if from.RetryConfig != nil || to.RetryConfig != nil {
		if from.RetryConfig == nil && to.RetryConfig != nil {
			changes = append(changes, "添加了重试配置")
		} else if from.RetryConfig != nil && to.RetryConfig == nil {
			changes = append(changes, "移除了重试配置")
		} else if from.RetryConfig != nil && to.RetryConfig != nil {
			if from.RetryConfig.MaxRetries != to.RetryConfig.MaxRetries {
				changes = append(changes, fmt.Sprintf("重试次数: %d → %d", from.RetryConfig.MaxRetries, to.RetryConfig.MaxRetries))
			}
		}
	}

	return changes
}