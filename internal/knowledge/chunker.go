package knowledge

import (
	"strings"
	"unicode/utf8"
)

// TextChunk 文本块（用于分块器）
type TextChunk struct {
	Content    string
	Index      int
	TokenCount int
	Metadata   Metadata
}

// Chunker 文本分块器接口
type Chunker interface {
	Chunk(text string, chunkSize, chunkOverlap int) []TextChunk
}

// ChunkStrategy 分块策略
type ChunkStrategy int

const (
	ChunkByFixedSize ChunkStrategy = iota   // 固定大小
	ChunkBySentence                           // 按句子
	ChunkByParagraph                          // 按段落
	ChunkByMarkdown                           // Markdown 标题感知
)

// NewChunker 创建分块器
func NewChunker(strategy ChunkStrategy) Chunker {
	switch strategy {
	case ChunkBySentence:
		return &SentenceChunker{}
	case ChunkByParagraph:
		return &ParagraphChunker{}
	case ChunkByMarkdown:
		return &MarkdownChunker{}
	default:
		return &FixedSizeChunker{}
	}
}

// FixedSizeChunker 固定大小分块器
type FixedSizeChunker struct{}

// Chunk 按固定大小分块
func (c *FixedSizeChunker) Chunk(text string, chunkSize, chunkOverlap int) []TextChunk {
	if chunkSize <= 0 {
		chunkSize = 500
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	if chunkOverlap >= chunkSize {
		chunkOverlap = chunkSize / 2
	}

	var chunks []TextChunk
	runes := []rune(text)
	length := len(runes)

	for start := 0; start < length; start += chunkSize - chunkOverlap {
		end := start + chunkSize
		if end > length {
			end = length
		}

		content := string(runes[start:end])
		chunks = append(chunks, TextChunk{
			Content:    content,
			Index:      len(chunks),
			TokenCount: estimateTokenCount(content),
			Metadata: Metadata{
				"start_char": start,
				"end_char":   end,
			},
		})

		if end == length {
			break
		}
	}

	return chunks
}

// SentenceChunker 按句子分块器
type SentenceChunker struct{}

// Chunk 按句子分块
func (c *SentenceChunker) Chunk(text string, chunkSize, chunkOverlap int) []TextChunk {
	if chunkSize <= 0 {
		chunkSize = 500
	}

	// 简单的句子分割（按句号、问号、感叹号）
	sentences := splitSentences(text)

	var chunks []TextChunk
	var currentChunk strings.Builder
	var currentSentences []string

	for _, sentence := range sentences {
		if currentChunk.Len()+len(sentence) > chunkSize && currentChunk.Len() > 0 {
			// 保存当前块
			content := currentChunk.String()
			chunks = append(chunks, TextChunk{
				Content:    content,
				Index:      len(chunks),
				TokenCount: estimateTokenCount(content),
				Metadata: Metadata{
					"sentences": len(currentSentences),
				},
			})

			// 处理重叠
			if chunkOverlap > 0 && len(currentSentences) > 0 {
				currentChunk.Reset()
				currentSentences = getOverlapSentences(currentSentences, chunkOverlap)
				for _, s := range currentSentences {
					currentChunk.WriteString(s)
				}
			} else {
				currentChunk.Reset()
				currentSentences = nil
			}
		}

		currentChunk.WriteString(sentence)
		currentSentences = append(currentSentences, sentence)
	}

	// 保存最后一个块
	if currentChunk.Len() > 0 {
		content := currentChunk.String()
		chunks = append(chunks, TextChunk{
			Content:    content,
			Index:      len(chunks),
			TokenCount: estimateTokenCount(content),
			Metadata: Metadata{
				"sentences": len(currentSentences),
			},
		})
	}

	return chunks
}

// ParagraphChunker 按段落分块器
type ParagraphChunker struct{}

// Chunk 按段落分块
func (c *ParagraphChunker) Chunk(text string, chunkSize, chunkOverlap int) []TextChunk {
	// 按空行分割段落
	paragraphs := strings.Split(text, "\n\n")

	var chunks []TextChunk
	var currentChunk strings.Builder
	var currentParagraphs []string

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		if currentChunk.Len()+len(paragraph) > chunkSize && currentChunk.Len() > 0 {
			// 保存当前块
			content := currentChunk.String()
			chunks = append(chunks, TextChunk{
				Content:    content,
				Index:      len(chunks),
				TokenCount: estimateTokenCount(content),
				Metadata: Metadata{
					"paragraphs": len(currentParagraphs),
				},
			})

			currentChunk.Reset()
			currentParagraphs = nil
		}

		currentChunk.WriteString(paragraph)
		currentChunk.WriteString("\n\n")
		currentParagraphs = append(currentParagraphs, paragraph)
	}

	// 保存最后一个块
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		chunks = append(chunks, TextChunk{
			Content:    content,
			Index:      len(chunks),
			TokenCount: estimateTokenCount(content),
			Metadata: Metadata{
				"paragraphs": len(currentParagraphs),
			},
		})
	}

	return chunks
}

