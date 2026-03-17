// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package focus

// Status represents the status of a focus task.
type Status int

const (
	// Pending means the task is waiting to be started.
	Pending Status = iota
	// InProgress means the task is currently being worked on.
	InProgress
	// Completed means the task is finished.
	Completed
)

// String returns the string representation of the status.
func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"
	case InProgress:
		return "in_progress"
	case Completed:
		return "completed"
	default:
		return "unknown"
	}
}

// Symbol returns the markdown symbol for the status.
func (s Status) Symbol() string {
	switch s {
	case Pending:
		return "[ ]"
	case InProgress:
		return "[/]"
	case Completed:
		return "[*]"
	default:
		return "[ ]"
	}
}

// ParseStatus parses a status string or symbol into a Status.
func ParseStatus(s string) Status {
	switch s {
	case "*", "[*]", "completed", "done":
		return Completed
	case "/", "[/]", "in_progress", "doing":
		return InProgress
	case "", " ", "[ ]", "pending", "todo":
		return Pending
	default:
		return Pending
	}
}

// Task represents a single focus task.
type Task struct {
	Title  string
	Status Status
}

// IsActive returns true if the task is in progress.
func (t *Task) IsActive() bool {
	return t.Status == InProgress
}

// IsCompleted returns true if the task is completed.
func (t *Task) IsCompleted() bool {
	return t.Status == Completed
}
