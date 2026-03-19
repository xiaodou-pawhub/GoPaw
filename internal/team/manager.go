// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package team

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Manager handles team-related operations.
type Manager struct {
	db     *sql.DB
	logger *zap.Logger
}

// NewManager creates a new team manager.
func NewManager(db *sql.DB, logger *zap.Logger) (*Manager, error) {
	m := &Manager{
		db:     db,
		logger: logger.Named("team_manager"),
	}

	if err := m.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize team schema: %w", err)
	}

	if err := m.seedDefaultData(); err != nil {
		return nil, fmt.Errorf("failed to seed default data: %w", err)
	}

	return m, nil
}

// initSchema creates the database tables for team management.
func (m *Manager) initSchema() error {
	schema := `
-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name TEXT,
    avatar TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

-- Teams table
CREATE TABLE IF NOT EXISTS teams (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT,
    avatar TEXT,
    owner_id TEXT NOT NULL,
    settings TEXT DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    FOREIGN KEY (owner_id) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_teams_slug ON teams(slug);
CREATE INDEX IF NOT EXISTS idx_teams_owner ON teams(owner_id);
CREATE INDEX IF NOT EXISTS idx_teams_status ON teams(status);

-- Team members table
CREATE TABLE IF NOT EXISTS team_members (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP NOT NULL,
    invited_by TEXT,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(team_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_team_members_team ON team_members(team_id);
CREATE INDEX IF NOT EXISTS idx_team_members_user ON team_members(user_id);

-- Roles table
CREATE TABLE IF NOT EXISTS roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_system BOOLEAN DEFAULT 0,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

-- Permissions table
CREATE TABLE IF NOT EXISTS permissions (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT,
    UNIQUE(resource, action)
);

-- Role permissions table
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id TEXT NOT NULL,
    permission_id TEXT NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- User roles table (team-scoped roles)
CREATE TABLE IF NOT EXISTS user_roles (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    team_id TEXT NOT NULL,
    role_id TEXT NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    UNIQUE(user_id, team_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_user ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_team ON user_roles(team_id);

-- Team resources table (tracks which resources belong to which team)
CREATE TABLE IF NOT EXISTS team_resources (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT NOT NULL,
    visibility TEXT NOT NULL DEFAULT 'team',
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    UNIQUE(resource_type, resource_id)
);
CREATE INDEX IF NOT EXISTS idx_team_resources_team ON team_resources(team_id);
CREATE INDEX IF NOT EXISTS idx_team_resources_resource ON team_resources(resource_type, resource_id);

-- Invitations table
CREATE TABLE IF NOT EXISTS invitations (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL,
    email TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    token TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    invited_by TEXT NOT NULL,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    FOREIGN KEY (invited_by) REFERENCES users(id)
);
CREATE INDEX IF NOT EXISTS idx_invitations_team ON invitations(team_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email ON invitations(email);
CREATE INDEX IF NOT EXISTS idx_invitations_token ON invitations(token);
CREATE INDEX IF NOT EXISTS idx_invitations_status ON invitations(status);
`
	_, err := m.db.Exec(schema)
	return err
}

