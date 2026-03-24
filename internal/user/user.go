// Copyright (C) 2026 luoxiaodou
// This file is part of GoPaw, licensed under the AGPL-3.0 License.
// See the LICENSE file in the project root for full license terms.

// Package user manages GoPaw users for team mode.
// In solo mode this package is not used.
package user

import (
	"time"
)

// Role defines the permission level of a user.
type Role string

const (
	// RoleAdmin can manage users, agents, and all system settings.
	RoleAdmin Role = "admin"
	// RoleMember has standard access to agents and personal sessions.
	RoleMember Role = "member"
)

// User represents a GoPaw account.
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email,omitempty"`
	Role         Role      `json:"role"`
	PasswordHash string    `json:"-"` // bcrypt hash, never serialised
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// IsAdmin reports whether the user has admin privileges.
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}
