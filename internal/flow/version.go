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