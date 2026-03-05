package builtin

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	ignore "github.com/sabhiram/go-gitignore"
	"github.com/gopaw/gopaw/internal/tool"
	"github.com/gopaw/gopaw/pkg/plugin"
)

func init() {
	tool.Register(&FileSearchTool{})
}

type FileSearchTool struct{}

func (t *FileSearchTool) Name() string { return "file_search" }

func (t *FileSearchTool) Description() string {
	return "Search for files by name (glob) or content (grep) within the workspace. " +
		"Supports .gitignore filtering and concurrent searching."
}

func (t *FileSearchTool) Parameters() plugin.ToolParameters {
	return plugin.ToolParameters{
		Type: "object",
		Properties: map[string]plugin.ToolProperty{
			"pattern": {
				Type:        "string",
				Description: "The search pattern (regex for grep, glob for file names).",
			},
			"type": {
				Type:        "string",
				Description: "Search type: 'grep' (content) or 'glob' (file name).",
				Enum:        []string{"grep", "glob"},
			},
			"path": {
				Type:        "string",
				Description: "Root path to search from (default is current directory).",
			},
			"respect_ignore": {
				Type:        "boolean",
				Description: "If true, respects .gitignore patterns (default true).",
			},
		},
		Required: []string{"pattern", "type"},
	}
}

func (t *FileSearchTool) Execute(ctx context.Context, args map[string]interface{}) *plugin.ToolResult {
	pattern, _ := args["pattern"].(string)
	searchType, _ := args["type"].(string)
	rootPath := "."
	if val, ok := args["path"].(string); ok && val != "" {
		rootPath = val
	}
	respectIgnore := true
	if val, ok := args["respect_ignore"].(bool); ok {
		respectIgnore = val
	}

	var ignorer *ignore.GitIgnore
	if respectIgnore {
		// Try to find .gitignore in rootPath or current directory
		ignorer, _ = ignore.CompileIgnoreFile(filepath.Join(rootPath, ".gitignore"))
		if ignorer == nil {
			ignorer, _ = ignore.CompileIgnoreFile(".gitignore")
		}
	}

	if searchType == "glob" {
		return t.searchGlob(rootPath, pattern, ignorer)
	}

	return t.searchGrep(ctx, rootPath, pattern, ignorer)
}

func (t *FileSearchTool) searchGlob(root, pattern string, ignorer *ignore.GitIgnore) *plugin.ToolResult {
	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ignorer != nil && ignorer.MatchesPath(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		matched, _ := filepath.Match(pattern, d.Name())
		if matched {
			matches = append(matches, path)
		}
		return nil
	})

	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("glob search failed: %v", err))
	}

	if len(matches) == 0 {
		return plugin.NewToolResult("No matches found.")
	}

	return plugin.NewToolResult(strings.Join(matches, "\n"))
}

func (t *FileSearchTool) searchGrep(ctx context.Context, root, pattern string, ignorer *ignore.GitIgnore) *plugin.ToolResult {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return plugin.ErrorResult(fmt.Sprintf("invalid regex: %v", err))
	}

	var results []string
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	// File queue for concurrency
	files := make(chan string, 100)
	
	// Worker pool
	numWorkers := 8
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range files {
				select {
				case <-ctx.Done():
					return
				default:
					matches := grepInFile(path, re)
					if len(matches) > 0 {
						mu.Lock()
						results = append(results, matches...)
						mu.Unlock()
					}
				}
			}
		}()
	}

	// Walk and feed workers
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ignorer != nil && ignorer.MatchesPath(path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.IsDir() {
			select {
			case <-ctx.Done():
				return context.Canceled
			case files <- path:
			}
		}
		return nil
	})
	close(files)
	wg.Wait()

	if err != nil && err != context.Canceled {
		return plugin.ErrorResult(fmt.Sprintf("grep search failed: %v", err))
	}

	if len(results) == 0 {
		return plugin.NewToolResult("No matches found.")
	}

	// Limit results to 100 lines
	if len(results) > 100 {
		results = append(results[:100], "... [truncated, too many matches]")
	}

	return plugin.NewToolResult(strings.Join(results, "\n"))
}

func grepInFile(path string, re *regexp.Regexp) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var matches []string
	scanner := bufio.NewScanner(file)
	// Skip binary files (naive check)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		text := scanner.Text()
		if lineNum == 1 && strings.Contains(text, "\x00") {
			return nil
		}
		if re.MatchString(text) {
			matches = append(matches, fmt.Sprintf("%s:%d: %s", path, lineNum, text))
		}
		if len(matches) > 50 { // Max matches per file to avoid bloat
			break
		}
	}
	return matches
}
