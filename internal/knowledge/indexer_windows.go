// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

//go:build windows

package knowledge

import (
	"context"
	"database/sql"
	"sync"

	"github.com/gopaw/gopaw/internal/embedding"
)

// VectorIndexer 向量索引管理器 (Windows 存根实现)
// 注意：由于 hnsw 库在 Windows 上不可用，此实现不提供向量索引功能
type VectorIndexer struct {
	mu      sync.RWMutex
	db      *sql.DB
	encoder embedding.Encoder
}

// NewVectorIndexer 创建向量索引管理器
func NewVectorIndexer(db *sql.DB, encoder embedding.Encoder) *VectorIndexer {
	return &VectorIndexer{
		db:      db,
		encoder: encoder,
	}
}

// GetOrCreateGraph 获取或创建知识库的向量图 (Windows 上不可用)
func (v *VectorIndexer) GetOrCreateGraph(ctx context.Context, kbID string) (interface{}, error) {
	return nil, nil // Windows 上返回 nil，不提供向量索引
}

// AddChunk 添加向量块 (Windows 上不可用)
func (v *VectorIndexer) AddChunk(ctx context.Context, kbID, chunkID, content string, vec []float32) error {
	return nil // 静默忽略，不阻塞流程
}

// RemoveChunk 移除向量块 (Windows 上不可用)
func (v *VectorIndexer) RemoveChunk(ctx context.Context, kbID, chunkID string) error {
	return nil
}

// Search 搜索相似向量 (Windows 上降级为数据库 LIKE 查询)
func (v *VectorIndexer) Search(ctx context.Context, kbID string, query string, k int) ([]SearchResult, error) {
	// Windows 上降级为数据库模糊搜索
	var results []SearchResult
	rows, err := v.db.QueryContext(ctx, `
		SELECT kc.chunk_id, kc.content, kc.document_id, COALESCE(d.name, '')
		FROM knowledge_chunks kc
		LEFT JOIN documents d ON kc.document_id = d.id
		WHERE kc.kb_id = ? AND kc.content LIKE ?
		LIMIT ?
	`, kbID, "%"+query+"%", k)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r SearchResult
		var docID, docName string
		if err := rows.Scan(&r.ChunkID, &r.Content, &docID, &docName); err != nil {
			continue
		}
		r.DocumentID = docID
		r.DocumentName = docName
		r.Distance = 0.5 // 默认距离
		results = append(results, r)
	}
	return results, nil
}

// DeleteGraph 删除知识库的向量图
func (v *VectorIndexer) DeleteGraph(kbID string) error {
	return nil
}

// DeleteIndex 删除知识库的向量索引
func (v *VectorIndexer) DeleteIndex(kbID string) {
	// Windows 上无操作
}