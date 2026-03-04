package memory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// DistilledItem is one memory entry extracted from daily notes by the LLM.
type DistilledItem struct {
	Key     string `json:"key"`
	Content string `json:"content"`
}

// DistillFn is called by HygieneRunner to distil raw note text into structured memories.
type DistillFn func(ctx context.Context, notes string) ([]DistilledItem, error)

// HygieneConfig controls the automatic memory cleanup schedule.
type HygieneConfig struct {
	// How often hygiene runs (default 24h).
	Interval time.Duration
	// Archive daily-notes files older than this many days (default 30).
	ArchiveDailyAfterDays int
	// Delete archived files older than this many days (default 90).
	PurgeArchiveAfterDays int
	// Delete conversation-category memories older than this many days (default 180).
	PruneConversationAfterDays int
	// Delete inactive sessions older than this many days (default 30).
	PruneSessionsAfterDays int
}

func defaultHygieneConfig() HygieneConfig {
	return HygieneConfig{
		Interval:                   24 * time.Hour,
		ArchiveDailyAfterDays:      30,
		PurgeArchiveAfterDays:      90,
		PruneConversationAfterDays: 180,
		PruneSessionsAfterDays:     30,
	}
}

// HygieneRunner runs periodic memory cleanup tasks in a background goroutine.
type HygieneRunner struct {
	store      *Store
	ltm        *LTMStore
	notesDir   string
	archiveDir string
	cfg        HygieneConfig
	distiller  DistillFn
	logger     *zap.Logger
}

// NewHygieneRunner creates a HygieneRunner.
func NewHygieneRunner(store *Store, ltm *LTMStore, notesDir, archiveDir string, cfg HygieneConfig, logger *zap.Logger) *HygieneRunner {
	if cfg.Interval == 0 {
		cfg = defaultHygieneConfig()
	}
	return &HygieneRunner{
		store:      store,
		ltm:        ltm,
		notesDir:   notesDir,
		archiveDir: archiveDir,
		cfg:        cfg,
		logger:     logger,
	}
}

// SetDistiller injects the LLM-backed distillation function.
func (r *HygieneRunner) SetDistiller(fn DistillFn) {
	r.distiller = fn
}

// Start begins the background hygiene loop.
func (r *HygieneRunner) Start(ctx context.Context) {
	go func() {
		r.run()
		ticker := time.NewTicker(r.cfg.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r.run()
			}
		}
	}()
}

func (r *HygieneRunner) run() {
	r.logger.Debug("memory hygiene: starting")

	// 1. Archive daily-notes
	if r.notesDir != "" {
		archived := r.archiveDailyNotes()
		if archived > 0 {
			r.logger.Info("memory hygiene: archived old daily notes", zap.Int("count", archived))
		}
	}

	// 2. Purge old archive files
	if r.archiveDir != "" {
		purged := r.purgeOldArchives()
		if purged > 0 {
			r.logger.Info("memory hygiene: purged old archives", zap.Int("count", purged))
		}
	}

	// 3. Prune old conversation-category memories
	if r.ltm != nil {
		cutoff := time.Now().AddDate(0, 0, -r.cfg.PruneConversationAfterDays)
		n, err := r.ltm.DeleteByCategory(CategoryConversation, cutoff)
		if err != nil {
			r.logger.Warn("memory hygiene: prune conversation failed", zap.Error(err))
		} else if n > 0 {
			r.logger.Info("memory hygiene: pruned conversation memories", zap.Int64("count", n))
		}
	}

	// 4. Prune inactive sessions (借鉴 ZeroClaw)
	if r.store != nil {
		cutoff := time.Now().AddDate(0, 0, -r.cfg.PruneSessionsAfterDays)
		n, err := r.store.DeleteInactiveSessions(cutoff)
		if err != nil {
			r.logger.Warn("memory hygiene: prune inactive sessions failed", zap.Error(err))
		} else if n > 0 {
			r.logger.Info("memory hygiene: pruned inactive sessions", zap.Int64("count", n))
		}
	}

	// 5. Distil recent daily notes
	if r.distiller != nil && r.ltm != nil && r.notesDir != "" {
		r.distillRecentNotes()
	}

	r.logger.Debug("memory hygiene: done")
}

// ParseDistilledItems parses a JSON array of DistilledItem from an LLM response.
func ParseDistilledItems(raw string) ([]DistilledItem, error) {
	start := -1
	for i, c := range raw {
		if c == '[' {
			start = i
			break
		}
	}
	end := -1
	for i := len(raw) - 1; i >= 0; i-- {
		if raw[i] == ']' {
			end = i
			break
		}
	}
	if start < 0 || end < 0 || end < start {
		return nil, nil
	}
	var items []DistilledItem
	if err := json.Unmarshal([]byte(raw[start:end+1]), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *HygieneRunner) distillRecentNotes() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	now := time.Now()
	for i := 1; i <= 3; i++ {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("20060102")
		done, err := r.ltm.IsDistilled(dateStr)
		if err != nil || done { continue }
		dayFile := filepath.Join(r.notesDir, date.Format("200601"), dateStr+".md")
		data, err := os.ReadFile(dayFile)
		if err != nil { _ = r.ltm.MarkDistilled(dateStr); continue }
		if len(data) < 50 { _ = r.ltm.MarkDistilled(dateStr); continue }
		items, err := r.distiller(ctx, string(data))
		if err != nil { continue }
		for _, item := range items {
			if item.Key != "" && item.Content != "" {
				_ = r.ltm.Store(item.Key, item.Content, CategoryDaily)
			}
		}
		_ = r.ltm.MarkDistilled(dateStr)
	}
}

func (r *HygieneRunner) archiveDailyNotes() int {
	cutoff := time.Now().AddDate(0, 0, -r.cfg.ArchiveDailyAfterDays)
	count := 0
	monthDirs, err := os.ReadDir(r.notesDir)
	if err != nil { return 0 }
	for _, md := range monthDirs {
		if !md.IsDir() { continue }
		monthPath := filepath.Join(r.notesDir, md.Name())
		dayFiles, err := os.ReadDir(monthPath)
		if err != nil { continue }
		for _, df := range dayFiles {
			if df.IsDir() || filepath.Ext(df.Name()) != ".md" { continue }
			fullPath := filepath.Join(monthPath, df.Name())
			info, err := df.Info()
			if err != nil || info.ModTime().After(cutoff) { continue }
			if err := r.appendFileToArchive(fullPath, df.Name()); err == nil {
				_ = os.Remove(fullPath)
				count++
			}
		}
		_ = os.Remove(monthPath)
	}
	return count
}

func (r *HygieneRunner) appendFileToArchive(srcPath, filename string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil { return err }
	if err := os.MkdirAll(r.archiveDir, 0o755); err != nil { return err }
	archFile := filepath.Join(r.archiveDir, "notes-"+time.Now().Format("2006-01")+".md")
	f, err := os.OpenFile(archFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil { return err }
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func (r *HygieneRunner) purgeOldArchives() int {
	cutoff := time.Now().AddDate(0, 0, -r.cfg.PurgeArchiveAfterDays)
	entries, err := os.ReadDir(r.archiveDir)
	if err != nil { return 0 }
	count := 0
	for _, e := range entries {
		if e.IsDir() { continue }
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) { continue }
		if err := os.Remove(filepath.Join(r.archiveDir, e.Name())); err == nil { count++ }
	}
	return count
}
