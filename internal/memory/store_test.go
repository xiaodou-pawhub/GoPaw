package memory

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestStore creates a test memory store.
func setupTestStore(t *testing.T) *Store {
	store, err := NewStore(":memory:")
	require.NoError(t, err)
	return store
}

// TestNewStore tests creating a new store.
func TestNewStore(t *testing.T) {
	store, err := NewStore(":memory:")

	require.NoError(t, err)
	assert.NotNil(t, store)
}

// TestStore_EnsureSession tests ensuring a session exists.
func TestStore_EnsureSession(t *testing.T) {
	store := setupTestStore(t)

	err := store.EnsureSession("session1", "user1", "test")
	require.NoError(t, err)

	// Verify session exists by listing
	sessions, err := store.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 1)
	assert.Equal(t, "user1", sessions[0].UserID)
	assert.Equal(t, "test", sessions[0].Channel)
}

// TestStore_AddMessage tests adding a message.
func TestStore_AddMessage(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")

	msg := StoredMessage{
		ID:        "msg1",
		SessionID: "session1",
		Role:      "user",
		Content:   "Hello",
		CreatedAt: 1000,
	}
	err := store.AddMessage(msg)
	require.NoError(t, err)

	// Verify message was added
	messages, err := store.GetRecentMessages("session1", 10)
	require.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "Hello", messages[0].Content)
}

// TestStore_GetRecentMessages tests retrieving messages.
func TestStore_GetRecentMessages(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")

	// Add multiple messages
	for i := 0; i < 5; i++ {
		_ = store.AddMessage(StoredMessage{
			ID:        fmt.Sprintf("msg%d", i),
			SessionID: "session1",
			Role:      "user",
			Content:   fmt.Sprintf("Message %d", i),
			CreatedAt: int64(1000 + i),
		})
	}

	// Get all messages
	messages, err := store.GetRecentMessages("session1", 10)
	require.NoError(t, err)
	assert.Len(t, messages, 5)

	// Get limited messages
	messages, err = store.GetRecentMessages("session1", 3)
	require.NoError(t, err)
	assert.Len(t, messages, 3)
}

// TestStore_GetRecentMessages_Order tests message order (DESC).
func TestStore_GetRecentMessages_Order(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")

	_ = store.AddMessage(StoredMessage{
		ID:        "msg1",
		SessionID: "session1",
		Role:      "user",
		Content:   "First",
		CreatedAt: 1000,
	})
	_ = store.AddMessage(StoredMessage{
		ID:        "msg2",
		SessionID: "session1",
		Role:      "assistant",
		Content:   "Second",
		CreatedAt: 2000,
	})

	messages, err := store.GetRecentMessages("session1", 10)
	require.NoError(t, err)
	assert.Len(t, messages, 2)
	// DESC order, so Second comes first
	assert.Equal(t, "Second", messages[0].Content)
	assert.Equal(t, "First", messages[1].Content)
}

// TestStore_DeleteSession tests deleting a session.
func TestStore_DeleteSession(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")
	_ = store.AddMessage(StoredMessage{
		ID:        "msg1",
		SessionID: "session1",
		Role:      "user",
		Content:   "Hello",
		CreatedAt: 1000,
	})

	err := store.DeleteSession("session1")
	require.NoError(t, err)

	// Verify session is deleted
	sessions, err := store.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 0)

	// Verify messages are deleted
	messages, err := store.GetRecentMessages("session1", 10)
	require.NoError(t, err)
	assert.Len(t, messages, 0)
}

// TestStore_MultipleSessions tests multiple sessions.
func TestStore_MultipleSessions(t *testing.T) {
	store := setupTestStore(t)

	_ = store.EnsureSession("session1", "user1", "test")
	_ = store.EnsureSession("session2", "user2", "test")

	_ = store.AddMessage(StoredMessage{
		ID:        "msg1",
		SessionID: "session1",
		Role:      "user",
		Content:   "Session 1",
		CreatedAt: 1000,
	})
	_ = store.AddMessage(StoredMessage{
		ID:        "msg2",
		SessionID: "session2",
		Role:      "user",
		Content:   "Session 2",
		CreatedAt: 2000,
	})

	messages1, _ := store.GetRecentMessages("session1", 10)
	messages2, _ := store.GetRecentMessages("session2", 10)

	assert.Len(t, messages1, 1)
	assert.Len(t, messages2, 1)
	assert.Equal(t, "Session 1", messages1[0].Content)
	assert.Equal(t, "Session 2", messages2[0].Content)
}

