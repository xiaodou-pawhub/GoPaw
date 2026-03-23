package handlers

import (
	"fmt"
	"io"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/knowledge"
	"github.com/gopaw/gopaw/pkg/api"
)

// KnowledgeHandler 知识库处理器
type KnowledgeHandler struct {
	service *knowledge.Service
}

// NewKnowledgeHandler 创建知识库处理器
func NewKnowledgeHandler(service *knowledge.Service) *KnowledgeHandler {
	return &KnowledgeHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *KnowledgeHandler) RegisterRoutes(router *gin.RouterGroup) {
	kb := router.Group("/knowledge/bases")
	{
		kb.POST("", h.CreateKnowledgeBase)
		kb.GET("", h.ListKnowledgeBases)
		kb.GET("/:id", h.GetKnowledgeBase)
		kb.PUT("/:id", h.UpdateKnowledgeBase)
		kb.DELETE("/:id", h.DeleteKnowledgeBase)
		kb.GET("/:id/stats", h.GetKnowledgeBaseStats)

		// 文档管理
		kb.POST("/:id/documents", h.UploadDocument)
		kb.GET("/:id/documents", h.ListDocuments)
		kb.DELETE("/:id/documents/:docId", h.DeleteDocument)
		kb.POST("/:id/documents/:docId/retry", h.RetryDocument)

		// 文档版本管理
		kb.GET("/:id/documents/:docId/versions", h.ListDocumentVersions)
		kb.POST("/:id/documents/:docId/versions", h.CreateDocumentVersion)
		kb.POST("/:id/documents/:docId/versions/:version/rollback", h.RollbackDocumentVersion)

		// 搜索
		kb.POST("/:id/search", h.Search)

		// 查询统计
		kb.GET("/:id/query-stats", h.GetQueryStats)
		kb.GET("/:id/query-stats/daily", h.GetDailyQueryStats)
	}
}

// CreateKnowledgeBase 创建知识库
func (h *KnowledgeHandler) CreateKnowledgeBase(c *gin.Context) {
	var req knowledge.CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	kb, err := h.service.CreateKnowledgeBase(c.Request.Context(), req)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to create knowledge base", err)
		return
	}

	api.Created(c, kb)
}

// ListKnowledgeBases 列出知识库
func (h *KnowledgeHandler) ListKnowledgeBases(c *gin.Context) {
	bases, err := h.service.ListKnowledgeBases(c.Request.Context())
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list knowledge bases", err)
		return
	}

	api.Success(c, bases)
}

// GetKnowledgeBase 获取知识库
func (h *KnowledgeHandler) GetKnowledgeBase(c *gin.Context) {
	id := c.Param("id")

	kb, err := h.service.GetKnowledgeBase(c.Request.Context(), id)
	if err != nil {
		api.NotFound(c, "knowledge base")
		return
	}

	api.Success(c, kb)
}

// UpdateKnowledgeBase 更新知识库
func (h *KnowledgeHandler) UpdateKnowledgeBase(c *gin.Context) {
	id := c.Param("id")

	var req knowledge.UpdateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	if err := h.service.UpdateKnowledgeBase(c.Request.Context(), id, req); err != nil {
		api.InternalErrorWithDetails(c, "failed to update knowledge base", err)
		return
	}

	api.SuccessWithMessage(c, "updated", nil)
}

// DeleteKnowledgeBase 删除知识库
func (h *KnowledgeHandler) DeleteKnowledgeBase(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteKnowledgeBase(c.Request.Context(), id); err != nil {
		api.InternalErrorWithDetails(c, "failed to delete knowledge base", err)
		return
	}

	api.SuccessWithMessage(c, "deleted", nil)
}

// GetKnowledgeBaseStats 获取知识库统计
func (h *KnowledgeHandler) GetKnowledgeBaseStats(c *gin.Context) {
	id := c.Param("id")

	stats, err := h.service.GetStats(c.Request.Context(), id)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to get stats", err)
		return
	}

	api.Success(c, stats)
}

