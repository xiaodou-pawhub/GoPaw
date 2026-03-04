package channel

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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

// Store registers an existing local file into the store and returns a virtual reference.
func (s *MediaStore) Store(localPath string, meta plugin.MediaMeta, scope string) (string, error) {
	if _, err := os.Stat(localPath); err != nil {
		return "", fmt.Errorf("media_store: file not found: %w", err)
	}

	refID := fmt.Sprintf("media://%s", uuid.New().String())

	s.mu.Lock()
	defer s.mu.Unlock()

	s.refs[refID] = MediaEntry{
		Path:     localPath,
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
func (s *MediaStore) Resolve(refID string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.refs[refID]
	if !ok {
		return "", fmt.Errorf("media_store: unknown reference: %s", refID)
	}
	return entry.Path, nil
}

// ReleaseAll deletes all files associated with a given scope.
func (s *MediaStore) ReleaseAll(scope string) {
	var pathsToDelete []string

	s.mu.Lock()
	refSet, ok := s.scopeToRefs[scope]
	if !ok {
		s.mu.Unlock()
		return
	}

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
	return filepath.Join(s.baseDir, fmt.Sprintf("tmp_%s%s", uuid.New().String(), ext))
}
