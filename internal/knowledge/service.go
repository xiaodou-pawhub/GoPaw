package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gopaw/gopaw/internal/embedding"
)

// Service 知识库服务
type Service struct {
	db       *sql.DB
	encoder  embedding.Encoder
	processor *DocumentProcessor
	indexer  *VectorIndexer
}

// NewService 创建知识库服务
func NewService(db *sql.DB) *Service {
	encoder := embedding.GetDefaultEncoder()
	indexer := NewVectorIndexer(db, encoder)
	return &Service{
		db:        db,
		encoder:   encoder,
		processor: NewDocumentProcessor(db, encoder, ChunkByMarkdown, indexer),
		indexer:   indexer,
	}
}

// CreateKnowledgeBase 创建知识库
func (s *Service) CreateKnowledgeBase(ctx context.Context, req CreateKnowledgeBaseRequest) (*KnowledgeBase, error) {
	// 自动生成 ID
	kbID := fmt.Sprintf("kb_%d", time.Now().UnixNano())
	
	kb := &KnowledgeBase{
		ID:          kbID,
		Name:        req.Name,
		Description: req.Description,
		Mode:        req.Mode,
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO knowledge_bases (id, name, description, mode, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, kb.ID, kb.Name, kb.Description, kb.Mode, kb.Status, kb.CreatedAt, kb.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge base: %w", err)
	}

	return kb, nil
}

// GetKnowledgeBase 获取知识库
func (s *Service) GetKnowledgeBase(ctx context.Context, id string) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, mode, status,
			document_count, chunk_count, total_tokens, created_at, updated_at
		FROM knowledge_bases WHERE id = ?
	`, id).Scan(
		&kb.ID, &kb.Name, &kb.Description, &kb.Mode, &kb.Status,
		&kb.DocumentCount, &kb.ChunkCount, &kb.TotalTokens, &kb.CreatedAt, &kb.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &kb, nil
}

// ListKnowledgeBases 列出知识库
func (s *Service) ListKnowledgeBases(ctx context.Context) ([]KnowledgeBase, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, mode, status,
			document_count, chunk_count, total_tokens, created_at, updated_at
		FROM knowledge_bases ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bases []KnowledgeBase
	for rows.Next() {
		var kb KnowledgeBase
		err := rows.Scan(
			&kb.ID, &kb.Name, &kb.Description, &kb.Mode, &kb.Status,
			&kb.DocumentCount, &kb.ChunkCount, &kb.TotalTokens, &kb.CreatedAt, &kb.UpdatedAt,
		)
		if err != nil {
			continue
		}
		bases = append(bases, kb)
	}

	return bases, rows.Err()
}

// UpdateKnowledgeBase 更新知识库
func (s *Service) UpdateKnowledgeBase(ctx context.Context, id string, req UpdateKnowledgeBaseRequest) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE knowledge_bases SET
			name = COALESCE(NULLIF(?, ''), name),
			description = COALESCE(NULLIF(?, ''), description),
			mode = COALESCE(NULLIF(?, ''), mode),
			status = COALESCE(NULLIF(?, ''), status),
			updated_at = ?
		WHERE id = ?
	`, req.Name, req.Description, req.Mode, req.Status, time.Now(), id)
	return err
}

// DeleteKnowledgeBase 删除知识库
func (s *Service) DeleteKnowledgeBase(ctx context.Context, id string) error {
	// 删除知识库会级联删除文档和块
	_, err := s.db.ExecContext(ctx, "DELETE FROM knowledge_bases WHERE id = ?", id)
	if err != nil {
		return err
	}

	// 删除向量索引
	s.indexer.DeleteIndex(id)
	return nil
}

