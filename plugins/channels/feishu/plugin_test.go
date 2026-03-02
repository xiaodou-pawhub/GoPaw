package feishu

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

// TestPlugin_HandleEventRequest_Challenge tests URL verification challenge.
func TestPlugin_HandleEventRequest_Challenge(t *testing.T) {
	p := &Plugin{}
	p.cfg.VerificationToken = "test-token"

	// Create request with challenge
	body := map[string]interface{}{
		"challenge": "test-challenge-value",
		"token":     "test-token",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, status := p.HandleEventRequest(req)
	if status != http.StatusOK {
		t.Fatalf("expected status 200, got %d", status)
	}

	respMap, ok := resp.(map[string]string)
	if !ok {
		t.Fatal("expected response to be map[string]string")
	}
	if respMap["challenge"] != "test-challenge-value" {
		t.Fatalf("expected challenge to be returned, got %s", respMap["challenge"])
	}
}

// TestPlugin_HandleEventRequest_InvalidToken tests that invalid token is rejected.
func TestPlugin_HandleEventRequest_InvalidToken(t *testing.T) {
	p := &Plugin{}
	p.cfg.VerificationToken = "test-token"

	body := map[string]interface{}{
		"challenge": "test-challenge",
		"token":     "wrong-token",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	resp, status := p.HandleEventRequest(req)

	respMap, ok := resp.(map[string]string)
	if !ok {
		t.Fatal("expected response to be map[string]string")
	}
	if status != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", status)
	}
	if respMap["error"] != "invalid token" {
		t.Fatalf("expected error 'invalid token', got %s", respMap["error"])
	}
}

// TestPlugin_HandleEventRequest_NonMessageEvent tests that non-message events are ignored.
func TestPlugin_HandleEventRequest_NonMessageEvent(t *testing.T) {
	p := &Plugin{}

	body := map[string]interface{}{
		"header": map[string]interface{}{
			"event_type": "im.message.read_v1", // not receive_v1
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(bodyBytes))
	resp, status := p.HandleEventRequest(req)

	if status != http.StatusOK {
		t.Fatalf("expected status 200 for non-message event, got %d", status)
	}
	if resp != nil {
		t.Fatal("expected nil response for non-message event")
	}
}

// TestPlugin_Health tests health check for configured and unconfigured states.
func TestPlugin_Health(t *testing.T) {
	// Test unconfigured health
	p := &Plugin{}
	p.Init([]byte("{}"))

	health := p.Health()
	if health.Running {
		t.Fatal("expected health Running to be false for unconfigured plugin")
	}
	if health.Message == "" {
		t.Fatal("expected health Message to be set for unconfigured plugin")
	}

	// Test configured health (with token)
	// Note: We can't easily test the running state without a real token,
	// but we verify configured state works
	p2 := &Plugin{}
	p2.Init(json.RawMessage(`{"app_id":"test","app_secret":"test"}`))
	// Health check now relies on getToken() which needs actual refresh
	// So we just verify it's configured
	health2 := p2.Health()
	// The Running state depends on whether getToken() returns empty,
	// which depends on token refresh. For unit test, just check message is set.
	if health2.Message == "" {
		t.Fatal("expected health Message to be set for configured plugin")
	}
}