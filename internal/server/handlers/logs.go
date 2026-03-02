package handlers

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// LogEntry represents a simplified log structure for the Web UI.
type LogEntry struct {
	Raw string `json:"raw"`
}

// ListLogs handles GET /api/system/logs.
// It reads the last N lines from the log file efficiently.
func ListLogs(c *gin.Context) {
	// TODO: Get log path from config
	logPath := "logs/gopaw.log"

	limit := 100
	if l, err := strconv.Atoi(c.DefaultQuery("limit", "100")); err == nil && l > 0 {
		limit = l
	}
	if limit > 500 {
		limit = 500
	}

	file, err := os.Open(logPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"logs": []LogEntry{{Raw: "无法打开日志文件: " + err.Error()}}})
		return
	}
	defer file.Close()

	// 中文：使用 Scanner 流式读取，仅在内存中保留最近 limit 行（类似环形缓冲区）
	// English: Use Scanner to stream lines, keeping only the last 'limit' lines in memory.
	var lastLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lastLines = append(lastLines, scanner.Text())
		if len(lastLines) > limit {
			lastLines = lastLines[1:]
		}
	}

	if err := scanner.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取日志失败: " + err.Error()})
		return
	}

	// 中文：逆序排列并进行敏感信息脱敏
	// English: Reverse order and desensitize sensitive info.
	sensitiveKeys := []string{"api_key", "api-key", "secret", "token", "password", "sk-", "bearer"}
	result := make([]LogEntry, 0, len(lastLines))
	for i := len(lastLines) - 1; i >= 0; i-- {
		raw := lastLines[i]
		lower := strings.ToLower(raw)
		masked := false
		for _, key := range sensitiveKeys {
			if strings.Contains(lower, key) {
				// 中文：检测到敏感词，进行掩码处理
				// English: Mask data if sensitive keys are detected.
				result = append(result, LogEntry{Raw: "[SENSITIVE DATA MASKED]"})
				masked = true
				break
			}
		}
		if !masked {
			result = append(result, LogEntry{Raw: raw})
		}
	}

	c.JSON(http.StatusOK, gin.H{"logs": result})
}



