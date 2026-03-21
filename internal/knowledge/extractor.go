package knowledge

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// Extractor 文本提取器接口
type Extractor interface {
	Extract(r io.Reader) (string, error)
}

// ExtractorRegistry 提取器注册表
type ExtractorRegistry struct {
	extractors map[string]Extractor
}

// NewExtractorRegistry 创建提取器注册表
func NewExtractorRegistry() *ExtractorRegistry {
	registry := &ExtractorRegistry{
		extractors: make(map[string]Extractor),
	}

	// 注册默认提取器
	registry.Register("txt", &TextExtractor{})
	registry.Register("md", &MarkdownExtractor{})
	registry.Register("markdown", &MarkdownExtractor{})
	registry.Register("pdf", &PDFExtractor{})

	return registry
}

// Register 注册提取器
func (r *ExtractorRegistry) Register(fileType string, extractor Extractor) {
	r.extractors[fileType] = extractor
}

// Get 获取提取器
func (r *ExtractorRegistry) Get(fileType string) (Extractor, bool) {
	extractor, ok := r.extractors[fileType]
	return extractor, ok
}

// Extract 提取文本
func (r *ExtractorRegistry) Extract(fileType string, reader io.Reader) (string, error) {
	extractor, ok := r.Get(fileType)
	if !ok {
		return "", fmt.Errorf("unsupported file type: %s", fileType)
	}
	return extractor.Extract(reader)
}

// TextExtractor 纯文本提取器
type TextExtractor struct{}

// Extract 提取纯文本
func (e *TextExtractor) Extract(r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// MarkdownExtractor Markdown 提取器
type MarkdownExtractor struct{}

// Extract 提取 Markdown 文本（保留结构）
func (e *MarkdownExtractor) Extract(r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	// 使用 goldmark 解析 Markdown
	md := goldmark.New()
	reader := text.NewReader(content)
	doc := md.Parser().Parse(reader)

	var result strings.Builder
	e.extractNode(doc, content, &result)

	return result.String(), nil
}

// extractNode 递归提取节点文本
func (e *MarkdownExtractor) extractNode(n ast.Node, source []byte, result *strings.Builder) {
	if n == nil {
		return
	}

	switch node := n.(type) {
	case *ast.Heading:
		// 标题添加标记
		result.WriteString("\n")
		result.WriteString(strings.Repeat("#", node.Level))
		result.WriteString(" ")
		result.WriteString(string(node.Text(source)))
		result.WriteString("\n")

	case *ast.Paragraph:
		result.WriteString(string(node.Text(source)))
		result.WriteString("\n")

	case *ast.List:
		result.WriteString("\n")
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			e.extractNode(child, source, result)
		}
		result.WriteString("\n")

	case *ast.ListItem:
		result.WriteString("- ")
		result.WriteString(string(node.Text(source)))
		result.WriteString("\n")

	case *ast.CodeBlock:
		result.WriteString("\n```\n")
		result.WriteString(string(node.Text(source)))
		result.WriteString("\n```\n")

	case *ast.FencedCodeBlock:
		result.WriteString("\n```")
		result.WriteString(string(node.Language(source)))
		result.WriteString("\n")
		result.WriteString(string(node.Text(source)))
		result.WriteString("\n```\n")

	default:
		// 递归处理子节点
		for child := n.FirstChild(); child != nil; child = child.NextSibling() {
			e.extractNode(child, source, result)
		}
	}

	// 处理下一个兄弟节点
	if n.NextSibling() != nil {
		e.extractNode(n.NextSibling(), source, result)
	}
}

// PDFExtractor PDF 提取器
type PDFExtractor struct{}

// Extract 提取 PDF 文本
func (e *PDFExtractor) Extract(r io.Reader) (string, error) {
	content, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	reader := bytes.NewReader(content)
	pdfReader, err := pdf.NewReader(reader, int64(len(content)))
	if err != nil {
		return "", fmt.Errorf("failed to read PDF: %w", err)
	}

	var result strings.Builder
	numPages := pdfReader.NumPage()
	textPages := 0

	for pageNum := 1; pageNum <= numPages; pageNum++ {
		page := pdfReader.Page(pageNum)
		if page.V.IsNull() {
			continue
		}

		text, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}

		// 检测页面是否有有效文本
		pageText := strings.TrimSpace(text)
		if len(pageText) > 0 {
			textPages++
			result.WriteString(fmt.Sprintf("\n--- Page %d ---\n", pageNum))
			result.WriteString(text)
			result.WriteString("\n")
		}
	}

	// 检测提取结果有效性
	extractedText := strings.TrimSpace(result.String())
	textLength := len(extractedText)

	// 如果文本太少，可能是扫描版 PDF
	if textLength < 100 {
		if numPages > 0 && textPages == 0 {
			return "", fmt.Errorf("PDF 无文本内容（可能是扫描版），请上传文本版本或使用 OCR 工具转换")
		}
		return "", fmt.Errorf("PDF 文本太少（仅 %d 字符），无法有效处理", textLength)
	}

	return extractedText, nil
}
