package handlers

import (
	"bufio"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// LogEntry 代表 Web UI 显示的单条日志
type LogEntry struct {
	Raw string `json:"raw"`
}

// ListLogs 处理 GET /api/system/logs
// 高效读取日志文件末尾 N 行
func ListLogs(c *gin.Context) {
	// 默认读取路径，后续可迁入 config
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法打开日志文件 / Failed to open log file"})
		return
	}
	defer file.Close()

	// 中文：使用 Scanner 流式读取最后 N 行，内存占用恒定
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取日志失败 / Failed to read log"})
		return
	}

	// 逆序排列并进行敏感信息掩码脱敏
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
