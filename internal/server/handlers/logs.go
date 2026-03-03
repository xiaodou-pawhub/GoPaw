package handlers

import (
	"io"
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
// 优化 P3: 实现真正的反向 Tail 算法，高效读取大文件末尾
func (h *SystemHandler) ListLogs(c *gin.Context) {
	logPath := h.cfg.Log.File
	if logPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "系统日志文件路径未配置 / Log file path not configured"})
		return
	}

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

	stat, err := file.Stat()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取文件状态 / Failed to stat log file"})
		return
	}

	filesize := stat.Size()
	if filesize == 0 {
		c.JSON(http.StatusOK, gin.H{"logs": []LogEntry{}})
		return
	}

	// 核心：反向按块读取算法
	var result []LogEntry
	var cursor int64 = filesize
	var buffer = make([]byte, 4096) // 4KB 缓冲区
	var leftover string

	for cursor > 0 && len(result) < limit {
		// 确定本次读取的起始位置和长度
		readSize := int64(len(buffer))
		if cursor < readSize {
			readSize = cursor
		}
		cursor -= readSize

		_, err := file.Seek(cursor, io.SeekStart)
		if err != nil {
			break
		}

		n, err := file.Read(buffer[:readSize])
		if err != nil && err != io.EOF {
			break
		}

		// 处理读取到的内容，拼接残余部分
		currBatch := string(buffer[:n]) + leftover
		lines := strings.Split(currBatch, "\n")

		// 第一个元素可能是被截断的行，存入 leftover
		if cursor > 0 {
			leftover = lines[0]
			lines = lines[1:]
		} else {
			leftover = ""
		}

		// 从后向前遍历当前块的行
		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]
			// 修复 P1-1: 保留空行，只 trim 右侧空白
			line = strings.TrimRight(line, "\r")
			
			// 修复 P1-3: 文件末尾换行产生的额外空行
			// 如果是第一个块（cursor+readSize==filesize）且是最后一行且为空，跳过
			isFileEnd := (cursor + readSize == filesize)
			if line == "" && i == len(lines)-1 && isFileEnd {
				continue
			}
			
			// 块中间的空行是真实的，保留

			// 脱敏逻辑
			if isSensitive(line) {
				result = append(result, LogEntry{Raw: "[SENSITIVE DATA MASKED]"})
			} else {
				result = append(result, LogEntry{Raw: line})
			}

			if len(result) >= limit {
				break
			}
		}
	}

	// 处理最后可能剩下的第一行
	if leftover != "" && len(result) < limit {
		line := strings.TrimRight(leftover, "\r")
		if line != "" {
			if isSensitive(line) {
				result = append(result, LogEntry{Raw: "[SENSITIVE DATA MASKED]"})
			} else {
				result = append(result, LogEntry{Raw: line})
			}
		}
	}

	// 修复 P1-4: 反转结果数组，保持"旧→新"顺序（与原实现兼容）
	// 原实现是顺序扫描，返回 [旧...新]
	// 本实现是从后向前读，需要反转为 [旧...新]
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	c.JSON(http.StatusOK, gin.H{"logs": result})
}

// 辅助脱敏函数
func isSensitive(line string) bool {
	lower := strings.ToLower(line)
	keys := []string{"api_key", "secret", "token", "password", "sk-", "bearer"}
	for _, key := range keys {
		if strings.Contains(lower, key) {
			return true
		}
	}
	return false
}
