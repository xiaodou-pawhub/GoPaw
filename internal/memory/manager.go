// Package memory implements the conversation history and long-term memory storage layer.
package memory

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
	"github.com/google/uuid"
	"github.com/gopaw/gopaw/internal/llm"
	"go.uber.org/zap"
)

// MemoryMessage is the normalised view of a stored conversation turn.
type MemoryMessage struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// MemorySnippet is a search result from the long-term memory store.
type MemorySnippet struct {
	Role      string
	Content   string
	CreatedAt time.Time
}

// Manager is the primary entry point for the memory subsystem.
// It coordinates the SQLite store, token counting and LLM-based compression.
type Manager struct {
	store         *Store
	client        llm.Client
	compressor    *Compressor
	contextLimit  int      // token limit that triggers compression
	historyLimit  int      // max messages returned for context window
	memoryArchDir string   // path to memory/archive/ for auto-archiving summaries
	compressing   sync.Map // sessionID → struct{}, prevents concurrent compression
	logger        *zap.Logger
}

// NewManager creates a Manager.
func NewManager(store *Store, llmClient llm.Client, contextLimit, historyLimit int, logger *zap.Logger) *Manager {
	return &Manager{
		store:        store,
		client:       llmClient,
		compressor:   NewCompressor(llmClient),
		contextLimit: contextLimit,
		historyLimit: historyLimit,
		logger:       logger,
	}
}
// SetArchiveDir 设置对话摘要归档目录（memory/archive/）。
func (m *Manager) SetArchiveDir(dir string) {
	m.memoryArchDir = dir
}

// Store returns the underlying store for direct SQL access.
func (m *Manager) Store() *Store {
	return m.store
}

// Add persists one user→assistant exchange to the store.
func (m *Manager) Add(sessionID, userID, channel, userMsg, assistantMsg string) error {
	if err := m.store.EnsureSession(sessionID, userID, channel); err != nil {
		return fmt.Errorf("memory: ensure session: %w", err)
	}

	now := time.Now().UnixMilli()
	
	// 中文：计算 token 数
	// English: Calculate token count
	userTokens := CountTokens([]MemoryMessage{{Role: "user", Content: userMsg}})
	assistTokens := CountTokens([]MemoryMessage{{Role: "assistant", Content: assistantMsg}})

	if err := m.store.AddMessage(StoredMessage{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		Role:       "user",
		Content:    userMsg,
		TokenCount: userTokens,
		CreatedAt:  now,
	}); err != nil {
		return fmt.Errorf("memory: add user message: %w", err)
	}

	if err := m.store.AddMessage(StoredMessage{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		Role:       "assistant",
		Content:    assistantMsg,
		TokenCount: assistTokens,
		CreatedAt:  now + 1, // ensure stable ordering
	}); err != nil {
		return fmt.Errorf("memory: add assistant message: %w", err)
	}
	return nil
}

// GetContext returns the most recent messages for the session, prefixed with any
// existing summary so the LLM is aware of earlier context.
func (m *Manager) GetContext(sessionID string, limit int) ([]MemoryMessage, error) {
	if limit <= 0 {
		limit = m.historyLimit
	}

	msgs, err := m.store.GetRecentMessages(sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("memory: get context: %w", err)
	}

	summary, err := m.store.GetLatestSummary(sessionID)
	if err != nil {
		return nil, fmt.Errorf("memory: get summary: %w", err)
	}

	var result []MemoryMessage
	if summary != "" {
		result = append(result, MemoryMessage{
			Role:    "system",
			Content: "[历史摘要] " + summary,
		})
	}
	for _, sm := range msgs {
		result = append(result, MemoryMessage{
			Role:      sm.Role,
			Content:   sm.Content,
			CreatedAt: time.UnixMilli(sm.CreatedAt),
		})
	}
	return result, nil
}

