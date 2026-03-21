package knowledge

import (
	"database/sql"
	"encoding/json"
	"time"
)

// KnowledgeBase 知识库
type KnowledgeBase struct {
	ID            string    `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Description   string    `json:"description" db:"description"`
	Status        string    `json:"status" db:"status"`
	DocumentCount int       `json:"document_count" db:"document_count"`
	ChunkCount    int       `json:"chunk_count" db:"chunk_count"`
	TotalTokens   int       `json:"total_tokens" db:"total_tokens"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// Document 知识库文档
type Document struct {
	ID             string         `json:"id" db:"id"`
	KnowledgeBaseID string        `json:"knowledge_base_id" db:"knowledge_base_id"`
	Filename       string         `json:"filename" db:"filename"`
	FileType       string         `json:"file_type" db:"file_type"`
	FileSize       int64          `json:"file_size" db:"file_size"`
	FileHash       string         `json:"file_hash" db:"file_hash"`
	Content        []byte         `json:"-" db:"content"` // 不返回给前端
	Metadata       Metadata       `json:"metadata" db:"metadata"`
	Status         string         `json:"status" db:"status"`
	ErrorMessage   string         `json:"error_message" db:"error_message"`
	ChunkCount     int            `json:"chunk_count" db:"chunk_count"`
	ProcessedAt    *time.Time     `json:"processed_at" db:"processed_at"`
	CreatedAt      time.Time      `json:"created_at" db:"created_at"`
}

// Chunk 文本块
type Chunk struct {
	ID              string    `json:"id" db:"id"`
	DocumentID      string    `json:"document_id" db:"document_id"`
	KnowledgeBaseID string    `json:"knowledge_base_id" db:"knowledge_base_id"`
	Content         string    `json:"content" db:"content"`
	TokenCount      int       `json:"token_count" db:"token_count"`
	ChunkIndex      int       `json:"chunk_index" db:"chunk_index"`
	Metadata        Metadata  `json:"metadata" db:"metadata"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
}

// Metadata 文档/块的元数据
type Metadata map[string]interface{}

// Value 实现 driver.Valuer 接口
func (m Metadata) Value() (interface{}, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 实现 sql.Scanner 接口
func (m *Metadata) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return nil
	}
}

// SearchResult 搜索结果
type SearchResult struct {
	ChunkID      string   `json:"chunk_id"`
	Content      string   `json:"content"`
	DocumentID   string   `json:"document_id"`
	DocumentName string   `json:"document_name"`
	Distance     float64  `json:"distance"`
	Metadata     Metadata `json:"metadata"`
}

// CreateKnowledgeBaseRequest 创建知识库请求
type CreateKnowledgeBaseRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

// UpdateKnowledgeBaseRequest 更新知识库请求
type UpdateKnowledgeBaseRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Query      string            `json:"query" binding:"required"`
	TopK       int               `json:"top_k"`
	SearchType string            `json:"search_type"` // vector / fulltext / hybrid
	Filters    map[string]string `json:"filters"`
}

// SearchResponse 搜索响应
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// InitSchema 初始化数据库表
func InitSchema(db *sql.DB) error {
	// 知识库表
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_bases (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			status TEXT DEFAULT 'active',
			document_count INTEGER DEFAULT 0,
			chunk_count INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 文档表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_documents (
			id TEXT PRIMARY KEY,
			knowledge_base_id TEXT NOT NULL,
			filename TEXT NOT NULL,
			file_type TEXT NOT NULL,
			file_size INTEGER,
			file_hash TEXT,
			content BLOB,
			metadata TEXT,
			status TEXT DEFAULT 'pending',
			error_message TEXT,
			chunk_count INTEGER DEFAULT 0,
			processed_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 文本块表
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS knowledge_chunks (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			knowledge_base_id TEXT NOT NULL,
			content TEXT NOT NULL,
			token_count INTEGER,
			chunk_index INTEGER,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (document_id) REFERENCES knowledge_documents(id) ON DELETE CASCADE,
			FOREIGN KEY (knowledge_base_id) REFERENCES knowledge_bases(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 向量嵌入表（使用 BLOB 存储向量）
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS chunk_embeddings (
			chunk_id TEXT PRIMARY KEY,
			embedding BLOB NOT NULL,
			FOREIGN KEY (chunk_id) REFERENCES knowledge_chunks(id) ON DELETE CASCADE
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_kb_docs_kb ON knowledge_documents(knowledge_base_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_kb_chunks_doc ON knowledge_chunks(document_id)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_kb_chunks_kb ON knowledge_chunks(knowledge_base_id)`)
	if err != nil {
		return err
	}

	return nil
}
