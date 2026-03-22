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
// Installs a skill from the market. Request body:
//
//	{
//	  "name":    "translator",  // required
//	  "version": "latest",      // optional, defaults to "latest"
//	  "source":  "market",      // optional
//	}
//
// Flow:
//  1. Call market API to get the zip download URL.
//  2. Download and extract zip to {skillsDir}/{name}/.
//  3. Reload skill manager.
func (h *SkillsHandler) Install(c *gin.Context) {
	var req struct {
		Name    string `json:"name"`
		Version string `json:"version"`
		Source  string `json:"source"`
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

	packageURL, err := h.fetchPackageURL(req.Name, req.Version, req.Source)
	if err != nil {
		h.logger.Error("market lookup failed",
			zap.String("name", req.Name), zap.Error(err))
		api.BadGatewayWithError(c, "market lookup failed", err)
		return
	}

	destDir := filepath.Join(h.skillsDir, req.Name)
	if err := downloadAndExtract(packageURL, destDir); err != nil {
		h.logger.Error("skill install failed",
			zap.String("name", req.Name), zap.Error(err))
		api.InternalErrorWithDetails(c, "skill install failed", err)
		return
	}

	if err := h.manager.Reload(); err != nil {
		h.logger.Warn("skill reload after install failed", zap.Error(err))
	}

	h.logger.Info("skill installed", zap.String("name", req.Name), zap.String("version", req.Version))
	api.Success(c, gin.H{"ok": true, "name": req.Name})
}

// ImportZip handles POST /api/skills/import.
// Accepts a multipart/form-data upload with a "file" field containing a zip archive.
// The zip must contain a valid skill directory (manifest.yaml required).
// The skill name is read from the manifest inside the zip.
func (h *SkillsHandler) ImportZip(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		api.BadRequest(c, "file field is required")
		return
	}
	if !strings.HasSuffix(strings.ToLower(fh.Filename), ".zip") {
		api.BadRequest(c, "only .zip files are supported")
		return
	}

	f, err := fh.Open()
	if err != nil {
		api.InternalErrorWithDetails(c, "cannot open uploaded file", err)
		return
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		api.InternalErrorWithDetails(c, "cannot read uploaded file", err)
		return
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		api.BadRequest(c, "invalid zip file")
		return
	}

	// Read manifest.yaml from zip to determine skill name.
	prefix := zipTopLevelPrefix(zr)
	skillName, err := readSkillNameFromZip(zr, prefix)
	if err != nil {
		api.BadRequest(c, "manifest.yaml not found or invalid: "+err.Error())
		return
	}

	destDir := filepath.Join(h.skillsDir, skillName)
	if err := extractZip(zr, prefix, destDir); err != nil {
		h.logger.Error("skill import failed",
			zap.String("name", skillName), zap.Error(err))
		api.InternalErrorWithDetails(c, "skill import failed", err)
		return
	}

	if err := h.manager.Reload(); err != nil {
		h.logger.Warn("skill reload after import failed", zap.Error(err))
	}

	h.logger.Info("skill imported", zap.String("name", skillName))
	api.Success(c, gin.H{"ok": true, "name": skillName})
}

// MarketList handles GET /api/skills/market.
// Proxies the GoPaw Market public skills list to avoid CORS issues.
// Query params: q, featured, page, page_size are forwarded transparently.
func (h *SkillsHandler) MarketList(c *gin.Context) {
	marketBase := marketBaseURL()

	// Build query string from incoming params.
	u := fmt.Sprintf("%s/api/public/skills", marketBase)
	q := c.Request.URL.Query()
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	resp, err := http.Get(u)
	if err != nil {
		h.logger.Error("market fetch failed", zap.Error(err))
		api.BadGatewayWithError(c, "market unavailable", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		api.InternalErrorWithDetails(c, "reading market response", err)
		return
	}

	c.Data(resp.StatusCode, "application/json; charset=utf-8", body)
}

// fetchPackageURL calls the skill market install API to get the zip download URL.
func (h *SkillsHandler) fetchPackageURL(name, version, source string) (string, error) {
	marketBase := marketBaseURL()

	payload, _ := json.Marshal(map[string]string{
		"version": version,
		"source":  source,
	})

	resp, err := http.Post(
		fmt.Sprintf("%s/api/public/skills/%s/install", marketBase, name),
		"application/json",
		bytes.NewReader(payload),
	)
	if err != nil {
		return "", fmt.Errorf("calling market API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("market returned status %d", resp.StatusCode)
	}

	// Market response: {"code":0,"data":{"ok":true,"package_url":"/data/skills/...","version":"..."}}
	var result struct {
		Code int `json:"code"`
		Data struct {
			OK         bool   `json:"ok"`
			PackageURL string `json:"package_url"`
			Version    string `json:"version"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding market response: %w", err)
	}
	if result.Code != 0 {
		return "", fmt.Errorf("market error code %d", result.Code)
	}
	if result.Data.PackageURL == "" {
		return "", fmt.Errorf("market returned empty package_url")
	}

	// package_url may be relative (e.g. "/data/skills/foo-1.0.zip")
	pkgURL := result.Data.PackageURL
	if strings.HasPrefix(pkgURL, "/") {
		pkgURL = marketBase + pkgURL
	}
	return pkgURL, nil
}

// marketBaseURL returns the market base URL from env or default.
func marketBaseURL() string {
	if v := os.Getenv("GOPAW_MARKET_URL"); v != "" {
		return v
	}
	return defaultMarketBaseURL
}

// downloadAndExtract downloads a zip from url and extracts it into destDir.
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

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	return extractZip(zr, zipTopLevelPrefix(zr), destDir)
}

// extractZip extracts zip entries (stripping the given prefix) into destDir.
func extractZip(zr *zip.Reader, prefix, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	for _, f := range zr.File {
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
		if err := extractZipFile(f, dest); err != nil {
			return err
		}
	}
	return nil
}

// extractZipFile extracts a single zip entry to dest path.
func extractZipFile(f *zip.File, dest string) error {
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

// zipTopLevelPrefix returns the common top-level directory prefix in the zip, if any.
func zipTopLevelPrefix(zr *zip.Reader) string {
	if len(zr.File) == 0 {
		return ""
	}
	first := zr.File[0].Name
	slash := strings.Index(first, "/")
	if slash < 0 {
		return ""
	}
	prefix := first[:slash+1]
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, prefix) {
			return ""
		}
	}
	return prefix
}

// readSkillNameFromZip finds manifest.yaml in the zip and returns the skill name.
func readSkillNameFromZip(zr *zip.Reader, prefix string) (string, error) {
	for _, f := range zr.File {
		rel := strings.TrimPrefix(f.Name, prefix)
		if rel != "manifest.yaml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			return "", err
		}

		// Quick YAML name extraction without full parse dependency.
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "name:") {
				name := strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				name = strings.Trim(name, `"'`)
				if name != "" {
					return name, nil
				}
			}
		}
		return "", fmt.Errorf("name field not found in manifest.yaml")
	}
	return "", fmt.Errorf("manifest.yaml not found in zip")
}
