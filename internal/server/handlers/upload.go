// Package handlers contains Gin route handlers for all GoPaw HTTP API endpoints.
package handlers

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/pkg/api"
	"go.uber.org/zap"
)

// UploadHandler handles file upload routes.
// UploadHandler 处理文件上传路由。
type UploadHandler struct {
	logger *zap.Logger
}

// NewUploadHandler creates an UploadHandler.
// NewUploadHandler 创建 UploadHandler。
func NewUploadHandler(logger *zap.Logger) *UploadHandler {
	return &UploadHandler{logger: logger}
}

// allowedExts defines allowed file extensions.
// allowedExts 定义允许的文件扩展名。
var allowedExts = map[string]bool{
	".txt":  true,
	".md":   true,
	".csv":  true,
	".json": true,
	".yaml": true,
	".yml":  true,
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
}

// UploadResponse represents the response for file upload.
// UploadResponse 文件上传响应结构。
type UploadResponse struct {
	Filename string `json:"filename"`
	Type     string `json:"type"` // "text" or "image"
	Content  string `json:"content"`
}

// Upload handles POST /api/agent/upload - receives a file and returns its content.
// Upload 处理 POST /api/agent/upload - 接收文件并返回其内容。
func (h *UploadHandler) Upload(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		api.BadRequest(c, "no file provided")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExts[ext] {
		api.BadRequest(c, fmt.Sprintf("file type %q not allowed, allowed types: .txt, .md, .csv, .json, .yaml, .yml, .png, .jpg, .jpeg, .gif", ext))
		return
	}

	// Limit file size to 5MB
	// 限制文件大小为 5MB
	const maxSize = 5 << 20 // 5MB
	data, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		h.logger.Error("failed to read file", zap.Error(err))
		api.InternalErrorWithDetails(c, "read file failed", err)
		return
	}
	if len(data) > maxSize {
		api.BadRequest(c, "file too large (max 5MB)")
		return
	}

	// Check if it's an image file
	// 检查是否为图片文件
	isImage := ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif"

	if isImage {
		// Return base64 encoded image
		// 返回 base64 编码的图片
		mimeType := http.DetectContentType(data)
		
		// 双重校验：扩展名与MIME类型必须一致，防止伪装攻击
		// Double validation: extension must match MIME type to prevent spoofing attacks
		if !strings.HasPrefix(mimeType, "image/") {
			api.BadRequest(c, fmt.Sprintf("file extension %q indicates image, but detected MIME type is %q (not an image)", ext, mimeType))
			return
		}
		
		base64Content := base64.StdEncoding.EncodeToString(data)
		api.Success(c, UploadResponse{
			Filename: header.Filename,
			Type:     "image",
			Content:  fmt.Sprintf("data:%s;base64,%s", mimeType, base64Content),
		})
		return
	}

	// Return text content
	// 返回文本内容
	api.Success(c, UploadResponse{
		Filename: header.Filename,
		Type:     "text",
		Content:  string(data),
	})
}
