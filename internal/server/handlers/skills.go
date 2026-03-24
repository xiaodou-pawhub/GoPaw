// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/llm"
	"github.com/gopaw/gopaw/internal/permission"
	"github.com/gopaw/gopaw/internal/skill"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const defaultMarketBaseURL = "https://skills.gopaw.top"

// SkillsHandler handles /api/skills routes.
type SkillsHandler struct {
	manager     *skill.Manager
	skillsDir   string
	llm         llm.Client // may be nil if LLM is not configured
	permChecker *permission.Checker
	logger      *zap.Logger
}

// NewSkillsHandler creates a SkillsHandler.
func NewSkillsHandler(m *skill.Manager, skillsDir string, llmClient llm.Client, permChecker *permission.Checker, logger *zap.Logger) *SkillsHandler {
	return &SkillsHandler{manager: m, skillsDir: skillsDir, llm: llmClient, permChecker: permChecker, logger: logger}
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
	
	// In team mode, filter skills based on user permissions
	userID, _ := c.Get("gopaw_user_id")
	isTeamMode := userID != nil && h.permChecker != nil
	
	for _, e := range entries {
		// In team mode, check if user has access to this skill
		if isTeamMode {
			hasAccess, err := h.permChecker.CanUseResource(c.Request.Context(), userID.(string), "skill", e.Manifest.Name)
			if err != nil || !hasAccess {
				// Skip skills the user doesn't have access to
				continue
			}
		}
		
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
//
// Accepts a multipart/form-data upload with a "file" field containing a zip archive.
// Supports two cases:
//  1. Zip contains a valid manifest.yaml — name is read from the manifest.
//  2. Zip has no manifest.yaml — reads SKILL.md / prompt.md / code files and uses the
//     LLM to auto-generate a manifest.yaml and prompt.md before importing.
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

	// Extract everything to a temp directory.
	tmpDir, err := os.MkdirTemp("", "gopaw-skill-*")
	if err != nil {
		api.InternalErrorWithDetails(c, "cannot create temp dir", err)
		return
	}
	defer os.RemoveAll(tmpDir)

	if err := extractZipToDir(zr, tmpDir); err != nil {
		api.InternalErrorWithDetails(c, "skill import failed", err)
		return
	}

	// Locate the skill root: the shallowest directory containing manifest.yaml,
	// SKILL.md, prompt.md, or any code file.
	skillRoot, err := findSkillRoot(tmpDir)
	if err != nil {
		api.BadRequest(c, "cannot locate skill files in zip: expected manifest.yaml, SKILL.md, or prompt.md")
		return
	}

	manifestPath := filepath.Join(skillRoot, "manifest.yaml")
	hasManifest := fileExists(manifestPath)

	var skillName string

	if hasManifest {
		// Read name directly from existing manifest.
		name, err := readSkillNameFromManifestFile(manifestPath)
		if err != nil {
			api.BadRequest(c, "manifest.yaml invalid: "+err.Error())
			return
		}
		skillName = name
		h.logger.Info("skill zip has manifest", zap.String("name", skillName))
	} else {
		// No manifest — auto-generate using LLM.
		if h.llm == nil {
			api.BadRequest(c, "skill zip has no manifest.yaml and LLM is not configured; cannot auto-generate manifest")
			return
		}
		name, err := h.autoGenerateManifest(c.Request.Context(), skillRoot)
		if err != nil {
			h.logger.Error("auto-generate manifest failed", zap.Error(err))
			api.InternalErrorWithDetails(c, "failed to auto-generate manifest.yaml", err)
			return
		}
		skillName = name
		h.logger.Info("auto-generated manifest for skill", zap.String("name", skillName))
	}

	// Copy skill root to skills directory.
	destDir := filepath.Join(h.skillsDir, skillName)

	// Duplicate detection: return 409 unless overwrite=true is passed as query param.
	overwrite := c.Query("overwrite") == "true"
	if !overwrite {
		if _, statErr := os.Stat(destDir); statErr == nil {
			c.JSON(http.StatusConflict, gin.H{
				"code":    409,
				"message": fmt.Sprintf("skill '%s' already exists", skillName),
				"name":    skillName,
			})
			return
		}
	}

	if err := copyDir(skillRoot, destDir); err != nil {
		h.logger.Error("skill copy failed", zap.String("name", skillName), zap.Error(err))
		api.InternalErrorWithDetails(c, "skill import failed", err)
		return
	}

	if err := h.manager.Reload(); err != nil {
		h.logger.Warn("skill reload after import failed", zap.Error(err))
	}

	status := "imported"
	if hasManifest {
		status = "imported_with_manifest"
	}

	h.logger.Info("skill imported", zap.String("name", skillName), zap.String("status", status))
	api.Success(c, gin.H{"ok": true, "name": skillName, "status": status})
}

// MarketList handles GET /api/skills/market.
// Proxies the GoPaw Market public skills list to avoid CORS issues.
func (h *SkillsHandler) MarketList(c *gin.Context) {
	marketBase := marketBaseURL()

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

// ---- Auto-generate manifest ----

// autoGenerateManifest reads skill files from skillRoot, calls the LLM to produce
// name/display_name/description/level, writes manifest.yaml and ensures prompt.md exists.
// Returns the generated skill name.
func (h *SkillsHandler) autoGenerateManifest(ctx context.Context, skillRoot string) (string, error) {
	const maxFileBytes = 20 * 1024
	const maxTotalBytes = 50 * 1024

	var parts []string
	var totalBytes int

	// Priority files: SKILL.md, prompt.md
	for _, name := range []string{"SKILL.md", "prompt.md", "README.md"} {
		if totalBytes >= maxTotalBytes {
			break
		}
		data, err := os.ReadFile(filepath.Join(skillRoot, name))
		if err != nil {
			continue
		}
		chunk := limitBytes(data, min(maxFileBytes, maxTotalBytes-totalBytes))
		parts = append(parts, "### "+name+"\n"+string(chunk))
		totalBytes += len(chunk)
	}

	// Meta files for name/description hints
	for _, name := range []string{"_meta.json", "skill.json", "package.json"} {
		if totalBytes >= maxTotalBytes {
			break
		}
		data, err := os.ReadFile(filepath.Join(skillRoot, name))
		if err != nil {
			continue
		}
		chunk := limitBytes(data, min(maxFileBytes, maxTotalBytes-totalBytes))
		parts = append(parts, "### "+name+"\n"+string(chunk))
		totalBytes += len(chunk)
	}

	// Detect code files to decide level; include first code file as context
	hasCode := false
	entries, _ := os.ReadDir(skillRoot)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".py" || ext == ".js" || ext == ".ts" || ext == ".sh" {
			hasCode = true
			if totalBytes < maxTotalBytes {
				data, err := os.ReadFile(filepath.Join(skillRoot, e.Name()))
				if err == nil {
					chunk := limitBytes(data, min(maxFileBytes, maxTotalBytes-totalBytes))
					parts = append(parts, "### "+e.Name()+"\n"+string(chunk))
					totalBytes += len(chunk)
				}
			}
			break
		}
	}

	if len(parts) == 0 {
		return "", fmt.Errorf("no recognizable skill files found (expected SKILL.md, prompt.md, README.md, or code files)")
	}

	level := 1
	if hasCode {
		level = 2
	}

	prompt := `You are analyzing a third-party AI skill package. Based on the files below, generate a JSON object with exactly these fields:
- "name": a lowercase snake_case identifier (e.g. "web_search_helper")
- "display_name": a human-readable name in the same language as the skill content
- "description": one sentence describing what this skill does

Respond ONLY with valid JSON, no explanation, no markdown fences.

Files:
` + strings.Join(parts, "\n\n")

	ctxTimeout, cancel := context.WithTimeout(ctx, 50*time.Second)
	defer cancel()

	resp, err := h.llm.Chat(ctxTimeout, llm.ChatRequest{
		Messages:  []llm.ChatMessage{{Role: llm.RoleUser, Content: prompt}},
		MaxTokens: 256,
	})
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	content := strings.TrimSpace(resp.Message.Content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var meta struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(content), &meta); err != nil {
		return "", fmt.Errorf("LLM returned invalid JSON: %w\ncontent: %s", err, content)
	}
	if meta.Name == "" {
		return "", fmt.Errorf("LLM did not return a skill name")
	}

	// Build and write manifest.yaml
	manifest := map[string]any{
		"name":         meta.Name,
		"display_name": meta.DisplayName,
		"description":  meta.Description,
		"version":      "1.0.0",
		"level":        level,
		"source":       "imported",
	}
	manifestBytes, err := yaml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath.Join(skillRoot, "manifest.yaml"), manifestBytes, 0o644); err != nil {
		return "", fmt.Errorf("write manifest.yaml: %w", err)
	}

	// Ensure prompt.md exists: prefer existing prompt.md, then SKILL.md, then README.md
	promptMDPath := filepath.Join(skillRoot, "prompt.md")
	if !fileExists(promptMDPath) {
		for _, src := range []string{"SKILL.md", "README.md"} {
			srcPath := filepath.Join(skillRoot, src)
			if data, err := os.ReadFile(srcPath); err == nil {
				_ = os.WriteFile(promptMDPath, data, 0o644)
				break
			}
		}
	}

	return meta.Name, nil
}

