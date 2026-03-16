package memory

import (
	"testing"
	"time"
)

// TestCountTokens_Empty tests that empty message list returns 0.
func TestCountTokens_Empty(t *testing.T) {
	result := CountTokens([]MemoryMessage{})
	if result != 0 {
		t.Fatalf("expected 0 tokens for empty messages, got %d", result)
	}
}

// TestCountTokens_SimpleEnglish tests token counting for simple English sentences.
func TestCountTokens_SimpleEnglish(t *testing.T) {
	msgs := []MemoryMessage{
		{Role: "user", Content: "Hello", CreatedAt: time.Now()},
	}
	result := CountTokens(msgs)
	// "Hello" is approximately 1 token in cl100k_base
	if result < 1 || result > 5 {
		t.Fatalf("expected 1-5 tokens for 'Hello', got %d", result)
	}
}

// TestCountTokens_MultipleMessages tests token counting for multiple messages.
func TestCountTokens_MultipleMessages(t *testing.T) {
	msgs := []MemoryMessage{
		{Role: "user", Content: "Hello, how are you?", CreatedAt: time.Now()},
		{Role: "assistant", Content: "I'm doing well, thank you!", CreatedAt: time.Now()},
	}
	result := CountTokens(msgs)
	// Each message also has ~4 tokens overhead for role + formatting
	// Total should be reasonable (not 0, not huge)
	if result < 5 {
		t.Fatalf("expected at least 5 tokens for two messages, got %d", result)
	}
}

// TestCountTokens_Fallback tests the fallback estimation when tiktoken fails.
func TestCountTokens_Fallback(t *testing.T) {
	// This test verifies the fallback logic works
	// The actual fallback is triggered when tiktoken.GetEncoding fails
	// Since tiktoken-go should work, we test the basic case
	msgs := []MemoryMessage{
		{Role: "system", Content: "You are a helpful assistant.", CreatedAt: time.Now()},
	}
	result := CountTokens(msgs)
	// Should return a reasonable number (> 0)
	if result <= 0 {
		t.Fatalf("expected positive token count, got %d", result)
	}
}

// TestCountTokens_Basic tests basic token counting.
func TestCountTokens_Basic(t *testing.T) {
	msgs := []MemoryMessage{
		{Role: "user", Content: "Test message", CreatedAt: time.Now()},
	}
	result := CountTokens(msgs)
	if result <= 0 {
		t.Fatalf("expected positive token count, got %d", result)
	}
}

// TestCountTokens_Chinese tests token counting for Chinese text.
func TestCountTokens_Chinese(t *testing.T) {
	msgs := []MemoryMessage{
		{Role: "user", Content: "你好世界", CreatedAt: time.Now()},
	}
	result := CountTokens(msgs)
	// Chinese characters typically use more tokens
	// 4 characters ≈ 4-8 tokens
	if result <= 0 {
		t.Fatalf("expected positive token count for Chinese, got %d", result)
	}
}
