package agent

import (
	"crypto/md5"
	"fmt"
)

// loopDetector catches two common infinite-loop patterns in the tool-call loop:
//  1. No-progress: the same (tool, args) pair is called maxSameCall times in a row.
//  2. Failure streak: maxFailures consecutive tool invocations all returned errors.
type loopDetector struct {
	// recentCalls is a rolling window of "toolName:argsHash" keys.
	recentCalls   []string
	failureStreak int
	maxSameCall   int
	maxFailures   int
}

func newLoopDetector(maxFailures, maxSameCall int) *loopDetector {
	return &loopDetector{
		maxFailures: maxFailures,
		maxSameCall: maxSameCall,
	}
}

// checkCall returns an error if the given call has been seen maxSameCall times consecutively.
// It must be called before executing the tool.
func (d *loopDetector) checkCall(toolName, argsJSON string) error {
	key := fmt.Sprintf("%s:%x", toolName, md5.Sum([]byte(argsJSON)))
	d.recentCalls = append(d.recentCalls, key)

	// Count how many of the last maxSameCall entries are identical to this key.
	n := d.maxSameCall
	if n > len(d.recentCalls) {
		n = len(d.recentCalls)
	}
	tail := d.recentCalls[len(d.recentCalls)-n:]
	count := 0
	for _, k := range tail {
		if k == key {
			count++
		}
	}
	if count >= d.maxSameCall {
		return fmt.Errorf("loop detected: tool %q called with identical arguments %d times consecutively", toolName, count)
	}
	return nil
}

// recordSuccess resets the failure streak counter.
func (d *loopDetector) recordSuccess() {
	d.failureStreak = 0
}

// recordFailure increments the failure streak counter.
func (d *loopDetector) recordFailure() {
	d.failureStreak++
}

// checkFailureStreak returns an error when too many consecutive tool calls have failed.
func (d *loopDetector) checkFailureStreak() error {
	if d.failureStreak >= d.maxFailures {
		return fmt.Errorf("agent aborted: %d consecutive tool failures", d.failureStreak)
	}
	return nil
}
