// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package focus

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseFile parses a FOCUS.md file and returns the tasks.
func ParseFile(path string) ([]Task, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty tasks if file doesn't exist
			return []Task{}, "", nil
		}
		return nil, "", err
	}

	return Parse(string(data))
}

// Parse parses focus content and returns tasks and notes.
func Parse(content string) ([]Task, string, error) {
	var tasks []Task
	var notes []string

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and headers
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Try to parse as task
		if task, ok := parseTaskLine(line); ok {
			tasks = append(tasks, task)
		} else {
			// Treat as note
			notes = append(notes, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, "", err
	}

	return tasks, strings.Join(notes, "\n"), nil
}

// parseTaskLine parses a single task line.
// Format: [symbol] title
// Examples:
//   [*] Completed task
//   [/] In progress task
//   [ ] Pending task
func parseTaskLine(line string) (Task, bool) {
	line = strings.TrimSpace(line)

	// Check for task pattern: [x] or [/] or [ ]
	if len(line) < 4 {
		return Task{}, false
	}

	if line[0] != '[' {
		return Task{}, false
	}

	// Find closing bracket
	closeIdx := strings.Index(line, "]")
	if closeIdx == -1 || closeIdx < 2 {
		return Task{}, false
	}

	// Extract status symbol
	symbol := line[:closeIdx+1]
	status := ParseStatus(symbol)

	// Extract title
	title := strings.TrimSpace(line[closeIdx+1:])
	if title == "" {
		return Task{}, false
	}

	return Task{
		Title:  title,
		Status: status,
	}, true
}

// FormatTasks formats tasks into markdown content.
func FormatTasks(tasks []Task, notes string) string {
	var sb strings.Builder

	sb.WriteString("# Current Focus\n\n")

	for _, task := range tasks {
		sb.WriteString(fmt.Sprintf("%s %s\n", task.Status.Symbol(), task.Title))
	}

	if notes != "" {
		sb.WriteString("\n")
		sb.WriteString(notes)
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetActiveTask returns the first in-progress task, or the first pending task if none in progress.
func GetActiveTask(tasks []Task) *Task {
	// First, look for in-progress task
	for i := range tasks {
		if tasks[i].Status == InProgress {
			return &tasks[i]
		}
	}

	// Then, look for first pending task
	for i := range tasks {
		if tasks[i].Status == Pending {
			return &tasks[i]
		}
	}

	return nil
}
