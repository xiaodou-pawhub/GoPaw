package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/memory"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// MemoryHandler handles /api/memories routes for structured long-term memory (memories.db).
type MemoryHandler struct {
	ltm    *memory.LTMStore
	logger *zap.Logger
}

// NewMemoryHandler creates a MemoryHandler.
func NewMemoryHandler(ltm *memory.LTMStore, logger *zap.Logger) *MemoryHandler {
	return &MemoryHandler{ltm: ltm, logger: logger}
}

// memoryEntryJSON is the JSON representation of a memory entry.
type memoryEntryJSON struct {
	ID        string `json:"id"`
	Key       string `json:"key"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
	Score     float64 `json:"score,omitempty"`
}

func entryToJSON(e memory.MemoryEntry) memoryEntryJSON {
	return memoryEntryJSON{
		ID:        e.ID,
		Key:       e.Key,
		Content:   e.Content,
		Category:  string(e.Category),
		CreatedAt: e.CreatedAt.UnixMilli(),
		UpdatedAt: e.UpdatedAt.UnixMilli(),
		Score:     e.Score,
	}
}

// List handles GET /api/memories
// Query params: category, q (search), limit (default 50), offset
func (h *MemoryHandler) List(c *gin.Context) {
	category := memory.Category(c.Query("category"))
	q := strings.TrimSpace(c.Query("q"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var (
		entries []memory.MemoryEntry
		err     error
	)
	if q != "" {
		entries, err = h.ltm.Recall(q, limit, category)
	} else {
		entries, err = h.ltm.List(category, limit)
	}
	if err != nil {
		h.logger.Error("memory list error", zap.Error(err))
		api.InternalErrorWithDetails(c, "memory list error", err)
		return
	}

	out := make([]memoryEntryJSON, len(entries))
	for i, e := range entries {
		out[i] = entryToJSON(e)
	}
	api.Success(c, gin.H{"memories": out, "total": len(out)})
}

// Create handles POST /api/memories
func (h *MemoryHandler) Create(c *gin.Context) {
	var req struct {
		Key      string `json:"key" binding:"required"`
		Content  string `json:"content" binding:"required"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}
	cat := memory.Category(req.Category)
	if cat == "" {
		cat = memory.CategoryCore
	}
	if err := h.ltm.Store(req.Key, req.Content, cat); err != nil {
		h.logger.Error("memory create error", zap.Error(err))
		api.InternalErrorWithDetails(c, "memory create error", err)
		return
	}
	entry, err := h.ltm.Get(req.Key)
	if err != nil || entry == nil {
		api.Success(c, gin.H{"ok": true})
		return
	}
	api.Success(c, gin.H{"memory": entryToJSON(*entry)})
}

// Update handles PUT /api/memories/:id
// Updates content and/or category for the entry identified by key (id = key).
func (h *MemoryHandler) Update(c *gin.Context) {
	key := c.Param("id")
	var req struct {
		Content  string `json:"content" binding:"required"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	// Get existing entry first to preserve category if not specified
	existing, err := h.ltm.Get(key)
	if err != nil {
		h.logger.Error("memory get error", zap.Error(err))
		api.InternalErrorWithDetails(c, "memory get error", err)
		return
	}
	if existing == nil {
		api.NotFound(c, "memory")
		return
	}

	cat := memory.Category(req.Category)
	if cat == "" {
		cat = existing.Category
	}
	if err := h.ltm.Store(key, req.Content, cat); err != nil {
		h.logger.Error("memory update error", zap.Error(err))
		api.InternalErrorWithDetails(c, "memory update error", err)
		return
	}
	entry, _ := h.ltm.Get(key)
	if entry == nil {
		api.Success(c, gin.H{"ok": true})
		return
	}
	api.Success(c, gin.H{"memory": entryToJSON(*entry)})
}

// Delete handles DELETE /api/memories/:id (id = key)
func (h *MemoryHandler) Delete(c *gin.Context) {
	key := c.Param("id")
	found, err := h.ltm.Forget(key)
	if err != nil {
		h.logger.Error("memory delete error", zap.Error(err))
		api.InternalErrorWithDetails(c, "memory delete error", err)
		return
	}
	if !found {
		api.NotFound(c, "memory")
		return
	}
	api.Success(c, gin.H{"deleted": key})
}

// Stats handles GET /api/memories/stats
func (h *MemoryHandler) Stats(c *gin.Context) {
	total, _ := h.ltm.List("", 10000)
	counts := map[string]int{
		"core":         0,
		"daily":        0,
		"conversation": 0,
		"custom":       0,
		"total":        len(total),
	}
	for _, e := range total {
		cat := string(e.Category)
		switch cat {
		case "core", "daily", "conversation":
			counts[cat]++
		default:
			counts["custom"]++
		}
	}
	api.Success(c, gin.H{"stats": counts})
}

// ImportMD handles POST /api/memories/import-md
// Splits a Markdown document by H2 headings and stores each section as a memory.
func (h *MemoryHandler) ImportMD(c *gin.Context) {
	var req struct {
		Content  string `json:"content" binding:"required"`
		Category string `json:"category"`
		Strategy string `json:"strategy"` // "db_only" | "file_only" | "both" (default: db_only)
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}
	cat := memory.Category(req.Category)
	if cat == "" {
		cat = memory.CategoryCore
	}

	sections := splitMarkdownByH2(req.Content)
	if len(sections) == 0 {
		// No H2 headings — import the whole content as one entry
		key := "import-" + time.Now().Format("20060102-150405")
		sections = []mdSection{{Key: key, Content: req.Content}}
	}

	var imported int
	var failures []string
	for _, sec := range sections {
		if err := h.ltm.Store(sec.Key, sec.Content, cat); err != nil {
			failures = append(failures, sec.Key)
		} else {
			imported++
		}
	}

	api.Success(c, gin.H{
		"imported": imported,
		"failures": failures,
		"total":    len(sections),
	})
}

type mdSection struct {
	Key     string
	Content string
}

// splitMarkdownByH2 splits a Markdown document into sections at each ## heading.
func splitMarkdownByH2(content string) []mdSection {
	lines := strings.Split(content, "\n")
	var sections []mdSection
	var currentKey string
	var currentLines []string

	flush := func() {
		if currentKey == "" {
			return
		}
		body := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if body != "" {
			sections = append(sections, mdSection{Key: currentKey, Content: body})
		}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			flush()
			currentKey = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentLines = nil
		} else if currentKey != "" {
			currentLines = append(currentLines, line)
		}
	}
	flush()
	return sections
}