// MaybeCompress checks if the current session's token count exceeds the context limit.
// If so, it launches an async goroutine to compress. Non-blocking; does not return errors.
// This should be called by the Agent before processing each user message.
func (m *Manager) MaybeCompress(sessionID string) {
	msgs, err := m.store.GetRecentMessages(sessionID, m.historyLimit)
	if err != nil {
		m.logger.Warn("memory: maybe compress: get messages", zap.Error(err))
		return
	}

	tokenCount := CountTokens(convertToMemoryMessages(msgs))
	if tokenCount < m.contextLimit {
		return
	}

	// 极端情况：超出 2 倍上限，执行同步强制压缩（不依赖 LLM）
	if tokenCount >= m.contextLimit*2 {
		m.logger.Warn("context far exceeds limit, force compressing",
			zap.String("session_id", sessionID),
			zap.Int("token_count", tokenCount),
		)
		m.forceCompression(sessionID)
		return
	}

	// 防止对同一 session 并发压缩
	if _, loaded := m.compressing.LoadOrStore(sessionID, struct{}{}); loaded {
		return
	}

	m.logger.Info("context token limit reached, async compressing",
		zap.String("session_id", sessionID),
		zap.Int("token_count", tokenCount),
		zap.Int("limit", m.contextLimit),
	)

	go func() {
		defer m.compressing.Delete(sessionID)
		if err := m.Compress(context.Background(), sessionID); err != nil {
			m.logger.Warn("memory: async compress failed", zap.Error(err))
		}
	}()
}

// forceCompression is an emergency fallback that drops the oldest 50% of messages
// without invoking the LLM. It adds a note to the session so the AI is aware.
func (m *Manager) forceCompression(sessionID string) {
	msgs, err := m.store.GetRecentMessages(sessionID, m.historyLimit*2)
	if err != nil || len(msgs) <= 4 {
		return
	}

	half := len(msgs) / 2
	older := msgs[:half]
	fromTime := older[0].CreatedAt
	toTime := older[len(older)-1].CreatedAt

	if _, err := m.store.DB().Exec(
		`DELETE FROM messages WHERE session_id = ? AND created_at BETWEEN ? AND ?`,
		sessionID, fromTime, toTime,
	); err != nil {
		m.logger.Error("memory: force compression delete failed", zap.Error(err))
		return
	}

	note := fmt.Sprintf("[系统：上下文超出限制，已自动丢弃最旧的 %d 条消息以释放空间]", half)
	_ = m.store.AddMessage(StoredMessage{
		ID:         uuid.New().String(),
		SessionID:  sessionID,
		Role:       "system",
		Content:    note,
		TokenCount: CountTokens([]MemoryMessage{{Role: "system", Content: note}}),
		CreatedAt:  time.Now().UnixMilli(),
	})

	m.logger.Info("force compression done", zap.String("session_id", sessionID), zap.Int("dropped", half))
}

// convertToMemoryMessages converts StoredMessage slice to MemoryMessage slice for token counting.
func convertToMemoryMessages(stored []StoredMessage) []MemoryMessage {
	result := make([]MemoryMessage, len(stored))
	for i, sm := range stored {
		result[i] = MemoryMessage{
			Role:      sm.Role,
			Content:   sm.Content,
			CreatedAt: time.UnixMilli(sm.CreatedAt),
		}
	}
	return result
}

