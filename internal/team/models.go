// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

package team

import (
	"time"
)

// User represents a user in the system.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // Never expose password hash
	DisplayName  string    `json:"display_name"`
	Avatar       string    `json:"avatar"`
	Status       string    `json:"status"` // active, inactive, suspended
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Team represents a team/workspace.
type Team struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"` // URL-friendly identifier
	Description string    `json:"description"`
	Avatar      string    `json:"avatar"`
	OwnerID     string    `json:"owner_id"`
	Settings    string    `json:"settings"` // JSON settings
	Status      string    `json:"status"`   // active, inactive
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// TeamMember represents a user's membership in a team.
type TeamMember struct {
	ID        string    `json:"id"`
	TeamID    string    `json:"team_id"`
	UserID    string    `json:"user_id"`
	Role      string    `json:"role"` // owner, admin, member, guest
	JoinedAt  time.Time `json:"joined_at"`
	InvitedBy string    `json:"invited_by,omitempty"`
}

// Role represents a role in the RBAC system.
type Role struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsSystem    bool      `json:"is_system"` // System roles cannot be deleted
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Permission represents a permission in the system.
type Permission struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Resource    string `json:"resource"` // agent, workflow, knowledge, etc.
	Action      string `json:"action"`   // create, read, update, delete, execute
	Description string `json:"description"`
}

// RolePermission associates a role with permissions.
type RolePermission struct {
	RoleID       string `json:"role_id"`
	PermissionID string `json:"permission_id"`
}

// UserRole associates a user with a role in a team context.
type UserRole struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	TeamID string `json:"team_id"`
	RoleID string `json:"role_id"`
}

// TeamResource represents a resource owned by a team.
type TeamResource struct {
	ID           string `json:"id"`
	TeamID       string `json:"team_id"`
	ResourceType string `json:"resource_type"` // agent, workflow, knowledge, etc.
	ResourceID   string `json:"resource_id"`
	Visibility   string `json:"visibility"` // public, team, private
}

// Invitation represents a team invitation.
type Invitation struct {
	ID        string     `json:"id"`
	TeamID    string     `json:"team_id"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Token     string     `json:"token"`
	Status    string     `json:"status"` // pending, accepted, declined, expired
	InvitedBy string     `json:"invited_by"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Predefined roles
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
	RoleGuest  = "guest"
)

// Predefined permissions
var DefaultPermissions = []Permission{
	// Agent permissions
	{ID: "agent:create", Name: "Create Agent", Resource: "agent", Action: "create", Description: "Create new agents"},
	{ID: "agent:read", Name: "View Agent", Resource: "agent", Action: "read", Description: "View agent details"},
	{ID: "agent:update", Name: "Update Agent", Resource: "agent", Action: "update", Description: "Modify agent configuration"},
	{ID: "agent:delete", Name: "Delete Agent", Resource: "agent", Action: "delete", Description: "Delete agents"},
	{ID: "agent:execute", Name: "Execute Agent", Resource: "agent", Action: "execute", Description: "Run agent tasks"},

	// Workflow permissions
	{ID: "workflow:create", Name: "Create Workflow", Resource: "workflow", Action: "create", Description: "Create new workflows"},
	{ID: "workflow:read", Name: "View Workflow", Resource: "workflow", Action: "read", Description: "View workflow details"},
	{ID: "workflow:update", Name: "Update Workflow", Resource: "workflow", Action: "update", Description: "Modify workflow configuration"},
	{ID: "workflow:delete", Name: "Delete Workflow", Resource: "workflow", Action: "delete", Description: "Delete workflows"},
	{ID: "workflow:execute", Name: "Execute Workflow", Resource: "workflow", Action: "execute", Description: "Run workflows"},

	// Knowledge permissions
	{ID: "knowledge:create", Name: "Create Knowledge Base", Resource: "knowledge", Action: "create", Description: "Create new knowledge bases"},
	{ID: "knowledge:read", Name: "View Knowledge Base", Resource: "knowledge", Action: "read", Description: "View knowledge base details"},
	{ID: "knowledge:update", Name: "Update Knowledge Base", Resource: "knowledge", Action: "update", Description: "Modify knowledge base"},
	{ID: "knowledge:delete", Name: "Delete Knowledge Base", Resource: "knowledge", Action: "delete", Description: "Delete knowledge bases"},

	// Team permissions
	{ID: "team:manage", Name: "Manage Team", Resource: "team", Action: "manage", Description: "Manage team settings and members"},
	{ID: "team:invite", Name: "Invite Members", Resource: "team", Action: "invite", Description: "Invite new team members"},
}

// Default role permissions mapping
var DefaultRolePermissions = map[string][]string{
	RoleOwner: {
		"agent:create", "agent:read", "agent:update", "agent:delete", "agent:execute",
		"workflow:create", "workflow:read", "workflow:update", "workflow:delete", "workflow:execute",
		"knowledge:create", "knowledge:read", "knowledge:update", "knowledge:delete",
		"team:manage", "team:invite",
	},
	RoleAdmin: {
		"agent:create", "agent:read", "agent:update", "agent:delete", "agent:execute",
		"workflow:create", "workflow:read", "workflow:update", "workflow:delete", "workflow:execute",
		"knowledge:create", "knowledge:read", "knowledge:update", "knowledge:delete",
		"team:invite",
	},
	RoleMember: {
		"agent:create", "agent:read", "agent:update", "agent:execute",
		"workflow:create", "workflow:read", "workflow:update", "workflow:execute",
		"knowledge:create", "knowledge:read", "knowledge:update",
	},
	RoleGuest: {
		"agent:read", "agent:execute",
		"workflow:read", "workflow:execute",
		"knowledge:read",
	},
}