// UploadDocument 上传文档
func (s *Service) UploadDocument(ctx context.Context, kbID string, filename string, fileType string, content []byte) (*Document, error) {
	docID := fmt.Sprintf("doc_%d", time.Now().UnixNano())

	doc := &Document{
		ID:              docID,
		KnowledgeBaseID: kbID,
		Filename:        filename,
		FileType:        fileType,
		FileSize:        int64(len(content)),
		FileHash:        CalculateFileHash(content),
		Content:         content,
		Status:          "pending",
		CreatedAt:       time.Now(),
	}

	// 检查是否已存在相同哈希的文档
	var existingID string
	err := s.db.QueryRowContext(ctx,
		"SELECT id FROM knowledge_documents WHERE knowledge_base_id = ? AND file_hash = ?",
		kbID, doc.FileHash).Scan(&existingID)
	
	// 如果存在相同哈希的文档，返回已存在的文档（不报错）
	if err == nil && existingID != "" {
		return s.GetDocument(ctx, existingID)
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO knowledge_documents (id, knowledge_base_id, filename, file_type, file_size, file_hash, content, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, doc.ID, doc.KnowledgeBaseID, doc.Filename, doc.FileType, doc.FileSize, doc.FileHash, doc.Content, doc.Status, doc.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to upload document: %w", err)
	}

	// 异步处理文档
	go s.processDocument(context.Background(), doc.ID)

	return doc, nil
}

// processDocument 处理文档
func (s *Service) processDocument(ctx context.Context, docID string) {
	if err := s.processor.Process(ctx, docID); err != nil {
		// 记录错误日志
		fmt.Printf("Failed to process document %s: %v\n", docID, err)
	}
}

// GetDocument 获取文档
func (s *Service) GetDocument(ctx context.Context, id string) (*Document, error) {
	var doc Document
	var metadataStr sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, knowledge_base_id, filename, file_type, file_size, file_hash, metadata, status, error_message, chunk_count, processed_at, created_at
		FROM knowledge_documents WHERE id = ?
	`, id).Scan(
		&doc.ID, &doc.KnowledgeBaseID, &doc.Filename, &doc.FileType, &doc.FileSize, &doc.FileHash,
		&metadataStr, &doc.Status, &doc.ErrorMessage, &doc.ChunkCount, &doc.ProcessedAt, &doc.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if metadataStr.Valid && metadataStr.String != "" {
		doc.Metadata.Scan(metadataStr.String)
	}

	return &doc, nil
}

// ListDocuments 列出知识库的文档
func (s *Service) ListDocuments(ctx context.Context, kbID string) ([]Document, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, knowledge_base_id, filename, file_type, file_size, file_hash, metadata, status, error_message, chunk_count, processed_at, created_at
		FROM knowledge_documents WHERE knowledge_base_id = ? ORDER BY created_at DESC
	`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var docs []Document
	for rows.Next() {
		var doc Document
		var metadataStr sql.NullString
		err := rows.Scan(
			&doc.ID, &doc.KnowledgeBaseID, &doc.Filename, &doc.FileType, &doc.FileSize, &doc.FileHash,
			&metadataStr, &doc.Status, &doc.ErrorMessage, &doc.ChunkCount, &doc.ProcessedAt, &doc.CreatedAt,
		)
		if err != nil {
			continue
		}

		if metadataStr.Valid && metadataStr.String != "" {
			doc.Metadata.Scan(metadataStr.String)
		}

		docs = append(docs, doc)
	}

	return docs, rows.Err()
}

// DeleteDocument 删除文档
func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 获取知识库 ID
	var kbID string
	err = tx.QueryRowContext(ctx, "SELECT knowledge_base_id FROM knowledge_documents WHERE id = ?", id).Scan(&kbID)
	if err != nil {
		return err
	}

	// 删除文档（级联删除块）
	_, err = tx.ExecContext(ctx, "DELETE FROM knowledge_documents WHERE id = ?", id)
	if err != nil {
		return err
	}

	// 更新知识库统计
	_, err = tx.ExecContext(ctx, `
		UPDATE knowledge_bases SET
			document_count = (SELECT COUNT(*) FROM knowledge_documents WHERE knowledge_base_id = ? AND status = 'completed'),
			chunk_count = (SELECT COUNT(*) FROM knowledge_chunks WHERE knowledge_base_id = ?),
			total_tokens = (SELECT COALESCE(SUM(token_count), 0) FROM knowledge_chunks WHERE knowledge_base_id = ?),
			updated_at = ?
		WHERE id = ?
	`, kbID, kbID, kbID, time.Now(), kbID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// RetryDocument 重试处理文档
func (s *Service) RetryDocument(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE knowledge_documents SET status = 'pending', error_message = '' WHERE id = ?",
		id)
	if err != nil {
		return err
	}

	// 异步处理
	go s.processDocument(context.Background(), id)
	return nil
}

// Search 搜索知识库
func (s *Service) Search(ctx context.Context, kbID string, req SearchRequest) (*SearchResponse, error) {
	if req.TopK <= 0 {
		req.TopK = 5
	}

	// 使用 HNSW 索引进行向量搜索
	results, err := s.indexer.Search(ctx, kbID, req.Query, req.TopK)
	if err != nil {
		return nil, err
	}

	return &SearchResponse{
		Results: results,
		Total:   len(results),
	}, nil
}

// GetStats 获取知识库统计
func (s *Service) GetStats(ctx context.Context, kbID string) (map[string]interface{}, error) {
	var stats struct {
		DocumentCount int
		ChunkCount    int
		TotalTokens   int
	}

	err := s.db.QueryRowContext(ctx, `
		SELECT document_count, chunk_count, total_tokens
		FROM knowledge_bases WHERE id = ?
	`, kbID).Scan(&stats.DocumentCount, &stats.ChunkCount, &stats.TotalTokens)
	if err != nil {
		return nil, err
	}

	// 获取文档状态统计
	var pendingCount, completedCount, failedCount int
	rows, err := s.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM knowledge_documents WHERE knowledge_base_id = ? GROUP BY status
	`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			continue
		}
		switch status {
		case "pending":
			pendingCount = count
		case "completed":
			completedCount = count
		case "failed":
			failedCount = count
		}
	}

	return map[string]interface{}{
		"document_count":   stats.DocumentCount,
		"chunk_count":      stats.ChunkCount,
		"total_tokens":     stats.TotalTokens,
		"pending_count":    pendingCount,
		"completed_count":  completedCount,
		"failed_count":     failedCount,
	}, nil
}

// GetGlobalInjectContent 获取所有全局注入模式的知识库内容
func (s *Service) GetGlobalInjectContent(ctx context.Context) (string, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT kb.name, d.content
		FROM knowledge_bases kb
		JOIN knowledge_documents d ON d.knowledge_base_id = kb.id
		WHERE kb.mode = 'inject' AND kb.status = 'active' AND d.status = 'completed'
		ORDER BY kb.updated_at DESC
	`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var parts []string
	for rows.Next() {
		var name string
		var content []byte
		if err := rows.Scan(&name, &content); err != nil {
			continue
		}
		parts = append(parts, fmt.Sprintf("## %s\n\n%s", name, string(content)))
	}

	if len(parts) == 0 {
		return "", nil
	}

	return strings.Join(parts, "\n\n---\n\n"), nil
}

// ========== 版本管理 ==========

// CreateDocumentVersion 创建文档版本
func (s *Service) CreateDocumentVersion(ctx context.Context, docID, changeNote, createdBy string) (*DocumentVersion, error) {
	// 获取文档
	doc, err := s.GetDocument(ctx, docID)
	if err != nil {
		return nil, err
	}

	// 获取下一个版本号
	var maxVersion int
	err = s.db.QueryRowContext(ctx,
		"SELECT COALESCE(MAX(version), 0) FROM document_versions WHERE document_id = ?",
		docID).Scan(&maxVersion)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	nextVersion := maxVersion + 1
	versionID := fmt.Sprintf("dv_%d", time.Now().UnixNano())

	version := &DocumentVersion{
		ID:         versionID,
		DocumentID: docID,
		Version:    nextVersion,
		FileHash:   doc.FileHash,
		Content:    doc.Content,
		ChangeType: "updated",
		ChangeNote: changeNote,
		CreatedAt:  time.Now(),
		CreatedBy:  createdBy,
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO document_versions (id, document_id, version, file_hash, content, change_type, change_note, created_at, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, version.ID, version.DocumentID, version.Version, version.FileHash, version.Content, version.ChangeType, version.ChangeNote, version.CreatedAt, version.CreatedBy)

	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	return version, nil
}

// ListDocumentVersions 列出文档版本
func (s *Service) ListDocumentVersions(ctx context.Context, docID string) ([]DocumentVersion, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, document_id, version, file_hash, change_type, change_note, created_at, created_by
		FROM document_versions
		WHERE document_id = ?
		ORDER BY version DESC
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var versions []DocumentVersion
	for rows.Next() {
		var v DocumentVersion
		var createdBy sql.NullString
		err := rows.Scan(&v.ID, &v.DocumentID, &v.Version, &v.FileHash, &v.ChangeType, &v.ChangeNote, &v.CreatedAt, &createdBy)
		if err != nil {
			continue
		}
		if createdBy.Valid {
			v.CreatedBy = createdBy.String
		}
		versions = append(versions, v)
	}

	return versions, nil
}

// RollbackDocumentVersion 回滚到指定版本
func (s *Service) RollbackDocumentVersion(ctx context.Context, docID string, version int) (*Document, error) {
	// 获取版本内容
	var content []byte
	var fileHash string
	err := s.db.QueryRowContext(ctx,
		"SELECT content, file_hash FROM document_versions WHERE document_id = ? AND version = ?",
		docID, version).Scan(&content, &fileHash)
	if err != nil {
		return nil, fmt.Errorf("version not found: %w", err)
	}

	// 更新文档
	_, err = s.db.ExecContext(ctx, `
		UPDATE knowledge_documents SET content = ?, file_hash = ?, status = 'pending', error_message = ''
		WHERE id = ?
	`, content, fileHash, docID)
	if err != nil {
		return nil, fmt.Errorf("failed to update document: %w", err)
	}

	// 创建回滚版本记录
	_, _ = s.CreateDocumentVersion(ctx, docID, fmt.Sprintf("Rollback to version %d", version), "")

	// 重新处理文档
	go s.processDocument(context.Background(), docID)

	return s.GetDocument(ctx, docID)
}

// ========== 统计功能 ==========

// GetKnowledgeStats 获取知识库统计
func (s *Service) GetKnowledgeStats(ctx context.Context, kbID string) (*KnowledgeStats, error) {
	stats := &KnowledgeStats{KnowledgeBaseID: kbID}

	// 文档统计
	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(file_size), 0) as total_size,
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0) as processed,
			COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END), 0) as pending,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed,
			MAX(updated_at) as last_updated
		FROM knowledge_documents WHERE knowledge_base_id = ?
	`, kbID).Scan(&stats.DocumentCount, &stats.TotalSize, &stats.ProcessedCount, &stats.PendingCount, &stats.FailedCount, &stats.LastUpdated)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// 切片统计
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(token_count), 0)
		FROM knowledge_chunks WHERE knowledge_base_id = ?
	`, kbID).Scan(&stats.ChunkCount, &stats.TotalTokens)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// 平均切片大小
	if stats.ChunkCount > 0 {
		stats.AvgChunkSize = float64(stats.TotalTokens) / float64(stats.ChunkCount)
	}

	return stats, nil
}