// seedDefaultData seeds default roles and permissions.
func (m *Manager) seedDefaultData() error {
	// Check if roles already exist
	var count int
	err := m.db.QueryRow("SELECT COUNT(*) FROM roles").Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil // Already seeded
	}

	now := time.Now()

	// Insert default roles
	roles := []Role{
		{ID: RoleOwner, Name: "Owner", Description: "Team owner with full access", IsSystem: true, CreatedAt: now, UpdatedAt: now},
		{ID: RoleAdmin, Name: "Admin", Description: "Team administrator", IsSystem: true, CreatedAt: now, UpdatedAt: now},
		{ID: RoleMember, Name: "Member", Description: "Regular team member", IsSystem: true, CreatedAt: now, UpdatedAt: now},
		{ID: RoleGuest, Name: "Guest", Description: "Limited access guest", IsSystem: true, CreatedAt: now, UpdatedAt: now},
	}

	for _, role := range roles {
		_, err := m.db.Exec(`
			INSERT INTO roles (id, name, description, is_system, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, role.ID, role.Name, role.Description, role.IsSystem, role.CreatedAt, role.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to insert role %s: %w", role.ID, err)
		}
	}

	// Insert default permissions
	for _, perm := range DefaultPermissions {
		_, err := m.db.Exec(`
			INSERT INTO permissions (id, name, resource, action, description)
			VALUES (?, ?, ?, ?, ?)
		`, perm.ID, perm.Name, perm.Resource, perm.Action, perm.Description)
		if err != nil {
			return fmt.Errorf("failed to insert permission %s: %w", perm.ID, err)
		}
	}

	// Insert default role-permission mappings
	for roleID, permIDs := range DefaultRolePermissions {
		for _, permID := range permIDs {
			_, err := m.db.Exec(`
				INSERT INTO role_permissions (role_id, permission_id)
				VALUES (?, ?)
			`, roleID, permID)
			if err != nil {
				return fmt.Errorf("failed to insert role permission %s -> %s: %w", roleID, permID, err)
			}
		}
	}

	m.logger.Info("Seeded default roles and permissions")
	return nil
}

// CreateUser creates a new user.
func (m *Manager) CreateUser(user *User) error {
	if user.ID == "" {
		user.ID = uuid.New().String()
	}
	now := time.Now()
	user.CreatedAt = now
	user.UpdatedAt = now
	if user.Status == "" {
		user.Status = "active"
	}

	_, err := m.db.Exec(`
		INSERT INTO users (id, username, email, password_hash, display_name, avatar, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, user.ID, user.Username, user.Email, user.PasswordHash, user.DisplayName, user.Avatar, user.Status, user.CreatedAt, user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUser retrieves a user by ID.
func (m *Manager) GetUser(id string) (*User, error) {
	user := &User{}
	var lastLogin sql.NullTime
	err := m.db.QueryRow(`
		SELECT id, username, email, display_name, avatar, status, last_login_at, created_at, updated_at
		FROM users WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.DisplayName, &user.Avatar, &user.Status, &lastLogin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username.
func (m *Manager) GetUserByUsername(username string) (*User, error) {
	user := &User{}
	var lastLogin sql.NullTime
	err := m.db.QueryRow(`
		SELECT id, username, email, password_hash, display_name, avatar, status, last_login_at, created_at, updated_at
		FROM users WHERE username = ?
	`, username).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName, &user.Avatar, &user.Status, &lastLogin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email.
func (m *Manager) GetUserByEmail(email string) (*User, error) {
	user := &User{}
	var lastLogin sql.NullTime
	err := m.db.QueryRow(`
		SELECT id, username, email, password_hash, display_name, avatar, status, last_login_at, created_at, updated_at
		FROM users WHERE email = ?
	`, email).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.DisplayName, &user.Avatar, &user.Status, &lastLogin, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if lastLogin.Valid {
		user.LastLoginAt = &lastLogin.Time
	}
	return user, nil
}

// UpdateUser updates a user.
func (m *Manager) UpdateUser(user *User) error {
	user.UpdatedAt = time.Now()
	_, err := m.db.Exec(`
		UPDATE users SET display_name = ?, avatar = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, user.DisplayName, user.Avatar, user.Status, user.UpdatedAt, user.ID)
	return err
}

// UpdateLastLogin updates the user's last login time.
func (m *Manager) UpdateLastLogin(userID string) error {
	_, err := m.db.Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`, time.Now(), userID)
	return err
}

// CreateTeam creates a new team.
func (m *Manager) CreateTeam(team *Team) error {
	if team.ID == "" {
		team.ID = uuid.New().String()
	}
	if team.Slug == "" {
		team.Slug = m.generateSlug(team.Name)
	}
	now := time.Now()
	team.CreatedAt = now
	team.UpdatedAt = now
	if team.Status == "" {
		team.Status = "active"
	}

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert team
	_, err = tx.Exec(`
		INSERT INTO teams (id, name, slug, description, avatar, owner_id, settings, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, team.ID, team.Name, team.Slug, team.Description, team.Avatar, team.OwnerID, team.Settings, team.Status, team.CreatedAt, team.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	// Add owner as team member
	memberID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO team_members (id, team_id, user_id, role, joined_at)
		VALUES (?, ?, ?, ?, ?)
	`, memberID, team.ID, team.OwnerID, RoleOwner, now)
	if err != nil {
		return fmt.Errorf("failed to add owner as team member: %w", err)
	}

	// Assign owner role
	roleID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_roles (id, user_id, team_id, role_id)
		VALUES (?, ?, ?, ?)
	`, roleID, team.OwnerID, team.ID, RoleOwner)
	if err != nil {
		return fmt.Errorf("failed to assign owner role: %w", err)
	}

	return tx.Commit()
}

// GetTeam retrieves a team by ID.
func (m *Manager) GetTeam(id string) (*Team, error) {
	team := &Team{}
	err := m.db.QueryRow(`
		SELECT id, name, slug, description, avatar, owner_id, settings, status, created_at, updated_at
		FROM teams WHERE id = ?
	`, id).Scan(&team.ID, &team.Name, &team.Slug, &team.Description, &team.Avatar, &team.OwnerID, &team.Settings, &team.Status, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return team, nil
}

// GetTeamBySlug retrieves a team by slug.
func (m *Manager) GetTeamBySlug(slug string) (*Team, error) {
	team := &Team{}
	err := m.db.QueryRow(`
		SELECT id, name, slug, description, avatar, owner_id, settings, status, created_at, updated_at
		FROM teams WHERE slug = ?
	`, slug).Scan(&team.ID, &team.Name, &team.Slug, &team.Description, &team.Avatar, &team.OwnerID, &team.Settings, &team.Status, &team.CreatedAt, &team.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return team, nil
}

// ListTeams lists all teams.
func (m *Manager) ListTeams() ([]*Team, error) {
	rows, err := m.db.Query(`
		SELECT id, name, slug, description, avatar, owner_id, settings, status, created_at, updated_at
		FROM teams ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*Team
	for rows.Next() {
		team := &Team{}
		err := rows.Scan(&team.ID, &team.Name, &team.Slug, &team.Description, &team.Avatar, &team.OwnerID, &team.Settings, &team.Status, &team.CreatedAt, &team.UpdatedAt)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}

// ListUserTeams lists all teams a user belongs to.
func (m *Manager) ListUserTeams(userID string) ([]*Team, error) {
	rows, err := m.db.Query(`
		SELECT t.id, t.name, t.slug, t.description, t.avatar, t.owner_id, t.settings, t.status, t.created_at, t.updated_at
		FROM teams t
		INNER JOIN team_members tm ON t.id = tm.team_id
		WHERE tm.user_id = ?
		ORDER BY t.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []*Team
	for rows.Next() {
		team := &Team{}
		err := rows.Scan(&team.ID, &team.Name, &team.Slug, &team.Description, &team.Avatar, &team.OwnerID, &team.Settings, &team.Status, &team.CreatedAt, &team.UpdatedAt)
		if err != nil {
			return nil, err
		}
		teams = append(teams, team)
	}
	return teams, nil
}

// UpdateTeam updates a team.
func (m *Manager) UpdateTeam(team *Team) error {
	team.UpdatedAt = time.Now()
	_, err := m.db.Exec(`
		UPDATE teams SET name = ?, slug = ?, description = ?, avatar = ?, settings = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, team.Name, team.Slug, team.Description, team.Avatar, team.Settings, team.Status, team.UpdatedAt, team.ID)
	return err
}

// DeleteTeam deletes a team.
func (m *Manager) DeleteTeam(id string) error {
	_, err := m.db.Exec(`DELETE FROM teams WHERE id = ?`, id)
	return err
}

// AddTeamMember adds a user to a team.
func (m *Manager) AddTeamMember(teamID, userID, role, invitedBy string) error {
	memberID := uuid.New().String()
	now := time.Now()

	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Add team member
	_, err = tx.Exec(`
		INSERT INTO team_members (id, team_id, user_id, role, joined_at, invited_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, memberID, teamID, userID, role, now, invitedBy)
	if err != nil {
		return fmt.Errorf("failed to add team member: %w", err)
	}

	// Assign role
	roleID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_roles (id, user_id, team_id, role_id)
		VALUES (?, ?, ?, ?)
	`, roleID, userID, teamID, role)
	if err != nil {
		return fmt.Errorf("failed to assign role: %w", err)
	}

	return tx.Commit()
}

// RemoveTeamMember removes a user from a team.
func (m *Manager) RemoveTeamMember(teamID, userID string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Remove user role
	_, err = tx.Exec(`DELETE FROM user_roles WHERE user_id = ? AND team_id = ?`, userID, teamID)
	if err != nil {
		return err
	}

	// Remove team member
	_, err = tx.Exec(`DELETE FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetTeamMembers lists all members of a team.
func (m *Manager) GetTeamMembers(teamID string) ([]*TeamMember, error) {
	rows, err := m.db.Query(`
		SELECT id, team_id, user_id, role, joined_at, invited_by
		FROM team_members WHERE team_id = ?
		ORDER BY joined_at ASC
	`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*TeamMember
	for rows.Next() {
		member := &TeamMember{}
		var invitedBy sql.NullString
		err := rows.Scan(&member.ID, &member.TeamID, &member.UserID, &member.Role, &member.JoinedAt, &invitedBy)
		if err != nil {
			return nil, err
		}
		if invitedBy.Valid {
			member.InvitedBy = invitedBy.String
		}
		members = append(members, member)
	}
	return members, nil
}

// UpdateMemberRole updates a team member's role.
func (m *Manager) UpdateMemberRole(teamID, userID, newRole string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Update team member role
	_, err = tx.Exec(`UPDATE team_members SET role = ? WHERE team_id = ? AND user_id = ?`, newRole, teamID, userID)
	if err != nil {
		return err
	}

	// Update user role
	_, err = tx.Exec(`UPDATE user_roles SET role_id = ? WHERE team_id = ? AND user_id = ?`, newRole, teamID, userID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// HasPermission checks if a user has a specific permission in a team.
func (m *Manager) HasPermission(userID, teamID, resource, action string) (bool, error) {
	var count int
	err := m.db.QueryRow(`
		SELECT COUNT(*)
		FROM user_roles ur
		INNER JOIN role_permissions rp ON ur.role_id = rp.role_id
		INNER JOIN permissions p ON rp.permission_id = p.id
		WHERE ur.user_id = ? AND ur.team_id = ? AND p.resource = ? AND p.action = ?
	`, userID, teamID, resource, action).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetUserPermissions gets all permissions for a user in a team.
func (m *Manager) GetUserPermissions(userID, teamID string) ([]Permission, error) {
	rows, err := m.db.Query(`
		SELECT p.id, p.name, p.resource, p.action, p.description
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = ? AND ur.team_id = ?
	`, userID, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []Permission
	for rows.Next() {
		var p Permission
		var desc sql.NullString
		err := rows.Scan(&p.ID, &p.Name, &p.Resource, &p.Action, &desc)
		if err != nil {
			return nil, err
		}
		if desc.Valid {
			p.Description = desc.String
		}
		permissions = append(permissions, p)
	}
	return permissions, nil
}

// CreateInvitation creates a team invitation.
func (m *Manager) CreateInvitation(teamID, email, role, invitedBy string, expiresAt *time.Time) (*Invitation, error) {
	inv := &Invitation{
		ID:        uuid.New().String(),
		TeamID:    teamID,
		Email:     email,
		Role:      role,
		Token:     m.generateToken(),
		Status:    "pending",
		InvitedBy: invitedBy,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	_, err := m.db.Exec(`
		INSERT INTO invitations (id, team_id, email, role, token, status, invited_by, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.ID, inv.TeamID, inv.Email, inv.Role, inv.Token, inv.Status, inv.InvitedBy, inv.ExpiresAt, inv.CreatedAt)
	if err != nil {
		return nil, err
	}

	return inv, nil
}

// GetInvitationByToken retrieves an invitation by token.
func (m *Manager) GetInvitationByToken(token string) (*Invitation, error) {
	inv := &Invitation{}
	var expiresAt sql.NullTime
	err := m.db.QueryRow(`
		SELECT id, team_id, email, role, token, status, invited_by, expires_at, created_at
		FROM invitations WHERE token = ?
	`, token).Scan(&inv.ID, &inv.TeamID, &inv.Email, &inv.Role, &inv.Token, &inv.Status, &inv.InvitedBy, &expiresAt, &inv.CreatedAt)
	if err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		inv.ExpiresAt = &expiresAt.Time
	}
	return inv, nil
}

// AcceptInvitation accepts an invitation.
func (m *Manager) AcceptInvitation(token, userID string) error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get invitation
	var inv Invitation
	var expiresAt sql.NullTime
	err = tx.QueryRow(`
		SELECT id, team_id, email, role, status, expires_at
		FROM invitations WHERE token = ?
	`, token).Scan(&inv.ID, &inv.TeamID, &inv.Email, &inv.Role, &inv.Status, &expiresAt)
	if err != nil {
		return err
	}

	// Check if invitation is valid
	if inv.Status != "pending" {
		return fmt.Errorf("invitation is not pending")
	}
	if expiresAt.Valid && expiresAt.Time.Before(time.Now()) {
		return fmt.Errorf("invitation has expired")
	}

	// Add user to team
	memberID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO team_members (id, team_id, user_id, role, joined_at, invited_by)
		VALUES (?, ?, ?, ?, ?, ?)
	`, memberID, inv.TeamID, userID, inv.Role, time.Now(), inv.InvitedBy)
	if err != nil {
		return err
	}

	// Assign role
	roleID := uuid.New().String()
	_, err = tx.Exec(`
		INSERT INTO user_roles (id, user_id, team_id, role_id)
		VALUES (?, ?, ?, ?)
	`, roleID, userID, inv.TeamID, inv.Role)
	if err != nil {
		return err
	}

	// Update invitation status
	_, err = tx.Exec(`UPDATE invitations SET status = 'accepted' WHERE id = ?`, inv.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// generateSlug generates a URL-friendly slug from a name.
func (m *Manager) generateSlug(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	// Remove non-alphanumeric characters
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// generateToken generates a random token.
func (m *Manager) generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}