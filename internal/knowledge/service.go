package knowledge

import (
	"context"
	"database/sql"
	"fmt"
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
		Status:      "active",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO knowledge_bases (id, name, description, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, kb.ID, kb.Name, kb.Description, kb.Status, kb.CreatedAt, kb.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create knowledge base: %w", err)
	}

	return kb, nil
}

// GetKnowledgeBase 获取知识库
func (s *Service) GetKnowledgeBase(ctx context.Context, id string) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, status,
			document_count, chunk_count, total_tokens, created_at, updated_at
		FROM knowledge_bases WHERE id = ?
	`, id).Scan(
		&kb.ID, &kb.Name, &kb.Description, &kb.Status,
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
		SELECT id, name, description, status,
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
			&kb.ID, &kb.Name, &kb.Description, &kb.Status,
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
			status = COALESCE(NULLIF(?, ''), status),
			updated_at = ?
		WHERE id = ?
	`, req.Name, req.Description, req.Status, time.Now(), id)
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