// ---- Helpers ----

// findSkillRoot does a BFS under root for the shallowest directory containing
// manifest.yaml, SKILL.md, prompt.md, or any script/code file.
func findSkillRoot(root string) (string, error) {
	queue := []string{root}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if name == "manifest.yaml" || name == "SKILL.md" || name == "prompt.md" ||
				name == "README.md" || name == "_meta.json" || name == "skill.json" ||
				ext == ".py" || ext == ".js" || ext == ".ts" || ext == ".sh" {
				return dir, nil
			}
		}

		for _, e := range entries {
			if e.IsDir() {
				queue = append(queue, filepath.Join(dir, e.Name()))
			}
		}
	}
	return "", fmt.Errorf("no skill files found")
}

// extractZipToDir extracts all entries of zr into destDir, preserving the zip's
// internal directory structure (top-level folder included).
func extractZipToDir(zr *zip.Reader, destDir string) error {
	for _, f := range zr.File {
		cleaned := filepath.Clean(f.Name)
		if strings.HasPrefix(cleaned, "..") {
			continue // zip-slip guard
		}
		target := filepath.Join(destDir, cleaned)
		if !strings.HasPrefix(target, filepath.Clean(destDir)+string(os.PathSeparator)) {
			continue // zip-slip guard
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractZipFile(f, target); err != nil {
			return err
		}
	}
	return nil
}

// copyDir recursively copies src directory to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func limitBytes(data []byte, max int) []byte {
	if len(data) > max {
		return data[:max]
	}
	return data
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// readSkillNameFromManifestFile reads name from a manifest.yaml file on disk.
func readSkillNameFromManifestFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var m struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal(data, &m); err != nil {
		return "", err
	}
	if m.Name == "" {
		return "", fmt.Errorf("name field is required")
	}
	return m.Name, nil
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

	pkgURL := result.Data.PackageURL
	if strings.HasPrefix(pkgURL, "/") {
		pkgURL = marketBase + pkgURL
	}
	return pkgURL, nil
}

// marketBaseURL returns the market base URL from env or default.
func marketBaseURL() string {
	if v := os.Getenv("GOPAW_MARKET_URL"); v != ""  {
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
// Kept for backward compatibility with Install flow.
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
