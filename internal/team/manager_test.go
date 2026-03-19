// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package team

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
	"go.uber.org/zap"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	return db
}

func TestNewManager(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}
}

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	user := &User{
		Username:    "testuser",
		Email:       "test@example.com",
		PasswordHash: "hashedpassword",
		DisplayName: "Test User",
	}

	err = manager.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == "" {
		t.Error("User ID should be generated")
	}

	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
}

func TestGetUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a user first
	user := &User{
		Username:    "testuser",
		Email:       "test@example.com",
		PasswordHash: "hashedpassword",
		DisplayName: "Test User",
	}
	err = manager.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Get the user
	retrieved, err := manager.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}

	if retrieved.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestGetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	user := &User{
		Username:    "testuser",
		Email:       "test@example.com",
		PasswordHash: "hashedpassword",
		DisplayName: "Test User",
	}
	err = manager.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	retrieved, err := manager.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if retrieved.ID != user.ID {
		t.Errorf("Expected user ID %s, got %s", user.ID, retrieved.ID)
	}
}

func TestCreateTeam(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create owner user first
	owner := &User{
		Username:    "owner",
		Email:       "owner@example.com",
		PasswordHash: "hashedpassword",
		DisplayName: "Owner",
	}
	err = manager.CreateUser(owner)
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}

	// Create team
	team := &Team{
		Name:        "Test Team",
		Slug:        "test-team",
		Description: "A test team",
		OwnerID:     owner.ID,
	}

	err = manager.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	if team.ID == "" {
		t.Error("Team ID should be generated")
	}

	// Verify owner is added as team member
	members, err := manager.GetTeamMembers(team.ID)
	if err != nil {
		t.Fatalf("Failed to get team members: %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}

	if members[0].Role != RoleOwner {
		t.Errorf("Expected role %s, got %s", RoleOwner, members[0].Role)
	}
}

func TestAddTeamMember(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create users
	owner := &User{
		Username:    "owner",
		Email:       "owner@example.com",
		PasswordHash: "hashedpassword",
	}
	err = manager.CreateUser(owner)
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}

	member := &User{
		Username:    "member",
		Email:       "member@example.com",
		PasswordHash: "hashedpassword",
	}
	err = manager.CreateUser(member)
	if err != nil {
		t.Fatalf("Failed to create member: %v", err)
	}

	// Create team
	team := &Team{
		Name:    "Test Team",
		OwnerID: owner.ID,
	}
	err = manager.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Add member
	err = manager.AddTeamMember(team.ID, member.ID, RoleMember, owner.ID)
	if err != nil {
		t.Fatalf("Failed to add team member: %v", err)
	}

	// Verify member was added
	members, err := manager.GetTeamMembers(team.ID)
	if err != nil {
		t.Fatalf("Failed to get team members: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestHasPermission(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create user
	user := &User{
		Username:    "testuser",
		Email:       "test@example.com",
		PasswordHash: "hashedpassword",
	}
	err = manager.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create team
	team := &Team{
		Name:    "Test Team",
		OwnerID: user.ID,
	}
	err = manager.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Check owner has all permissions
	tests := []struct {
		resource string
		action   string
		expected bool
	}{
		{"agent", "create", true},
		{"agent", "read", true},
		{"agent", "delete", true},
		{"workflow", "execute", true},
		{"team", "manage", true},
	}

	for _, tt := range tests {
		has, err := manager.HasPermission(user.ID, team.ID, tt.resource, tt.action)
		if err != nil {
			t.Fatalf("Failed to check permission: %v", err)
		}
		if has != tt.expected {
			t.Errorf("Expected permission %s:%s to be %v, got %v", tt.resource, tt.action, tt.expected, has)
		}
	}
}

func TestCreateInvitation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create user
	user := &User{
		Username:    "testuser",
		Email:       "test@example.com",
		PasswordHash: "hashedpassword",
	}
	err = manager.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Create team
	team := &Team{
		Name:    "Test Team",
		OwnerID: user.ID,
	}
	err = manager.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Create invitation
	expiresAt := time.Now().Add(24 * time.Hour)
	inv, err := manager.CreateInvitation(team.ID, "invitee@example.com", RoleMember, user.ID, &expiresAt)
	if err != nil {
		t.Fatalf("Failed to create invitation: %v", err)
	}

	if inv.ID == "" {
		t.Error("Invitation ID should be generated")
	}

	if inv.Token == "" {
		t.Error("Invitation token should be generated")
	}

	if inv.Status != "pending" {
		t.Errorf("Expected status pending, got %s", inv.Status)
	}
}

func TestAcceptInvitation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logger := zap.NewNop()
	manager, err := NewManager(db, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create users
	owner := &User{
		Username:    "owner",
		Email:       "owner@example.com",
		PasswordHash: "hashedpassword",
	}
	err = manager.CreateUser(owner)
	if err != nil {
		t.Fatalf("Failed to create owner: %v", err)
	}

	invitee := &User{
		Username:    "invitee",
		Email:       "invitee@example.com",
		PasswordHash: "hashedpassword",
	}
	err = manager.CreateUser(invitee)
	if err != nil {
		t.Fatalf("Failed to create invitee: %v", err)
	}

	// Create team
	team := &Team{
		Name:    "Test Team",
		OwnerID: owner.ID,
	}
	err = manager.CreateTeam(team)
	if err != nil {
		t.Fatalf("Failed to create team: %v", err)
	}

	// Create invitation
	expiresAt := time.Now().Add(24 * time.Hour)
	inv, err := manager.CreateInvitation(team.ID, invitee.Email, RoleMember, owner.ID, &expiresAt)
	if err != nil {
		t.Fatalf("Failed to create invitation: %v", err)
	}

	// Accept invitation
	err = manager.AcceptInvitation(inv.Token, invitee.ID)
	if err != nil {
		t.Fatalf("Failed to accept invitation: %v", err)
	}

	// Verify member was added
	members, err := manager.GetTeamMembers(team.ID)
	if err != nil {
		t.Fatalf("Failed to get team members: %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}

	// Verify invitation status
	retrieved, err := manager.GetInvitationByToken(inv.Token)
	if err != nil {
		t.Fatalf("Failed to get invitation: %v", err)
	}

	if retrieved.Status != "accepted" {
		t.Errorf("Expected status accepted, got %s", retrieved.Status)
	}
}