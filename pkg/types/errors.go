// Package types defines shared data types used across GoPaw modules.
package types

import "fmt"

// ErrorCode represents a categorized error code.
type ErrorCode string

const (
	// ErrCodeInternal represents an unexpected internal error.
	ErrCodeInternal ErrorCode = "INTERNAL"
	// ErrCodeNotFound represents a resource-not-found error.
	ErrCodeNotFound ErrorCode = "NOT_FOUND"
	// ErrCodeInvalidInput represents a validation or bad-input error.
	ErrCodeInvalidInput ErrorCode = "INVALID_INPUT"
	// ErrCodeUnauthorized represents an authentication failure.
	ErrCodeUnauthorized ErrorCode = "UNAUTHORIZED"
	// ErrCodeTimeout represents an operation timeout.
	ErrCodeTimeout ErrorCode = "TIMEOUT"
	// ErrCodeLLMError represents an error returned by the LLM provider.
	ErrCodeLLMError ErrorCode = "LLM_ERROR"
	// ErrCodeToolError represents an error during tool execution.
	ErrCodeToolError ErrorCode = "TOOL_ERROR"
	// ErrCodeChannelError represents a channel-related error.
	ErrCodeChannelError ErrorCode = "CHANNEL_ERROR"
	// ErrCodeMaxSteps represents the ReAct agent reaching its step limit.
	ErrCodeMaxSteps ErrorCode = "MAX_STEPS"
)

// GoPawError is the standard error type used throughout the application.
type GoPawError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *GoPawError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func (e *GoPawError) Unwrap() error {
	return e.Cause
}

// NewError creates a new GoPawError without a cause.
func NewError(code ErrorCode, message string) *GoPawError {
	return &GoPawError{Code: code, Message: message}
}

// WrapError wraps an existing error into a GoPawError.
func WrapError(code ErrorCode, message string, cause error) *GoPawError {
	return &GoPawError{Code: code, Message: message, Cause: cause}
}

// IsCode returns true when the error (or any unwrapped error) has the given code.
func IsCode(err error, code ErrorCode) bool {
	if err == nil {
		return false
	}
	if ge, ok := err.(*GoPawError); ok {
		return ge.Code == code
	}
	return false
}
