// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package auth

import (
	"testing"
	"time"
)

func TestHashPassword(t *testing.T) {
	service := NewService(nil)

	password := "testpassword123"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if hash == password {
		t.Error("Hash should not equal password")
	}
}

func TestCheckPassword(t *testing.T) {
	service := NewService(nil)

	password := "testpassword123"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Correct password
	if !service.CheckPassword(password, hash) {
		t.Error("CheckPassword should return true for correct password")
	}

	// Wrong password
	if service.CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestGenerateToken(t *testing.T) {
	config := &Config{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  time.Hour,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	}
	service := NewService(config)

	tokenPair, err := service.GenerateToken("user123", "testuser", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	if tokenPair.AccessToken == "" {
		t.Error("Access token should not be empty")
	}

	if tokenPair.RefreshToken == "" {
		t.Error("Refresh token should not be empty")
	}

	if tokenPair.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}

	if tokenPair.TokenType != "Bearer" {
		t.Errorf("Expected token type Bearer, got %s", tokenPair.TokenType)
	}
}

func TestValidateToken(t *testing.T) {
	config := &Config{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  time.Hour,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	}
	service := NewService(config)

	// Generate token
	tokenPair, err := service.GenerateToken("user123", "testuser", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate access token
	claims, err := service.ValidateToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("Expected user ID user123, got %s", claims.UserID)
	}

	if claims.Username != "testuser" {
		t.Errorf("Expected username testuser, got %s", claims.Username)
	}

	if claims.Email != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %s", claims.Email)
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	service := NewService(nil)

	// Invalid token
	_, err := service.ValidateToken("invalid-token")
	if err == nil {
		t.Error("ValidateToken should return error for invalid token")
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	config1 := &Config{
		JWTSecret:          "secret1",
		AccessTokenExpiry:  time.Hour,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	}
	service1 := NewService(config1)

	config2 := &Config{
		JWTSecret:          "secret2",
		AccessTokenExpiry:  time.Hour,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	}
	service2 := NewService(config2)

	// Generate token with service1
	tokenPair, err := service1.GenerateToken("user123", "testuser", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Validate with service2 (different secret)
	_, err = service2.ValidateToken(tokenPair.AccessToken)
	if err == nil {
		t.Error("ValidateToken should return error for token signed with different secret")
	}
}

func TestRefreshToken(t *testing.T) {
	config := &Config{
		JWTSecret:          "test-secret-key",
		AccessTokenExpiry:  time.Hour,
		RefreshTokenExpiry: 24 * time.Hour,
		Issuer:             "test",
	}
	service := NewService(config)

	// Generate token pair
	tokenPair, err := service.GenerateToken("user123", "testuser", "test@example.com", "Test User")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Refresh token
	newTokenPair, err := service.RefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if newTokenPair.AccessToken == "" {
		t.Error("New access token should not be empty")
	}

	if newTokenPair.RefreshToken == "" {
		t.Error("New refresh token should not be empty")
	}

	// New tokens should be different
	if newTokenPair.AccessToken == tokenPair.AccessToken {
		t.Error("New access token should be different from old one")
	}

	// Validate new access token
	claims, err := service.ValidateToken(newTokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate new token: %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("Expected user ID user123, got %s", claims.UserID)
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	service := NewService(nil)

	_, err := service.RefreshToken("invalid-refresh-token")
	if err == nil {
		t.Error("RefreshToken should return error for invalid token")
	}
}