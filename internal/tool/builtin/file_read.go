package builtin

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
	"github.com/nguyenthenguyen/docx"
	"github.com/xuri/excelize/v2"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileReadTool{})
}

type FileReadTool struct{}

func (t *FileReadTool) Name() string { return "read_file" }

func (t *FileReadTool) Description() string {
	return "Read the contents of a file. Supports plain text, PDF, DOCX, and Excel (XLSX). " +
		"For text files, supports line ranges. For Excel, supports sheet selection and pagination."
}

func (t *FileReadTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"path": {
				Type:        "string",
				Description: "Path to the file.",
			},
			"start_line": {
				Type:        "integer",
				Description: "For text files: 1-based start line (inclusive). For Excel: start row (inclusive).",
			},
			"end_line": {
				Type:        "integer",
				Description: "For text files: 1-based end line (inclusive). For Excel: end row (inclusive).",
			},
			"sheet": {
				Type:        "string",
				Description: "For Excel files: name of the sheet to read.",
			},
		},
		Required: []string{"path"},
	}
}

func (t *FileReadTool) Execute(_ context.Context, args map[string]interface{}) *plugin.ToolResult {
	path, _ := args["path"].(string)
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".xlsx":
		return t.readXLSX(path, args)
	case ".pdf":
		return t.readPDF(path)
	case ".docx":
		return t.readDOCX(path)
	default:
		return t.readText(path, args)
	}
}

func (t *FileReadTool) readText(path string, args map[string]interface{}) *plugin.ToolResult {
	startLine := 1
	if val, ok := args["start_line"].(float64); ok {
		startLine = int(val)
	}
	endLine := 0
	if val, ok := args["end_line"].(float64); ok {
		endLine = int(val)
	}

	file, err := os.Open(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()

	var sb strings.Builder
	scanner := bufio.NewScanner(file)
	currentLine := 0
	for scanner.Scan() {
		currentLine++
		if currentLine < startLine {
			continue
		}
		if endLine > 0 && currentLine > endLine {
			break
		}
		sb.WriteString(fmt.Sprintf("%6d | %s\n", currentLine, scanner.Text()))
		if sb.Len() > 50000 { // Safety cap for text reading
			sb.WriteString("\n... [truncated due to length]")
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return plugin.ErrorResult(fmt.Sprintf("error reading file: %v", err))
	}

	return plugin.NewToolResult(sb.String())
}

func (t *FileReadTool) readXLSX(path string, args map[string]interface{}) *plugin.ToolResult {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open Excel file: %v", err))
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	if val, ok := args["sheet"].(string); ok && val != "" {
		sheet = val
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to read sheet %q: %v", sheet, err))
	}

	startRow := 1
	if val, ok := args["start_line"].(float64); ok {
		startRow = int(val)
	}
	endRow := startRow + 50 // Default limit 50 rows
	if val, ok := args["end_line"].(float64); ok {
		endRow = int(val)
	}

	if startRow > len(rows) {
		return plugin.NewToolResult(fmt.Sprintf("Start row %d is beyond total rows (%d).", startRow, len(rows)))
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Reading sheet %q (Rows %d to %d of %d):\n\n", sheet, startRow, endRow, len(rows))

	for i := startRow - 1; i < endRow && i < len(rows); i++ {
		row := rows[i]
		sb.WriteString("| ")
		for _, cell := range row {
			sb.WriteString(cell)
			sb.WriteString(" | ")
		}
		sb.WriteString("\n")
	}

	if endRow < len(rows) {
		sb.WriteString("\n... [more rows available, use end_line to read further]")
	}

	return plugin.NewToolResult(sb.String())
}

func (t *FileReadTool) readPDF(path string) *plugin.ToolResult {
	f, r, err := pdf.Open(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open PDF file: %v", err))
	}
	_ = f // Unused but returned by Open

	var sb strings.Builder
	totalPage := r.NumPage()

	for i := 1; i <= totalPage; i++ {
		p := r.Page(i)
		if p.V.IsNull() {
			continue
		}
		text, _ := p.GetPlainText(nil)
		sb.WriteString(fmt.Sprintf("--- Page %d ---\n", i))
		sb.WriteString(text)
		sb.WriteString("\n\n")
		if sb.Len() > 100000 {
			sb.WriteString("... [truncated due to length]")
			break
		}
	}

	content := sb.String()
	if strings.TrimSpace(content) == "" {
		return plugin.NewToolResult("PDF seems to be empty or contains only scanned images. Text extraction failed.")
	}

	return plugin.NewToolResult(content)
}

func (t *FileReadTool) readDOCX(path string) *plugin.ToolResult {
	r, err := docx.ReadDocxFile(path)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("failed to open DOCX file: %v", err))
	}
	defer r.Close()

	content := r.Editable().GetContent()
	// docx library returns the raw XML or text depending on the implementation.
	// Simple text extraction for now.
	if strings.TrimSpace(content) == "" {
		return plugin.NewToolResult("DOCX seems to be empty or extraction failed.")
	}

	return plugin.NewToolResult(content)
}
