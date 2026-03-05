package channel

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gopaw/gopaw/pkg/plugin"
	"go.uber.org/zap"
)

// MediaEntry represents a record in the MediaStore.
type MediaEntry struct {
	Path      string
	Meta      plugin.MediaMeta
	StoredAt  time.Time
}

// MediaStore provides a unified way to manage temporary media files.
// It uses SHA-256 hashing for content-addressable storage (deduplication).
type MediaStore struct {
	mu          sync.RWMutex
	refs        map[string]MediaEntry            // refID -> entry
	scopeToRefs map[string]map[string]struct{}   // scopeID -> set of refIDs
	refToScope  map[string]string                // refID -> scopeID

	baseDir     string
	logger      *zap.Logger
	maxAge      time.Duration
	stopJanitor chan struct{}
}

// NewMediaStore creates a new MediaStore.
func NewMediaStore(baseDir string, maxAge time.Duration, logger *zap.Logger) *MediaStore {
	s := &MediaStore{
		refs:        make(map[string]MediaEntry),
		scopeToRefs: make(map[string]map[string]struct{}),
		refToScope:  make(map[string]string),
		baseDir:     baseDir,
		logger:      logger.Named("media_store"),
		maxAge:      maxAge,
		stopJanitor: make(chan struct{}),
	}

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		s.logger.Error("failed to create media base directory", zap.Error(err))
	}

	go s.runJanitor()
	return s
}

// Store calculates the SHA-256 hash of the file at localPath, moves it to the
// managed directory with its hash as the filename, and returns a media:// reference.
// This ensures deduplication and consistency between IDs and filenames.
func (s *MediaStore) Store(localPath string, meta plugin.MediaMeta, scope string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("media_store: cannot open source: %w", err)
	}
	defer file.Close()

	// 1. Calculate SHA-256 Hash
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("media_store: hash failed: %w", err)
	}
	fullHash := hex.EncodeToString(hash.Sum(nil))
	// Use a truncated hash (12 chars) for more readable references, similar to Git short SHAs.
	hashSum := fullHash[:12]
	
	// Use the short hash as the unique ID
	refID := fmt.Sprintf("media://%s", hashSum)
	
	// Determine permanent destination path in the media directory
	ext := filepath.Ext(localPath)
	if ext == "" {
		ext = filepath.Ext(meta.Filename)
	}
	permPath := filepath.Join(s.baseDir, hashSum+ext)

	// 2. Check if we already have this content
	s.mu.RLock()
	_, exists := s.refs[refID]
	s.mu.RUnlock()

	if !exists {
		// Only move the file if it doesn't already exist in our store
		// We use Rename for speed, fallback to Copy if on different partitions
		err := os.Rename(localPath, permPath)
		if err != nil {
			// Fallback to copy+delete
			if err := copyFile(localPath, permPath); err != nil {
				return "", fmt.Errorf("media_store: failed to persist file: %w", err)
			}
			_ = os.Remove(localPath)
		}
	} else {
		// If it exists, we just delete the incoming temp file
		_ = os.Remove(localPath)
	}

	// 3. Register entry
	s.mu.Lock()
	defer s.mu.Unlock()

	s.refs[refID] = MediaEntry{
		Path:     permPath,
		Meta:     meta,
		StoredAt: time.Now(),
	}

	if s.scopeToRefs[scope] == nil {
		s.scopeToRefs[scope] = make(map[string]struct{})
	}
	s.scopeToRefs[scope][refID] = struct{}{}
	s.refToScope[refID] = scope

	return refID, nil
}

// Resolve returns the absolute local path for a given reference.
// If the memory record is missing, it attempts to find the file physically in the base directory.
func (s *MediaStore) Resolve(refID string) (string, error) {
	s.mu.RLock()
	entry, ok := s.refs[refID]
	s.mu.RUnlock()

	if ok {
		return entry.Path, nil
	}

	// Fallback: Try to find the file physically. 
	// We use a prefix match (hash*) to find the file regardless of whether it was stored 
	// with a short (12-char) or long (64-char) filename.
	hashSum := strings.TrimPrefix(refID, "media://")
	if len(hashSum) > 12 {
		hashSum = hashSum[:12]
	}
	
	matches, _ := filepath.Glob(filepath.Join(s.baseDir, hashSum+"*"))
	if len(matches) > 0 {
		return matches[0], nil
	}

	return "", fmt.Errorf("media_store: unknown reference: %s", refID)
}

// ResolveWithMeta returns both path and metadata for a given reference.
func (s *MediaStore) ResolveWithMeta(refID string) (string, plugin.MediaMeta, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.refs[refID]
	if !ok {
		return "", plugin.MediaMeta{}, fmt.Errorf("media_store: unknown reference: %s", refID)
	}
	return entry.Path, entry.Meta, nil
}

// Delete removes a single reference and its associated file.
func (s *MediaStore) Delete(refID string) error {
	s.mu.Lock()
	entry, ok := s.refs[refID]
	if !ok {
		s.mu.Unlock()
		return nil
	}

	path := entry.Path
	scope := s.refToScope[refID]

	delete(s.refs, refID)
	delete(s.refToScope, refID)
	if set, ok := s.scopeToRefs[scope]; ok {
		delete(set, refID)
		if len(set) == 0 {
			delete(s.scopeToRefs, scope)
		}
	}
	s.mu.Unlock()

	return os.Remove(path)
}

// ReleaseAll deletes all files associated with a given scope.
func (s *MediaStore) ReleaseAll(scope string) {
	s.mu.Lock()
	refSet, ok := s.scopeToRefs[scope]
	if !ok {
		s.mu.Unlock()
		return
	}

	var pathsToDelete []string
	for refID := range refSet {
		if entry, exists := s.refs[refID]; exists {
			pathsToDelete = append(pathsToDelete, entry.Path)
		}
		delete(s.refs, refID)
		delete(s.refToScope, refID)
	}
	delete(s.scopeToRefs, scope)
	s.mu.Unlock()

	for _, path := range pathsToDelete {
		_ = os.Remove(path)
	}
}

func (s *MediaStore) Close() {
	close(s.stopJanitor)
}

func (s *MediaStore) runJanitor() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.cleanExpired()
		case <-s.stopJanitor:
			return
		}
	}
}

func (s *MediaStore) cleanExpired() {
	if s.maxAge <= 0 {
		return
	}

	var pathsToDelete []string
	now := time.Now()

	s.mu.Lock()
	for refID, entry := range s.refs {
		if now.Sub(entry.StoredAt) > s.maxAge {
			pathsToDelete = append(pathsToDelete, entry.Path)
			if scope, ok := s.refToScope[refID]; ok {
				if set, ok := s.scopeToRefs[scope]; ok {
					delete(set, refID)
					if len(set) == 0 {
						delete(s.scopeToRefs, scope)
					}
				}
			}
			delete(s.refs, refID)
			delete(s.refToScope, refID)
		}
	}
	s.mu.Unlock()

	for _, path := range pathsToDelete {
		_ = os.Remove(path)
	}
}

func (s *MediaStore) TempPath(ext string) string {
	if ext != "" && !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}
	// Note: We still use UUID for the initial temporary creation to avoid collisions before hashing
	return filepath.Join(s.baseDir, fmt.Sprintf("upload_%d_%s", time.Now().UnixNano(), ext))
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}
