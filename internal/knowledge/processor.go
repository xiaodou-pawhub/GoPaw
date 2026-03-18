package knowledge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
	"unsafe"
)

// DocumentProcessor 文档处理器
type DocumentProcessor struct {
	db          *sql.DB
	embedder    Embedder
	extractors  *ExtractorRegistry
	chunker     Chunker
}

// NewDocumentProcessor 创建文档处理器
func NewDocumentProcessor(db *sql.DB, embedder Embedder, chunkStrategy ChunkStrategy) *DocumentProcessor {
	return &DocumentProcessor{
		db:         db,
		embedder:   embedder,
		extractors: NewExtractorRegistry(),
		chunker:    NewChunker(chunkStrategy),
	}
}

// Process 处理文档
func (p *DocumentProcessor) Process(ctx context.Context, docID string) error {
	// 获取文档信息
	doc, err := p.getDocument(ctx, docID)
	if err != nil {
		return err
	}

	// 更新状态为处理中
	if err := p.updateDocumentStatus(ctx, docID, "processing", ""); err != nil {
		return err
	}

	// 获取知识库配置
	kb, err := p.getKnowledgeBase(ctx, doc.KnowledgeBaseID)
	if err != nil {
		p.updateDocumentStatus(ctx, docID, "failed", err.Error())
		return err
	}

	// 提取文本
	text, err := p.extractText(doc)
	if err != nil {
		p.updateDocumentStatus(ctx, docID, "failed", err.Error())
		return err
	}

	// 文本分块
	chunks := p.chunker.Chunk(text, kb.ChunkSize, kb.ChunkOverlap)

	// 生成 Embedding 并保存
	if err := p.saveChunks(ctx, doc, chunks); err != nil {
		p.updateDocumentStatus(ctx, docID, "failed", err.Error())
		return err
	}

	// 更新文档状态
	if err := p.updateDocumentStatus(ctx, docID, "completed", ""); err != nil {
		return err
	}

	// 更新知识库统计
	if err := p.updateKnowledgeBaseStats(ctx, kb.ID); err != nil {
		return err
	}

	return nil
}

// extractText 提取文档文本
func (p *DocumentProcessor) extractText(doc *Document) (string, error) {
	reader := bytes.NewReader(doc.Content)
	return p.extractors.Extract(doc.FileType, reader)
}

// saveChunks 保存文本块和 Embedding
func (p *DocumentProcessor) saveChunks(ctx context.Context, doc *Document, chunks []TextChunk) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 先删除旧的 Embedding（通过 JOIN 获取 chunk_id）
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM chunk_embeddings 
		WHERE chunk_id IN (
			SELECT id FROM knowledge_chunks WHERE document_id = ?
		)
	`, doc.ID); err != nil {
		return err
	}

	// 删除旧的块
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM knowledge_chunks WHERE document_id = ?",
		doc.ID); err != nil {
		return err
	}

	// 准备 Embedding 生成
	contents := make([]string, len(chunks))
	for i, chunk := range chunks {
		contents[i] = chunk.Content
	}

	// 批量生成 Embedding
	embeddings, err := p.embedder.BatchEmbed(ctx, contents)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// 检查 embeddings 长度是否匹配
	if len(embeddings) != len(chunks) {
		return fmt.Errorf("embeddings count mismatch: expected %d, got %d", len(chunks), len(embeddings))
	}

	// 保存块和 Embedding
	for i, chunk := range chunks {
		chunkID := generateChunkID(doc.ID, i)

		// 保存块
		_, err := tx.ExecContext(ctx, `
			INSERT INTO knowledge_chunks (id, document_id, knowledge_base_id, content, token_count, chunk_index, metadata)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, chunkID, doc.ID, doc.KnowledgeBaseID, chunk.Content, chunk.TokenCount, i, chunk.Metadata)
		if err != nil {
			return err
		}

		// 保存 Embedding
		embedding := embeddings[i]
		if len(embedding) > 0 {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO chunk_embeddings (chunk_id, embedding)
				VALUES (?, vec_normalize(vec_f32(?)))
			`, chunkID, float32SliceToBlob(embedding))
			if err != nil {
				return fmt.Errorf("failed to save embedding: %w", err)
			}
		}
	}

	// 更新文档块数量
	_, err = tx.ExecContext(ctx,
		"UPDATE knowledge_documents SET chunk_count = ?, processed_at = ? WHERE id = ?",
		len(chunks), time.Now(), doc.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// getDocument 获取文档
func (p *DocumentProcessor) getDocument(ctx context.Context, id string) (*Document, error) {
	var doc Document
	var metadataStr string

	err := p.db.QueryRowContext(ctx, `
		SELECT id, knowledge_base_id, filename, file_type, file_size, file_hash, content, metadata, status
		FROM knowledge_documents WHERE id = ?
	`, id).Scan(
		&doc.ID, &doc.KnowledgeBaseID, &doc.Filename, &doc.FileType,
		&doc.FileSize, &doc.FileHash, &doc.Content, &metadataStr, &doc.Status,
	)
	if err != nil {
		return nil, err
	}

	if metadataStr != "" {
		doc.Metadata.Scan(metadataStr)
	}

	return &doc, nil
}

// getKnowledgeBase 获取知识库
func (p *DocumentProcessor) getKnowledgeBase(ctx context.Context, id string) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	err := p.db.QueryRowContext(ctx, `
		SELECT id, name, description, embedding_model, chunk_size, chunk_overlap, status
		FROM knowledge_bases WHERE id = ?
	`, id).Scan(
		&kb.ID, &kb.Name, &kb.Description, &kb.EmbeddingModel,
		&kb.ChunkSize, &kb.ChunkOverlap, &kb.Status,
	)
	if err != nil {
		return nil, err
	}
	return &kb, nil
}

// updateDocumentStatus 更新文档状态
func (p *DocumentProcessor) updateDocumentStatus(ctx context.Context, id, status, errorMsg string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE knowledge_documents SET status = ?, error_message = ? WHERE id = ?
	`, status, errorMsg, id)
	return err
}

// updateKnowledgeBaseStats 更新知识库统计
func (p *DocumentProcessor) updateKnowledgeBaseStats(ctx context.Context, kbID string) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE knowledge_bases SET
			document_count = (SELECT COUNT(*) FROM knowledge_documents WHERE knowledge_base_id = ? AND status = 'completed'),
			chunk_count = (SELECT COUNT(*) FROM knowledge_chunks WHERE knowledge_base_id = ?),
			total_tokens = (SELECT COALESCE(SUM(token_count), 0) FROM knowledge_chunks WHERE knowledge_base_id = ?),
			updated_at = ?
		WHERE id = ?
	`, kbID, kbID, kbID, time.Now(), kbID)
	return err
}

// CalculateFileHash 计算文件哈希
func CalculateFileHash(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

// generateChunkID 生成块 ID
func generateChunkID(docID string, index int) string {
	return fmt.Sprintf("%s_chunk_%d", docID, index)
}

// float32SliceToBlob 将 float32 切片转换为 blob
func float32SliceToBlob(slice []float32) []byte {
	blob := make([]byte, len(slice)*4)
	for i, v := range slice {
		// 简单的二进制转换
		*(*float32)(unsafe.Pointer(&blob[i*4])) = v
	}
	return blob
}
