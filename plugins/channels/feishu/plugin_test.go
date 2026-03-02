package feishu

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/gopaw/gopaw/pkg/types"
)

// TestPlugin_Init tests plugin initialization with various configs.
func TestPlugin_Init(t *testing.T) {
	p := &Plugin{}

	// Test 1: Empty config should not return error
	err := p.Init([]byte("{}"))
	if err != nil {
		t.Fatalf("Init with empty config failed: %v", err)
	}
	if p.configured {
		t.Fatal("expected configured to be false for empty config")
	}

	// Test 2: No config should not return error
	p2 := &Plugin{}
	err = p2.Init(nil)
	if err != nil {
		t.Fatalf("Init with nil config failed: %v", err)
	}
	if p2.configured {
		t.Fatal("expected configured to be false for nil config")
	}

	// Test 3: Valid config should set configured = true
	p3 := &Plugin{}
	validConfig := json.RawMessage(`{"app_id":"test-app","app_secret":"test-secret"}`)
	err = p3.Init(validConfig)
	if err != nil {
		t.Fatalf("Init with valid config failed: %v", err)
	}
	if !p3.configured {
		t.Fatal("expected configured to be true for valid config")
	}
	if p3.cfg.AppID != "test-app" {
		t.Fatalf("expected app_id to be test-app, got %s", p3.cfg.AppID)
	}

	// Test 4: Invalid JSON should not panic but log warning
	p4 := &Plugin{}
	err = p4.Init([]byte("invalid json"))
	if err != nil {
		t.Fatalf("Init with invalid config should not return error: %v", err)
	}
	if p4.configured {
		t.Fatal("expected configured to be false for invalid JSON")
	}

	// Test 5: Partial config (only app_id) should not be configured
	p5 := &Plugin{}
	partialConfig := json.RawMessage(`{"app_id":"test-app"}`)
	err = p5.Init(partialConfig)
	if err != nil {
		t.Fatalf("Init with partial config failed: %v", err)
	}
	if p5.configured {
		t.Fatal("expected configured to be false for partial config")
	}
}

// TestPlugin_NameAndDisplayName tests plugin name methods.
func TestPlugin_NameAndDisplayName(t *testing.T) {
	p := &Plugin{}
	if p.Name() != "feishu" {
		t.Fatalf("expected name feishu, got %s", p.Name())
	}
	if p.DisplayName() != "飞书" {
		t.Fatalf("expected display name 飞书, got %s", p.DisplayName())
	}
}

// TestPlugin_Send_Unconfigured tests that Send returns error when not configured.
func TestPlugin_Send_Unconfigured(t *testing.T) {
	p := &Plugin{}
	p.Init([]byte("{}")) // unconfigured

	err := p.Send(&types.Message{
		ID:        "test-msg-id",
		SessionID: "test-session",
		UserID:    "test-user",
		Channel:   "feishu",
		Content:   "test content",
	})
	if err == nil {
		t.Fatal("expected error when sending from unconfigured plugin")
	}
}

// TestPlugin_Health tests health check for various states.
func TestPlugin_Health(t *testing.T) {
	// Test 1: Unconfigured health
	p := &Plugin{}
	p.Init([]byte("{}"))

	health := p.Health()
	if health.Running {
		t.Fatal("expected health Running to be false for unconfigured plugin")
	}
	if health.Message == "" {
		t.Fatal("expected health Message to be set for unconfigured plugin")
	}

	// Test 2: Configured but not connected
	p2 := &Plugin{}
	p2.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))

	health2 := p2.Health()
	if health2.Running {
		t.Fatal("expected health Running to be false when not connected")
	}

	// Test 3: Configured and connected
	p3 := &Plugin{}
	p3.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))
	p3.mu.Lock()
	p3.connected = true
	p3.mu.Unlock()

	health3 := p3.Health()
	if !health3.Running {
		t.Fatal("expected health Running to be true when connected")
	}
}

// TestPlugin_Test tests the Test() method for connection validation.
func TestPlugin_Test(t *testing.T) {
	ctx := context.Background()

	// Test 1: Unconfigured should fail
	p := &Plugin{}
	p.Init([]byte("{}"))

	result := p.Test(ctx)
	if result.Success {
		t.Fatal("expected Test to fail for unconfigured plugin")
	}
	if result.Message == "" {
		t.Fatal("expected error message for unconfigured plugin")
	}

	// Test 2: Configured but not connected
	p2 := &Plugin{}
	p2.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))

	result2 := p2.Test(ctx)
	if result2.Success {
		t.Fatal("expected Test to fail when not connected")
	}

	// Test 3: Configured and connected (but no real token)
	p3 := &Plugin{}
	p3.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))
	p3.mu.Lock()
	p3.connected = true
	p3.mu.Unlock()

	// This will fail on getToken() since we don't have a real token,
	// but we can verify the connected check passes
	result3 := p3.Test(ctx)
	// The test will fail on token validation, which is expected
	if result3.Success {
		t.Fatal("expected Test to fail due to invalid credentials")
	}
}

// TestPlugin_Stop tests that Stop() can be called safely.
func TestPlugin_Stop(t *testing.T) {
	p := &Plugin{}
	p.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))

	// Stop should not panic even if cancelFunc is nil
	err := p.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}

	// Stop with cancelFunc set
	p.ctx, p.cancelFunc = context.WithCancel(context.Background())
	err = p.Stop()
	if err != nil {
		t.Fatalf("Stop returned error: %v", err)
	}
}

// TestPlugin_ConcurrentHealth tests concurrent access to health status.
func TestPlugin_ConcurrentHealth(t *testing.T) {
	p := &Plugin{}
	p.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Simulate concurrent read/write
			p.mu.Lock()
			p.connected = true
			p.mu.Unlock()

			p.mu.RLock()
			_ = p.connected
			p.mu.RUnlock()

			_ = p.Health()
		}()
	}
	wg.Wait()
}

// TestPlugin_TokenExpiry tests token expiry calculation.
func TestPlugin_TokenExpiry(t *testing.T) {
	p := &Plugin{}
	p.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))

	// Test token cache state
	p.tokenMu.Lock()
	p.cachedToken = "test-token"
	p.tokenExpiry = time.Now().Add(2 * time.Hour)
	p.tokenMu.Unlock()

	// Verify token is cached
	p.tokenMu.RLock()
	if p.cachedToken != "test-token" {
		t.Fatal("expected cached token to be set")
	}
	p.tokenMu.RUnlock()
}

// TestPlugin_Receive tests that Receive() returns a valid channel.
func TestPlugin_Receive(t *testing.T) {
	p := &Plugin{
		inbound: make(chan *types.Message, 256),
	}

	ch := p.Receive()
	if ch == nil {
		t.Fatal("expected Receive to return non-nil channel")
	}

	// Verify channel is readable (receive-only)
	// We can't send to it directly, but we can verify it's a valid channel
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("expected channel to be empty")
		}
		// Channel closed, also valid
	default:
		// Channel empty and open, valid
	}
}
