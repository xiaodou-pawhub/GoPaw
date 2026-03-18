package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gopaw/gopaw/internal/knowledge"
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

		// 搜索
		kb.POST("/:id/search", h.Search)
	}
}

// CreateKnowledgeBase 创建知识库
func (h *KnowledgeHandler) CreateKnowledgeBase(c *gin.Context) {
	var req knowledge.CreateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kb, err := h.service.CreateKnowledgeBase(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, kb)
}

// ListKnowledgeBases 列出知识库
func (h *KnowledgeHandler) ListKnowledgeBases(c *gin.Context) {
	bases, err := h.service.ListKnowledgeBases(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, bases)
}

// GetKnowledgeBase 获取知识库
func (h *KnowledgeHandler) GetKnowledgeBase(c *gin.Context) {
	id := c.Param("id")

	kb, err := h.service.GetKnowledgeBase(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "knowledge base not found"})
		return
	}

	c.JSON(http.StatusOK, kb)
}

// UpdateKnowledgeBase 更新知识库
func (h *KnowledgeHandler) UpdateKnowledgeBase(c *gin.Context) {
	id := c.Param("id")

	var req knowledge.UpdateKnowledgeBaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateKnowledgeBase(c.Request.Context(), id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DeleteKnowledgeBase 删除知识库
func (h *KnowledgeHandler) DeleteKnowledgeBase(c *gin.Context) {
	id := c.Param("id")

	if err := h.service.DeleteKnowledgeBase(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// GetKnowledgeBaseStats 获取知识库统计
func (h *KnowledgeHandler) GetKnowledgeBaseStats(c *gin.Context) {
	id := c.Param("id")

	stats, err := h.service.GetStats(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// UploadDocument 上传文档
func (h *KnowledgeHandler) UploadDocument(c *gin.Context) {
	kbID := c.Param("id")

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	// 读取文件内容
	content := make([]byte, header.Size)
	_, err = file.Read(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	// 获取文件类型
	fileType := c.PostForm("file_type")
	if fileType == "" {
		// 从文件名推断
		fileType = getFileTypeFromName(header.Filename)
	}

	doc, err := h.service.UploadDocument(c.Request.Context(), kbID, header.Filename, fileType, content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, doc)
}

// ListDocuments 列出文档
func (h *KnowledgeHandler) ListDocuments(c *gin.Context) {
	kbID := c.Param("id")

	docs, err := h.service.ListDocuments(c.Request.Context(), kbID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, docs)
}

// DeleteDocument 删除文档
func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
	docID := c.Param("docId")

	if err := h.service.DeleteDocument(c.Request.Context(), docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// RetryDocument 重试处理文档
func (h *KnowledgeHandler) RetryDocument(c *gin.Context) {
	docID := c.Param("docId")

	if err := h.service.RetryDocument(c.Request.Context(), docID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "retrying"})
}

// Search 搜索知识库
func (h *KnowledgeHandler) Search(c *gin.Context) {
	kbID := c.Param("id")

	var req knowledge.SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.service.Search(c.Request.Context(), kbID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
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
