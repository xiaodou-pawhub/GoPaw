// Package memory implements the conversation history and long-term memory storage layer.
package memory

import (
	"context"
	"math"
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
	Embedding []float32
}

// Manager is the primary entry point for the memory subsystem.
type Manager struct {
	store         *Store
	client        llm.Client
	compressor    *Compressor
	contextLimit  int      // token limit that triggers compression
	historyLimit  int      // max messages returned for context window
	memoryArchDir string   // path to memory/archive/
	compressing   sync.Map // sessionID → struct{}
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

func (m *Manager) SetArchiveDir(dir string) { m.memoryArchDir = dir }
func (m *Manager) Store() *Store { return m.store }

// Add persists one exchange with embedding support.
func (m *Manager) Add(sessionID, userID, channel, userMsg, assistantMsg string) error {
	if err := m.store.EnsureSession(sessionID, userID, channel); err != nil {
		return err
	}

	now := time.Now().UnixMilli()
	var uVec, aVec []float32
	if embedder, ok := m.client.(llm.Embedder); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		uVec, _ = embedder.Embed(ctx, userMsg)
		aVec, _ = embedder.Embed(ctx, assistantMsg)
	}

	_ = m.store.AddMessage(StoredMessage{
		ID: uuid.New().String(), SessionID: sessionID, Role: "user", Content: userMsg,
		TokenCount: CountTokens([]MemoryMessage{{Role: "user", Content: userMsg}}),
		Embedding: uVec, CreatedAt: now,
	})
	_ = m.store.AddMessage(StoredMessage{
		ID: uuid.New().String(), SessionID: sessionID, Role: "assistant", Content: assistantMsg,
		TokenCount: CountTokens([]MemoryMessage{{Role: "assistant", Content: assistantMsg}}),
		Embedding: aVec, CreatedAt: now + 1,
	})
	return nil
}

// GetContext retrieves messages and latest summary.
func (m *Manager) GetContext(sessionID string, limit int) ([]MemoryMessage, error) {
	if limit <= 0 { limit = m.historyLimit }
	msgs, err := m.store.GetRecentMessages(sessionID, limit)
	if err != nil { return nil, err }
	summary, _ := m.store.GetLatestSummary(sessionID)
	var result []MemoryMessage
	if summary != "" { result = append(result, MemoryMessage{Role: "system", Content: "[摘要] " + summary}) }
	for _, sm := range msgs {
		result = append(result, MemoryMessage{Role: sm.Role, Content: sm.Content, CreatedAt: time.UnixMilli(sm.CreatedAt)})
	}
	return result, nil
}

// Search performs hybrid FTS5 + Vector reranking.
func (m *Manager) Search(ctx context.Context, sessionID, query string, limit int, minScore float64) ([]MemorySnippet, error) {
	msgs, err := m.store.SearchMessages(sessionID, query, limit*5)
	if err != nil { return nil, err }
	if len(msgs) == 0 { return nil, nil }

	embedder, ok := m.client.(llm.Embedder)
	if !ok { return convertToSnippets(msgs)[:min(limit, len(msgs))], nil }

	queryVec, err := embedder.Embed(ctx, query)
	if err != nil { return convertToSnippets(msgs)[:min(limit, len(msgs))], nil }

	type scored struct { s MemorySnippet; score float64 }
	var results []scored
	for _, m := range msgs {
		score := 0.0
		if len(m.Embedding) > 0 { score = cosineSimilarity(queryVec, m.Embedding) }
		if score >= minScore || len(m.Embedding) == 0 {
			results = append(results, scored{MemorySnippet{Role: m.Role, Content: m.Content, CreatedAt: time.UnixMilli(m.CreatedAt)}, score})
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].score > results[j].score })
	final := make([]MemorySnippet, 0, limit)
	for i := 0; i < len(results) && i < limit; i++ { final = append(final, results[i].s) }
	return final, nil
}

// MaybeCompress checks if the current session's token count exceeds the limit.
func (m *Manager) MaybeCompress(sessionID string) {
	msgs, err := m.store.GetRecentMessages(sessionID, m.historyLimit)
	if err != nil { return }
	tokenCount := CountTokens(convertToMemoryMessages(msgs))
	if tokenCount < m.contextLimit { return }

	if tokenCount >= m.contextLimit*2 {
		m.forceCompression(sessionID)
		return
	}

	if _, loaded := m.compressing.LoadOrStore(sessionID, struct{}{}); loaded { return }
	go func() {
		defer m.compressing.Delete(sessionID)
		_ = m.Compress(context.Background(), sessionID)
	}()
}

func (m *Manager) Compress(ctx context.Context, sessionID string) error {
	msgs, err := m.store.GetRecentMessages(sessionID, m.historyLimit*2)
	if err != nil || len(msgs) < 4 { return err }
	half := len(msgs) / 2
	older := msgs[:half]
	
	// FIX: Pass []StoredMessage directly to Summarise
	summary, err := m.compressor.Summarise(ctx, older)
	if err != nil || summary == "" { return err }

	from, to := older[0].CreatedAt, older[len(older)-1].CreatedAt
	_ = m.store.StoreSummary(uuid.New().String(), sessionID, summary, from, to)
	_, _ = m.store.DB().Exec(`DELETE FROM messages WHERE session_id = ? AND created_at BETWEEN ? AND ?`, sessionID, from, to)
	return nil
}

func (m *Manager) forceCompression(sessionID string) {
	msgs, err := m.store.GetRecentMessages(sessionID, m.historyLimit*2)
	if err != nil || len(msgs) <= 4 { return }
	half := len(msgs) / 2
	from, to := msgs[0].CreatedAt, msgs[half-1].CreatedAt
	_, _ = m.store.DB().Exec(`DELETE FROM messages WHERE session_id = ? AND created_at BETWEEN ? AND ?`, sessionID, from, to)
}

func (m *Manager) Clear(sessionID string) error { return m.store.DeleteSession(sessionID) }
func (m *Manager) GetSessionStats(sessionID string) (int, int, int, int, error) { return m.store.GetSessionStats(sessionID) }

func convertToSnippets(msgs []StoredMessage) []MemorySnippet {
	res := make([]MemorySnippet, len(msgs))
	for i, m := range msgs { res[i] = MemorySnippet{Role: m.Role, Content: m.Content, CreatedAt: time.UnixMilli(m.CreatedAt)} }
	return res
}

func convertToMemoryMessages(stored []StoredMessage) []MemoryMessage {
	res := make([]MemoryMessage, len(stored))
	for i, sm := range stored { res[i] = MemoryMessage{Role: sm.Role, Content: sm.Content} }
	return res
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 { return 0 }
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}
	if normA == 0 || normB == 0 { return 0 }
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

func min(a, b int) int { if a < b { return a }; return b }
