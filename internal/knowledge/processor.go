package knowledge

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/gopaw/gopaw/internal/embedding"
)

// DocumentProcessor 文档处理器
type DocumentProcessor struct {
	db         *sql.DB
	encoder    embedding.Encoder
	extractors *ExtractorRegistry
	chunker    Chunker
	indexer    *VectorIndexer
}

// NewDocumentProcessor 创建文档处理器
func NewDocumentProcessor(db *sql.DB, encoder embedding.Encoder, chunkStrategy ChunkStrategy, indexer *VectorIndexer) *DocumentProcessor {
	return &DocumentProcessor{
		db:         db,
		encoder:    encoder,
		extractors: NewExtractorRegistry(),
		chunker:    NewChunker(chunkStrategy),
		indexer:    indexer,
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

	// 获取知识库
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

	// 检测文本有效性
	textLength := len(strings.TrimSpace(text))
	if textLength < 50 {
		errMsg := fmt.Sprintf("文档文本太少（仅 %d 字符），无法有效处理", textLength)
		p.updateDocumentStatus(ctx, docID, "no_text", errMsg)
		return fmt.Errorf(errMsg)
	}

	// 根据知识库模式进行不同处理
	if kb.Mode == "inject" {
		// 全局注入模式：直接保存内容，不进行分块和向量化
		if err := p.saveInjectContent(ctx, doc, text); err != nil {
			p.updateDocumentStatus(ctx, docID, "failed", err.Error())
			return err
		}
	} else {
		// 向量检索模式：文本分块并生成 Embedding
		chunks := p.chunker.Chunk(text, 500, 50)
		if err := p.saveChunks(ctx, doc, chunks); err != nil {
			p.updateDocumentStatus(ctx, docID, "failed", err.Error())
			return err
		}
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

	// 删除旧的块
	if _, err := tx.ExecContext(ctx,
		"DELETE FROM knowledge_chunks WHERE document_id = ?",
		doc.ID); err != nil {
		return err
	}

	// 保存块和 Embedding
	for i, chunk := range chunks {
		chunkID := generateChunkID(doc.ID, i)

		// 生成向量
		embedding, err := p.encoder.Encode(chunk.Content)
		if err != nil {
			return fmt.Errorf("failed to generate embedding: %w", err)
		}

		// 保存块
		metadataValue, err := chunk.Metadata.Value()
		if err != nil {
			return fmt.Errorf("failed to serialize metadata: %w", err)
		}
		
		_, err = tx.ExecContext(ctx, `
			INSERT INTO knowledge_chunks (id, document_id, knowledge_base_id, content, token_count, chunk_index, metadata)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, chunkID, doc.ID, doc.KnowledgeBaseID, chunk.Content, chunk.TokenCount, i, metadataValue)
		if err != nil {
			return err
		}

		// 保存 Embedding
		if len(embedding) > 0 {
			_, err := tx.ExecContext(ctx, `
				INSERT INTO chunk_embeddings (chunk_id, embedding)
				VALUES (?, ?)
			`, chunkID, float32SliceToBlob(embedding))
			if err != nil {
				return fmt.Errorf("failed to save embedding: %w", err)
			}

			// 更新 HNSW 索引
			if p.indexer != nil {
				p.indexer.AddChunk(ctx, doc.KnowledgeBaseID, chunkID, chunk.Content, embedding)
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

// saveInjectContent 保存全局注入内容（不进行分块和向量化）
func (p *DocumentProcessor) saveInjectContent(ctx context.Context, doc *Document, text string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 更新文档内容（直接保存完整文本）
	_, err = tx.ExecContext(ctx,
		"UPDATE knowledge_documents SET content = ?, chunk_count = 1, processed_at = ? WHERE id = ?",
		[]byte(text), time.Now(), doc.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// getDocument 获取文档
func (p *DocumentProcessor) getDocument(ctx context.Context, id string) (*Document, error) {
	var doc Document
	var metadataStr sql.NullString

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

	if metadataStr.Valid && metadataStr.String != "" {
		doc.Metadata.Scan(metadataStr.String)
	}

	return &doc, nil
}

// getKnowledgeBase 获取知识库
func (p *DocumentProcessor) getKnowledgeBase(ctx context.Context, id string) (*KnowledgeBase, error) {
	var kb KnowledgeBase
	err := p.db.QueryRowContext(ctx, `
		SELECT id, name, description, mode, status
		FROM knowledge_bases WHERE id = ?
	`, id).Scan(
		&kb.ID, &kb.Name, &kb.Description, &kb.Mode, &kb.Status,
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

// blobToFloat32Slice 将 blob 转换为 float32 切片
func blobToFloat32Slice(blob []byte) []float32 {
	if len(blob) == 0 || len(blob)%4 != 0 {
		return nil
	}
	slice := make([]float32, len(blob)/4)
	for i := range slice {
		slice[i] = *(*float32)(unsafe.Pointer(&blob[i*4]))
	}
	return slice
}

// cosineSimilarity 计算两个向量的余弦相似度
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

// sqrt 简单的平方根函数
func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// 牛顿迭代法
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
