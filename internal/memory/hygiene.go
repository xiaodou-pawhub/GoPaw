package memory

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

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
}

func defaultHygieneConfig() HygieneConfig {
	return HygieneConfig{
		Interval:                   24 * time.Hour,
		ArchiveDailyAfterDays:      30,
		PurgeArchiveAfterDays:      90,
		PruneConversationAfterDays: 180,
	}
}

// HygieneRunner runs periodic memory cleanup tasks in a background goroutine.
type HygieneRunner struct {
	ltm        *LTMStore
	notesDir   string // memory/notes/
	archiveDir string // memory/archive/
	cfg        HygieneConfig
	logger     *zap.Logger
}

// NewHygieneRunner creates a HygieneRunner. Use zero HygieneConfig for defaults.
func NewHygieneRunner(ltm *LTMStore, notesDir, archiveDir string, cfg HygieneConfig, logger *zap.Logger) *HygieneRunner {
	if cfg.Interval == 0 {
		cfg = defaultHygieneConfig()
	}
	return &HygieneRunner{
		ltm:        ltm,
		notesDir:   notesDir,
		archiveDir: archiveDir,
		cfg:        cfg,
		logger:     logger,
	}
}

// Start begins the background hygiene loop. It runs once immediately, then on the configured interval.
// Cancel ctx to stop the loop.
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

	// 1. Archive daily-notes files older than ArchiveDailyAfterDays
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

	// 3. Prune old conversation-category memories from memories.db
	if r.ltm != nil {
		cutoff := time.Now().AddDate(0, 0, -r.cfg.PruneConversationAfterDays)
		n, err := r.ltm.DeleteByCategory(CategoryConversation, cutoff)
		if err != nil {
			r.logger.Warn("memory hygiene: prune conversation failed", zap.Error(err))
		} else if n > 0 {
			r.logger.Info("memory hygiene: pruned conversation memories", zap.Int64("count", n))
		}
	}

	r.logger.Debug("memory hygiene: done")
}

// archiveDailyNotes moves daily note files older than ArchiveDailyAfterDays
// from memory/notes/ to memory/archive/notes-YYYY-MM.md (append mode).
// Returns the number of files processed.
func (r *HygieneRunner) archiveDailyNotes() int {
	cutoff := time.Now().AddDate(0, 0, -r.cfg.ArchiveDailyAfterDays)
	count := 0

	// Walk month directories under notesDir
	monthDirs, err := os.ReadDir(r.notesDir)
	if err != nil {
		return 0
	}

	for _, md := range monthDirs {
		if !md.IsDir() {
			continue
		}
		monthPath := filepath.Join(r.notesDir, md.Name())
		dayFiles, err := os.ReadDir(monthPath)
		if err != nil {
			continue
		}
		for _, df := range dayFiles {
			if df.IsDir() || filepath.Ext(df.Name()) != ".md" {
				continue
			}
			fullPath := filepath.Join(monthPath, df.Name())
			info, err := df.Info()
			if err != nil || info.ModTime().After(cutoff) {
				continue
			}
			// Append content to archive, then delete
			if err := r.appendFileToArchive(fullPath, df.Name()); err == nil {
				_ = os.Remove(fullPath)
				count++
			}
		}
		// Remove empty month directory
		_ = os.Remove(monthPath) // silently fails if not empty
	}
	return count
}

// appendFileToArchive appends the contents of srcPath to the monthly archive file.
func (r *HygieneRunner) appendFileToArchive(srcPath, filename string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(r.archiveDir, 0o755); err != nil {
		return err
	}
	// Archive file named notes-YYYY-MM.md (derived from daily note filename YYYYMMDD.md)
	archFile := filepath.Join(r.archiveDir, "notes-"+time.Now().Format("2006-01")+".md")
	f, err := os.OpenFile(archFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

// purgeOldArchives deletes archive files in archiveDir older than PurgeArchiveAfterDays.
func (r *HygieneRunner) purgeOldArchives() int {
	cutoff := time.Now().AddDate(0, 0, -r.cfg.PurgeArchiveAfterDays)
	entries, err := os.ReadDir(r.archiveDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		if err := os.Remove(filepath.Join(r.archiveDir, e.Name())); err == nil {
			count++
		}
	}
	return count
}