// UploadDocument 上传文档
func (h *KnowledgeHandler) UploadDocument(c *gin.Context) {
	kbID := c.Param("id")

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		api.BadRequest(c, "no file uploaded")
		return
	}
	defer file.Close()

	// 读取文件内容
	content, err := io.ReadAll(file)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to read file", err)
		return
	}

	// 从文件名推断文件类型
	fileType := getFileTypeFromName(header.Filename)

	doc, err := h.service.UploadDocument(c.Request.Context(), kbID, header.Filename, fileType, content)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to upload document", err)
		return
	}

	api.Created(c, doc)
}

// ListDocuments 列出文档
func (h *KnowledgeHandler) ListDocuments(c *gin.Context) {
	kbID := c.Param("id")

	docs, err := h.service.ListDocuments(c.Request.Context(), kbID)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list documents", err)
		return
	}

	api.Success(c, docs)
}

// DeleteDocument 删除文档
func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
	docID := c.Param("docId")

	if err := h.service.DeleteDocument(c.Request.Context(), docID); err != nil {
		api.InternalErrorWithDetails(c, "failed to delete document", err)
		return
	}

	api.SuccessWithMessage(c, "deleted", nil)
}

// RetryDocument 重试处理文档
func (h *KnowledgeHandler) RetryDocument(c *gin.Context) {
	docID := c.Param("docId")

	if err := h.service.RetryDocument(c.Request.Context(), docID); err != nil {
		api.InternalErrorWithDetails(c, "failed to retry document", err)
		return
	}

	api.SuccessWithMessage(c, "retrying", nil)
}

// Search 搜索知识库
func (h *KnowledgeHandler) Search(c *gin.Context) {
	kbID := c.Param("id")

	var req knowledge.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.BadRequestWithError(c, "invalid request", err)
		return
	}

	resp, err := h.service.Search(c.Request.Context(), kbID, req)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to search", err)
		return
	}

	api.Success(c, resp)
}

// getFileTypeFromName 从文件名获取文件类型
func getFileTypeFromName(filename string) string {
	// 从文件名后缀推断类型
	if len(filename) < 4 {
		return "txt"
	}

	// 获取后缀
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			switch filename[i+1:] {
			case "pdf":
				return "pdf"
			case "md", "markdown":
				return "md"
			case "txt":
				return "txt"
			case "doc", "docx":
				return "docx"
			default:
				return "txt"
			}
		}
	}

	return "txt"
}

// ========== 版本管理 ==========

// ListDocumentVersions 列出文档版本
func (h *KnowledgeHandler) ListDocumentVersions(c *gin.Context) {
	docID := c.Param("docId")

	versions, err := h.service.ListDocumentVersions(c.Request.Context(), docID)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to list versions", err)
		return
	}

	api.Success(c, versions)
}

// CreateDocumentVersion 创建文档版本
func (h *KnowledgeHandler) CreateDocumentVersion(c *gin.Context) {
	docID := c.Param("docId")

	var req struct {
		ChangeNote string `json:"change_note"`
	}
	c.ShouldBindJSON(&req)

	version, err := h.service.CreateDocumentVersion(c.Request.Context(), docID, req.ChangeNote, "")
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to create version", err)
		return
	}

	api.Created(c, version)
}

// RollbackDocumentVersion 回滚文档版本
func (h *KnowledgeHandler) RollbackDocumentVersion(c *gin.Context) {
	docID := c.Param("docId")
	versionStr := c.Param("version")

	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		api.BadRequest(c, "invalid version number")
		return
	}

	doc, err := h.service.RollbackDocumentVersion(c.Request.Context(), docID, version)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to rollback version", err)
		return
	}

	api.SuccessWithMessage(c, "rolled back successfully", doc)
}

// ========== 统计功能 ==========

// GetQueryStats 获取查询统计
func (h *KnowledgeHandler) GetQueryStats(c *gin.Context) {
	kbID := c.Param("id")
	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := h.service.GetQueryStats(c.Request.Context(), kbID, days)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to get query stats", err)
		return
	}

	api.Success(c, stats)
}

// GetDailyQueryStats 获取每日查询统计
func (h *KnowledgeHandler) GetDailyQueryStats(c *gin.Context) {
	kbID := c.Param("id")
	days := 7
	if d := c.Query("days"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			days = parsed
		}
	}

	stats, err := h.service.GetDailyQueryStats(c.Request.Context(), kbID, days)
	if err != nil {
		api.InternalErrorWithDetails(c, "failed to get daily query stats", err)
		return
	}

	api.Success(c, stats)
}
