package knowledge

import (
	"context"
	"database/sql"
	"sync"

	"github.com/coder/hnsw"
	"github.com/gopaw/gopaw/internal/embedding"
)

// VectorIndexer 向量索引管理器
type VectorIndexer struct {
	mu       sync.RWMutex
	db       *sql.DB
	encoder  embedding.Encoder
	graphs   map[string]*hnsw.Graph[string] // kbID -> graph
	chunkMap map[string]map[string]string   // kbID -> chunkID -> content
}

// NewVectorIndexer 创建向量索引管理器
func NewVectorIndexer(db *sql.DB, encoder embedding.Encoder) *VectorIndexer {
	return &VectorIndexer{
		db:       db,
		encoder:  encoder,
		graphs:   make(map[string]*hnsw.Graph[string]),
		chunkMap: make(map[string]map[string]string),
	}
}

// GetOrCreateGraph 获取或创建知识库的向量图
func (v *VectorIndexer) GetOrCreateGraph(ctx context.Context, kbID string) (*hnsw.Graph[string], error) {
	v.mu.RLock()
	graph, exists := v.graphs[kbID]
	v.mu.RUnlock()

	if exists {
		return graph, nil
	}

	v.mu.Lock()
	defer v.mu.Unlock()

	// 双重检查
	if graph, exists := v.graphs[kbID]; exists {
		return graph, nil
	}

	// 从数据库加载向量构建图
	graph, err := v.buildGraph(ctx, kbID)
	if err != nil {
		return nil, err
	}

	v.graphs[kbID] = graph
	return graph, nil
}

// buildGraph 从数据库构建向量图
func (v *VectorIndexer) buildGraph(ctx context.Context, kbID string) (*hnsw.Graph[string], error) {
	// 创建 HNSW 图
	graph := hnsw.NewGraph[string]()

	// 从数据库读取所有块和向量
	rows, err := v.db.QueryContext(ctx, `
		SELECT c.id, c.content, e.embedding
		FROM knowledge_chunks c
		JOIN chunk_embeddings e ON c.id = e.chunk_id
		WHERE c.knowledge_base_id = ?
	`, kbID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// 初始化 chunkMap
	if v.chunkMap[kbID] == nil {
		v.chunkMap[kbID] = make(map[string]string)
	}

	for rows.Next() {
		var chunkID, content string
		var embeddingBlob []byte

		if err := rows.Scan(&chunkID, &content, &embeddingBlob); err != nil {
			continue
		}

		// 解码向量
		vec := blobToFloat32Slice(embeddingBlob)
		if len(vec) != 384 {
			continue
		}

		// 添加到图
		node := hnsw.MakeNode(chunkID, vec)
		graph.Add(node)

		v.chunkMap[kbID][chunkID] = content
	}

	return graph, nil
}

// AddChunk 添加块到索引
func (v *VectorIndexer) AddChunk(ctx context.Context, kbID, chunkID, content string, vec []float32) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 获取或创建图
	graph, exists := v.graphs[kbID]
	if !exists {
		graph = hnsw.NewGraph[string]()
		v.graphs[kbID] = graph
	}

	// 添加到图
	node := hnsw.MakeNode(chunkID, vec)
	graph.Add(node)

	// 更新 chunkMap
	if v.chunkMap[kbID] == nil {
		v.chunkMap[kbID] = make(map[string]string)
	}
	v.chunkMap[kbID][chunkID] = content

	return nil
}

// RemoveChunk 从索引中删除块
func (v *VectorIndexer) RemoveChunk(kbID, chunkID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if graph, exists := v.graphs[kbID]; exists {
		graph.Delete(chunkID)
	}

	if v.chunkMap[kbID] != nil {
		delete(v.chunkMap[kbID], chunkID)
	}
}

// DeleteIndex 删除知识库的索引
func (v *VectorIndexer) DeleteIndex(kbID string) {
	v.mu.Lock()
	defer v.mu.Unlock()

	delete(v.graphs, kbID)
	delete(v.chunkMap, kbID)
}

// RebuildIndex 重建知识库的向量索引
func (v *VectorIndexer) RebuildIndex(ctx context.Context, kbID string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	// 删除旧索引
	delete(v.graphs, kbID)
	delete(v.chunkMap, kbID)

	// 构建新索引
	graph, err := v.buildGraph(ctx, kbID)
	if err != nil {
		return err
	}

	v.graphs[kbID] = graph
	return nil
}

// Search 执行向量搜索
func (v *VectorIndexer) Search(ctx context.Context, kbID string, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	// 获取图
	graph, err := v.GetOrCreateGraph(ctx, kbID)
	if err != nil {
		return nil, err
	}

	// 生成查询向量
	queryVec, err := v.encoder.Encode(query)
	if err != nil {
		return nil, err
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	// 执行搜索
	nodes := graph.Search(queryVec, topK)

	// 转换结果
	var searchResults []SearchResult
	for _, node := range nodes {
		content := v.chunkMap[kbID][node.Key]
		// 计算余弦距离
		distance := 1.0 - cosineSimilarity(queryVec, node.Value)
		searchResults = append(searchResults, SearchResult{
			ChunkID:  node.Key,
			Content:  content,
			Distance: distance,
		})
	}

	return searchResults, nil
}