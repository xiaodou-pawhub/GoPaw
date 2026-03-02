package memory

import (
	"os"
	"testing"
)

// TestStore_CRUD tests basic CRUD operations on the memory store.
func TestStore_CRUD(t *testing.T) {
	// Use an in-memory database for testing
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	sessionID := "test-session-001"
	userID := "test-user"
	channel := "console"

	// Test EnsureSession (idempotent)
	err = store.EnsureSession(sessionID, userID, channel)
	if err != nil {
		t.Fatalf("EnsureSession failed: %v", err)
	}

	// EnsureSession should be idempotent (call again should not error)
	err = store.EnsureSession(sessionID, userID, channel)
	if err != nil {
		t.Fatalf("EnsureSession (idempotent) failed: %v", err)
	}

	// Test AddMessage
	msgID := "msg-001"
	userMsg := StoredMessage{
		ID:        msgID,
		SessionID: sessionID,
		Role:      "user",
		Content:   "Hello, world!",
		CreatedAt: 1000,
	}
	err = store.AddMessage(userMsg)
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}

	// Test GetRecentMessages
	msgs, err := store.GetRecentMessages(sessionID, 10)
	if err != nil {
		t.Fatalf("GetRecentMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Content != "Hello, world!" {
		t.Fatalf("unexpected message content: %s", msgs[0].Content)
	}

	// Test AddMessage for assistant
	assistantMsg := StoredMessage{
		ID:        "msg-002",
		SessionID: sessionID,
		Role:      "assistant",
		Content:   "Hi there!",
		CreatedAt: 1001,
	}
	err = store.AddMessage(assistantMsg)
	if err != nil {
		t.Fatalf("AddMessage (assistant) failed: %v", err)
	}

	// Test GetRecentMessages order (should be newest first, then reversed to oldest-first)
	msgs, err = store.GetRecentMessages(sessionID, 10)
	if err != nil {
		t.Fatalf("GetRecentMessages failed: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	// msgs are reversed to oldest-first in GetRecentMessages
	// First message should be user (oldest), second should be assistant (newest)
	if msgs[0].Role != "user" {
		t.Fatalf("expected user message first (oldest), got %s", msgs[0].Role)
	}
	if msgs[1].Role != "assistant" {
		t.Fatalf("expected assistant message second (newest), got %s", msgs[1].Role)
	}

	// Test SearchMessages (FTS5)
	results, err := store.SearchMessages(sessionID, "hello", 10)
	if err != nil {
		t.Fatalf("SearchMessages failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 search result, got %d", len(results))
	}

	// Test DeleteSession
	err = store.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify deletion
	msgs, err = store.GetRecentMessages(sessionID, 10)
	if err != nil {
		t.Fatalf("GetRecentMessages after delete failed: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("expected 0 messages after delete, got %d", len(msgs))
	}
}

// TestStore_Summary tests the summary storage functionality.
func TestStore_Summary(t *testing.T) {
	store, err := NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	sessionID := "test-session-summary"
	err = store.EnsureSession(sessionID, "user", "console")
	if err != nil {
		t.Fatalf("EnsureSession failed: %v", err)
	}

	// Store a summary
	summaryID := "summary-001"
	summary := "This is a summary of previous conversation."
	err = store.StoreSummary(summaryID, sessionID, summary, 1000, 2000)
	if err != nil {
		t.Fatalf("StoreSummary failed: %v", err)
	}

	// Get the latest summary
	got, err := store.GetLatestSummary(sessionID)
	if err != nil {
		t.Fatalf("GetLatestSummary failed: %v", err)
	}
	if got != summary {
		t.Fatalf("expected summary %q, got %q", summary, got)
	}

	// Test GetLatestSummary when no summary exists
	emptySummary, err := store.GetLatestSummary("non-existent-session")
	if err != nil {
		t.Fatalf("GetLatestSummary for empty session failed: %v", err)
	}
	if emptySummary != "" {
		t.Fatalf("expected empty summary, got %q", emptySummary)
	}
}

// TestStore_FilePersistence tests that the store can persist to a file.
func TestStore_FilePersistence(t *testing.T) {
	tmpFile := t.TempDir() + "/test.db"
	defer os.Remove(tmpFile)

	// Create store and add data
	store, err := NewStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	sessionID := "persist-session"
	err = store.EnsureSession(sessionID, "user", "console")
	if err != nil {
		t.Fatalf("EnsureSession failed: %v", err)
	}
	err = store.AddMessage(StoredMessage{
		ID:        "msg-001",
		SessionID: sessionID,
		Role:      "user",
		Content:   "Test message",
		CreatedAt: 1000,
	})
	if err != nil {
		t.Fatalf("AddMessage failed: %v", err)
	}
	store.Close()

	// Reopen store and verify data
	store, err = NewStore(tmpFile)
	if err != nil {
		t.Fatalf("failed to reopen store: %v", err)
	}
	defer store.Close()

	msgs, err := store.GetRecentMessages(sessionID, 10)
	if err != nil {
		t.Fatalf("GetRecentMessages failed: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message after reopen, got %d", len(msgs))
	}
}