// Search performs a hybrid search: keyword-based via FTS5, then reranked by semantic similarity if an embedder is available.
func (m *Manager) Search(ctx context.Context, sessionID, query string, limit int, minScore float64) ([]MemorySnippet, error) {
	// 1. Initial keyword recall via FTS5
	// We fetch more results than the limit to provide room for reranking
	recallLimit := limit * 3
	msgs, err := m.store.SearchMessages(sessionID, query, recallLimit)
	if err != nil {
		return nil, fmt.Errorf("memory: keyword recall: %w", err)
	}

	snippets := make([]MemorySnippet, 0, len(msgs))
	for _, sm := range msgs {
		snippets = append(snippets, MemorySnippet{
			Role:      sm.Role,
			Content:   sm.Content,
			CreatedAt: time.UnixMilli(sm.CreatedAt),
		})
	}

	// 2. Semantic Reranking (if supported)
	embedder, ok := m.client.(llm.Embedder)
	if !ok || len(snippets) == 0 {
		// Fallback to pure keyword search if no embedder
		if len(snippets) > limit {
			snippets = snippets[:limit]
		}
		return snippets, nil
	}

	// Get query embedding
	queryVec, err := embedder.Embed(ctx, query)
	if err != nil {
		m.logger.Warn("semantic search: query embedding failed", zap.Error(err))
		return snippets[:min(limit, len(snippets))], nil
	}

	type scoredSnippet struct {
		snippet MemorySnippet
		score   float64
	}
	scored := make([]scoredSnippet, 0, len(snippets))

	// Rerank snippets
	// Note: For large scale, we should cache embeddings in DB. For now, we do it real-time.
	for _, s := range snippets {
		vec, err := embedder.Embed(ctx, s.Content)
		if err != nil {
			continue
		}
		score := cosineSimilarity(queryVec, vec)
		if score >= minScore {
			scored = append(scored, scoredSnippet{s, score})
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].score > scored[j].score
	})

	finalResults := make([]MemorySnippet, 0, limit)
	for i := 0; i < len(scored) && i < limit; i++ {
		finalResults = append(finalResults, scored[i].snippet)
	}

	return finalResults, nil
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}
// Compress compresses old messages into a summary when the context is too long.
// It targets the oldest 50% of messages in the session.
func (m *Manager) Compress(ctx context.Context, sessionID string) error {
	msgs, err := m.store.GetRecentMessages(sessionID, m.historyLimit*2)
	if err != nil {
		return fmt.Errorf("memory: compress: fetch: %w", err)
	}
	if len(msgs) < 4 {
		return nil // nothing meaningful to compress
	}

	half := len(msgs) / 2
	older := msgs[:half]

	summary, err := m.compressor.Summarise(ctx, older)
	if err != nil {
		return fmt.Errorf("memory: compress: summarise: %w", err)
	}
	if summary == "" {
		return nil
	}

	// Determine rowid range for the older half.
	// We approximate by deleted via the stored message IDs list.
	// Re-fetch to get actual rowids is unnecessary here; we identify by session+timestamp range.
	fromTime := older[0].CreatedAt
	toTime := older[len(older)-1].CreatedAt

	summaryID := uuid.New().String()
	if err := m.store.StoreSummary(summaryID, sessionID, summary, fromTime, toTime); err != nil {
		return fmt.Errorf("memory: compress: store summary: %w", err)
	}

	// Delete compressed messages from the messages table.
	if _, err := m.store.DB().Exec(
		`DELETE FROM messages WHERE session_id = ? AND created_at BETWEEN ? AND ?`,
		sessionID, fromTime, toTime,
	); err != nil {
		return fmt.Errorf("memory: compress: delete old messages: %w", err)
	}

	m.logger.Info("memory compressed",
		zap.String("session_id", sessionID),
		zap.Int("compressed_count", half),
		zap.String("summary_id", summaryID),
	)

	// 将摘要追加到 memory/archive/YYYY-MM.md
	if m.memoryArchDir != "" {
		if err := m.appendToArchive(summary); err != nil {
			m.logger.Warn("memory: archive summary failed", zap.Error(err))
		}
	}

	return nil
}

// appendToArchive 将摘要追加到当月归档文件 memory/archive/YYYY-MM.md。
func (m *Manager) appendToArchive(summary string) error {
	if err := os.MkdirAll(m.memoryArchDir, 0o755); err != nil {
		return fmt.Errorf("archive: create dir: %w", err)
	}
	now := time.Now()
	archFile := filepath.Join(m.memoryArchDir, now.Format("2006-01")+".md")
	entry := fmt.Sprintf("\n## [%s]\n%s\n", now.Format("2006-01-02 15:04"), summary)
	f, err := os.OpenFile(archFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("archive: open %q: %w", archFile, err)
	}
	defer f.Close()
	_, err = f.WriteString(entry)
	return err
}

// Clear removes all memory for a session.
func (m *Manager) Clear(sessionID string) error {
	if err := m.store.DeleteSession(sessionID); err != nil {
		return fmt.Errorf("memory: clear: %w", err)
	}
	return nil
}

// GetSessionStats returns the message count and token usage for a session.
func (m *Manager) GetSessionStats(sessionID string) (count, total, user, assist int, err error) {
	return m.store.GetSessionStats(sessionID)
}

// EstimateTokens returns the precise token count for the messages.
func EstimateTokens(msgs []MemoryMessage) int {
	return CountTokens(msgs)
}

