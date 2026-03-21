package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/gopaw/gopaw/internal/embedding"
)

// KnowledgeSearcher 知识库搜索器
type KnowledgeSearcher struct {
	db      *sql.DB
	encoder embedding.Encoder
}

// NewKnowledgeSearcher 创建搜索器
func NewKnowledgeSearcher(db *sql.DB, encoder embedding.Encoder) *KnowledgeSearcher {
	return &KnowledgeSearcher{
		db:      db,
		encoder: encoder,
	}
}

// Search 执行向量搜索
func (s *KnowledgeSearcher) Search(ctx context.Context, kbID string, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	// 生成查询向量
	queryVec, err := s.encoder.Encode(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// 执行向量搜索（使用余弦相似度）
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			c.id,
			c.content,
			c.metadata,
			c.document_id,
			d.filename as document_name,
			1.0 - vec_cosine(e.embedding, vec_normalize(vec_f32(?))) as distance
		FROM knowledge_chunks c
		JOIN chunk_embeddings e ON c.id = e.chunk_id
		JOIN knowledge_documents d ON c.document_id = d.id
		WHERE c.knowledge_base_id = ?
		ORDER BY distance ASC
		LIMIT ?
	`, float32SliceToBlob(queryVec), kbID, topK)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var metadataStr string
		err := rows.Scan(
			&result.ChunkID,
			&result.Content,
			&metadataStr,
			&result.DocumentID,
			&result.DocumentName,
			&result.Distance,
		)
		if err != nil {
			continue
		}

		if metadataStr != "" {
			result.Metadata.Scan(metadataStr)
		}

		results = append(results, result)
	}

	return results, rows.Err()
}

// FullTextSearch 执行全文搜索
func (s *KnowledgeSearcher) FullTextSearch(ctx context.Context, kbID string, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	// 使用 SQLite FTS5 或 LIKE 进行全文搜索
	// 这里使用简单的 LIKE 搜索，生产环境建议使用 FTS5
	pattern := "%" + query + "%"

	rows, err := s.db.QueryContext(ctx, `
		SELECT 
			c.id,
			c.content,
			c.metadata,
			c.document_id,
			d.filename as document_name,
			0.0 as distance
		FROM knowledge_chunks c
		JOIN knowledge_documents d ON c.document_id = d.id
		WHERE c.knowledge_base_id = ? AND c.content LIKE ?
		ORDER BY LENGTH(c.content) - LENGTH(REPLACE(LOWER(c.content), LOWER(?), '')) DESC
		LIMIT ?
	`, kbID, pattern, query, topK)
	if err != nil {
		return nil, fmt.Errorf("fulltext search failed: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var result SearchResult
		var metadataStr string
		err := rows.Scan(
			&result.ChunkID,
			&result.Content,
			&metadataStr,
			&result.DocumentID,
			&result.DocumentName,
			&result.Distance,
		)
		if err != nil {
			continue
		}

		if metadataStr != "" {
			result.Metadata.Scan(metadataStr)
		}

		// 计算简单的相关性分数（出现次数）
		result.Distance = 1.0 - float64(countOccurrences(result.Content, query))/float64(len(result.Content))

		results = append(results, result)
	}

	return results, rows.Err()
}

// HybridSearch 混合搜索（向量 + 全文）
func (s *KnowledgeSearcher) HybridSearch(ctx context.Context, kbID string, query string, topK int, vectorWeight float64) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}
	if vectorWeight < 0 || vectorWeight > 1 {
		vectorWeight = 0.7
	}

	textWeight := 1.0 - vectorWeight

	// 向量搜索（获取更多结果用于融合）
	vecResults, err := s.Search(ctx, kbID, query, topK*2)
	if err != nil {
		return nil, err
	}

	// 全文搜索
	textResults, err := s.FullTextSearch(ctx, kbID, query, topK*2)
	if err != nil {
		return nil, err
	}

	// RRF 融合排序
	return s.reciprocalRankFusion(vecResults, textResults, topK, vectorWeight, textWeight), nil
}

// reciprocalRankFusion RRF 融合排序
func (s *KnowledgeSearcher) reciprocalRankFusion(vecResults, textResults []SearchResult, topK int, vecWeight, textWeight float64) []SearchResult {
	// 计算每个文档的 RRF 分数
	scores := make(map[string]float64)
	vecRank := make(map[string]int)
	textRank := make(map[string]int)

	// 记录向量搜索排名
	for rank, result := range vecResults {
		vecRank[result.ChunkID] = rank + 1
	}

	// 记录全文搜索排名
	for rank, result := range textResults {
		textRank[result.ChunkID] = rank + 1
	}

	// 合并所有结果
	allResults := make(map[string]SearchResult)
	for _, r := range vecResults {
		allResults[r.ChunkID] = r
	}
	for _, r := range textResults {
		allResults[r.ChunkID] = r
	}

	// 计算 RRF 分数
	k := 60.0 // RRF 常数
	for chunkID := range allResults {
		score := 0.0

		if rank, ok := vecRank[chunkID]; ok {
			score += vecWeight / (k + float64(rank))
		}

		if rank, ok := textRank[chunkID]; ok {
			score += textWeight / (k + float64(rank))
		}

		scores[chunkID] = score
	}

	// 按分数排序
	var results []SearchResult
	for chunkID, result := range allResults {
		result.Distance = 1.0 - scores[chunkID] // 转换为距离（越小越好）
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// 返回前 topK 个结果
	if len(results) > topK {
		results = results[:topK]
	}

	return results
}

// SearchWithFilters 带过滤条件的搜索
func (s *KnowledgeSearcher) SearchWithFilters(ctx context.Context, kbID string, query string, topK int, filters map[string]string) ([]SearchResult, error) {
	// 基础搜索
	results, err := s.Search(ctx, kbID, query, topK*2)
	if err != nil {
		return nil, err
	}

	// 应用过滤条件
	var filtered []SearchResult
	for _, result := range results {
		if s.matchesFilters(result, filters) {
			filtered = append(filtered, result)
		}
		if len(filtered) >= topK {
			break
		}
	}

	return filtered, nil
}

// matchesFilters 检查是否匹配过滤条件
func (s *KnowledgeSearcher) matchesFilters(result SearchResult, filters map[string]string) bool {
	for key, value := range filters {
		switch key {
		case "document_id":
			if result.DocumentID != value {
				return false
			}
		default:
			// 检查元数据
			if metaValue, ok := result.Metadata[key]; ok {
				if fmt.Sprintf("%v", metaValue) != value {
					return false
				}
			}
		}
	}
	return true
}

// countOccurrences 计算子串出现次数
func countOccurrences(s, substr string) int {
	count := 0
	for {
		idx := indexOf(s, substr)
		if idx == -1 {
			break
		}
		count++
		s = s[idx+len(substr):]
	}
	return count
}

// indexOf 查找子串位置（不区分大小写）
func indexOf(s, substr string) int {
	// 简单的实现，生产环境建议使用 strings.Index
	lowerS := toLower(s)
	lowerSubstr := toLower(substr)
	return index(lowerS, lowerSubstr)
}

// toLower 转换为小写
func toLower(s string) string {
	// 简单实现，支持 ASCII
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		result[i] = c
	}
	return string(result)
}

// index 查找子串位置
func index(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}
	if len(substr) > len(s) {
		return -1
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
