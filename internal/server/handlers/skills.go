// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

const defaultMarketBaseURL = "https://market.gopaw.dev"

// SkillsHandler handles /api/skills routes.
type SkillsHandler struct {
	manager   *skill.Manager
	skillsDir string
	logger    *zap.Logger
}

// NewSkillsHandler creates a SkillsHandler.
func NewSkillsHandler(m *skill.Manager, skillsDir string, logger *zap.Logger) *SkillsHandler {
	return &SkillsHandler{manager: m, skillsDir: skillsDir, logger: logger}
}

type skillInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Version     string `json:"version"`
	Level       int    `json:"level"`
	Enabled     bool   `json:"enabled"`
}

// List handles GET /api/skills.
func (h *SkillsHandler) List(c *gin.Context) {
	entries := h.manager.Registry().All()
	out := make([]skillInfo, 0, len(entries))
	for _, e := range entries {
		out = append(out, skillInfo{
			Name:        e.Manifest.Name,
			DisplayName: e.Manifest.DisplayName,
			Description: e.Manifest.Description,
			Author:      e.Manifest.Author,
			Version:     e.Manifest.Version,
			Level:       int(e.Manifest.Level),
			Enabled:     e.Enabled,
		})
	}
	api.Success(c, gin.H{"skills": out})
}

// Reload handles POST /api/skills/reload.
func (h *SkillsHandler) Reload(c *gin.Context) {
	if err := h.manager.Reload(); err != nil {
		h.logger.Error("skill reload failed", zap.Error(err))
		api.InternalErrorWithDetails(c, "skill reload failed", err)
		return
	}
	h.logger.Info("skills reloaded")
	api.Success(c, gin.H{"ok": true, "count": len(h.manager.Registry().All())})
}

// SetEnabled handles PUT /api/skills/:name/enabled.
func (h *SkillsHandler) SetEnabled(c *gin.Context) {
	name := c.Param("name")

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		api.BadRequestWithError(c, "invalid request body", err)
		return
	}

	if err := h.manager.Registry().SetEnabled(name, body.Enabled); err != nil {
		api.NotFound(c, "skill")
		return
	}

	h.logger.Info("skill enabled state changed",
		zap.String("name", name), zap.Bool("enabled", body.Enabled))
	api.Success(c, gin.H{"ok": true})
}

// Install handles POST /api/skills/install.
//
// Request body:
//
//	{
//	  "name":        "translator",   // required
//	  "version":     "latest",       // optional, defaults to "latest"
//	  "source":      "market",       // optional, defaults to "market"
//	  "package_url": "https://..."   // optional; if set, skips market lookup
//	}
//
// Flow:
//  1. If package_url is provided, download directly.
//  2. Otherwise, call the skill market API to get the download URL.
//  3. Download the zip and extract to {skillsDir}/{name}/.
//  4. Reload the skill manager so the new skill is immediately available.
func (h *SkillsHandler) Install(c *gin.Context) {
	var req struct {
		Name       string `json:"name"`
		Version    string `json:"version"`
		Source     string `json:"source"`
		PackageURL string `json:"package_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequest(c, "invalid request body")
		return
	}
	if req.Name == "" {
		api.BadRequest(c, "name is required")
		return
	}
	if req.Version == "" {
		req.Version = "latest"
	}
	if req.Source == "" {
		req.Source = "market"
	}

	// Resolve package URL
	packageURL := req.PackageURL
	if packageURL == "" {
		var err error
		packageURL, err = h.fetchPackageURL(req.Name, req.Version, req.Source)
		if err != nil {
			h.logger.Error("skill market lookup failed",
				zap.String("name", req.Name), zap.Error(err))
			api.BadGatewayWithError(c, "market lookup failed", err)
			return
		}
	}

	// Download and extract
	destDir := filepath.Join(h.skillsDir, req.Name)
	if err := downloadAndExtract(packageURL, destDir); err != nil {
		h.logger.Error("skill install failed",
			zap.String("name", req.Name), zap.Error(err))
		api.InternalErrorWithDetails(c, "skill install failed", err)
		return
	}

	// Reload so the skill is immediately active
	if err := h.manager.Reload(); err != nil {
		h.logger.Warn("skill reload after install failed", zap.Error(err))
	}

	h.logger.Info("skill installed", zap.String("name", req.Name), zap.String("version", req.Version))
	api.Success(c, gin.H{"ok": true, "name": req.Name})
}

// fetchPackageURL calls the skill market API to get the download URL for a skill.
// Market base URL defaults to https://market.gopaw.dev, override with GOPAW_MARKET_URL env.
func (h *SkillsHandler) fetchPackageURL(name, version, source string) (string, error) {
	marketBase := os.Getenv("GOPAW_MARKET_URL")
	if marketBase == "" {
		marketBase = defaultMarketBaseURL
	}

	body, _ := json.Marshal(map[string]string{
		"version": version,
		"source":  source,
	})

	resp, err := http.Post(
		fmt.Sprintf("%s/api/skills/%s/install", marketBase, name),
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return "", fmt.Errorf("calling market API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("market returned status %d", resp.StatusCode)
	}

	var result struct {
		PackageURL string `json:"packageUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding market response: %w", err)
	}
	if result.PackageURL == "" {
		return "", fmt.Errorf("market returned empty packageUrl")
	}
	return result.PackageURL, nil
}

// downloadAndExtract downloads a zip file from url and extracts it into destDir.
// If the zip has a single top-level directory, its contents are extracted directly
// into destDir (so destDir/{name}/manifest.yaml, not destDir/{name}/{name}/manifest.yaml).
func downloadAndExtract(url, destDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	// Detect common top-level prefix to strip (e.g. "translator-1.0.0/")
	prefix := zipTopLevelPrefix(r)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	for _, f := range r.File {
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel == "" || rel == "/" {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(rel))

		// Guard against zip-slip
		if !strings.HasPrefix(dest, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(dest, 0o755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := extractFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// zipTopLevelPrefix returns the common top-level directory prefix in a zip,
// e.g. "translator-1.0.0/" if all entries start with that. Returns "" if none.
func zipTopLevelPrefix(r *zip.Reader) string {
	if len(r.File) == 0 {
		return ""
	}
	first := r.File[0].Name
	slash := strings.Index(first, "/")
	if slash < 0 {
		return ""
	}
	prefix := first[:slash+1]
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, prefix) {
			return ""
		}
	}
	return prefix
}
