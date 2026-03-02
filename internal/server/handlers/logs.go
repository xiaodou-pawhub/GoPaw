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
// Optional query param: limit (default 100, max 500).
func ListLogs(c *gin.Context) {
	// TODO: Get log path from config. Defaulting to logs/gopaw.log
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开日志文件: " + err.Error()})
		return
	}
	defer file.Close()

	// 中文：流式读取最后 N 行
	// English: Stream read last N lines
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

	// 中文：逆序排列并脱敏
	// English: Reverse and desensitize
	sensitiveKeys := []string{"api_key", "secret", "token", "password", "sk-", "bearer"}
	result := make([]LogEntry, 0, len(lastLines))
	for i := len(lastLines) - 1; i >= 0; i-- {
		raw := lastLines[i]
		lower := strings.ToLower(raw)
		masked := false
		for _, key := range sensitiveKeys {
			if strings.Contains(lower, key) {
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