// MarkdownChunker Markdown 感知分块器
type MarkdownChunker struct{}

// Chunk 按 Markdown 标题分块
func (c *MarkdownChunker) Chunk(text string, chunkSize, chunkOverlap int) []TextChunk {
	if chunkSize <= 0 {
		chunkSize = 500
	}

	lines := strings.Split(text, "\n")
	var chunks []TextChunk
	var currentChunk strings.Builder
	var currentLevel int
	var currentTitle string
	var startLine int

	for i, line := range lines {
		// 检测标题
		level := getHeadingLevel(line)

		if level > 0 {
			// 遇到新标题，保存当前块
			if currentChunk.Len() > 0 {
				content := strings.TrimSpace(currentChunk.String())
				if len(content) > 0 {
					chunks = append(chunks, TextChunk{
						Content:    content,
						Index:      len(chunks),
						TokenCount: estimateTokenCount(content),
						Metadata: Metadata{
							"heading_level": currentLevel,
							"heading_title": currentTitle,
							"start_line":    startLine,
							"end_line":      i - 1,
						},
					})
				}
			}

			// 开始新块
			currentChunk.Reset()
			currentChunk.WriteString(line)
			currentChunk.WriteString("\n")
			currentLevel = level
			currentTitle = strings.TrimSpace(strings.TrimLeft(line, "#"))
			startLine = i
		} else {
			currentChunk.WriteString(line)
			currentChunk.WriteString("\n")
		}
	}

	// 保存最后一个块
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		if len(content) > 0 {
			chunks = append(chunks, TextChunk{
				Content:    content,
				Index:      len(chunks),
				TokenCount: estimateTokenCount(content),
				Metadata: Metadata{
					"heading_level": currentLevel,
					"heading_title": currentTitle,
					"start_line":    startLine,
					"end_line":      len(lines) - 1,
				},
			})
		}
	}

	// 如果块太大，进一步分割
	return c.splitLargeChunks(chunks, chunkSize)
}

// splitLargeChunks 分割过大的块
func (c *MarkdownChunker) splitLargeChunks(chunks []TextChunk, chunkSize int) []TextChunk {
	var result []TextChunk

	for _, chunk := range chunks {
		if len(chunk.Content) <= chunkSize {
			result = append(result, chunk)
			continue
		}

		// 使用固定大小分块器进一步分割
		fixedChunker := &FixedSizeChunker{}
		subChunks := fixedChunker.Chunk(chunk.Content, chunkSize, 0)

		for _, subChunk := range subChunks {
			subChunk.Index = len(result)
			subChunk.Metadata = mergeMetadata(chunk.Metadata, subChunk.Metadata)
			result = append(result, subChunk)
		}
	}

	return result
}

// 辅助函数

// splitSentences 分割句子
func splitSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		// 检测句子结束
		if r == '.' || r == '?' || r == '!' || r == '。' || r == '？' || r == '！' {
			// 检查下一个字符是否是空格或结束
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' {
				sentence := strings.TrimSpace(current.String())
				if len(sentence) > 0 {
					sentences = append(sentences, sentence+" ")
				}
				current.Reset()
			}
		}
	}

	// 保存最后的内容
	if current.Len() > 0 {
		sentence := strings.TrimSpace(current.String())
		if len(sentence) > 0 {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// getOverlapSentences 获取重叠的句子
func getOverlapSentences(sentences []string, overlapSize int) []string {
	if len(sentences) == 0 {
		return nil
	}

	var overlap []string
	var totalLen int

	// 从后往前取句子，直到达到重叠大小
	for i := len(sentences) - 1; i >= 0; i-- {
		overlap = append([]string{sentences[i]}, overlap...)
		totalLen += len(sentences[i])
		if totalLen >= overlapSize {
			break
		}
	}

	return overlap
}

// getHeadingLevel 获取 Markdown 标题级别
func getHeadingLevel(line string) int {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "#") {
		return 0
	}

	level := 0
	for _, r := range line {
		if r == '#' {
			level++
		} else {
			break
		}
	}

	// 检查后面是否有空格
	if level < len(line) && line[level] == ' ' {
		return level
	}

	return 0
}

// estimateTokenCount 估算 Token 数量
func estimateTokenCount(text string) int {
	// 简单的估算：英文约 4 字符/token，中文约 1 字符/token
	// 这里使用保守估计
	runes := utf8.RuneCountInString(text)
	return runes / 3
}

// mergeMetadata 合并元数据
func mergeMetadata(base, override Metadata) Metadata {
	result := make(Metadata)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range override {
		result[k] = v
	}
	return result
}