// GetQueryStats 获取查询统计
func (s *Service) GetQueryStats(ctx context.Context, kbID string, days int) (*QueryStats, error) {
	stats := &QueryStats{KnowledgeBaseID: kbID}

	err := s.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*) as total,
			COALESCE(AVG(latency_ms), 0) as avg_latency,
			COALESCE(AVG(result_count), 0) as avg_results,
			MAX(created_at) as last_queried
		FROM knowledge_query_stats
		WHERE knowledge_base_id = ? AND created_at >= datetime('now', '-' || ? || ' days')
	`, kbID, days).Scan(&stats.TotalQueries, &stats.AvgLatencyMs, &stats.AvgResultCount, &stats.LastQueriedAt)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return stats, nil
}

// RecordQuery 记录查询
func (s *Service) RecordQuery(ctx context.Context, kbID, queryText string, resultCount int, latencyMs int64) error {
	id := fmt.Sprintf("qs_%d", time.Now().UnixNano())
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO knowledge_query_stats (id, knowledge_base_id, query_text, result_count, latency_ms)
		VALUES (?, ?, ?, ?, ?)
	`, id, kbID, queryText, resultCount, latencyMs)
	return err
}

// GetDailyQueryStats 获取每日查询统计
func (s *Service) GetDailyQueryStats(ctx context.Context, kbID string, days int) ([]map[string]interface{}, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			date(created_at) as date,
			COUNT(*) as queries,
			AVG(latency_ms) as avg_latency,
			AVG(result_count) as avg_results
		FROM knowledge_query_stats
		WHERE knowledge_base_id = ? AND created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY date(created_at)
		ORDER BY date DESC
	`, kbID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var date string
		var queries int64
		var avgLatency, avgResults float64
		if err := rows.Scan(&date, &queries, &avgLatency, &avgResults); err != nil {
			continue
		}
		results = append(results, map[string]interface{}{
			"date":        date,
			"queries":     queries,
			"avg_latency": avgLatency,
			"avg_results": avgResults,
		})
	}

	return results, nil
}