// TestStore_SearchMessages tests searching messages.
func TestStore_SearchMessages(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")

	_ = store.AddMessage(StoredMessage{
		ID:        "msg1",
		SessionID: "session1",
		Role:      "user",
		Content:   "I love programming in Go",
		CreatedAt: 1000,
	})
	_ = store.AddMessage(StoredMessage{
		ID:        "msg2",
		SessionID: "session1",
		Role:      "user",
		Content:   "Python is also nice",
		CreatedAt: 2000,
	})

	results, err := store.SearchMessages("session1", "Go programming", 10)
	require.NoError(t, err)
	assert.NotEmpty(t, results)

	// Check if the first message is found
	found := false
	for _, r := range results {
		if r.Content == "I love programming in Go" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

// TestStore_SearchMessages_NoResults tests search with no matches.
func TestStore_SearchMessages_NoResults(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")

	_ = store.AddMessage(StoredMessage{
		ID:        "msg1",
		SessionID: "session1",
		Role:      "user",
		Content:   "Hello world",
		CreatedAt: 1000,
	})

	results, err := store.SearchMessages("session1", "nonexistent xyz", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

// TestStore_ListSessions tests listing sessions.
func TestStore_ListSessions(t *testing.T) {
	store := setupTestStore(t)

	_ = store.EnsureSession("session1", "user1", "test")
	_ = store.EnsureSession("session2", "user2", "test")

	sessions, err := store.ListSessions()
	require.NoError(t, err)
	assert.Len(t, sessions, 2)
}

// TestStore_UpdateSessionName tests updating session name.
func TestStore_UpdateSessionName(t *testing.T) {
	store := setupTestStore(t)
	_ = store.EnsureSession("session1", "user1", "test")

	err := store.UpdateSessionName("session1", "My Session")
	require.NoError(t, err)

	sessions, _ := store.ListSessions()
	assert.Equal(t, "My Session", sessions[0].Name)
}

// BenchmarkStore_AddMessage benchmarks adding messages.
func BenchmarkStore_AddMessage(b *testing.B) {
	store, _ := NewStore(":memory:")
	_ = store.EnsureSession("session1", "user1", "test")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.AddMessage(StoredMessage{
			ID:        fmt.Sprintf("msg%d", i),
			SessionID: "session1",
			Role:      "user",
			Content:   fmt.Sprintf("Message %d", i),
			CreatedAt: int64(i),
		})
	}
}

// BenchmarkStore_GetRecentMessages benchmarks retrieving messages.
func BenchmarkStore_GetRecentMessages(b *testing.B) {
	store, _ := NewStore(":memory:")
	_ = store.EnsureSession("session1", "user1", "test")

	// Add 100 messages
	for i := 0; i < 100; i++ {
		_ = store.AddMessage(StoredMessage{
			ID:        fmt.Sprintf("msg%d", i),
			SessionID: "session1",
			Role:      "user",
			Content:   fmt.Sprintf("Message %d", i),
			CreatedAt: int64(i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.GetRecentMessages("session1", 50)
	}
}

// BenchmarkStore_SearchMessages benchmarks searching.
func BenchmarkStore_SearchMessages(b *testing.B) {
	store, _ := NewStore(":memory:")
	_ = store.EnsureSession("session1", "user1", "test")

	// Add messages
	for i := 0; i < 100; i++ {
		_ = store.AddMessage(StoredMessage{
			ID:        fmt.Sprintf("msg%d", i),
			SessionID: "session1",
			Role:      "user",
			Content:   fmt.Sprintf("This is message %d about programming", i),
			CreatedAt: int64(i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.SearchMessages("session1", "programming", 10)
	}
}
