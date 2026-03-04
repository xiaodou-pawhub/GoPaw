package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/workspace"
	"go.uber.org/zap"
)

// MemoryFilesHandler handles /api/memory-files routes for MD-based memory files.
type MemoryFilesHandler struct {
	paths  *workspace.Paths
	logger *zap.Logger
}

// NewMemoryFilesHandler creates a MemoryFilesHandler.
func NewMemoryFilesHandler(paths *workspace.Paths, logger *zap.Logger) *MemoryFilesHandler {
	return &MemoryFilesHandler{paths: paths, logger: logger}
}

// --- MEMORY.md ---

// GetMemoryMD handles GET /api/memory-files/memory — reads MEMORY.md.
func (h *MemoryFilesHandler) GetMemoryMD(c *gin.Context) {
	data, err := os.ReadFile(h.paths.MemoryMDFile)
	if os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"content": ""})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(data)})
}

// PutMemoryMD handles PUT /api/memory-files/memory — writes MEMORY.md.
func (h *MemoryFilesHandler) PutMemoryMD(c *gin.Context) {
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := os.WriteFile(h.paths.MemoryMDFile, []byte(req.Content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Daily Notes ---

// noteFileInfo represents a daily note file entry.
type noteFileInfo struct {
	Date     string `json:"date"`     // YYYYMMDD
	Month    string `json:"month"`    // YYYYMM
	Path     string `json:"path"`     // relative path inside notes dir
	ModTime  int64  `json:"mod_time"` // unix milli
	Size     int64  `json:"size"`
}

// ListNotes handles GET /api/memory-files/notes
// Returns list of daily note files, grouped by month.
func (h *MemoryFilesHandler) ListNotes(c *gin.Context) {
	monthDirs, err := os.ReadDir(h.paths.MemoryNotesDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"notes": []noteFileInfo{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var notes []noteFileInfo
	for _, md := range monthDirs {
		if !md.IsDir() {
			continue
		}
		month := md.Name()
		dayFiles, err := os.ReadDir(filepath.Join(h.paths.MemoryNotesDir, month))
		if err != nil {
			continue
		}
		for _, df := range dayFiles {
			if df.IsDir() || filepath.Ext(df.Name()) != ".md" {
				continue
			}
			info, err := df.Info()
			if err != nil {
				continue
			}
			date := strings.TrimSuffix(df.Name(), ".md")
			notes = append(notes, noteFileInfo{
				Date:    date,
				Month:   month,
				Path:    filepath.Join(month, df.Name()),
				ModTime: info.ModTime().UnixMilli(),
				Size:    info.Size(),
			})
		}
	}
	// Sort by date descending
	sort.Slice(notes, func(i, j int) bool {
		return notes[i].Date > notes[j].Date
	})
	c.JSON(http.StatusOK, gin.H{"notes": notes})
}

// GetNote handles GET /api/memory-files/notes/:date (date = YYYYMMDD)
func (h *MemoryFilesHandler) GetNote(c *gin.Context) {
	date := c.Param("date")
	if !isValidDate(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYYMMDD"})
		return
	}
	month := date[:6]
	path := filepath.Join(h.paths.MemoryNotesDir, month, date+".md")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"content": "", "date": date})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(data), "date": date})
}

// PutNote handles PUT /api/memory-files/notes/:date — overwrites a daily note.
func (h *MemoryFilesHandler) PutNote(c *gin.Context) {
	date := c.Param("date")
	if !isValidDate(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYYMMDD"})
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	month := date[:6]
	dir := filepath.Join(h.paths.MemoryNotesDir, month)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Join(dir, date+".md")
	if err := os.WriteFile(path, []byte(req.Content), 0o644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "date": date})
}

// AppendNote handles POST /api/memory-files/notes/:date/append — appends content to a daily note.
func (h *MemoryFilesHandler) AppendNote(c *gin.Context) {
	date := c.Param("date")
	if !isValidDate(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYYMMDD"})
		return
	}
	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	month := date[:6]
	dir := filepath.Join(h.paths.MemoryNotesDir, month)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Join(dir, date+".md")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()

	// Check file is new (empty) to write a header
	info, _ := f.Stat()
	if info != nil && info.Size() == 0 {
		t, _ := time.Parse("20060102", date)
		header := "# " + t.Format("2006-01-02") + "\n\n"
		f.WriteString(header) //nolint:errcheck
	}
	f.WriteString(req.Content + "\n") //nolint:errcheck

	c.JSON(http.StatusOK, gin.H{"ok": true, "date": date})
}

// DeleteNote handles DELETE /api/memory-files/notes/:date
func (h *MemoryFilesHandler) DeleteNote(c *gin.Context) {
	date := c.Param("date")
	if !isValidDate(date) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}
	month := date[:6]
	path := filepath.Join(h.paths.MemoryNotesDir, month, date+".md")
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "note not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": date})
}

// --- Archive files ---

type archiveFileInfo struct {
	Name    string `json:"name"`    // e.g. "notes-2026-02.md"
	ModTime int64  `json:"mod_time"`
	Size    int64  `json:"size"`
}

// ListArchives handles GET /api/memory-files/archives
func (h *MemoryFilesHandler) ListArchives(c *gin.Context) {
	entries, err := os.ReadDir(h.paths.MemoryArchDir)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(http.StatusOK, gin.H{"archives": []archiveFileInfo{}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	var archives []archiveFileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		archives = append(archives, archiveFileInfo{
			Name:    e.Name(),
			ModTime: info.ModTime().UnixMilli(),
			Size:    info.Size(),
		})
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].Name > archives[j].Name
	})
	c.JSON(http.StatusOK, gin.H{"archives": archives})
}

// GetArchive handles GET /api/memory-files/archives/:name
func (h *MemoryFilesHandler) GetArchive(c *gin.Context) {
	name := filepath.Base(c.Param("name")) // sanitize
	path := filepath.Join(h.paths.MemoryArchDir, name)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "archive not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"content": string(data), "name": name})
}

// isValidDate checks that s is an 8-digit string (YYYYMMDD).
func isValidDate(s string) bool {
	if len(s) != 8 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
